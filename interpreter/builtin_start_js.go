//go:build js

package interpreter

import (
	"chronospp/ast"
	"chronospp/object"
)

// TODO(并发): WASM 是单线程环境，start() 在这里退化为顺序执行，
// 行为一致（返回结果数组），但没有真正的并行。

// evalStartCall 处理 start(...)：WASM 版顺序执行
//   start(fn)          → 调用 fn
//   start(fn(a, b))    → 求值调用表达式
//   start(fn1, fn2)    → 逐个跑
func evalStartCall(args []ast.Expression, env *object.Environment) object.Object {
	if len(args) == 0 {
		return newError("start() 至少需要一个函数或调用")
	}

	results := make([]object.Object, 0, len(args))
	for i, arg := range args {
		// 调用表达式：直接求值
		if call, ok := arg.(*ast.CallExpression); ok {
			if id, ok := call.Function.(*ast.Identifier); ok && id.Value == "start" {
				return newError("start() 里面不能再嵌套 start()")
			}
			results = append(results, Eval(arg, env))
			continue
		}

		val := Eval(arg, env)
		if isError(val) {
			return val
		}
		fn, ok := val.(*object.Function)
		if !ok {
			return newError("start() 第 %d 个参数不是函数或函数调用，收到 %s", i+1, val.Type())
		}
		results = append(results, applyFunction(fn, nil, env))
	}
	return &object.Array{Elements: results}
}
