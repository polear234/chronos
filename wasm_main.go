//go:build js && wasm

package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
	"syscall/js"

	"chronos/interpreter"
	"chronos/lexer"
	"chronos/object"
	"chronos/parser"
)

var version = "0.3.2"

type runResult struct {
	Output string `json:"output"`
	Error  string `json:"error"`
}

func main() {
	// 输入：调用 JS 的 prompt()（Android 里由 WebChromeClient 弹框）
	interpreter.SetInput(func(prompt string) string {
		v := js.Global().Call("prompt", prompt)
		if v.Type() == js.TypeNull || v.Type() == js.TypeUndefined {
			return ""
		}
		return v.String()
	})

	js.Global().Set("chronosRun", js.FuncOf(runJS))
	js.Global().Set("chronosCheck", js.FuncOf(checkJS))
	// 虚拟文件系统：JS 侧维护 { "lib.chr": "源码..." }，import 时从这里取
	interpreter.SetModuleReader(func(path string) ([]byte, error) {
		files := js.Global().Get("chronosFiles")
		if files.Type() != js.TypeObject {
			return nil, fmt.Errorf("module not found: %s", path)
		}
		// 依次尝试完整路径、文件名、带/不带扩展名
		cands := []string{path, filepath.Base(path)}
		base := filepath.Base(path)
		cands = append(cands,
			strings.TrimSuffix(base, filepath.Ext(base)),
			strings.TrimSuffix(base, filepath.Ext(base))+".chr",
			base+".chr",
		)
		for _, c := range cands {
			v := files.Get(c)
			if v.Type() == js.TypeString {
				return []byte(v.String()), nil
			}
		}
		return nil, fmt.Errorf("module not found: %s", path)
	})
	interpreter.SetModuleExists(func(path string) bool {
		files := js.Global().Get("chronosFiles")
		if files.Type() != js.TypeObject {
			return false
		}
		if files.Get(path).Type() == js.TypeString {
			return true
		}
		base := filepath.Base(path)
		if files.Get(base).Type() == js.TypeString {
			return true
		}
		stem := strings.TrimSuffix(base, filepath.Ext(base))
		return files.Get(stem).Type() == js.TypeString ||
			files.Get(stem+".chr").Type() == js.TypeString
	})
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

// checkJS 只做语法解析，不执行程序（用于编译前校验，等价于 C++ 的编译期检查）
func checkJS(this js.Value, args []js.Value) any {
	src := ""
	if len(args) > 0 {
		src = args[0].String()
	}
	l := lexer.New(src)
	p := parser.New(l)
	p.ParseProgram()
	errs := p.Errors()
	b, _ := json.Marshal(struct {
		OK    bool     `json:"ok"`
		Error string   `json:"error"`
		Count int      `json:"count"`
		Errs  []string `json:"errors"`
	}{OK: len(errs) == 0, Count: len(errs), Errs: errs, Error: strings.Join(errs, "\n")})
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
