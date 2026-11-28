//go:build js

package interpreter

import (
	"path/filepath"
	"strings"

	"chronospp/object"
)

// builtinModules 内置模块：import "os" 直接拿到，不走文件系统
// WASM 环境没有文件系统和子进程，os 只提供纯内存/只读操作
var builtinModules = map[string]func() *object.Module{
	"os":       newOSModule,
	"math":     newMathModule,
	"requests": newRequestsModule,
	"process":  newProcessModule,
}

func lookupBuiltinModule(name string) *object.Module {
	base := strings.TrimSuffix(filepath.Base(name), filepath.Ext(name))
	if fn, ok := builtinModules[base]; ok {
		return fn()
	}
	return nil
}

// newOSModule WASM 版：文件系统操作不可用，调用时报错提示
func newOSModule() *object.Module {
	env := object.NewEnvironment()

	unavailable := func(name string) func(args ...object.Object) object.Object {
		return func(args ...object.Object) object.Object {
			return newError("os.%s 在浏览器/WASM 环境不可用（需要本机命令行版本）", name)
		}
	}

	for _, name := range []string{
		"readFile", "writeFile", "appendFile", "exists", "remove",
		"mkdir", "listDir", "abs", "cwd", "chdir", "run", "exec",
	} {
		env.Define(name, &object.Builtin{Fn: unavailable(name)})
	}

	// 这些是纯字符串操作，WASM 也能用
	env.Define("join", &object.Builtin{Fn: func(args ...object.Object) object.Object {
		parts := make([]string, 0, len(args))
		for _, a := range args {
			parts = append(parts, a.Inspect())
		}
		return &object.String{Value: filepath.Join(parts...)}
	}})
	env.Define("basename", &object.Builtin{Fn: func(args ...object.Object) object.Object {
		if len(args) < 1 {
			return newError("os.basename 需要一个路径")
		}
		return &object.String{Value: filepath.Base(args[0].Inspect())}
	}})
	env.Define("dirname", &object.Builtin{Fn: func(args ...object.Object) object.Object {
		if len(args) < 1 {
			return newError("os.dirname 需要一个路径")
		}
		return &object.String{Value: filepath.Dir(args[0].Inspect())}
	}})
	env.Define("getenv", &object.Builtin{Fn: func(args ...object.Object) object.Object {
		return &object.String{Value: ""}
	}})
	env.Define("setenv", &object.Builtin{Fn: unavailable("setenv")})

	return &object.Module{
		Name:    "os",
		Env:     env,
		Exports: env.Snapshot(),
	}
}
