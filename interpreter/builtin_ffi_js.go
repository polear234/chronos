//go:build js

package interpreter

import "chronospp/object"

// newFfiModule WASM 版：浏览器无法加载动态库
func newFfiModule() *object.Module {
	env := object.NewEnvironment()
	unavailable := func(name string) func(args ...object.Object) object.Object {
		return func(args ...object.Object) object.Object {
			return newError("ffi.%s 在浏览器/WASM 环境不可用", name)
		}
	}
	for _, n := range []string{"load", "ptr", "str", "alloc"} {
		env.Define(n, &object.Builtin{Fn: unavailable(n)})
	}
	return &object.Module{Name: "ffi", Env: env, Exports: env.Snapshot()}
}
