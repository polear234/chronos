//go:build js && wasm

package main

import (
	"bytes"
	"encoding/json"
	"strings"
	"syscall/js"

	"chronos/interpreter"
	"chronos/lexer"
	"chronos/object"
	"chronos/parser"
)

var version = "0.2.1"

type runResult struct {
	Output string `json:"output"`
	Error  string `json:"error"`
}

func main() {
	js.Global().Set("chronosRun", js.FuncOf(runJS))
	js.Global().Set("chronosVersion", js.FuncOf(func(this js.Value, args []js.Value) any {
		return "Chronos " + version
	}))
	// 通知宿主已就绪（若宿主注册了 onChronosReady）
	if cb := js.Global().Get("onChronosReady"); cb.Type() == js.TypeFunction {
		cb.Invoke("Chronos " + version)
	}
	// 阻塞，保持 Go 运行时存活以便持续接收 JS 调用
	select {}
}

// runJS 接收脚本源码，返回 JSON: {"output": "...", "error": "..."}
func runJS(this js.Value, args []js.Value) any {
	src := ""
	if len(args) > 0 {
		src = args[0].String()
	}
	res := execute(src)
	b, _ := json.Marshal(res)
	return string(b)
}

func execute(src string) (result runResult) {
	var buf bytes.Buffer
	old := interpreter.SetOutput(&buf)
	defer interpreter.SetOutput(old)

	l := lexer.New(src)
	p := parser.New(l)
	prog := p.ParseProgram()
	if errs := p.Errors(); len(errs) > 0 {
		result.Output = buf.String()
		result.Error = "语法错误:\n" + strings.Join(errs, "\n")
		return
	}

	env := object.NewEnvironment()
	r := interpreter.Eval(prog, env)
	result.Output = buf.String()
	if r != nil && r.Type() == object.ERROR_OBJ {
		result.Error = r.Inspect()
	}
	return
}
