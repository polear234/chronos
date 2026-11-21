#!/usr/bin/env bash
# PicoX 一键安装脚本（Linux）
# 用法：sudo ./install.sh          （安装到 /usr/local/bin，系统级）
#       ./install.sh --user        （安装到 ~/.local/bin，用户级，无需 root）
#       ./install.sh --prefix=/opt （自定义目录）
#
# 自动检测 CPU 架构（amd64/arm64），选择对应二进制，
# 安装到目标目录并写入 PATH。

set -euo pipefail

VERSION="0.1.0"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
BIN_DIR="$SCRIPT_DIR/../bin"

PREFIX="/usr/local"
INSTALL_BIN_DIR="$PREFIX/bin"

for arg in "$@"; do
    case "$arg" in
        --user|-u)
            PREFIX="$HOME/.local"
            INSTALL_BIN_DIR="$PREFIX/bin"
            ;;
        --prefix=*)
            PREFIX="${arg#*=}"
            INSTALL_BIN_DIR="$PREFIX/bin"
            ;;
        -h|--help)
            cat <<'EOF'
PicoX 安装脚本

用法:
  sudo ./install.sh            安装到 /usr/local/bin（系统级，推荐）
  ./install.sh --user          安装到 ~/.local/bin（用户级，无需 root）
  ./install.sh --prefix=/opt   安装到自定义目录

说明:
  自动检测 CPU 架构（amd64 / arm64），安装 picox 并写入 PATH。
EOF
            exit 0
            ;;
    esac
done

# ---- 检测架构 ----
ARCH="$(uname -m)"
case "$ARCH" in
    x86_64|amd64)   BINARY="picox-linux-amd64" ;;
    aarch64|arm64)  BINARY="picox-linux-arm64" ;;
    *)
        echo "✗ 不支持的架构: $ARCH" >&2
        echo "  支持: x86_64 (amd64), aarch64 (arm64)" >&2
        exit 1
        ;;
esac

SRC="$BIN_DIR/$BINARY"
if [ ! -f "$SRC" ]; then
    echo "✗ 找不到二进制: $SRC" >&2
    echo "  请先在项目根目录执行: make dist" >&2
    exit 1
fi

echo "==> PicoX $VERSION 安装"
echo "    架构: $ARCH"
echo "    目标: $INSTALL_BIN_DIR"

mkdir -p "$INSTALL_BIN_DIR"

if [ -w "$INSTALL_BIN_DIR" ]; then
    cp "$SRC" "$INSTALL_BIN_DIR/picox"
    chmod +x "$INSTALL_BIN_DIR/picox"
else
    echo "==> 需要权限，尝试 sudo..."
    sudo cp "$SRC" "$INSTALL_BIN_DIR/picox"
    sudo chmod +x "$INSTALL_BIN_DIR/picox"
fi

# 用户级安装时写 PATH
if [ "$PREFIX" = "$HOME/.local" ]; then
    EXPORT_LINE='export PATH="$HOME/.local/bin:$PATH"'
    RC_FILE=""
    for f in "$HOME/.bashrc" "$HOME/.zshrc" "$HOME/.profile"; do
        if [ -f "$f" ]; then RC_FILE="$f"; break; fi
    done
    if [ -n "$RC_FILE" ] && ! grep -qF "$EXPORT_LINE" "$RC_FILE"; then
        printf '\n# PicoX PATH\n%s\n' "$EXPORT_LINE" >> "$RC_FILE"
        echo "==> 已写入 PATH 到 $RC_FILE"
    fi
fi

if command -v picox >/dev/null 2>&1; then
    echo ""
    echo "✓ 安装成功！位置: $(command -v picox)"
    "$INSTALL_BIN_DIR/picox" version 2>/dev/null || true
else
    echo ""
    echo "✓ 已安装到 $INSTALL_BIN_DIR/picox"
    echo "  ⚠ 请重新打开终端或执行: export PATH=\"$INSTALL_BIN_DIR:\$PATH\""
fi

echo ""
echo "快速开始:"
echo "  picox                   # 启动 REPL"
echo "  picox -run hello.px     # 运行脚本"
