//go:build !js

package main

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"chronos/lexer"
	"chronos/parser"
)

// 可执行文件尾部布局（payload 区域 = base 之后全部内容）：
//
//	明文模式: [脚本源码][8 字节小端 payload 长度][16 字节 magic]
//	加密模式: [盐(16)][AES-256-CTR 密文][8 字节小端 payload 长度][16 字节 magic]
//
// 加密模式下源码在二进制里是密文，strings / hexdump 无法直接读出脚本内容。
var (
	magicPlain = []byte("CHRONOS_PAYLOAD_V1") // 18 字节：明文
	magicEnc   = []byte("CHRONOS_PAYLOAD_E1") // 18 字节：AES-CTR 加密
)

const (
	magicLen = 18
	sizeLen  = 8
	saltLen  = 16
)

// payloadSecret 参与密钥派生。注意：它必须随二进制分发，
// 因此这层保护属于「混淆」而非「密码学安全」。
var payloadSecret = []byte("Chronos::payload::v1::2026")

// deriveKey 由盐派生 32 字节 AES-256 密钥
func deriveKey(salt []byte) []byte {
	h := sha256.New()
	h.Write(payloadSecret)
	h.Write(salt)
	h.Write([]byte("key"))
	return h.Sum(nil)
}

// deriveIV 由盐派生 16 字节初始计数器
func deriveIV(salt []byte) []byte {
	h := sha256.New()
	h.Write(payloadSecret)
	h.Write(salt)
	h.Write([]byte("iv"))
	return h.Sum(nil)[:aes.BlockSize]
}

// encryptPayload 加密脚本，返回 [盐][密文]
func encryptPayload(src []byte) ([]byte, error) {
	salt := make([]byte, saltLen)
	if _, err := rand.Read(salt); err != nil {
		return nil, fmt.Errorf("生成随机盐失败: %w", err)
	}
	block, err := aes.NewCipher(deriveKey(salt))
	if err != nil {
		return nil, err
	}
	ct := make([]byte, len(src))
	cipher.NewCTR(block, deriveIV(salt)).XORKeyStream(ct, src)

	out := make([]byte, 0, saltLen+len(ct))
	out = append(out, salt...)
	out = append(out, ct...)
	return out, nil
}

// decryptPayload 解密 [盐][密文]
func decryptPayload(blob []byte) ([]byte, error) {
	if len(blob) < saltLen {
		return nil, errors.New("payload 长度不足")
	}
	salt := blob[:saltLen]
	ct := blob[saltLen:]
	block, err := aes.NewCipher(deriveKey(salt))
	if err != nil {
		return nil, err
	}
	pt := make([]byte, len(ct))
	cipher.NewCTR(block, deriveIV(salt)).XORKeyStream(pt, ct)
	return pt, nil
}

// splitPayload 把二进制切成「解释器本体」和「payload」。
// encrypted 表示 payload 是否为密文；ok 表示是否确实带有 payload。
func splitPayload(bin []byte) (base, payload []byte, encrypted, ok bool) {
	if len(bin) < magicLen+sizeLen {
		return bin, nil, false, false
	}
	tail := bin[len(bin)-magicLen:]
	switch {
	case bytes.Equal(tail, magicPlain):
		encrypted = false
	case bytes.Equal(tail, magicEnc):
		encrypted = true
	default:
		return bin, nil, false, false
	}
	sizePos := len(bin) - magicLen - sizeLen
	n := binary.LittleEndian.Uint64(bin[sizePos : sizePos+sizeLen])
	if n == 0 || n > uint64(sizePos) {
		return bin, nil, false, false
	}
	start := sizePos - int(n)
	return bin[:start], bin[start:sizePos], encrypted, true
}

// selfPayload 读取当前可执行文件里嵌入的脚本（自动解密）。
func selfPayload() ([]byte, bool) {
	exe, err := os.Executable()
	if err != nil {
		return nil, false
	}
	bin, err := os.ReadFile(exe)
	if err != nil {
		return nil, false
	}
	_, payload, encrypted, ok := splitPayload(bin)
	if !ok {
		return nil, false
	}
	if !encrypted {
		return payload, true
	}
	script, err := decryptPayload(payload)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: 内嵌脚本解密失败: %v\n", err)
		os.Exit(1)
	}
	return script, true
}

// runEmbedded 执行编译进二进制的脚本。
// import 的相对路径基准为可执行文件所在目录。
func runEmbedded(script []byte) int {
	dir := "."
	if exe, err := os.Executable(); err == nil {
		dir = filepath.Dir(exe)
	}
	if err := runSource(string(script), os.Stdout, dir); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		return 1
	}
	return 0
}

// cmdBuild 实现 chr build <file.chr> [-o out] [--plain]：
// 把脚本嵌入解释器副本，产出一个独立可执行文件（默认加密混淆）。
func cmdBuild(args []string, stdout, stderr io.Writer) int {
	var outPath, srcPath string
	encrypt := true // 默认开启混淆加密

	for i := 0; i < len(args); i++ {
		a := args[i]
		switch {
		case a == "-o" || a == "--output":
			if i+1 >= len(args) {
				fmt.Fprintln(stderr, "error: -o 需要跟输出文件名")
				return 2
			}
			i++
			outPath = args[i]
		case strings.HasPrefix(a, "-o="):
			outPath = strings.TrimPrefix(a, "-o=")
		case strings.HasPrefix(a, "--output="):
			outPath = strings.TrimPrefix(a, "--output=")
		case a == "--plain" || a == "--no-encrypt" || a == "-p":
			encrypt = false
		case a == "--encrypt" || a == "-e":
			encrypt = true
		case a == "-h" || a == "--help":
			fmt.Fprintln(stdout, `用法: chr build <file.chr> [-o 输出文件名] [--plain]

  -o, --output <name>   指定输出文件名（默认取脚本名）
  --plain, -p           不做加密混淆，直接嵌入明文脚本（便于调试）
  --encrypt, -e         加密混淆（默认）`)
			return 0
		case strings.HasPrefix(a, "-"):
			fmt.Fprintf(stderr, "error: 未知选项 %s\n", a)
			return 2
		default:
			if srcPath != "" {
				fmt.Fprintln(stderr, "error: build 一次只能编译一个脚本")
				return 2
			}
			srcPath = a
		}
	}

	if srcPath == "" {
		fmt.Fprintln(stderr, "error: 缺少输入文件（用法: chr build <file.chr> [-o 输出]）")
		return 2
	}

	src, err := os.ReadFile(srcPath)
	if err != nil {
		fmt.Fprintf(stderr, "error: 读取 %s 失败: %v\n", srcPath, err)
		return 1
	}

	// 编译前语法检查，避免产出「一跑就报错」的二进制
	if perrs := parseErrors(string(src)); len(perrs) > 0 {
		for _, e := range perrs {
			fmt.Fprintf(stderr, "语法错误: %s\n", e)
		}
		fmt.Fprintf(stderr, "error: %s 存在 %d 处语法错误，已中止\n", srcPath, len(perrs))
		return 1
	}

	// 组装 payload
	var payload []byte
	magic := magicPlain
	if encrypt {
		payload, err = encryptPayload(src)
		if err != nil {
			fmt.Fprintf(stderr, "error: 加密失败: %v\n", err)
			return 1
		}
		magic = magicEnc
	} else {
		payload = src
	}

	// 读取自身二进制，剥掉可能已存在的旧 payload
	self, err := os.Executable()
	if err != nil {
		fmt.Fprintf(stderr, "error: 无法定位当前可执行文件: %v\n", err)
		return 1
	}
	bin, err := os.ReadFile(self)
	if err != nil {
		fmt.Fprintf(stderr, "error: 读取自身失败: %v\n", err)
		return 1
	}
	base, _, _, _ := splitPayload(bin)

	// 默认输出名：脚本文件名去掉扩展名
	if outPath == "" {
		outPath = strings.TrimSuffix(filepath.Base(srcPath), filepath.Ext(srcPath))
		if outPath == "" {
			outPath = "a.out"
		}
		if runtime.GOOS == "windows" && !strings.HasSuffix(strings.ToLower(outPath), ".exe") {
			outPath += ".exe"
		}
	}

	// 拼接：[解释器][payload][长度][magic]
	var buf bytes.Buffer
	buf.Write(base)
	buf.Write(payload)
	var sizeBuf [sizeLen]byte
	binary.LittleEndian.PutUint64(sizeBuf[:], uint64(len(payload)))
	buf.Write(sizeBuf[:])
	buf.Write(magic)

	if err := os.WriteFile(outPath, buf.Bytes(), 0o755); err != nil {
		fmt.Fprintf(stderr, "error: 写出 %s 失败: %v\n", outPath, err)
		return 1
	}
	_ = os.Chmod(outPath, 0o755)

	mode := "加密混淆 (AES-256-CTR)"
	if !encrypt {
		mode = "明文"
	}
	abs, _ := filepath.Abs(outPath)
	fmt.Fprintf(stdout, "✓ 已编译: %s\n", outPath)
	fmt.Fprintf(stdout, "  源文件: %s (%d 字节)\n", srcPath, len(src))
	fmt.Fprintf(stdout, "  模式:   %s\n", mode)
	fmt.Fprintf(stdout, "  产物:   %s (%d 字节)\n", abs, buf.Len())
	fmt.Fprintf(stdout, "  运行:   ./%s\n", filepath.Base(outPath))
	return 0
}

// parseErrors 只做词法+语法检查，返回错误列表。
func parseErrors(src string) []string {
	l := lexer.New(src)
	p := parser.New(l)
	p.ParseProgram()
	return p.Errors()
}
