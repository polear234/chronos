//go:build js

package interpreter

import "chronospp/object"

// newProcessModule WASM 版：浏览器无法访问进程内存
func newProcessModule() *object.Module {
	env := object.NewEnvironment()
	unavailable := func(name string) func(args ...object.Object) object.Object {
		return func(args ...object.Object) object.Object {
			return newError("process.%s 在浏览器/WASM 环境不可用（需要本机命令行版本）", name)
		}
	}
	for _, n := range []string{"find", "list", "maps", "readBytes", "writeBytes",
		"readInt", "writeInt", "readFloat", "writeFloat", "scanInt"} {
		env.Define(n, &object.Builtin{Fn: unavailable(n)})
	}
	return &object.Module{Name: "process", Env: env, Exports: env.Snapshot()}
}
