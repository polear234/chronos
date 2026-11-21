#!/usr/bin/env bash
# 交叉编译 PicoX 为多平台独立二进制
# 用法: ./scripts/build-all.sh
set -euo pipefail

cd "$(dirname "$0")/.."

VERSION="0.1.0"
OUT="dist"
LDFLAGS="-s -w -X main.version=${VERSION}"

mkdir -p "$OUT"

targets=(
  "linux/amd64"
  "linux/arm64"
  "darwin/amd64"
  "darwin/arm64"
  "windows/amd64"
)

for t in "${targets[@]}"; do
  goos="${t%/*}"
  goarch="${t#*/}"
  outname="picox-${goos}-${goarch}"
  if [ "$goos" = "windows" ]; then
    outname="${outname}.exe"
  fi
  echo ">>> 编译 ${goos}/${goarch} -> ${OUT}/${outname}"
  CGO_ENABLED=0 GOOS="$goos" GOARCH="$goarch" \
    go build -trimpath -ldflags "$LDFLAGS" -o "${OUT}/${outname}" .
done

echo ""
echo "=== 完成 ==="
ls -lh "$OUT"
