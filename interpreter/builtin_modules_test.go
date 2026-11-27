//go:build !js

package interpreter

import (
	"os"
	"path/filepath"
	"testing"

	"chronospp/object"
)

func TestBuiltinModulesRegistered(t *testing.T) {
	for _, name := range []string{"os", "math"} {
		if lookupBuiltinModule(name) == nil {
			t.Errorf("内置模块 %q 没有注册", name)
		}
	}
	if lookupBuiltinModule("nonexistent") != nil {
		t.Error("不存在的模块不应返回对象")
	}
}

func TestOSModuleFileOps(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "test.txt")

	mod := lookupBuiltinModule("os")
	if mod == nil {
		t.Fatal("os 模块不可用")
	}

	get := func(name string) *object.Builtin {
		obj, ok := mod.Exports[name]
		if !ok {
			t.Fatalf("os.%s 不存在", name)
		}
		b, ok := obj.(*object.Builtin)
		if !ok {
			t.Fatalf("os.%s 不是函数", name)
		}
		return b
	}

	// 写入
	res := get("writeFile").Fn(&object.String{Value: file}, &object.String{Value: "hello"})
	if isError(res) {
		t.Fatalf("writeFile 失败: %s", res.Inspect())
	}

	// 读取
	res = get("readFile").Fn(&object.String{Value: file})
	s, ok := res.(*object.String)
	if !ok || s.Value != "hello" {
		t.Fatalf("readFile 返回 %v，期望 hello", res.Inspect())
	}

	// 追加
	get("appendFile").Fn(&object.String{Value: file}, &object.String{Value: " world"})
	res = get("readFile").Fn(&object.String{Value: file})
	if s := res.(*object.String); s.Value != "hello world" {
		t.Fatalf("appendFile 后内容为 %q", s.Value)
	}

	// 存在性
	res = get("exists").Fn(&object.String{Value: file})
	if b, ok := res.(*object.Boolean); !ok || !b.Value {
		t.Error("exists 对已存在的文件应返回 true")
	}

	// 删除
	get("remove").Fn(&object.String{Value: file})
	if _, err := os.Stat(file); !os.IsNotExist(err) {
		t.Error("remove 后文件仍存在")
	}
}

func TestMathModule(t *testing.T) {
	mod := lookupBuiltinModule("math")
	if mod == nil {
		t.Fatal("math 模块不可用")
	}

	sqrt, ok := mod.Exports["sqrt"].(*object.Builtin)
	if !ok {
		t.Fatal("math.sqrt 不存在")
	}
	res := sqrt.Fn(&object.Integer{Value: 16})
	f, ok := res.(*object.Float)
	if !ok || f.Value != 4 {
		t.Fatalf("sqrt(16) = %v，期望 4", res.Inspect())
	}

	// 常量
	pi, ok := mod.Exports["PI"].(*object.Float)
	if !ok || pi.Value < 3.14 || pi.Value > 3.15 {
		t.Errorf("PI 值异常: %v", pi.Inspect())
	}

	// abs 保持整数类型
	abs, _ := mod.Exports["abs"].(*object.Builtin)
	res = abs.Fn(&object.Integer{Value: -42})
	i, ok := res.(*object.Integer)
	if !ok || i.Value != 42 {
		t.Fatalf("abs(-42) = %v，期望整数 42", res.Inspect())
	}
}
