#!/usr/bin/env bash
# Chronospp 一键安装脚本（Linux）
# 用法：
#   sudo ./install.sh             安装到 /usr/local/bin（系统级）
#   ./install.sh --user           安装到 ~/.local/bin（用户级，无需 root）
#   ./install.sh --prefix=/opt    安装到自定义目录
#   ./install.sh --force          跳过版本校验（安装旧二进制）
#
# 自动检测 CPU 架构（amd64/arm64），选择对应二进制安装并写入 PATH。
# 安装前会执行 `chpp version` 校验版本号，避免装上空版本号的旧构建。

set -euo pipefail

VERSION="1.3.0"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

PREFIX="/usr/local"
INSTALL_BIN_DIR="$PREFIX/bin"
FORCE=0

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
        --force|-f)
            FORCE=1
            ;;
        -h|--help)
            cat <<'EOF'
Chronospp 安装脚本

用法:
  sudo ./install.sh            安装到 /usr/local/bin（系统级，推荐）
  ./install.sh --user          安装到 ~/.local/bin（用户级，无需 root）
  ./install.sh --prefix=/opt   安装到自定义目录
  ./install.sh --force         跳过二进制版本校验

说明:
  自动检测 CPU 架构（amd64 / arm64），安装 chpp 并写入 PATH。
  安装前会运行 `chpp version` 校验版本号；若二进制是旧构建（版本号不匹配），
  脚本会拒绝安装并提示重新编译（make dist）或从 release 重新下载。
EOF
            exit 0
            ;;
    esac
done

# ---- 定位二进制目录：优先脚本同目录，其次 ../bin ----
if ls "$SCRIPT_DIR"/chpp-linux-* >/dev/null 2>&1; then
    BIN_DIR="$SCRIPT_DIR"
else
    BIN_DIR="$SCRIPT_DIR/../bin"
fi

# ---- 检测架构 ----
ARCH="$(uname -m)"
case "$ARCH" in
    x86_64|amd64)   BINARY="chpp-linux-amd64" ;;
    aarch64|arm64)  BINARY="chpp-linux-arm64" ;;
    *)
        echo "✗ 不支持的架构: $ARCH" >&2
        echo "  支持: x86_64 (amd64), aarch64 (arm64)" >&2
        exit 1
        ;;
esac

SRC="$BIN_DIR/$BINARY"
if [ ! -f "$SRC" ]; then
    echo "✗ 找不到二进制: $SRC" >&2
    echo "  源码安装：在项目根目录执行 make dist" >&2
    echo "  发布包：确保 install.sh 与 chpp-linux-* 在同一目录" >&2
    exit 1
fi

[ -x "$SRC" ] || chmod +x "$SRC" 2>/dev/null || true

# ---- 校验二进制版本（编译期注入，旧构建会显示 0.1.0 之类的旧值）----
EXPECT="Chronospp $VERSION"
ACTUAL="$("$SRC" version 2>/dev/null | head -n1 | tr -d '\r' || true)"
if [ "$ACTUAL" != "$EXPECT" ]; then
    echo "✗ 二进制版本不匹配，已中止安装" >&2
    echo "    期望: $EXPECT" >&2
    echo "    实际: ${ACTUAL:-（无法执行该文件）}" >&2
    echo "    文件: $SRC" >&2
    echo "" >&2
    if [ "$FORCE" != "1" ]; then
        echo "  原因：版本号在编译期由 -ldflags 注入，旧二进制不会自动变新。" >&2
        echo "  修复（源码）：在项目根目录执行 make dist" >&2
        echo "  修复（发布包）：重新下载 v$VERSION 的文件，别混用旧目录" >&2
        echo "  确认要装旧二进制：加 --force" >&2
        exit 1
    fi
    echo "  --force 已指定，忽略版本不符继续安装。" >&2
fi

echo "==> Chronospp $VERSION 安装"
echo "    架构: $ARCH"
echo "    源:   $SRC"
echo "    目标: $INSTALL_BIN_DIR"

mkdir -p "$INSTALL_BIN_DIR"

if [ -w "$INSTALL_BIN_DIR" ]; then
    cp "$SRC" "$INSTALL_BIN_DIR/chpp"
    chmod +x "$INSTALL_BIN_DIR/chpp"
else
    echo "==> 需要权限，尝试 sudo..."
    sudo cp "$SRC" "$INSTALL_BIN_DIR/chpp"
    sudo chmod +x "$INSTALL_BIN_DIR/chpp"
fi

# 用户级安装时写 PATH
if [ "$PREFIX" = "$HOME/.local" ]; then
    EXPORT_LINE='export PATH="$HOME/.local/bin:$PATH"'
    RC_FILE=""
    for f in "$HOME/.bashrc" "$HOME/.zshrc" "$HOME/.profile"; do
        if [ -f "$f" ]; then RC_FILE="$f"; break; fi
    done
    if [ -n "$RC_FILE" ] && ! grep -qF "$EXPORT_LINE" "$RC_FILE"; then
        printf '\n# Chronospp PATH\n%s\n' "$EXPORT_LINE" >> "$RC_FILE"
        echo "==> 已写入 PATH 到 $RC_FILE"
    fi
fi

INSTALLED="$("$INSTALL_BIN_DIR/chpp" version 2>/dev/null | head -n1 || true)"
if command -v chpp >/dev/null 2>&1; then
    echo ""
    echo "✓ 安装成功！位置: $(command -v chpp)"
else
    echo ""
    echo "✓ 已安装到 $INSTALL_BIN_DIR/chpp"
    echo "  ⚠ 请重新打开终端或执行: export PATH=\"$INSTALL_BIN_DIR:\$PATH\""
fi

echo "  版本: ${INSTALLED:-未知}"
if [ "$INSTALLED" != "$EXPECT" ]; then
    echo "  ⚠ 注意：装上的版本与脚本期望不符（$EXPECT）" >&2
fi

echo ""
echo "快速开始:"
echo "  chpp                       # 启动 REPL"
echo "  chpp run hello.chpp      # 运行脚本"
echo "  chpp build app.chpp      # 打包成单文件可执行"