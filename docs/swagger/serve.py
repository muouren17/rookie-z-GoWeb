#!/usr/bin/env python3
"""Serve the local Swagger UI and proxy the documented API using only Python 3."""

import sys

sys.dont_write_bytecode = True

import argparse
import http.client
import json
import re
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer
from pathlib import Path
from urllib.parse import urlsplit


MAX_BODY_BYTES = 16 * 1024 * 1024
TIMEOUT_SECONDS = 30
DOCUMENT_ROOT = Path(__file__).resolve().parent
STATIC_FILES = {
    "/": ("index.html", "text/html; charset=utf-8"),
    "/index.html": ("index.html", "text/html; charset=utf-8"),
    "/init.js": ("init.js", "text/javascript; charset=utf-8"),
    "/openapi.json": ("openapi.json", "application/json; charset=utf-8"),
    "/openapi.yaml": ("openapi.yaml", "application/yaml; charset=utf-8"),
    "/swagger.json": ("swagger.json", "application/json; charset=utf-8"),
    "/swagger.yaml": ("swagger.yaml", "application/yaml; charset=utf-8"),
    "/vendor/swagger-ui-bundle.js": (
        "vendor/swagger-ui-bundle.js", "text/javascript; charset=utf-8"
    ),
    "/vendor/swagger-ui.css": ("vendor/swagger-ui.css", "text/css; charset=utf-8"),
    "/vendor/LICENSE.txt": ("vendor/LICENSE.txt", "text/plain; charset=utf-8"),
}
GET_ROUTES = {"/api/v1/community", "/api/v1/posts", "/api/v1/posts2"}
POST_ROUTES = {
    "/api/v1/signup", "/api/v1/login", "/api/v1/post", "/api/v1/vote", "/refresh"
}
ID_ROUTE = re.compile(r"/api/v1/(?:community|post)/[^/]+\Z")


def backend_origin(value):
    """An origin keeps the proxy destination independent of the request path."""
    try:
        parsed = urlsplit(value)
        valid = (
            parsed.scheme in ("http", "https")
            and parsed.hostname
            and parsed.username is None
            and parsed.password is None
            and parsed.path in ("", "/")
            and "?" not in value
            and "#" not in value
            and not any(character.isspace() for character in value)
        )
        port = parsed.port
        if not valid or (port is not None and not 1 <= port <= 65535):
            raise ValueError
        return parsed
    except ValueError:
        raise argparse.ArgumentTypeError(
            "backend must be an http:// or https:// origin with no credentials, "
            "path, query, or fragment (for example http://127.0.0.1:8081)"
        ) from None


def listen_port(value):
    try:
        port = int(value)
        if 1 <= port <= 65535:
            return port
    except ValueError:
        pass
    raise argparse.ArgumentTypeError("port must be between 1 and 65535")


class DocsServer(ThreadingHTTPServer):
    daemon_threads = True

    def __init__(self, port, backend):
        self.backend = backend
        super().__init__(("127.0.0.1", port), DocsHandler)


class DocsHandler(BaseHTTPRequestHandler):
    server_version = "SwaggerDocs"
    sys_version = ""

    def setup(self):
        super().setup()
        self.connection.settimeout(TIMEOUT_SECONDS)

    def log_message(self, format, *args):
        # Access logs could expose values in query strings; startup output is enough.
        pass

    def _send(self, status, body, content_type, extra_headers=None, head=False):
        self.close_connection = True
        try:
            self.send_response(status)
            self.send_header("Content-Type", content_type)
            self.send_header("Content-Length", str(len(body)))
            self.send_header("Cache-Control", "no-store")
            self.send_header("Connection", "close")
            for name, value in (extra_headers or {}).items():
                self.send_header(name, value)
            self.end_headers()
            if not head:
                self.wfile.write(body)
        except (OSError, TimeoutError):
            # Closing a browser tab must not produce a traceback or leak a request.
            pass

    def _error(self, status, message, extra_headers=None):
        body = (json.dumps({"error": message}) + "\n").encode("utf-8")
        self._send(
            status, body, "application/json; charset=utf-8", extra_headers,
            head=self.command == "HEAD",
        )

    def _request_body(self):
        if self.headers.get("Transfer-Encoding") is not None:
            self._error(400, "Transfer-Encoding is unsupported; send Content-Length.")
            return None
        lengths = self.headers.get_all("Content-Length", [])
        if len(lengths) > 1 or (lengths and not re.fullmatch(r"[0-9]+", lengths[0])):
            self._error(400, "Invalid Content-Length.")
            return None
        try:
            length = int(lengths[0]) if lengths else 0
        except ValueError:
            self._error(400, "Invalid Content-Length.")
            return None
        if length > MAX_BODY_BYTES:
            self._error(413, "Request body exceeds the 16 MiB limit.")
            return None
        try:
            body = self.rfile.read(length)
        except (OSError, TimeoutError):
            self._error(400, "Could not read the complete request body.")
            return None
        if len(body) != length:
            self._error(400, "Incomplete request body.")
            return None
        return body

    def _proxy(self):
        body = self._request_body()
        if body is None:
            return
        backend = self.server.backend
        connection_class = (
            http.client.HTTPSConnection if backend.scheme == "https"
            else http.client.HTTPConnection
        )
        connection = connection_class(
            backend.hostname, backend.port, timeout=TIMEOUT_SECONDS
        )
        headers = {}
        for name in ("Authorization", "Content-Type"):
            value = self.headers.get(name)
            if value is not None:
                headers[name] = value
        try:
            # http.client does not follow redirects. Preserve the original query
            # and escaped path, and forward JSON bytes without parsing them.
            connection.request(self.command, self.path, body=body, headers=headers)
            response = connection.getresponse()
            response_body = response.read()
            status = response.status
            content_type = response.getheader("Content-Type", "application/octet-stream")
        except (OSError, TimeoutError, http.client.HTTPException):
            self._error(
                502, "Backend request failed. Start the Go API and check --backend."
            )
            return
        finally:
            connection.close()
        self._send(status, response_body, content_type)

    def _dispatch(self):
        try:
            target = urlsplit(self.path)
        except ValueError:
            self._error(400, "Invalid request target.")
            return
        if not self.path.startswith("/") or target.netloc or target.scheme or target.fragment:
            self._error(400, "Use a local path without a fragment.")
            return
        path = target.path
        allowed = set()
        if path in GET_ROUTES or ID_ROUTE.fullmatch(path):
            allowed.add("GET")
        if path in POST_ROUTES:
            allowed.add("POST")
        if allowed:
            if self.command not in allowed:
                self._error(405, "Method not allowed.", {"Allow": ", ".join(sorted(allowed))})
                return
            self._proxy()
            return
        asset = STATIC_FILES.get(path)
        if asset is None:
            self._error(404, "No documented API route or static file at this path.")
            return
        if self.command not in ("GET", "HEAD"):
            self._error(405, "Method not allowed.", {"Allow": "GET, HEAD"})
            return
        filename, content_type = asset
        try:
            body = (DOCUMENT_ROOT / filename).read_bytes()
        except OSError:
            self._error(404, "Documentation file is missing.")
            return
        self._send(200, body, content_type, head=self.command == "HEAD")

    do_GET = _dispatch
    do_HEAD = _dispatch
    do_POST = _dispatch
    do_PUT = _dispatch
    do_PATCH = _dispatch
    do_DELETE = _dispatch
    do_OPTIONS = _dispatch


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--port", type=listen_port, default=8090, help="local docs port (default: 8090)")
    parser.add_argument(
        "--backend", type=backend_origin, default="http://127.0.0.1:8081",
        help="Go API origin (default: http://127.0.0.1:8081)",
    )
    args = parser.parse_args()
    try:
        with DocsServer(args.port, args.backend) as server:
            print("Swagger UI: http://127.0.0.1:{}/".format(args.port), flush=True)
            print("API backend: {}".format(args.backend.geturl().rstrip("/")), flush=True)
            print("Press Ctrl+C to stop.", flush=True)
            try:
                server.serve_forever(poll_interval=0.25)
            except KeyboardInterrupt:
                print("\nStopped.", flush=True)
    except OSError as error:
        parser.exit(1, "Cannot start the local documentation server: {}\n".format(error))


if __name__ == "__main__":
    main()
