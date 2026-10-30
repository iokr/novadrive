#!/usr/bin/env bash
# NovaDrive 构建脚本：构建前端 → 输出静态链接的单二进到 bin/novadrive。
# 用法： scripts/build.sh            # 当前平台
#        GOOS=linux scripts/build.sh  # 交叉编译
#        SKIP_WEB=1 scripts/build.sh  # 跳过前端构建，直接用现有 web/dist
set -euo pipefail

cd "$(dirname "$0")/.."

APP_NAME="novadrive"
OUTPUT_DIR="bin"
GOOS_TARGET="${GOOS:-$(go env GOOS)}"
GOARCH_TARGET="${GOARCH:-$(go env GOARCH)}"
VERSION="${VERSION:-$(git describe --tags --always --dirty 2>/dev/null || echo dev)}"

mkdir -p "${OUTPUT_DIR}"

if [ "${SKIP_WEB:-0}" != "1" ]; then
  if [ -d "frontend/node_modules" ]; then
    echo ">> 构建前端 → web/dist"
    ( cd frontend && npm run build )
  else
    echo ">> 跳过前端构建（未安装依赖，先执行 cd frontend && npm install）"
  fi
fi

echo ">> 编译 ${APP_NAME} ${VERSION} (${GOOS_TARGET}/${GOARCH_TARGET})"
CGO_ENABLED=0 GOOS="${GOOS_TARGET}" GOARCH="${GOARCH_TARGET}" \
  go build -trimpath -ldflags "-s -w" \
  -o "${OUTPUT_DIR}/${APP_NAME}" ./cmd/novadrive

echo ">> 构建完成: ${OUTPUT_DIR}/${APP_NAME}"
