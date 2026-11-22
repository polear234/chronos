package interpreter

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"
)

// outputWriter 是 print/println 的目标。
// 默认标准输出；在 WASM（浏览器/安卓 WebView）环境中由宿主重定向到缓冲区。
var outputWriter io.Writer = os.Stdout

// SetOutput 重定向 print 输出，返回旧值以便恢复。
func SetOutput(w io.Writer) io.Writer {
	old := outputWriter
	outputWriter = w
	return old
}

// GetOutput 返回当前输出目标。
func GetOutput() io.Writer { return outputWriter }

// ── 输入 ─────────────────────────────────────────────

// inputFunc 是可替换的输入源：CLI 用 stdin，WASM 用 JS prompt。
var inputFunc func(prompt string) string

var stdinReader = bufio.NewReader(os.Stdin)

// SetInput 替换输入实现（WASM 环境用）。
func SetInput(f func(prompt string) string) { inputFunc = f }

// ReadInput 读取一行输入。prompt 非空时先打印提示。
func ReadInput(prompt string) string {
	if inputFunc != nil {
		return inputFunc(prompt)
	}
	if prompt != "" {
		fmt.Fprint(outputWriter, prompt)
	}
	line, err := stdinReader.ReadString('\n')
	if err != nil && line == "" {
		return ""
	}
	return strings.TrimRight(line, "\r\n")
}
