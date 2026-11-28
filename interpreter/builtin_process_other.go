//go:build (darwin || freebsd || openbsd || netbsd) && !js

package interpreter

import "chronospp/object"

// newProcessModule 其它 Unix 系统：暂不实现
func newProcessModule() *object.Module {
	env := object.NewEnvironment()
	unavailable := func(name string) func(args ...object.Object) object.Object {
		return func(args ...object.Object) object.Object {
			return newError("process.%s 暂不支持当前系统", name)
		}
	}
	for _, n := range []string{"find", "list", "maps", "readBytes", "writeBytes",
		"readInt", "writeInt", "readFloat", "writeFloat", "scanInt"} {
		env.Define(n, &object.Builtin{Fn: unavailable(n)})
	}
	return &object.Module{Name: "process", Env: env, Exports: env.Snapshot()}
}
