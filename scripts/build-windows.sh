#!/usr/bin/env bash
# 生成 Windows 分发包：MSI 安装包 + 便携 ZIP
#
# 依赖：
#   - wixl（msitools 的 wixl 包）用于生成 MSI
#   - zip
#   - 若主机是 aarch64，wixl 需通过 qemu-x86_64-static + /tmp/x64 运行
#
# 用法: ./scripts/build-windows.sh [输出目录]
set -euo pipefail

HERE="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
OUT="${1:-$HERE/dist}"
VERSION="$(grep -E '^VERSION *:=' "$HERE/Makefile" | sed 's/.*:= *//')"
WXS="$HERE/scripts/msi/chronos.wxs"

mkdir -p "$OUT"

# 定位 wixl（可能在 PATH，也可能是手动解压的 msitools）
WIXL="${WIXL:-$(command -v wixl || true)}"
if [ -z "$WIXL" ]; then
    for cand in /tmp/msitools-x/root/usr/bin/wixl "$HOME/.local/opt/msitools/usr/bin/wixl"; do
        [ -x "$cand" ] && WIXL="$cand" && break
    done
fi
if [ -z "$WIXL" ]; then
    echo "错误：找不到 wixl。" >&2
    echo "  Ubuntu: sudo apt install wixl" >&2
    echo "  或设置 WIXL=/path/to/wixl" >&2
    exit 1
fi

# aarch64 主机上需要 qemu 跑 x86-64 的 wixl
RUN=""
if [ "$(uname -m)" != "x86_64" ] && head -c4 "$WIXL" 2>/dev/null | grep -q $'\x7fELF'; then
    if [ -e /tmp/x64/lib64/ld-linux-x86-64.so.2 ]; then
        RUN="qemu-x86_64-static -L /tmp/x64"
    fi
fi

for ARCH in amd64 arm64; do
    EXE="$HERE/bin/chronos-windows-$ARCH.exe"
    if [ ! -f "$EXE" ]; then
        echo "跳过 $ARCH：缺少 $EXE（先跑 make dist）" >&2
        continue
    fi

    echo "==> $ARCH: MSI"
    $RUN "$WIXL" -D "source=$EXE" -D "version=$VERSION" \
        -o "$OUT/chronos-$VERSION-windows-$ARCH.msi" "$WXS"

    echo "==> $ARCH: 便携 ZIP"
    TMP="$(mktemp -d)"
    cp "$EXE" "$TMP/chronos.exe"
    cp "$HERE/README.md" "$HERE/LICENSE" "$TMP/" 2>/dev/null || true
    mkdir -p "$TMP/examples"
    cp "$HERE"/examples/*.chr "$TMP/examples/" 2>/dev/null || true
    ( cd "$TMP" && zip -q -r "$OUT/chronos-$VERSION-windows-$ARCH.zip" . )
    rm -rf "$TMP"
done

echo ""
ls -lh "$OUT"/chronos-"$VERSION"-windows-*.msi "$OUT"/chronos-"$VERSION"-windows-*.zip 2>/dev/null
