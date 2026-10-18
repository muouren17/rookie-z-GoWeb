#!/bin/sh
# 仅从文档注释生成 JSON/YAML，不生成 docs.go，不引入运行时依赖。
set -eu
DOC_DIR=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
PROJECT_DIR=$(CDPATH= cd -- "$DOC_DIR/../.." && pwd)
cd "$PROJECT_DIR"
if [ -x /usr/local/go/bin/go ]; then
    PATH="/usr/local/go/bin:$PATH"
    export PATH
fi
if command -v swag >/dev/null 2>&1; then
    SWAG_COMMAND=$(command -v swag)
elif [ -x "$HOME/.local/bin/swag" ]; then
    SWAG_COMMAND="$HOME/.local/bin/swag"
else
    printf '%s\n' '找不到 swag，请先安装 swaggo/swag CLI。' >&2
    exit 1
fi
"$SWAG_COMMAND" init -g main.go -o docs/swagger --outputTypes json,yaml --exclude docs,tmp,logs,vendor
# 兼容之前的文档下载地址，文件内容与注释生成的 Swagger 2.0 完全一致。
cp "$DOC_DIR/swagger.json" "$DOC_DIR/openapi.json"
cp "$DOC_DIR/swagger.yaml" "$DOC_DIR/openapi.yaml"
