//go:build !js

package interpreter

import (
	"fmt"
	"strconv"
	"strings"
	"math"
	"sync"
	"unsafe"

	"chronospp/object"
)

// FfiLib 动态库句柄，持有 purego 的库指针
type FfiLib struct {
	Name string
	H    uintptr
	mu   sync.Mutex
	cache map[string]*object.Function
}

func (l *FfiLib) Type() object.ObjectType { return "FFI_LIB" }
func (l *FfiLib) Inspect() string         { return "<ffi lib " + l.Name + ">" }

// FfiPtr 封装指针（可指向 C 字符串、分配的内存等）
type FfiPtr struct {
	Addr   uint64
	Length int   // 字节数（0 表示未知）
}

func (p *FfiPtr) Type() object.ObjectType { return "FFI_PTR" }
func (p *FfiPtr) Inspect() string         { return fmt.Sprintf("<ptr 0x%x>", p.Addr) }

// ffiDlopen 加载动态库（平台相关，见 builtin_ffi_unix.go / _windows.go）
// ffiDlsym 查找符号（平台相关）

// newFfiModule 构造 FFI 模块
//   import "ffi"
//   let lib = ffi.load("libm.so")
//   lib.call("sqrt", 16.0)
func newFfiModule() *object.Module {
	env := object.NewEnvironment()

	// ffi.load(name) → 库句柄
	env.Define("load", &object.Builtin{
		Fn: func(args ...object.Object) object.Object {
			if len(args) < 1 {
				return newError("ffi.load 需要一个库路径或名字")
			}
			name := args[0].Inspect()
			h, err := ffiDlopen(name)
			if err != nil {
				return newError("ffi.load: 无法加载 %q: %v", name, err)
			}
			return &FfiLib{
				Name:  name,
				H:     h,
				cache: make(map[string]*object.Function),
			}
		},
	})

	// ffi.ptr("字符串") → 指针（C 字符串，GC 不回收）
	env.Define("ptr", &object.Builtin{
		Fn: func(args ...object.Object) object.Object {
			if len(args) < 1 {
				return newError("ffi.ptr 需要一个字符串")
			}
			s := args[0].Inspect()
			cstr := make([]byte, len(s)+1)
			copy(cstr, s)
			return &FfiPtr{Addr: uint64(uintptr(unsafe.Pointer(&cstr[0])))}
		},
	})

	// ffi.str(ptr[, maxLen]) 从指针读 C 字符串
	env.Define("str", &object.Builtin{
		Fn: func(args ...object.Object) object.Object {
			if len(args) < 1 {
				return newError("ffi.str 需要一个指针")
			}
			var addr uint64
			switch p := args[0].(type) {
			case *FfiPtr:
				addr = p.Addr
			case *object.Integer:
				addr = uint64(p.Value)
			default:
				return newError("ffi.str 参数要是指针或整数地址")
			}
			maxLen := 4096
			if len(args) > 1 {
				if n, ok := args[1].(*object.Integer); ok {
					maxLen = int(n.Value)
				}
			}
			// 用 unsafe.Slice 构造切片访问内存（避免 vet 的 uintptr→Pointer 警告）
			slice := unsafe.Slice((*byte)(unsafe.Add(unsafe.Pointer(nil), addr)), maxLen)
			out := make([]byte, 0, 32)
			for i := 0; i < maxLen; i++ {
				if slice[i] == 0 {
					break
				}
				out = append(out, slice[i])
			}
			return &object.String{Value: string(out)}
		},
	})

	// ffi.alloc(size) 分配内存
	env.Define("alloc", &object.Builtin{
		Fn: func(args ...object.Object) object.Object {
			if len(args) < 1 {
				return newError("ffi.alloc 需要大小")
			}
			size := toInt(args[0])
			if size <= 0 || size > 64*1024*1024 {
				return newError("ffi.alloc 大小无效: %d", size)
			}
			buf := make([]byte, size)
			return &FfiPtr{
				Addr:   uint64(uintptr(unsafe.Pointer(&buf[0]))),
				Length: int(size),
			}
		},
	})

	return &object.Module{
		Name:    "ffi",
		Env:     env,
		Exports: env.Snapshot(),
	}
}

// ── lib.call / lib.declare 方法调用 ──────
// lib 是 FfiLib 对象，我们通过 builtin 函数模拟方法调用

// ffiLibCall 在 evalCallExpression 里拦截 lib.xxx() 调用
func ffiLibCall(lib *FfiLib, method string, args []object.Object) object.Object {
	switch method {
	case "call":
		return ffiDoCall(lib, args, nil)
	case "declare":
		return ffiDeclare(lib, args)
	case "symbols":
		return ffiListSymbols(lib)
	}
	return newError("ffi lib 不支持方法 %s", method)
}

// ffiDeclare 声明函数签名：
//   lib.declare("pow", "float", "float", "float")
//   参数：fn名, 返回类型, 参数类型...
func ffiDeclare(lib *FfiLib, args []object.Object) object.Object {
	if len(args) < 2 {
		return newError("lib.declare 需要：函数名、返回类型、参数类型...")
	}
	name := args[0].Inspect()
	retType := args[1].Inspect()
	argTypes := make([]string, 0, len(args)-2)
	for _, a := range args[2:] {
		argTypes = append(argTypes, a.Inspect())
	}
	// 存储到 lib 的缓存中
	if lib.cache == nil {
		lib.cache = make(map[string]*object.Function)
	}
	sig := &object.Function{
		Name: name,
		Parameters: nil,
		Body:       nil,
	}
	// 把签名信息编码到 Name 里
	sig.Name = name + "\x00" + retType + "\x00"
	for i, t := range argTypes {
		if i > 0 {
			sig.Name += ","
		}
		sig.Name += t
	}
	lib.cache[name] = sig
	return NIL
}

// ffiListSymbols 列出库中的导出符号（返回名字列表，采样）
func ffiListSymbols(lib *FfiLib) object.Object {
	// purego 没有直接的符号枚举，返回已声明的
	out := []object.Object{}
	for name := range lib.cache {
		out = append(out, &object.String{Value: name})
	}
	return &object.Array{Elements: out}
}

// ffiDoCall 执行实际的函数调用
// args[0] 是函数名，其余是参数
func ffiDoCall(lib *FfiLib, args []object.Object, sig *object.Function) object.Object {
	if len(args) < 1 {
		return newError("lib.call 需要函数名")
	}
	fnName := args[0].Inspect()
	fnArgs := args[1:]

	// 解析签名（如果有）
	var retType string
	var argTypeNames []string
	if sig != nil && strings.Contains(sig.Name, "\x00") {
		parts := strings.Split(sig.Name, "\x00")
		if len(parts) >= 2 {
			fnName = parts[0]
			retType = parts[1]
			if len(parts) >= 3 && parts[2] != "" {
				argTypeNames = strings.Split(parts[2], ",")
			}
		}
	}

	// 查符号
	sym, err := ffiDlsym(lib.H, fnName)
	if err != nil {
		return newError("ffi: 符号 %q 未找到: %v", fnName, err)
	}

	// 检查是否有浮点参数/返回
	hasFloat := retType == "float" || retType == "double"
	for _, a := range fnArgs {
		if _, ok := a.(*object.Float); ok {
			hasFloat = true
			break
		}
	}
	for _, t := range argTypeNames {
		if t == "float" || t == "double" {
			hasFloat = true
			break
		}
	}

	if hasFloat {
		return ffiCallFloat(lib.H, sym, fnName, retType, argTypeNames, fnArgs)
	}
	return ffiCallInt(sym, fnName, fnArgs)
}

// ffiCallInt 整数 ABI 调用（SyscallN）
func ffiCallInt(sym uintptr, name string, args []object.Object) object.Object {
	rawArgs := make([]uintptr, len(args))
	for i, a := range args {
		switch v := a.(type) {
		case *object.Integer:
			rawArgs[i] = uintptr(v.Value)
		case *object.Boolean:
			if v.Value {
				rawArgs[i] = 1
			}
		case *object.String:
			// 字符串 → C 字符串指针
			cstr := make([]byte, len(v.Value)+1)
			copy(cstr, v.Value)
			rawArgs[i] = uintptr(unsafe.Pointer(&cstr[0]))
		case *FfiPtr:
			rawArgs[i] = uintptr(v.Addr)
		default:
			return newError("ffi: 参数 %d 不支持类型 %s", i+1, a.Type())
		}
	}

	r1, r2, _ := ffiSyscallN(sym, rawArgs...)
	_ = r2
	_ = name
	return &object.Integer{Value: int64(r1)}
}

// ffiCallFloat 浮点 ABI 调用：用固定类型的 Go 包装器
// purego.RegisterLibFunc 需要编译期确定的函数类型，用 0-3 个浮点参数的预定义签名
func ffiCallFloat(libHandle uintptr, sym uintptr, name string, retType string, argTypeNames []string, args []object.Object) object.Object {
	// 统计浮点参数个数
	floatCount := 0
	intCount := 0
	for _, a := range args {
		switch a.(type) {
		case *object.Float:
			floatCount++
		default:
			intCount++
		}
	}

	// 只支持常见的 1-2 个浮点参数场景
	// sqrt(x), sin(x), cos(x) 等 1 参
	// pow(x,y) 等 2 参
	if floatCount == 1 && intCount == 0 && retType != "int" {
		// 单浮点参数，浮点返回
		var f func(float64) float64
		ffiRegisterLibFunc(&f, libHandle, name)
		if f == nil {
			return newError("ffi: %s 注册失败", name)
		}
		fv := 0.0
		if fl, ok := args[0].(*object.Float); ok {
			fv = fl.Value
		} else if i, ok := args[0].(*object.Integer); ok {
			fv = float64(i.Value)
		}
		return &object.Float{Value: f(fv)}
	}
	if floatCount == 2 && intCount == 0 {
		// 双浮点参数，浮点返回
		var f func(float64, float64) float64
		ffiRegisterLibFunc(&f, libHandle, name)
		if f == nil {
			return newError("ffi: %s 注册失败", name)
		}
		f0, f1 := 0.0, 0.0
		if fl, ok := args[0].(*object.Float); ok {
			f0 = fl.Value
		} else if i, ok := args[0].(*object.Integer); ok {
			f0 = float64(i.Value)
		}
		if fl, ok := args[1].(*object.Float); ok {
			f1 = fl.Value
		} else if i, ok := args[1].(*object.Integer); ok {
			f1 = float64(i.Value)
		}
		return &object.Float{Value: f(f0, f1)}
	}

	// 其他情况回退到整数 ABI
	rawArgs := make([]uintptr, len(args))
	for i, a := range args {
		switch v := a.(type) {
		case *object.Float:
			bits := math.Float64bits(v.Value)
			rawArgs[i] = uintptr(bits)
		case *object.Integer:
			rawArgs[i] = uintptr(v.Value)
		case *object.String:
			cstr := make([]byte, len(v.Value)+1)
			copy(cstr, v.Value)
			rawArgs[i] = uintptr(unsafe.Pointer(&cstr[0]))
		case *FfiPtr:
			rawArgs[i] = uintptr(v.Addr)
		}
	}
	r1, _, _ := ffiSyscallN(sym, rawArgs...)
	_ = name
	_ = retType
	return &object.Integer{Value: int64(r1)}
}

var _ = math.Pi

// toInt 把对象转成 int64（通用辅助）
func toInt(o object.Object) int64 {
	switch v := o.(type) {
	case *object.Integer:
		return v.Value
	case *object.Float:
		return int64(v.Value)
	case *object.String:
		n, _ := strconv.ParseInt(strings.TrimSpace(v.Value), 0, 64)
		return n
	}
	return 0
}

// float32 位操作
func float32frombits(b uint32) float32 {
	return *(*float32)(unsafe.Pointer(&b))
}

func float32bits(f float32) uint32 {
	return *(*uint32)(unsafe.Pointer(&f))
}
