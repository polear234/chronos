//go:build !js

package interpreter

import (
	"strings"
	"sync"

	"chronos/ast"
	"chronos/object"
)

// TODO(并发): start() 用 goroutine 并行执行，每个任务在隔离子环境里跑。
// 限制与后续方向：
//   - 任务对全局变量的改动只在各自副本里，互不可见（避免 data race）
//   - print 输出加了锁，顺序不保证，但不会串行错乱
//   - 空闲时可以换成共享环境 + 读写锁，让任务之间能交换数据
//   - WASM 单线程环境没有真正的并行，那里是顺序执行（见 builtin_start_js.go）

// outputMu 保护并行任务同时写输出，避免行与行互相穿插
var outputMu sync.Mutex

// startTask 一个待并行执行的任务
type startTask struct {
	// 二选一：fn 不为空表示调用函数；expr 不为空表示求值表达式（如 add(3,4)）
	fn   *object.Function
	expr ast.Expression
}

// runParallel 并行执行多个任务，按传入顺序返回结果数组。
// 任一任务出错时，该位置放 Error 对象，不影响其它任务。
func runParallel(tasks []startTask, env *object.Environment) object.Object {
	results := make([]object.Object, len(tasks))
	var wg sync.WaitGroup
	wg.Add(len(tasks))

	for i, task := range tasks {
		go func(idx int, t startTask) {
			defer wg.Done()
			defer func() {
				// 单个任务 panic 不应该拖垮整个程序
				if r := recover(); r != nil {
					results[idx] = newError("start(): 任务 %d 崩溃: %v", idx+1, r)
				}
			}()
			// 每个任务在自己的子环境里跑，避免并发写同一张 map
			childEnv := object.NewEnclosedEnvironment(env)
			if t.fn != nil {
				results[idx] = applyFunction(t.fn, nil, childEnv)
			} else {
				// 调用表达式在这里才求值——这是并行的关键
				results[idx] = Eval(t.expr, childEnv)
			}
		}(i, task)
	}

	wg.Wait()
	return &object.Array{Elements: results}
}

// evalStartCall 处理 start(...)：
//   start(fn)              → 零参数调用 fn
//   start(fn(a, b))        → 并行求值调用表达式
//   start(fn1, fn2)        → 并行跑多个
//   start(fn(1), fn(2))    → 混合
// 参数可以是函数，也可以是调用表达式；其它值报错。
func evalStartCall(args []ast.Expression, env *object.Environment) object.Object {
	if len(args) == 0 {
		return newError("start() 至少需要一个函数或调用")
	}

	tasks := make([]startTask, 0, len(args))
	for i, arg := range args {
		// 先看语法形态：调用表达式直接当作并行任务，不求值
		if call, ok := arg.(*ast.CallExpression); ok {
			// 确认被调用的是函数（而不是 start 自己嵌套），避免无限递归
			if id, ok := call.Function.(*ast.Identifier); ok && id.Value == "start" {
				return newError("start() 里面不能再嵌套 start()")
			}
			tasks = append(tasks, startTask{expr: arg})
			continue
		}

		// 其它形态先求值，必须是函数
		val := Eval(arg, env)
		if isError(val) {
			return val
		}
		fn, ok := val.(*object.Function)
		if !ok {
			return newError("start() 第 %d 个参数不是函数或函数调用，收到 %s", i+1, val.Type())
		}
		tasks = append(tasks, startTask{fn: fn})
	}

	return runParallel(tasks, env)
}

// parallelPrint 供并行任务使用：加锁后写一行输出
func parallelPrint(s string) {
	outputMu.Lock()
	defer outputMu.Unlock()
	outputWriter.Write([]byte(strings.TrimRight(s, "\n") + "\n"))
}
