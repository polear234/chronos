//go:build !js

package interpreter

import (
	"strings"
	"sync"

	"chronos/ast"
	"chronos/object"
)

// TODO(并发): 目前 start() 用 goroutine 并行跑两个函数，各自在隔离的子环境里执行，
// 结束后把返回值合并成数组。限制与后续方向：
//   - 两个函数共享的全局变量只在各自副本里改，互不可见（避免 data race）
//   - print 输出加了锁，顺序不保证，但不会串行错乱
//   - 空闲时可以换成真正的共享环境 + 读写锁，让 fn 之间能交换数据
//   - WASM 单线程环境没有真正的并行，那里是顺序执行（见 builtin_start_js.go）

// outputMu 保护并行任务同时写输出，避免行与行互相穿插
var outputMu sync.Mutex

// runParallel 并行执行多个函数，按传入顺序返回结果数组。
// 任一任务出错时，该位置放 Error 对象，不影响其它任务。
func runParallel(fns []*object.Function, env *object.Environment) object.Object {
	results := make([]object.Object, len(fns))
	var wg sync.WaitGroup
	wg.Add(len(fns))

	for i, fn := range fns {
		go func(idx int, f *object.Function) {
			defer wg.Done()
			defer func() {
				// 单个任务 panic 不应该拖垮整个程序
				if r := recover(); r != nil {
					results[idx] = newError("start(): 任务 %d 崩溃: %v", idx+1, r)
				}
			}()
			// 每个任务在自己的子环境里跑，避免并发写同一张 map
			childEnv := object.NewEnclosedEnvironment(env)
			results[idx] = applyFunction(f, nil, childEnv)
		}(i, fn)
	}

	wg.Wait()
	return &object.Array{Elements: results}
}

// evalStartCall 处理 start(fn1, fn2, ...)：
// 并行执行传入的函数，返回结果数组。参数必须全是函数。
func evalStartCall(args []ast.Expression, env *object.Environment) object.Object {
	if len(args) == 0 {
		return newError("start() 至少需要一个函数")
	}

	// 先把所有参数求值成函数对象（这一步是串行的，保证参数表达式按顺序求值）
	fns := make([]*object.Function, 0, len(args))
	for i, arg := range args {
		val := Eval(arg, env)
		if isError(val) {
			return val
		}
		fn, ok := val.(*object.Function)
		if !ok {
			return newError("start() 第 %d 个参数不是函数，收到 %s", i+1, val.Type())
		}
		fns = append(fns, fn)
	}

	return runParallel(fns, env)
}

// parallelPrint 供并行任务使用：加锁后写一行输出
func parallelPrint(s string) {
	outputMu.Lock()
	defer outputMu.Unlock()
	outputWriter.Write([]byte(strings.TrimRight(s, "\n") + "\n"))
}
