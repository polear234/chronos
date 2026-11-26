package interpreter

import (
	"fmt"
	"math"
	"math/rand"
	"strings"

	"chronos/object"
)

// newMathModule 构造 Python 风格的 math 模块（跨平台，WASM 也能用）
func newMathModule() *object.Module {
	env := object.NewEnvironment()

	def := func(name string, fn func(args ...object.Object) object.Object) {
		env.Define(name, &object.Builtin{Fn: fn})
	}

	// 单参数数学函数
	unary := func(name string, f func(float64) float64) {
		def(name, func(args ...object.Object) object.Object {
			if len(args) < 1 {
				return newError("math.%s 需要一个数字", name)
			}
			x, ok := toFloat(args[0])
			if !ok {
				return newError("math.%s 需要数字，收到 %s", name, args[0].Type())
			}
			return &object.Float{Value: f(x)}
		})
	}

	unary("sqrt", math.Sqrt)
	unary("floor", math.Floor)
	unary("ceil", math.Ceil)
	unary("sin", math.Sin)
	unary("cos", math.Cos)
	unary("tan", math.Tan)
	unary("asin", math.Asin)
	unary("acos", math.Acos)
	unary("atan", math.Atan)
	unary("log", math.Log)
	unary("log2", math.Log2)
	unary("log10", math.Log10)
	unary("exp", math.Exp)

	// abs 返回保持原类型
	def("abs", func(args ...object.Object) object.Object {
		if len(args) < 1 {
			return newError("math.abs 需要一个数字")
		}
		if i, ok := args[0].(*object.Integer); ok {
			if i.Value < 0 {
				return &object.Integer{Value: -i.Value}
			}
			return i
		}
		x, ok := toFloat(args[0])
		if !ok {
			return newError("math.abs 需要数字，收到 %s", args[0].Type())
		}
		return &object.Float{Value: math.Abs(x)}
	})

	// pow(x, y) / round(x) / min / max
	def("pow", func(args ...object.Object) object.Object {
		if len(args) < 2 {
			return newError("math.pow 需要两个数字")
		}
		x, ok1 := toFloat(args[0])
		y, ok2 := toFloat(args[1])
		if !ok1 || !ok2 {
			return newError("math.pow 需要数字")
		}
		return &object.Float{Value: math.Pow(x, y)}
	})

	def("round", func(args ...object.Object) object.Object {
		if len(args) < 1 {
			return newError("math.round 需要一个数字")
		}
		x, ok := toFloat(args[0])
		if !ok {
			return newError("math.round 需要数字")
		}
		return &object.Integer{Value: int64(math.Round(x))}
	})

	def("min", func(args ...object.Object) object.Object {
		if len(args) == 0 {
			return newError("math.min 至少需要一个数字")
		}
		best, ok := toFloat(args[0])
		if !ok {
			return newError("math.min 需要数字")
		}
		for _, a := range args[1:] {
			x, ok := toFloat(a)
			if !ok {
				return newError("math.min 需要数字")
			}
			if x < best {
				best = x
			}
		}
		return &object.Float{Value: best}
	})

	def("max", func(args ...object.Object) object.Object {
		if len(args) == 0 {
			return newError("math.max 至少需要一个数字")
		}
		best, ok := toFloat(args[0])
		if !ok {
			return newError("math.max 需要数字")
		}
		for _, a := range args[1:] {
			x, ok := toFloat(a)
			if !ok {
				return newError("math.max 需要数字")
			}
			if x > best {
				best = x
			}
		}
		return &object.Float{Value: best}
	})

	// 随机数
	def("random", func(args ...object.Object) object.Object {
		return &object.Float{Value: rand.Float64()}
	})

	def("randInt", func(args ...object.Object) object.Object {
		if len(args) < 2 {
			return newError("math.randInt 需要最小值和最大值")
		}
		lo, ok1 := toFloat(args[0])
		hi, ok2 := toFloat(args[1])
		if !ok1 || !ok2 {
			return newError("math.randInt 需要数字")
		}
		if hi <= lo {
			return newError("math.randInt 最大值必须大于最小值")
		}
		return &object.Integer{Value: int64(lo) + rand.Int63n(int64(hi-lo))}
	})

	// 常量
	env.Define("PI", &object.Float{Value: math.Pi})
	env.Define("E", &object.Float{Value: math.E})
	env.Define("inf", &object.Float{Value: math.Inf(1)})

	return &object.Module{
		Name:    "math",
		Env:     env,
		Exports: env.Snapshot(),
	}
}

// toFloat 把数字对象转成 float64
func toFloat(o object.Object) (float64, bool) {
	switch v := o.(type) {
	case *object.Integer:
		return float64(v.Value), true
	case *object.Float:
		return v.Value, true
	case *object.String:
		var f float64
		if _, err := fmt.Sscanf(strings.TrimSpace(v.Value), "%g", &f); err == nil {
			return f, true
		}
	}
	return 0, false
}

// nativeBool 把 Go bool 转成 Chronos 布尔对象
func nativeBool(b bool) object.Object {
	if b {
		return TRUE
	}
	return FALSE
}
