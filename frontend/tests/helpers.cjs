const fs = require('node:fs');
const path = require('node:path');
const vm = require('node:vm');

// Load the application's small ES modules with explicit browser/network mocks.
// This keeps the test runner dependency-free and never contacts the backend.
function loadModule(relativePath, imports = {}, globals = {}) {
  const filename = path.resolve(__dirname, '..', 'src', relativePath);
  let source = fs.readFileSync(filename, 'utf8');
  if (relativePath.endsWith('.vue')) {
    const script = source.match(/<script[^>]*>([\s\S]*?)<\/script>/);
    if (!script) throw new Error('Vue component has no script: ' + relativePath);
    source = script[1];
  }
  const names = [];
  source = source.replace(/^[ \t]*import\s+([\s\S]*?)\s+from\s+['"]([^'"]+)['"]\s*;?/gm,
    (_, binding, specifier) => {
      const lookup = 'require(' + JSON.stringify(specifier) + ')';
      if (binding.trim().startsWith('{')) {
        return 'const ' + binding.replace(/\bas\b/g, ':') + ' = ' + lookup + ';';
      }
      return 'const ' + binding.trim() + ' = ' + lookup + '.default;';
    });
  source = source.replace(/\bexport\s+((?:async\s+)?(?:function|const|let|class))\s+(\w+)/g,
    (_, declaration, name) => {
      names.push(name);
      return declaration + ' ' + name;
    });
  source = source.replace(/\bexport\s+default\s+/g, 'module.exports.default = ');
  source = source.replace(/\bexport\s*\{([^}]+)\}\s*;?/g, (_, list) => list.split(',').map(item => {
    const [name, exported] = item.trim().split(/\s+as\s+/);
    return 'module.exports.' + (exported || name) + ' = ' + name + ';';
  }).join('\n'));
  source += '\n' + names.map(name => 'module.exports.' + name + ' = ' + name + ';').join('\n');
  const module = { exports: {} };
  const context = {
    module,
    console,
    Promise,
    setTimeout,
    clearTimeout,
    ...globals,
    require(specifier) {
      if (!Object.hasOwn(imports, specifier)) {
        throw new Error('Unexpected module dependency: ' + specifier);
      }
      return imports[specifier];
    }
  };
  vm.runInNewContext(source, context, { filename });
  return module.exports;
}

function makeBrowser(initialStorage = {}) {
  const values = new Map(Object.entries(initialStorage));
  const listeners = new Map();
  const localStorage = {
    getItem: key => values.has(key) ? values.get(key) : null,
    setItem: (key, value) => values.set(key, String(value)),
    removeItem: key => values.delete(key)
  };
  const window = {
    localStorage,
    addEventListener(type, handler) {
      if (!listeners.has(type)) listeners.set(type, []);
      listeners.get(type).push(handler);
    },
    removeEventListener(type, handler) {
      listeners.set(type, (listeners.get(type) || []).filter(item => item !== handler));
    },
    dispatchEvent(event) {
      (listeners.get(event.type) || []).forEach(handler => handler(event));
      return true;
    }
  };
  return { window, localStorage };
}

function makeAxios(adapter) {
  const requests = [];
  const instances = [];
  const axios = {
    create(defaults = {}) {
      const interceptors = { request: [], response: [] };
      const instance = config => instance.request(config);
      instance.defaults = defaults;
      instance.interceptors = Object.fromEntries(['request', 'response'].map(kind => [kind, {
        use(fulfilled, rejected) {
          interceptors[kind].push({ fulfilled, rejected });
          return interceptors[kind].length - 1;
        }
      }]));
      instance.request = config => {
        const actual = { ...defaults, ...config, headers: { ...defaults.headers, ...config.headers } };
        let result = Promise.resolve(actual);
        for (const handler of [...interceptors.request].reverse()) {
          result = result.then(handler.fulfilled, handler.rejected);
        }
        result = result.then(async request => {
          requests.push({ ...request, headers: { ...request.headers } });
          const data = await adapter(request);
          return { status: 200, config: request, data };
        });
        for (const handler of interceptors.response) {
          result = result.then(handler.fulfilled, handler.rejected);
        }
        return result;
      };
      instance.get = (url, config = {}) => instance.request({ ...config, method: 'get', url });
      instance.post = (url, data, config = {}) => instance.request({ ...config, method: 'post', url, data });
      instances.push(instance);
      return instance;
    }
  };
  return { axios, requests, instances };
}

const loginResult = {
  userID: '9223372036854775806',
  username: 'zed',
  accessTokne: 'access-old',
  refreshToken: 'refresh-old'
};

function setup(adapter = async () => ({ code: 1000, data: {} }), stored = {}) {
  const browser = makeBrowser(stored);
  const session = loadModule('session.js', {}, browser);
  const network = makeAxios(adapter);
  const api = loadModule('service/api.js', {
    axios: { default: network.axios },
    '../session': session,
    './session': session,
    './session.js': session
  }, browser).default;
  return { ...browser, ...network, session, api };
}

const plain = value => JSON.parse(JSON.stringify(value));
const tick = () => new Promise(resolve => setImmediate(resolve));
function deferred() {
  let resolve;
  let reject;
  const promise = new Promise((accept, fail) => { resolve = accept; reject = fail; });
  return { promise, resolve, reject };
}

module.exports = { loadModule, makeBrowser, loginResult, setup, plain, tick, deferred };
