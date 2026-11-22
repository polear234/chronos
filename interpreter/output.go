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

// ── 模块文件系统（import 用）────────────────────────────

// moduleReader 是可替换的模块文件读取器。
// 为 nil 时用真实文件系统（CLI）；WASM 环境由宿主提供虚拟文件表。
var moduleReader func(path string) ([]byte, error)

// moduleExists 可选的"存在性"检查；nil 时退回真实文件系统。
var moduleExists func(path string) bool

// SetModuleReader 设置虚拟模块读取器，返回旧的读取器以便恢复。
// reader 为 nil 时恢复真实文件系统行为。
func SetModuleReader(reader func(path string) ([]byte, error)) func(string) ([]byte, error) {
	old := moduleReader
	moduleReader = reader
	return old
}

// SetModuleExists 设置虚拟模块存在性检查，返回旧值。
func SetModuleExists(fn func(path string) bool) func(string) bool {
	old := moduleExists
	moduleExists = fn
	return old
}

// readModuleFile 读取模块源码：优先虚拟文件系统，否则读真实磁盘。
func readModuleFile(path string) ([]byte, error) {
	if moduleReader != nil {
		return moduleReader(path)
	}
	return os.ReadFile(path)
}

// moduleFileExists 判断模块文件是否存在：优先虚拟文件系统，否则查磁盘。
func moduleFileExists(path string) bool {
	if moduleExists != nil {
		return moduleExists(path)
	}
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}
