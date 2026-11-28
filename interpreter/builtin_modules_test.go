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

func TestRequestsModuleRegistered(t *testing.T) {
	mod := lookupBuiltinModule("requests")
	if mod == nil {
		t.Fatal("requests 模块没有注册")
	}
	for _, fn := range []string{"get", "post", "put", "delete", "head", "getJson", "download", "quote"} {
		if _, ok := mod.Exports[fn]; !ok {
			t.Errorf("requests.%s 不存在", fn)
		}
	}
}

func TestJSONConversion(t *testing.T) {
	// JSON → Chronos 对象
	obj := jsonToObject(`{"name":"test","n":42,"ok":true,"arr":[1,2,3]}`)
	h, ok := obj.(*object.Hash)
	if !ok {
		t.Fatalf("期望 Hash，得到 %T", obj)
	}

	// 找 name 字段
	found := false
	for _, pair := range h.Pairs {
		if k, ok := pair.Key.(*object.String); ok && k.Value == "name" {
			if s, ok := pair.Value.(*object.String); ok && s.Value == "test" {
				found = true
			}
		}
	}
	if !found {
		t.Error("name 字段解析错误")
	}
}

func TestObjectToGo(t *testing.T) {
	// Chronos 对象 → Go 值（用于 POST JSON）
	h := &object.Hash{Pairs: map[object.HashKey]object.HashPair{}}
	key := &object.String{Value: "n"}
	h.Pairs[key.HashKey()] = object.HashPair{Key: key, Value: &object.Integer{Value: 42}}

	v := objectToGo(h)
	m, ok := v.(map[string]any)
	if !ok {
		t.Fatalf("期望 map，得到 %T", v)
	}
	if m["n"] != int64(42) {
		t.Errorf("n = %v，期望 42", m["n"])
	}
}
