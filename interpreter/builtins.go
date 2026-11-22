package interpreter

import (
	"fmt"
	"strconv"
	"strings"

	"chronos/object"
)

// builtins 全局内置函数
var builtins = map[string]*object.Builtin{
	"print": {
		Fn: func(args ...object.Object) object.Object {
			parts := make([]string, 0, len(args))
			for _, arg := range args {
				parts = append(parts, arg.Inspect())
			}
			fmt.Fprintln(outputWriter, strings.Join(parts, " "))
			return NIL
		},
	},
	"input": {
		Fn: func(args ...object.Object) object.Object {
			prompt := ""
			if len(args) > 0 {
				prompt = args[0].Inspect()
			}
			return &object.String{Value: ReadInput(prompt)}
		},
	},

	"println": {
		Fn: func(args ...object.Object) object.Object {
			parts := make([]string, 0, len(args))
			for _, arg := range args {
				parts = append(parts, arg.Inspect())
			}
			fmt.Fprintln(outputWriter, strings.Join(parts, " "))
			return NIL
		},
	},
	"len": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return newError("len expects 1 argument, got %d", len(args))
			}
			switch arg := args[0].(type) {
			case *object.String:
				return &object.Integer{Value: int64(len([]rune(arg.Value)))}
			case *object.Array:
				return &object.Integer{Value: int64(len(arg.Elements))}
			case *object.Hash:
				return &object.Integer{Value: int64(len(arg.Pairs))}
			default:
				return newError("len not supported on %s", args[0].Type())
			}
		},
	},
	"push": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 2 {
				return newError("push expects 2 arguments, got %d", len(args))
			}
			arr, ok := args[0].(*object.Array)
			if !ok {
				return newError("push requires array, got %s", args[0].Type())
			}
			arr.Elements = append(arr.Elements, args[1])
			return arr
		},
	},
	"pop": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return newError("pop expects 1 argument, got %d", len(args))
			}
			arr, ok := args[0].(*object.Array)
			if !ok {
				return newError("pop requires array, got %s", args[0].Type())
			}
			if len(arr.Elements) == 0 {
				return NIL
			}
			last := arr.Elements[len(arr.Elements)-1]
			arr.Elements = arr.Elements[:len(arr.Elements)-1]
			return last
		},
	},
	"shift": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return newError("shift expects 1 argument, got %d", len(args))
			}
			arr, ok := args[0].(*object.Array)
			if !ok {
				return newError("shift requires array, got %s", args[0].Type())
			}
			if len(arr.Elements) == 0 {
				return NIL
			}
			first := arr.Elements[0]
			arr.Elements = arr.Elements[1:]
			return first
		},
	},
	"unshift": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 2 {
				return newError("unshift expects 2 arguments, got %d", len(args))
			}
			arr, ok := args[0].(*object.Array)
			if !ok {
				return newError("unshift requires array, got %s", args[0].Type())
			}
			arr.Elements = append([]object.Object{args[1]}, arr.Elements...)
			return arr
		},
	},
	"keys": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return newError("keys expects 1 argument, got %d", len(args))
			}
			hash, ok := args[0].(*object.Hash)
			if !ok {
				return newError("keys requires hash, got %s", args[0].Type())
			}
			keys := make([]object.Object, 0, len(hash.Pairs))
			for _, pair := range hash.Pairs {
				keys = append(keys, pair.Key)
			}
			return &object.Array{Elements: keys}
		},
	},
	"values": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return newError("values expects 1 argument, got %d", len(args))
			}
			hash, ok := args[0].(*object.Hash)
			if !ok {
				return newError("values requires hash, got %s", args[0].Type())
			}
			values := make([]object.Object, 0, len(hash.Pairs))
			for _, pair := range hash.Pairs {
				values = append(values, pair.Value)
			}
			return &object.Array{Elements: values}
		},
	},
	"type": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return newError("type expects 1 argument, got %d", len(args))
			}
			return &object.String{Value: string(args[0].Type())}
		},
	},
	"int": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return newError("int expects 1 argument, got %d", len(args))
			}
			switch arg := args[0].(type) {
			case *object.Integer:
				return arg
			case *object.Float:
				return &object.Integer{Value: int64(arg.Value)}
			case *object.Boolean:
				if arg.Value {
					return &object.Integer{Value: 1}
				}
				return &object.Integer{Value: 0}
			case *object.String:
				if v, err := strconv.ParseInt(strings.TrimSpace(arg.Value), 10, 64); err == nil {
					return &object.Integer{Value: v}
				}
				return newError("cannot convert %q to int", arg.Value)
			default:
				return newError("cannot convert %s to int", args[0].Type())
			}
		},
	},
	"float": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return newError("float expects 1 argument, got %d", len(args))
			}
			switch arg := args[0].(type) {
			case *object.Integer:
				return &object.Float{Value: float64(arg.Value)}
			case *object.Float:
				return arg
			case *object.String:
				if v, err := strconv.ParseFloat(strings.TrimSpace(arg.Value), 64); err == nil {
					return &object.Float{Value: v}
				}
				return newError("cannot convert %q to float", arg.Value)
			default:
				return newError("cannot convert %s to float", args[0].Type())
			}
		},
	},
	"str": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return newError("str expects 1 argument, got %d", len(args))
			}
			if s, ok := args[0].(*object.String); ok {
				return s
			}
			return &object.String{Value: args[0].Inspect()}
		},
	},
	"range": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) < 1 || len(args) > 3 {
				return newError("range expects 1-3 arguments, got %d", len(args))
			}
			for _, a := range args {
				if _, ok := a.(*object.Integer); !ok {
					return newError("range arguments must be integers, got %s", a.Type())
				}
			}
			var start, end, step int64 = 0, 0, 1
			switch len(args) {
			case 1:
				end = args[0].(*object.Integer).Value
			case 2:
				start = args[0].(*object.Integer).Value
				end = args[1].(*object.Integer).Value
			case 3:
				start = args[0].(*object.Integer).Value
				end = args[1].(*object.Integer).Value
				step = args[2].(*object.Integer).Value
			}
			if step == 0 {
				return newError("range step cannot be zero")
			}
			elements := []object.Object{}
			if step > 0 {
				for i := start; i < end; i += step {
					elements = append(elements, &object.Integer{Value: i})
				}
			} else {
				for i := start; i > end; i += step {
					elements = append(elements, &object.Integer{Value: i})
				}
			}
			return &object.Array{Elements: elements}
		},
	},
}

// stringMethods 字符串对象的方法
var stringMethods = map[string]*object.Builtin{
	"length": {
		Fn: func(args ...object.Object) object.Object {
			s := args[0].(*object.String)
			return &object.Integer{Value: int64(len([]rune(s.Value)))}
		},
	},
	"upper": {
		Fn: func(args ...object.Object) object.Object {
			s := args[0].(*object.String)
			return &object.String{Value: strings.ToUpper(s.Value)}
		},
	},
	"lower": {
		Fn: func(args ...object.Object) object.Object {
			s := args[0].(*object.String)
			return &object.String{Value: strings.ToLower(s.Value)}
		},
	},
	"trim": {
		Fn: func(args ...object.Object) object.Object {
			s := args[0].(*object.String)
			return &object.String{Value: strings.TrimSpace(s.Value)}
		},
	},
	"contains": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 2 {
				return newError("contains expects 1 argument")
			}
			s := args[0].(*object.String)
			sub, ok := args[1].(*object.String)
			if !ok {
				return newError("contains requires string argument")
			}
			return nativeBoolToBooleanObject(strings.Contains(s.Value, sub.Value))
		},
	},
	"split": {
		Fn: func(args ...object.Object) object.Object {
			s := args[0].(*object.String)
			sep := ""
			if len(args) >= 2 {
				if ss, ok := args[1].(*object.String); ok {
					sep = ss.Value
				}
			}
			parts := strings.Split(s.Value, sep)
			elems := make([]object.Object, 0, len(parts))
			for _, p := range parts {
				elems = append(elems, &object.String{Value: p})
			}
			return &object.Array{Elements: elems}
		},
	},
	"replace": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 3 {
				return newError("replace expects 2 arguments")
			}
			s := args[0].(*object.String)
			old, ok1 := args[1].(*object.String)
			newS, ok2 := args[2].(*object.String)
			if !ok1 || !ok2 {
				return newError("replace requires string arguments")
			}
			return &object.String{Value: strings.ReplaceAll(s.Value, old.Value, newS.Value)}
		},
	},
}

// arrayMethods 数组对象的方法
var arrayMethods = map[string]*object.Builtin{
	"length": {
		Fn: func(args ...object.Object) object.Object {
			a := args[0].(*object.Array)
			return &object.Integer{Value: int64(len(a.Elements))}
		},
	},
	"push": {
		Fn: func(args ...object.Object) object.Object {
			a := args[0].(*object.Array)
			for _, v := range args[1:] {
				a.Elements = append(a.Elements, v)
			}
			return a
		},
	},
	"pop": {
		Fn: func(args ...object.Object) object.Object {
			a := args[0].(*object.Array)
			if len(a.Elements) == 0 {
				return NIL
			}
			last := a.Elements[len(a.Elements)-1]
			a.Elements = a.Elements[:len(a.Elements)-1]
			return last
		},
	},
	"shift": {
		Fn: func(args ...object.Object) object.Object {
			a := args[0].(*object.Array)
			if len(a.Elements) == 0 {
				return NIL
			}
			first := a.Elements[0]
			a.Elements = a.Elements[1:]
			return first
		},
	},
	"unshift": {
		Fn: func(args ...object.Object) object.Object {
			a := args[0].(*object.Array)
			a.Elements = append(args[1:], a.Elements...)
			return a
		},
	},
	"insert": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 3 {
				return newError("insert expects 2 arguments (index, value), got %d", len(args)-1)
			}
			a := args[0].(*object.Array)
			idx, ok := args[1].(*object.Integer)
			if !ok {
				return newError("insert index must be integer, got %s", args[1].Type())
			}
			i := idx.Value
			if i < 0 {
				i = 0
			}
			if i > int64(len(a.Elements)) {
				i = int64(len(a.Elements))
			}
			a.Elements = append(a.Elements, NIL)
			copy(a.Elements[i+1:], a.Elements[i:])
			a.Elements[i] = args[2]
			return a
		},
	},
	"removeAt": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 2 {
				return newError("removeAt expects 1 argument (index), got %d", len(args)-1)
			}
			a := args[0].(*object.Array)
			idx, ok := args[1].(*object.Integer)
			if !ok {
				return newError("removeAt index must be integer, got %s", args[1].Type())
			}
			i := idx.Value
			if i < 0 || i >= int64(len(a.Elements)) {
				return NIL
			}
			removed := a.Elements[i]
			a.Elements = append(a.Elements[:i], a.Elements[i+1:]...)
			return removed
		},
	},
	"indexOf": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 2 {
				return newError("indexOf expects 1 argument, got %d", len(args)-1)
			}
			a := args[0].(*object.Array)
			for i, e := range a.Elements {
				if objectsEqual(e, args[1]) {
					return &object.Integer{Value: int64(i)}
				}
			}
			return &object.Integer{Value: -1}
		},
	},
	"contains": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 2 {
				return newError("contains expects 1 argument, got %d", len(args)-1)
			}
			a := args[0].(*object.Array)
			for _, e := range a.Elements {
				if objectsEqual(e, args[1]) {
					return TRUE
				}
			}
			return FALSE
		},
	},
	"slice": {
		Fn: func(args ...object.Object) object.Object {
			a := args[0].(*object.Array)
			n := int64(len(a.Elements))
			var start, end int64 = 0, n
			if len(args) >= 2 {
				if s, ok := args[1].(*object.Integer); ok {
					start = s.Value
				}
			}
			if len(args) >= 3 {
				if e, ok := args[2].(*object.Integer); ok {
					end = e.Value
				}
			}
			if start < 0 {
				start = 0
			}
			if end > n {
				end = n
			}
			if start > end {
				start = end
			}
			return &object.Array{Elements: a.Elements[start:end]}
		},
	},
	"concat": {
		Fn: func(args ...object.Object) object.Object {
			a := args[0].(*object.Array)
			result := make([]object.Object, 0, len(a.Elements))
			result = append(result, a.Elements...)
			for _, other := range args[1:] {
				if oa, ok := other.(*object.Array); ok {
					result = append(result, oa.Elements...)
				} else {
					result = append(result, other)
				}
			}
			return &object.Array{Elements: result}
		},
	},
	"clear": {
		Fn: func(args ...object.Object) object.Object {
			a := args[0].(*object.Array)
			a.Elements = []object.Object{}
			return a
		},
	},
	"join": {
		Fn: func(args ...object.Object) object.Object {
			a := args[0].(*object.Array)
			sep := ""
			if len(args) >= 2 {
				if s, ok := args[1].(*object.String); ok {
					sep = s.Value
				}
			}
			parts := make([]string, 0, len(a.Elements))
			for _, e := range a.Elements {
				parts = append(parts, e.Inspect())
			}
			return &object.String{Value: strings.Join(parts, sep)}
		},
	},
	"reverse": {
		Fn: func(args ...object.Object) object.Object {
			a := args[0].(*object.Array)
			for i, j := 0, len(a.Elements)-1; i < j; i, j = i+1, j-1 {
				a.Elements[i], a.Elements[j] = a.Elements[j], a.Elements[i]
			}
			return a
		},
	},
}

// hashMethods 哈希对象的方法
var hashMethods = map[string]*object.Builtin{
	"length": {
		Fn: func(args ...object.Object) object.Object {
			h := args[0].(*object.Hash)
			return &object.Integer{Value: int64(len(h.Pairs))}
		},
	},
	"keys": {
		Fn: func(args ...object.Object) object.Object {
			h := args[0].(*object.Hash)
			keys := make([]object.Object, 0, len(h.Pairs))
			for _, pair := range h.Pairs {
				keys = append(keys, pair.Key)
			}
			return &object.Array{Elements: keys}
		},
	},
	"values": {
		Fn: func(args ...object.Object) object.Object {
			h := args[0].(*object.Hash)
			values := make([]object.Object, 0, len(h.Pairs))
			for _, pair := range h.Pairs {
				values = append(values, pair.Value)
			}
			return &object.Array{Elements: values}
		},
	},
	"has": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 2 {
				return newError("has expects 1 argument")
			}
			h := args[0].(*object.Hash)
			key, ok := args[1].(object.Hashable)
			if !ok {
				return newError("unusable as hash key: %s", args[1].Type())
			}
			_, exists := h.Pairs[key.HashKey()]
			return nativeBoolToBooleanObject(exists)
		},
	},
}
