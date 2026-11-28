//go:build js

package interpreter

import "chronospp/object"

// newRequestsModule WASM 版：浏览器环境用 fetch 太复杂，先提供存根
// TODO: 后续可以用 syscall/js 包装 fetch 实现真正的请求
func newRequestsModule() *object.Module {
	env := object.NewEnvironment()

	unavailable := func(name string) func(args ...object.Object) object.Object {
		return func(args ...object.Object) object.Object {
			return newError("requests.%s 在浏览器/WASM 环境不可用（需要本机命令行版本）", name)
		}
	}

	for _, name := range []string{"get", "post", "put", "delete", "head", "getJson", "download"} {
		env.Define(name, &object.Builtin{Fn: unavailable(name)})
	}

	// quote 是纯字符串操作，能用
	env.Define("quote", &object.Builtin{
		Fn: func(args ...object.Object) object.Object {
			if len(args) < 1 {
				return newError("requests.quote 需要字符串")
			}
			s := args[0].Inspect()
			// 简单 URL 编码
			out := ""
			for _, c := range []byte(s) {
				if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') ||
					(c >= '0' && c <= '9') || c == '-' || c == '_' || c == '.' || c == '~' {
					out += string(c)
				} else {
					out += "%" + hexByte(c)
				}
			}
			return &object.String{Value: out}
		},
	})

	return &object.Module{
		Name:    "requests",
		Env:     env,
		Exports: env.Snapshot(),
	}
}

func hexByte(b byte) string {
	const hex = "0123456789ABCDEF"
	return string([]byte{hex[b>>4], hex[b&0xF]})
}
