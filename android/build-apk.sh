#!/usr/bin/env bash
# Chronos IDE —— 手动构建 APK（无需 Gradle）
#
# 用法:
#   ./build-apk.sh [输出路径]          构建完整版（含 template.apk + keystore）
#   ./build-apk.sh --bare [输出路径]   构建裸版（作为 template.apk 的源）
#
# 依赖: Android SDK(build-tools 34.0.0, platforms android-34)、JDK、
#       qemu-x86_64-static（aarch64 主机上运行 x86-64 的 aapt2/zipalign）
set -euo pipefail

HERE="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SDK="${ANDROID_HOME:-$HOME/Android/Sdk}"
BT="$SDK/build-tools/34.0.0"
PLAT="$SDK/platforms/android-34/android.jar"
APKSIG="$BT/lib/apksigner.jar"
SRC="$HERE/app/src/main"
BUILD=/tmp/abuild

BARE=0
TEMPLATE=0
if [ "${1:-}" = "--bare" ]; then BARE=1; shift; fi
if [ "${1:-}" = "--template" ]; then TEMPLATE=1; BARE=1; shift; fi
OUT="${1:-$HERE/chronos-ide.apk}"

QEMU=""
if [ "$(uname -m)" != "x86_64" ]; then
    QEMU="qemu-x86_64-static -L /tmp/x64"
    if [ ! -e /tmp/x64/lib64/ld-linux-x86-64.so.2 ]; then
        echo "错误：缺少 x86-64 glibc（/tmp/x64），无法运行 aapt2。" >&2
        exit 1
    fi
fi
run() { if [ -n "$QEMU" ]; then $QEMU "$@"; else "$@"; fi; }

rm -rf "$BUILD"
mkdir -p "$BUILD/gen" "$BUILD/classes" "$BUILD/dex"

echo "==> 1/6 编译资源"
run "$BT/aapt2" compile --dir "$SRC/res" -o "$BUILD/res.zip"

echo "==> 2/6 链接资源"
MANIFEST="$SRC/AndroidManifest.xml"
if [ "$TEMPLATE" = 1 ]; then
    MANIFEST="$HERE/template/AndroidManifest.xml"
fi

run "$BT/aapt2" link -o "$BUILD/base.apk" -I "$PLAT" \
    --manifest "$MANIFEST" \
    -R "$BUILD/res.zip" --java "$BUILD/gen" --auto-add-overlay

echo "==> 3/6 编译 Java"
# R.java 的包名取决于 manifest 的 package（模板是 com.chronos.script），用 find 定位
find "$BUILD/gen" -name "R.java" > "$BUILD/rjava.txt"
if [ ! -s "$BUILD/rjava.txt" ]; then
    echo "错误：aapt2 未生成 R.java" >&2
    exit 1
fi
javac -source 8 -target 8 -bootclasspath "$PLAT" -classpath "$PLAT:$APKSIG" \
    -d "$BUILD/classes" \
    @"$BUILD/rjava.txt" \
    "$SRC"/java/com/chronos/ide/*.java

echo "==> 4/6 转 dex（含 apksig 签名库）"
find "$BUILD/classes" -name "*.class" > "$BUILD/classes.txt"
"$BT/d8" --lib "$PLAT" --min-api 26 --output "$BUILD/dex" \
    @"$BUILD/classes.txt" "$APKSIG"

echo "==> 5/6 组装 APK"
python3 - "$BUILD" "$SRC/assets" "$BARE" <<'PY'
import sys, os, shutil, zipfile
build, assets, bare = sys.argv[1], sys.argv[2], sys.argv[3] == "1"
out = os.path.join(build, "out.apk")
shutil.copy(os.path.join(build, "base.apk"), out)
skip = {"template.apk", "chronos.keystore"} if bare else set()
if bare:
    # 模板/裸版不需要 sh 编译内核（只有 IDE 需要）
    skip.add("chronos-linux-arm64")
with zipfile.ZipFile(out, "a", zipfile.ZIP_DEFLATED) as z:
    z.write(os.path.join(build, "dex", "classes.dex"), "classes.dex")
    for root, _, files in os.walk(assets):
        for f in files:
            if f in skip:
                continue
            p = os.path.join(root, f)
            z.write(p, "assets/" + os.path.relpath(p, assets).replace(os.sep, "/"))
PY

echo "==> 6/6 对齐 + 签名"
run "$BT/zipalign" -f 4 "$BUILD/out.apk" "$BUILD/aligned.apk"

KS="$HERE/chronos.keystore"
if [ ! -f "$KS" ]; then
    keytool -genkeypair -keystore "$KS" -alias chronos -keyalg RSA -keysize 2048 \
        -storetype PKCS12 -validity 10000 -storepass android -keypass android \
        -dname "CN=Chronos IDE, OU=Dev, O=Chronos, L=CN, ST=CN, C=CN" >/dev/null 2>&1
fi
"$BT/apksigner" sign --ks "$KS" --ks-pass pass:android --key-pass pass:android \
    --out "$OUT" "$BUILD/aligned.apk"

echo ""
if [ "$TEMPLATE" = 1 ]; then MODE="模板版(com.chronos.script)"
elif [ "$BARE" = 1 ]; then MODE="裸版"
else MODE="完整版(com.chronos.ide)"; fi
echo "✓ 构建完成: $OUT ($MODE)"
ls -lh "$OUT"
