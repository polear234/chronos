package object

import (
	"fmt"
	"strconv"
	"strings"

	"chronospp/ast"
)

// Integer 整数对象
type Integer struct {
	Value int64
}

func (i *Integer) Type() ObjectType { return INTEGER_OBJ }
func (i *Integer) Inspect() string  { return strconv.FormatInt(i.Value, 10) }
func (i *Integer) HashKey() HashKey {
	return HashKey{Type: i.Type(), Value: uint64(i.Value)}
}

// Float 浮点数对象
type Float struct {
	Value float64
}

func (f *Float) Type() ObjectType { return FLOAT_OBJ }
func (f *Float) Inspect() string  { return strconv.FormatFloat(f.Value, 'f', -1, 64) }
func (f *Float) HashKey() HashKey {
	return HashKey{Type: f.Type(), Value: uint64(f.Value)}
}

// String 字符串对象
type String struct {
	Value string
}

func (s *String) Type() ObjectType { return STRING_OBJ }
func (s *String) Inspect() string  { return s.Value }
func (s *String) HashKey() HashKey {
	var h uint64
	for i := 0; i < len(s.Value); i++ {
		h = h*31 + uint64(s.Value[i])
	}
	return HashKey{Type: s.Type(), Value: h}
}

// Boolean 布尔对象
type Boolean struct {
	Value bool
}

func (b *Boolean) Type() ObjectType { return BOOLEAN_OBJ }
func (b *Boolean) Inspect() string  { return fmt.Sprintf("%t", b.Value) }
func (b *Boolean) HashKey() HashKey {
	var v uint64
	if b.Value {
		v = 1
	}
	return HashKey{Type: b.Type(), Value: v}
}

// Nil 空值对象
type Nil struct{}

func (n *Nil) Type() ObjectType { return NIL_OBJ }
func (n *Nil) Inspect() string  { return "nil" }

// ReturnValue 返回值包装器
type ReturnValue struct {
	Value Object
}

func (rv *ReturnValue) Type() ObjectType { return RETURN_VALUE_OBJ }
func (rv *ReturnValue) Inspect() string {
	if rv.Value != nil {
		return rv.Value.Inspect()
	}
	return "nil"
}

// Error 错误对象
type Error struct {
	Message string
}

func (e *Error) Type() ObjectType { return ERROR_OBJ }
func (e *Error) Inspect() string  { return "ERROR: " + e.Message }

// Function 用户函数对象
type Function struct {
	Parameters []*ast.Identifier
	Body       *ast.BlockStatement
	Env        *Environment
	Name       string
	ReturnType string // C++ 风格返回类型，空串表示不检查
}

func (f *Function) Type() ObjectType { return FUNCTION_OBJ }
func (f *Function) Inspect() string {
	params := make([]string, 0, len(f.Parameters))
	for _, p := range f.Parameters {
		params = append(params, p.Value)
	}
	return "fn " + f.Name + "(" + strings.Join(params, ", ") + ") {...}"
}

// Builtin 内置函数对象
type Builtin struct {
	Fn func(args ...Object) Object
}

func (b *Builtin) Type() ObjectType { return BUILTIN_OBJ }
func (b *Builtin) Inspect() string  { return "builtin function" }

// Array 数组对象
type Array struct {
	Elements []Object
}

func (a *Array) Type() ObjectType { return ARRAY_OBJ }
func (a *Array) Inspect() string {
	elems := make([]string, 0, len(a.Elements))
	for _, e := range a.Elements {
		elems = append(elems, e.Inspect())
	}
	return "[" + strings.Join(elems, ", ") + "]"
}

// HashPair 哈希键值对
type HashPair struct {
	Key   Object
	Value Object
}

// Hash 哈希/字典对象
type Hash struct {
	Pairs map[HashKey]HashPair
}

func (h *Hash) Type() ObjectType { return HASH_OBJ }
func (h *Hash) Inspect() string {
	pairs := make([]string, 0, len(h.Pairs))
	for _, pair := range h.Pairs {
		pairs = append(pairs, pair.Key.Inspect()+": "+pair.Value.Inspect())
	}
	return "{" + strings.Join(pairs, ", ") + "}"
}

// Class 类对象
type Class struct {
	Name       string
	SuperClass *Class
	Methods    map[string]*Function
	Properties map[string]Object
}

func (c *Class) Type() ObjectType { return CLASS_OBJ }
func (c *Class) Inspect() string  { return "<class " + c.Name + ">" }

// Instance 实例对象
type Instance struct {
	Class  *Class
	Fields map[string]Object
}

func (i *Instance) Type() ObjectType { return INSTANCE_OBJ }
func (i *Instance) Inspect() string  { return "<instance of " + i.Class.Name + ">" }

// Method 绑定实例的方法
type Method struct {
	Fn       *Function
	Receiver Object
	Class    *Class // 方法所属的类，用于 super 从正确的父类开始查找
}

func (m *Method) Type() ObjectType { return METHOD_OBJ }
func (m *Method) Inspect() string  { return "<method " + m.Fn.Name + ">" }

// Module 模块对象
type Module struct {
	Name    string
	Env     *Environment
	Exports map[string]Object
}

func (m *Module) Type() ObjectType { return MODULE_OBJ }
func (m *Module) Inspect() string  { return "<module " + m.Name + ">" }

// Interface 接口：只记录方法名，运行时检查实现类
type Interface struct {
	Name    string
	Methods []string
}

func (i *Interface) Type() ObjectType { return INTERFACE_OBJ }
func (i *Interface) Inspect() string {
	return "<interface " + i.Name + ">"
}

// Pointer 指针对象：指向某个变量（按名字引用），支持 *p 读写
type Pointer struct {
	Env  *Environment // 目标变量所在的作用域
	Name string       // 目标变量名
}

func (p *Pointer) Type() ObjectType { return POINTER_OBJ }
func (p *Pointer) Inspect() string  { return "<pointer to " + p.Name + ">" }

// Deref 取指针指向的值
func (p *Pointer) Deref() (Object, bool) {
	return p.Env.Get(p.Name)
}

// Assign 通过指针写值
func (p *Pointer) Assign(val Object) bool {
	p.Env.Set(p.Name, val)
	return true
}

// Break 循环中断信号
type Break struct{}

func (b *Break) Type() ObjectType { return BREAK_OBJ }
func (b *Break) Inspect() string  { return "break" }

// Continue 循环继续信号
type Continue struct{}

func (c *Continue) Type() ObjectType { return CONTINUE_OBJ }
func (c *Continue) Inspect() string  { return "continue" }

// Snapshot 目标状态快照（数组记录长度，字典浅拷贝）
type Snapshot struct {
	Kind      SnapshotKind
	ArrayLen  int
	HashPairs map[HashKey]HashPair // 浅拷贝：值仍是指针，但 map 本身独立
}

type SnapshotKind int

const (
	SnapshotArray SnapshotKind = iota
	SnapshotHash
)

// Timeline 版本历史对象，支持 commit/undo
type Timeline struct {
	Target  Object
	History []Snapshot // 历史栈，栈顶为最近一次 commit
}

func (t *Timeline) Type() ObjectType { return TIMELINE_OBJ }
func (t *Timeline) Inspect() string  { return "<timeline>" }

// CaptureSnapshot 对目标取快照
func CaptureSnapshot(target Object) *Snapshot {
	switch t := target.(type) {
	case *Array:
		return &Snapshot{Kind: SnapshotArray, ArrayLen: len(t.Elements)}
	case *Hash:
		pairs := make(map[HashKey]HashPair, len(t.Pairs))
		for k, v := range t.Pairs {
			pairs[k] = v
		}
		return &Snapshot{Kind: SnapshotHash, HashPairs: pairs}
	default:
		return nil
	}
}

// RestoreSnapshot 原地恢复目标到快照状态（保持目标引用不变）
func RestoreSnapshot(target Object, snap *Snapshot) {
	switch snap.Kind {
	case SnapshotArray:
		if a, ok := target.(*Array); ok {
			if snap.ArrayLen <= len(a.Elements) {
				a.Elements = a.Elements[:snap.ArrayLen]
			}
		}
	case SnapshotHash:
		if h, ok := target.(*Hash); ok {
			h.Pairs = make(map[HashKey]HashPair, len(snap.HashPairs))
			for k, v := range snap.HashPairs {
				h.Pairs[k] = v
			}
		}
	}
}

// Environment 变量作用域环境
type Environment struct {
	store    map[string]Object
	consts   map[string]bool     // 标记常量，赋值时拒绝
	history  map[string][]Object // 变量历史旧值栈，用于 rollback
	varTypes map[string]string   // 变量类型约束（"int"/"string"/"dict"/"array"），空串无约束
	outer    *Environment

	ImportDir   string                 // import 解析的基准目录（相对路径基于此）
	ImportChain map[string]bool        // 共享的导入链，用于循环 import 检测
}

func NewEnvironment() *Environment {
	return &Environment{
		store:       make(map[string]Object),
		consts:      make(map[string]bool),
		history:     make(map[string][]Object),
		varTypes:    make(map[string]string),
		ImportChain: make(map[string]bool),
	}
}

func NewEnclosedEnvironment(outer *Environment) *Environment {
	env := &Environment{
		store:       make(map[string]Object),
		consts:      make(map[string]bool),
		history:     make(map[string][]Object),
		varTypes:    make(map[string]string),
		outer:       outer,
		ImportChain: make(map[string]bool),
	}
	if outer != nil {
		// 继承导入目录，但 import 链独立（子环境有自己的链副本，避免污染）
		env.ImportDir = outer.ImportDir
		for k := range outer.ImportChain {
			env.ImportChain[k] = true
		}
	}
	return env
}

// findScope 定位变量实际所在的作用域（沿外层链查找），不存在返回 nil
func (e *Environment) findScope(name string) *Environment {
	if _, ok := e.store[name]; ok {
		return e
	}
	if e.outer != nil {
		return e.outer.findScope(name)
	}
	return nil
}

func (e *Environment) Get(name string) (Object, bool) {
	obj, ok := e.store[name]
	if !ok && e.outer != nil {
		return e.outer.Get(name)
	}
	return obj, ok
}

// Set 在当前作用域（或已存在变量的最近作用域）赋值
func (e *Environment) Set(name string, val Object) Object {
	scope := e.findScope(name)
	if scope != nil {
		if scope.isConst(name) {
			return &Error{Message: "cannot reassign to constant: " + name}
		}
		// 记录历史：把旧值压栈，供 rollback 使用
		scope.history[name] = append(scope.history[name], scope.store[name])
		scope.store[name] = val
		return val
	}
	// 变量不存在：在当前作用域新建（不记录历史）
	e.store[name] = val
	return val
}

// Define 在当前作用域强制定义（用于函数参数、let 局部变量）
// 重新声明会重置该变量的历史栈
func (e *Environment) Define(name string, val Object) Object {
	e.store[name] = val
	delete(e.history, name)
	delete(e.varTypes, name)
	return val
}

// DefineTyped 定义带类型约束的变量
func (e *Environment) DefineTyped(name string, val Object, varType string) Object {
	e.store[name] = val
	delete(e.history, name)
	if varType != "" {
		e.varTypes[name] = varType
	} else {
		delete(e.varTypes, name)
	}
	return val
}

// GetVarType 返回变量声明的类型约束，空串表示无约束
func (e *Environment) GetVarType(name string) string {
	scope := e.findScope(name)
	if scope == nil {
		return ""
	}
	return scope.varTypes[name]
}

// DefineVarType 为已存在变量单独设置类型约束（不覆盖值）
func (e *Environment) DefineVarType(name string, varType string) {
	if varType != "" {
		e.varTypes[name] = varType
	} else {
		delete(e.varTypes, name)
	}
}

// DefineConst 在当前作用域定义常量（不可重新赋值）
func (e *Environment) DefineConst(name string, val Object) Object {
	e.store[name] = val
	e.consts[name] = true
	delete(e.history, name)
	delete(e.varTypes, name)
	return val
}

// Rollback 回滚变量到上一次赋值前的值。
// 返回回滚后的旧值和是否成功（无历史时返回 false）。
func (e *Environment) Rollback(name string) (Object, bool) {
	scope := e.findScope(name)
	if scope == nil {
		return nil, false
	}
	hist := scope.history[name]
	if len(hist) == 0 {
		return nil, false
	}
	oldVal := hist[len(hist)-1]
	scope.history[name] = hist[:len(hist)-1]
	scope.store[name] = oldVal
	return oldVal, true
}

func (e *Environment) isConst(name string) bool {
	return e.consts[name]
}

// Snapshot 返回当前作用域所有变量的只读副本（用于模块导出）
func (e *Environment) Snapshot() map[string]Object {
	out := make(map[string]Object, len(e.store))
	for k, v := range e.store {
		out[k] = v
	}
	return out
}
