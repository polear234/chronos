package interpreter

import (
	"io"
	"os"
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
