//go:build js

package interpreter

import (
	"chronos/ast"
	"chronos/object"
)

// TODO(并发): WASM 是单线程环境，start() 在这里退化为顺序执行，
// 行为一致（返回结果数组），但没有真正的并行。参数按顺序逐个跑。

// evalStartCall 处理 start(fn1, fn2, ...)：WASM 版顺序执行
func evalStartCall(args []ast.Expression, env *object.Environment) object.Object {
	if len(args) == 0 {
		return newError("start() 至少需要一个函数")
	}

	results := make([]object.Object, 0, len(args))
	for i, arg := range args {
		val := Eval(arg, env)
		if isError(val) {
			return val
		}
		fn, ok := val.(*object.Function)
		if !ok {
			return newError("start() 第 %d 个参数不是函数，收到 %s", i+1, val.Type())
		}
		results = append(results, applyFunction(fn, nil, env))
	}
	return &object.Array{Elements: results}
}
