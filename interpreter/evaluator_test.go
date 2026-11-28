package interpreter

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"chronospp/lexer"
	"chronospp/object"
	"chronospp/parser"
)

func testEval(t *testing.T, input string) object.Object {
	l := lexer.New(input)
	p := parser.New(l)
	program := p.ParseProgram()
	if len(p.Errors()) > 0 {
		t.Fatalf("parse errors: %v", p.Errors())
	}
	env := object.NewEnvironment()
	return Eval(program, env)
}

func TestIntegerExpression(t *testing.T) {
	tests := []struct {
		input    string
		expected int64
	}{
		{"5", 5},
		{"10", 10},
		{"-5", -5},
		{"-10", -10},
		{"5 + 5 + 5 + 5 - 10", 10},
		{"2 * 2 * 2 * 2 * 2", 32},
		{"-50 + 100 + -50", 0},
		{"5 * 2 + 10", 20},
		{"5 + 2 * 10", 25},
		{"20 + 2 * -10", 0},
		{"50 / 2 * 2 + 10", 60},
		{"2 * (5 + 10)", 30},
		{"3 * 3 * 3 + 10", 37},
		{"(5 + 10 * 2 + 15 / 3) * 2 + -10", 50},
		{"5 % 3", 2},
		{"2 ** 10", 1024},
	}

	for _, tt := range tests {
		evaluated := testEval(t, tt.input)
		testIntegerObject(t, evaluated, tt.expected)
	}
}

func testIntegerObject(t *testing.T, obj object.Object, expected int64) bool {
	result, ok := obj.(*object.Integer)
	if !ok {
		t.Errorf("object is not Integer. got=%T (%+v)", obj, obj)
		return false
	}
	if result.Value != expected {
		t.Errorf("object has wrong value. got=%d, want=%d", result.Value, expected)
		return false
	}
	return true
}

func TestFloatExpression(t *testing.T) {
	tests := []struct {
		input    string
		expected float64
	}{
		{"3.14", 3.14},
		{"1.5 + 2.5", 4.0},
		{"10.0 / 4.0", 2.5},
	}

	for _, tt := range tests {
		evaluated := testEval(t, tt.input)
		result, ok := evaluated.(*object.Float)
		if !ok {
			t.Errorf("object is not Float. got=%T", evaluated)
			continue
		}
		if result.Value != tt.expected {
			t.Errorf("wrong value. got=%f, want=%f", result.Value, tt.expected)
		}
	}
}

func TestBooleanExpression(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{"true", true},
		{"false", false},
		{"1 < 2", true},
		{"1 > 2", false},
		{"1 == 1", true},
		{"1 != 1", false},
		{"true == true", true},
		{"false == false", true},
		{"true != false", true},
		{"(1 < 2) == true", true},
		{"!true", false},
		{"!false", true},
		{"not true", false},
	}

	for _, tt := range tests {
		evaluated := testEval(t, tt.input)
		result, ok := evaluated.(*object.Boolean)
		if !ok {
			t.Errorf("object is not Boolean. got=%T (%+v)", evaluated, evaluated)
			continue
		}
		if result.Value != tt.expected {
			t.Errorf("wrong value. got=%t, want=%t", result.Value, tt.expected)
		}
	}
}

func TestStringExpression(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{`"hello"`, "hello"},
		{`"hello" + " " + "world"`, "hello world"},
	}

	for _, tt := range tests {
		evaluated := testEval(t, tt.input)
		result, ok := evaluated.(*object.String)
		if !ok {
			t.Errorf("object is not String. got=%T", evaluated)
			continue
		}
		if result.Value != tt.expected {
			t.Errorf("wrong value. got=%q, want=%q", result.Value, tt.expected)
		}
	}
}

func TestIfElseExpressions(t *testing.T) {
	tests := []struct {
		input    string
		expected interface{}
	}{
		{"if true { 10 }", 10},
		{"if false { 10 }", nil},
		{"if 1 < 2 { 10 }", 10},
		{"if 1 > 2 { 10 } else { 20 }", 20},
		{"if 1 > 2 { 10 } else { 20 }", 20},
	}

	for _, tt := range tests {
		evaluated := testEval(t, tt.input)
		integer, ok := tt.expected.(int)
		if ok {
			testIntegerObject(t, evaluated, int64(integer))
		} else {
			if evaluated.Type() != object.NIL_OBJ {
				t.Errorf("object is not NIL. got=%T (%+v)", evaluated, evaluated)
			}
		}
	}
}

func TestReturnStatements(t *testing.T) {
	tests := []struct {
		input    string
		expected int64
	}{
		{"return 10;", 10},
		{"return 10; 9;", 10},
		{"return 2 * 5; 9;", 10},
		{"9; return 2 * 5; 9;", 10},
	}

	for _, tt := range tests {
		evaluated := testEval(t, tt.input)
		testIntegerObject(t, evaluated, tt.expected)
	}
}

func TestLetStatements(t *testing.T) {
	tests := []struct {
		input    string
		expected int64
	}{
		{"let a = 5; a;", 5},
		{"let a = 5 * 5; a;", 25},
		{"let a = 5; let b = a; b;", 5},
		{"let a = 5; let b = a; let c = a + b + 5; c;", 15},
	}

	for _, tt := range tests {
		testIntegerObject(t, testEval(t, tt.input), tt.expected)
	}
}

func TestFunctionApplication(t *testing.T) {
	tests := []struct {
		input    string
		expected int64
	}{
		{"let identity = fn(x) { x; }; identity(5);", 5},
		{"let add = fn(a, b) { a + b; }; add(2, 3);", 5},
		{"fn add(a, b) { a + b; } add(10, 5);", 15},
		{"let add = fn(a, b) { a + b; }; add(5 + 5, add(5, 5));", 20},
	}

	for _, tt := range tests {
		testIntegerObject(t, testEval(t, tt.input), tt.expected)
	}
}

func TestRecursion(t *testing.T) {
	input := `
fn factorial(n) {
    if n <= 1 {
        return 1
    }
    return n * factorial(n - 1)
}
factorial(5)`
	testIntegerObject(t, testEval(t, input), 120)
}

func TestClosure(t *testing.T) {
	input := `
let makeCounter = fn() {
    let count = 0
    fn() {
        count = count + 1
        count
    }
}
let counter = makeCounter()
counter()
counter()
counter()`
	testIntegerObject(t, testEval(t, input), 3)
}

func TestArrayLiterals(t *testing.T) {
	input := `[1, 2 * 2, 3 + 3]`
	evaluated := testEval(t, input)
	result, ok := evaluated.(*object.Array)
	if !ok {
		t.Fatalf("object is not Array. got=%T", evaluated)
	}
	if len(result.Elements) != 3 {
		t.Fatalf("array has wrong num of elements. got=%d", len(result.Elements))
	}
	testIntegerObject(t, result.Elements[0], 1)
	testIntegerObject(t, result.Elements[1], 4)
	testIntegerObject(t, result.Elements[2], 6)
}

func TestArrayIndexExpressions(t *testing.T) {
	tests := []struct {
		input    string
		expected interface{}
	}{
		{"[1, 2, 3][0]", 1},
		{"[1, 2, 3][1]", 2},
		{"[1, 2, 3][2]", 3},
		{"let i = 0; [1][i];", 1},
		{"[1, 2, 3][1 + 1];", 3},
		{"let myArray = [1, 2, 3]; myArray[2];", 3},
	}

	for _, tt := range tests {
		evaluated := testEval(t, tt.input)
		integer, ok := tt.expected.(int)
		if ok {
			testIntegerObject(t, evaluated, int64(integer))
		} else {
			if evaluated.Type() != object.NIL_OBJ {
				t.Errorf("object is not NIL. got=%T", evaluated)
			}
		}
	}
}

func TestHashLiterals(t *testing.T) {
	input := `
let two = "two"
let hash = {
    "one": 10 - 9,
    two: 1 + 1,
    "thr" + "ee": 6 / 2,
}
hash["one"] + hash["two"] + hash["three"]`
	testIntegerObject(t, testEval(t, input), 6)
}

func TestBuiltinFunctions(t *testing.T) {
	tests := []struct {
		input    string
		expected interface{}
	}{
		{`len("")`, 0},
		{`len("four")`, 4},
		{`len([1, 2, 3])`, 3},
		{`len({"a": 1})`, 1},
		{`type(1)`, "INTEGER"},
		{`type("hi")`, "STRING"},
		{`str(42)`, "42"},
		{`int(3.9)`, 3},
		{`float(3)`, 3.0},
	}

	for _, tt := range tests {
		evaluated := testEval(t, tt.input)
		switch expected := tt.expected.(type) {
		case int:
			testIntegerObject(t, evaluated, int64(expected))
		case string:
			if evaluated.Type() != object.STRING_OBJ {
				t.Errorf("expected string, got %T", evaluated)
				continue
			}
			if evaluated.Inspect() != expected {
				t.Errorf("wrong string. got=%q, want=%q", evaluated.Inspect(), expected)
			}
		case float64:
			result, ok := evaluated.(*object.Float)
			if !ok {
				t.Errorf("expected float, got %T", evaluated)
				continue
			}
			if result.Value != expected {
				t.Errorf("wrong float. got=%f, want=%f", result.Value, expected)
			}
		}
	}
}

func TestWhileLoop(t *testing.T) {
	input := `
let i = 0
let sum = 0
while i < 10 {
    sum = sum + i
    i = i + 1
}
sum`
	testIntegerObject(t, testEval(t, input), 45)
}

func TestForLoop(t *testing.T) {
	input := `
let sum = 0
for x in [1, 2, 3, 4, 5] {
    sum = sum + x
}
sum`
	testIntegerObject(t, testEval(t, input), 15)
}

func TestBreakContinue(t *testing.T) {
	input := `
let sum = 0
for x in [1, 2, 3, 4, 5, 6, 7, 8, 9, 10] {
    if x > 5 {
        break
    }
    if x % 2 == 0 {
        continue
    }
    sum = sum + x
}
sum`
	testIntegerObject(t, testEval(t, input), 9)
}

func TestClassAndInheritance(t *testing.T) {
	input := `
class Animal {
    name = ""
    init(name) { this.name = name }
    speak() { return this.name + " makes a sound" }
}
class Dog extends Animal {
    init(name) { super.init(name) }
    speak() { return this.name + " barks" }
}
let d = new Dog("Buddy")
d.speak()`
	evaluated := testEval(t, input)
	result, ok := evaluated.(*object.String)
	if !ok {
		t.Fatalf("expected string, got %T (%+v)", evaluated, evaluated)
	}
	if result.Value != "Buddy barks" {
		t.Errorf("wrong result. got=%q", result.Value)
	}
}

func TestErrorHandling(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"5 + true;", "type mismatch"},
		{"-true", "unknown operator"},
		{"let a = 5; a = b;", "identifier not found"},
		{"10 / 0", "division by zero"},
		{"const c = 1; c = 2;", "cannot reassign to constant"},
	}

	for _, tt := range tests {
		evaluated := testEval(t, tt.input)
		errObj, ok := evaluated.(*object.Error)
		if !ok {
			t.Errorf("no error object returned. got=%T (%+v)", evaluated, evaluated)
			continue
		}
		if errObj.Message == "" {
			t.Errorf("empty error message")
		}
	}
}

func TestConstImmutability(t *testing.T) {
	// 正常读取 const
	evaluated := testEval(t, "const c = 42; c;")
	testIntegerObject(t, evaluated, 42)

	// 重新赋值应报错
	evaluated = testEval(t, "const c = 42; c = 100;")
	if errObj, ok := evaluated.(*object.Error); !ok {
		t.Errorf("expected error on const reassignment, got=%T (%+v)", evaluated, evaluated)
	} else if errObj.Message != "cannot reassign to constant: c" {
		t.Errorf("wrong error message: %q", errObj.Message)
	}

	// 复合赋值也应报错
	evaluated = testEval(t, "const c = 1; c += 1;")
	if _, ok := evaluated.(*object.Error); !ok {
		t.Errorf("expected error on const +=, got=%T (%+v)", evaluated, evaluated)
	}
}

func TestArrayMethods(t *testing.T) {
	// lst.push() 方法调用
	tests := []struct {
		input    string
		expected int64
	}{
		{"let lst = [1, 2]; lst.push(3); lst.length();", 3},
		{"let lst = [1, 2, 3]; lst.pop(); lst.length();", 2},
		{"let lst = [1, 2, 3]; lst.shift(); lst.length();", 2},
		{"let lst = [1]; lst.unshift(0); lst.length();", 2},
		{"let lst = [1, 2, 3]; lst.indexOf(2);", 1},
		{"let lst = [1, 2, 3]; lst.indexOf(99);", -1},
	}
	for _, tt := range tests {
		testIntegerObject(t, testEval(t, tt.input), tt.expected)
	}

	// contains 返回布尔
	evaluated := testEval(t, "let lst = [1, 2, 3]; lst.contains(2);")
	if b, ok := evaluated.(*object.Boolean); !ok || !b.Value {
		t.Errorf("expected true, got=%T (%+v)", evaluated, evaluated)
	}

	// 链式调用
	evaluated = testEval(t, "let lst = [1, 2]; lst.push(3).push(4).length();")
	testIntegerObject(t, evaluated, 4)
}

func TestArrayHigherOrderMethods(t *testing.T) {
	// map
	evaluated := testEval(t, "let lst = [1, 2, 3]; lst.map(fn(x) { x * 2 });")
	arr, ok := evaluated.(*object.Array)
	if !ok {
		t.Fatalf("expected array, got=%T", evaluated)
	}
	if len(arr.Elements) != 3 {
		t.Fatalf("wrong length: %d", len(arr.Elements))
	}
	testIntegerObject(t, arr.Elements[0], 2)
	testIntegerObject(t, arr.Elements[1], 4)
	testIntegerObject(t, arr.Elements[2], 6)

	// filter
	evaluated = testEval(t, "let lst = [1, 2, 3, 4]; lst.filter(fn(x) { x % 2 == 0 });")
	arr, ok = evaluated.(*object.Array)
	if !ok {
		t.Fatalf("expected array, got=%T", evaluated)
	}
	if len(arr.Elements) != 2 {
		t.Fatalf("wrong length: %d", len(arr.Elements))
	}

	// reduce
	evaluated = testEval(t, "let lst = [1, 2, 3, 4]; lst.reduce(fn(acc, x) { acc + x }, 0);")
	testIntegerObject(t, evaluated, 10)

	// reduce 无初始值
	evaluated = testEval(t, "let lst = [1, 2, 3, 4]; lst.reduce(fn(acc, x) { acc + x });")
	testIntegerObject(t, evaluated, 10)

	// find
	evaluated = testEval(t, "let lst = [1, 2, 3, 4]; lst.find(fn(x) { x > 2 });")
	testIntegerObject(t, evaluated, 3)
}

func TestTryCatch(t *testing.T) {
	input := `
let result = "ok"
try {
    throw "boom"
} catch (e) {
    result = "caught"
}
result`
	evaluated := testEval(t, input)
	result, ok := evaluated.(*object.String)
	if !ok {
		t.Fatalf("expected string, got %T", evaluated)
	}
	if result.Value != "caught" {
		t.Errorf("wrong result. got=%q", result.Value)
	}
}

func TestInheritanceCycle(t *testing.T) {
	// B 先定义，A extends B，再重新定义 B extends A → 第二次定义 B 时沿父类链发现 A 的链上有 B，构成环
	input := `
class B {
    b() { return "b" }
}
class A extends B {
    a() { return "a" }
}
class B extends A {
    b2() { return "b2" }
}`
	evaluated := testEval(t, input)
	errObj, ok := evaluated.(*object.Error)
	if !ok {
		t.Fatalf("expected error on inheritance cycle, got=%T (%+v)", evaluated, evaluated)
	}
	if !strings.Contains(errObj.Message, "cycle") {
		t.Errorf("expected cycle error, got=%q", errObj.Message)
	}
}

func TestSelfInheritance(t *testing.T) {
	// class A extends A → 直接自继承
	input := `
class A { foo() { return 1 } }
class A extends A { bar() { return 2 } }`
	evaluated := testEval(t, input)
	if _, ok := evaluated.(*object.Error); !ok {
		t.Fatalf("expected error on self inheritance, got=%T (%+v)", evaluated, evaluated)
	}
}

func TestNormalInheritanceChain(t *testing.T) {
	// 正常三层继承不应报错
	input := `
class A { a() { return 1 } }
class B extends A { b() { return 2 } }
class C extends B { c() { return 3 } }
let x = new C()
x.a() + x.b() + x.c()`
	testIntegerObject(t, testEval(t, input), 6)
}

func TestVariableRollback(t *testing.T) {
	// 基本用法：let x = 5; x = 6; x.rollback() → 5
	input := `let x = 5; x = 6; x.rollback()`
	testIntegerObject(t, testEval(t, input), 5)

	// rollback 后 x 的值确实变回 5
	input = `let x = 5; x = 6; x.rollback(); x`
	testIntegerObject(t, testEval(t, input), 5)

	// 多次赋值可多次回滚（栈式）
	input = `let x = 1; x = 2; x = 3; x.rollback(); x.rollback()`
	testIntegerObject(t, testEval(t, input), 1)

	// 回滚后再赋值，新历史正确记录
	input = `let x = 1; x = 2; x.rollback(); x = 99; x.rollback()`
	testIntegerObject(t, testEval(t, input), 1)
}

func TestVariableRollbackNoHistory(t *testing.T) {
	// 从未重新赋值，rollback 应报错
	input := `let x = 5; x.rollback()`
	evaluated := testEval(t, input)
	if _, ok := evaluated.(*object.Error); !ok {
		t.Fatalf("expected error, got=%T (%+v)", evaluated, evaluated)
	}
}

func TestVariableRollbackInFunction(t *testing.T) {
	// 闭包内回滚外层变量
	input := `
let x = 5
fn setX() {
    x = 100
}
setX()
x.rollback()`
	testIntegerObject(t, testEval(t, input), 5)
}

func TestVariableRollbackString(t *testing.T) {
	// 非数值类型也支持回滚
	input := `let name = "a"; name = "b"; name.rollback()`
	evaluated := testEval(t, input)
	if s, ok := evaluated.(*object.String); !ok || s.Value != "a" {
		t.Fatalf("expected 'a', got=%T (%+v)", evaluated, evaluated)
	}
}

func TestWithSnapshotArray(t *testing.T) {
	// 数组：callback 里 push 后应被恢复到原长度
	input := `
let arr = [1, 2, 3]
let result = withSnapshot([arr], fn() {
    arr.push(4)
    arr.push(5)
    return arr.length()
})
arr.length()`
	// 最后一句是 arr.length()，应回到 3
	testIntegerObject(t, testEval(t, input), 3)
}

func TestWithSnapshotRestoresInPlace(t *testing.T) {
	// 验证原地恢复：同一个引用 arr 恢复后内容正确
	input := `
let arr = [1, 2, 3]
withSnapshot([arr], fn() {
    arr.push(99)
})
arr`
	evaluated := testEval(t, input)
	a, ok := evaluated.(*object.Array)
	if !ok {
		t.Fatalf("expected array, got=%T", evaluated)
	}
	if len(a.Elements) != 3 {
		t.Fatalf("expected length 3, got %d", len(a.Elements))
	}
	testIntegerObject(t, a.Elements[2], 3)
}

func TestWithSnapshotHash(t *testing.T) {
	// 字典：callback 里加键后应被恢复
	input := `
let d = {"a": 1}
withSnapshot([d], fn() {
    d["b"] = 2
})
d["b"]`
	// 恢复后 d["b"] 应为 nil
	evaluated := testEval(t, input)
	if evaluated.Type() != object.NIL_OBJ {
		t.Fatalf("expected nil, got=%T (%+v)", evaluated, evaluated)
	}
}

func TestWithSnapshotMultipleTargets(t *testing.T) {
	// 多个 targets 同时快照恢复
	input := `
let a = [1]
let b = [1, 2]
withSnapshot([a, b], fn() {
    a.push(2)
    b.push(3)
})
a.length() + b.length()`
	testIntegerObject(t, testEval(t, input), 3)
}

func TestTimelineCommitUndoArray(t *testing.T) {
	// Timeline：commit 后修改，undo 恢复
	input := `
let arr = [1, 2, 3]
let tl = Timeline(arr)
tl.commit()
arr.push(4)
tl.undo()
arr.length()`
	testIntegerObject(t, testEval(t, input), 3)
}

func TestTimelineMultipleUndo(t *testing.T) {
	// 多次 commit/undo
	input := `
let arr = [1]
let tl = Timeline(arr)
tl.commit()      // 历史: [1]
arr.push(2)
tl.commit()      // 历史: [1], [1,2]
arr.push(3)
tl.undo()        // 回到 [1,2]
arr.length()`
	testIntegerObject(t, testEval(t, input), 2)
}

func TestTimelineUndoHash(t *testing.T) {
	// Timeline 作用于字典
	input := `
let d = {"x": 1}
let tl = Timeline(d)
tl.commit()
d["y"] = 2
tl.undo()
d["y"]`
	evaluated := testEval(t, input)
	if evaluated.Type() != object.NIL_OBJ {
		t.Fatalf("expected nil, got=%T (%+v)", evaluated, evaluated)
	}
}

func TestTimelineUndoEmpty(t *testing.T) {
	// 无 commit 时 undo 应报错
	input := `
let arr = [1]
let tl = Timeline(arr)
tl.undo()`
	evaluated := testEval(t, input)
	if _, ok := evaluated.(*object.Error); !ok {
		t.Fatalf("expected error, got=%T (%+v)", evaluated, evaluated)
	}
}

func TestTravellerBasic(t *testing.T) {
	// let traveller int x = 1; x = 2; x.rollback() → 1
	input := `let traveller int x = 1; x = 2; x.rollback()`
	testIntegerObject(t, testEval(t, input), 1)

	// rollback 后 x 的值确实变回 1（透明读取）
	input = `let traveller int x = 1; x = 2; x.rollback(); x`
	testIntegerObject(t, testEval(t, input), 1)

	// 多次赋值多次回滚
	input = `let traveller int x = 1; x = 2; x = 3; x.rollback(); x.rollback()`
	testIntegerObject(t, testEval(t, input), 1)

	// traveller 变量参与运算（透明解包）
	input = `let traveller int x = 10; x = x + 5; x`
	testIntegerObject(t, testEval(t, input), 15)
}

func TestTravellerTypeCheck(t *testing.T) {
	// 声明 int，赋 string 应报错
	input := `let traveller int x = 1; x = "hello"`
	evaluated := testEval(t, input)
	if _, ok := evaluated.(*object.Error); !ok {
		t.Fatalf("expected type error, got=%T (%+v)", evaluated, evaluated)
	}

	// 声明 string，赋 int 应报错
	input = `let traveller string s = "a"; s = 42`
	evaluated = testEval(t, input)
	if _, ok := evaluated.(*object.Error); !ok {
		t.Fatalf("expected type error, got=%T (%+v)", evaluated, evaluated)
	}

	// 初值类型不匹配也应报错
	input = `let traveller int x = "not an int"`
	evaluated = testEval(t, input)
	if _, ok := evaluated.(*object.Error); !ok {
		t.Fatalf("expected type error, got=%T (%+v)", evaluated, evaluated)
	}
}

func TestTravellerEmptyHistory(t *testing.T) {
	// 从未赋值就 rollback → 报 "Traveller history is empty"
	input := `let traveller int x = 1; x.rollback()`
	evaluated := testEval(t, input)
	errObj, ok := evaluated.(*object.Error)
	if !ok {
		t.Fatalf("expected error, got=%T (%+v)", evaluated, evaluated)
	}
	if errObj.Message != "Traveller history is empty" {
		t.Errorf("wrong error message: %q", errObj.Message)
	}
}

func TestTravellerString(t *testing.T) {
	// 字符串 traveller
	input := `let traveller string s = "a"; s = "b"; s.rollback()`
	evaluated := testEval(t, input)
	if s, ok := evaluated.(*object.String); !ok || s.Value != "a" {
		t.Fatalf("expected 'a', got=%T (%+v)", evaluated, evaluated)
	}
}

func TestTypedVarWithoutTraveller(t *testing.T) {
	// 纯类型标注（无 traveller）：let int x = 1
	input := `let int x = 1; x`
	testIntegerObject(t, testEval(t, input), 1)

	// 类型不符赋值报错
	input = `let int x = 1; x = "s"`
	evaluated := testEval(t, input)
	if _, ok := evaluated.(*object.Error); !ok {
		t.Fatalf("expected type error, got=%T (%+v)", evaluated, evaluated)
	}
}

func TestTravellerMultiRollbackScenario(t *testing.T) {
	// 完整场景：let traveller int b = 1; b=2; b=3; b=4
	// b.rollback()→3; b.rollback()→2; print(b)→2; b.rollback()→1; b.rollback()→报错
	input := `
let traveller int b = 1
b = 2
b = 3
b = 4
b.rollback()
b.rollback()
b.rollback()
b.rollback()
b.rollback()`

	// 连续 5 次 rollback，最后一次应报错
	evaluated := testEval(t, input)
	if _, ok := evaluated.(*object.Error); !ok {
		t.Fatalf("expected error on empty history, got=%T (%+v)", evaluated, evaluated)
	}
}

func TestTravellerMultiRollbackValues(t *testing.T) {
	// 逐步验证回滚值
	input := `
let traveller int b = 1
b = 2
b = 3
b = 4
b.rollback()
b.rollback()
b`
	// b.rollback()→3, b.rollback()→2, 最后 b=2
	testIntegerObject(t, testEval(t, input), 2)
}

func TestImportModule(t *testing.T) {
	// 在临时目录创建模块文件和主文件，验证 import
	dir := t.TempDir()

	// 写模块文件
	modPath := filepath.Join(dir, "math.chpp")
	modSrc := `
let PI = 3.14159
fn square(x) { return x * x }
fn add(a, b) { return a + b }
let msg = "hello from math"`
	if err := os.WriteFile(modPath, []byte(modSrc), 0644); err != nil {
		t.Fatalf("write module: %v", err)
	}

	// 主文件 import 并调用
	mainSrc := `
import "math" as m
m.square(5) + m.add(1, 2)`

	l := lexer.New(mainSrc)
	p := parser.New(l)
	program := p.ParseProgram()
	if len(p.Errors()) > 0 {
		t.Fatalf("parse errors: %v", p.Errors())
	}
	env := object.NewEnvironment()
	env.ImportDir = dir
	result := Eval(program, env)

	// m.square(5)=25, m.add(1,2)=3, 25+3=28
	testIntegerObject(t, result, 28)
}

func TestImportDefaultAlias(t *testing.T) {
	dir := t.TempDir()
	modPath := filepath.Join(dir, "util.chpp")
	modSrc := `fn double(x) { return x * 2 }`
	if err := os.WriteFile(modPath, []byte(modSrc), 0644); err != nil {
		t.Fatalf("write module: %v", err)
	}

	// 无 as 别名，默认用文件名 util
	mainSrc := `import "util" util.double(21)`
	l := lexer.New(mainSrc)
	p := parser.New(l)
	program := p.ParseProgram()
	if len(p.Errors()) > 0 {
		t.Fatalf("parse errors: %v", p.Errors())
	}
	env := object.NewEnvironment()
	env.ImportDir = dir
	result := Eval(program, env)
	testIntegerObject(t, result, 42)
}

func TestImportModuleAccessString(t *testing.T) {
	dir := t.TempDir()
	modPath := filepath.Join(dir, "msg.chpp")
	modSrc := `let greeting = "hi"`
	if err := os.WriteFile(modPath, []byte(modSrc), 0644); err != nil {
		t.Fatalf("write module: %v", err)
	}

	mainSrc := `import "msg" as m
m.greeting`
	l := lexer.New(mainSrc)
	p := parser.New(l)
	program := p.ParseProgram()
	if len(p.Errors()) > 0 {
		t.Fatalf("parse errors: %v", p.Errors())
	}
	env := object.NewEnvironment()
	env.ImportDir = dir
	result := Eval(program, env)
	if s, ok := result.(*object.String); !ok || s.Value != "hi" {
		t.Fatalf("expected 'hi', got=%T (%+v)", result, result)
	}
}

func TestCircularImport(t *testing.T) {
	dir := t.TempDir()
	// a.chpp 导入 b.chpp，b.chpp 导入 a.chpp → 循环
	os.WriteFile(filepath.Join(dir, "a.chpp"), []byte(`import "b"`), 0644)
	os.WriteFile(filepath.Join(dir, "b.chpp"), []byte(`import "a"`), 0644)

	mainSrc := `import "a" as a`
	l := lexer.New(mainSrc)
	p := parser.New(l)
	program := p.ParseProgram()
	if len(p.Errors()) > 0 {
		t.Fatalf("parse errors: %v", p.Errors())
	}
	env := object.NewEnvironment()
	env.ImportDir = dir
	result := Eval(program, env)
	if _, ok := result.(*object.Error); !ok {
		t.Fatalf("expected circular import error, got=%T (%+v)", result, result)
	}
}

func TestChineseIdentifiers(t *testing.T) {
	// 中文变量名 / 函数名 / 类名 / 循环变量
	input := `
let 问候 = "你好"
let 数字 = 42
fn 加倍(数) { return 数 * 2 }
class 猫 {
    speak() { return "喵" }
}
let 水果 = ["苹果", "香蕉"]
let 计数 = 0
for 果 in 水果 {
    计数 = 计数 + 1
}
let 喵 = new 猫()
加倍(21) + 计数`
	// 42 + 2 = 44
	testIntegerObject(t, testEval(t, input), 44)
}

func TestChineseStringRunes(t *testing.T) {
	// 字符串按 rune 计数，中文长度正确
	input := `len("你好")`
	testIntegerObject(t, testEval(t, input), 2)

	input = `"你好"[1]`
	if s, ok := testEval(t, input).(*object.String); !ok || s.Value != "好" {
		t.Fatalf("expected 好, got %T", testEval(t, input))
	}
}

func TestGlobalArrayHOF(t *testing.T) {
	// 全局函数形态：map(arr, fn)
	input := `map([1, 2, 3], fn(x) { x * 2 })`
	arr, ok := testEval(t, input).(*object.Array)
	if !ok || len(arr.Elements) != 3 {
		t.Fatalf("expected 3 elements, got %v", arr)
	}
	testIntegerObject(t, arr.Elements[2], 6)

	// 管道形态：arr |> map(fn)
	input = `[1, 2, 3] |> map(fn(x) { x + 1 })`
	arr, ok = testEval(t, input).(*object.Array)
	if !ok || len(arr.Elements) != 3 {
		t.Fatalf("expected 3 elements, got %v", arr)
	}
	testIntegerObject(t, arr.Elements[0], 2)

	// filter 全局形态
	input = `filter([1, 2, 3, 4], fn(x) { x % 2 == 0 })`
	arr, ok = testEval(t, input).(*object.Array)
	if !ok || len(arr.Elements) != 2 {
		t.Fatalf("expected 2 elements, got %v", arr)
	}

	// reduce 全局形态
	input = `reduce([1, 2, 3], fn(acc, x) { acc + x }, 0)`
	testIntegerObject(t, testEval(t, input), 6)
}

func TestUserOverrideGlobalHOF(t *testing.T) {
	// 用户自定义同名函数应优先于内置全局 HOF
	input := `
fn map(a, b) { return "custom" }
map(1, 2)`
	evaluated := testEval(t, input)
	if s, ok := evaluated.(*object.String); !ok || s.Value != "custom" {
		t.Fatalf("expected 'custom', got %T (%+v)", evaluated, evaluated)
	}
}

func TestForElse(t *testing.T) {
	// 自然结束：else 执行，标记置 1
	t.Run("natural", func(t *testing.T) {
		input := `
let flag = 0
for x in [1, 2] {
} else {
    flag = 1
}
flag`
		testIntegerObject(t, testEval(t, input), 1)
	})
	// break 跳过 else：flag 保持 0
	t.Run("break", func(t *testing.T) {
		input := `
let flag = 0
for x in [1, 2, 3] {
    if x == 2 { break }
} else {
    flag = 1
}
flag`
		testIntegerObject(t, testEval(t, input), 0)
	})
	// 空迭代：else 执行
	t.Run("empty", func(t *testing.T) {
		input := `
let flag = 0
for x in [] {
} else {
    flag = 1
}
flag`
		testIntegerObject(t, testEval(t, input), 1)
	})
}

func TestWhileElse(t *testing.T) {
	// 自然结束：else 执行
	t.Run("natural", func(t *testing.T) {
		input := `
let i = 0
let flag = 0
while i < 2 {
    i = i + 1
} else {
    flag = 1
}
flag`
		testIntegerObject(t, testEval(t, input), 1)
	})
	// break 跳过 else：flag 保持 0
	t.Run("break", func(t *testing.T) {
		input := `
let i = 0
let flag = 0
while i < 5 {
    i = i + 1
    if i == 2 { break }
} else {
    flag = 1
}
flag`
		testIntegerObject(t, testEval(t, input), 0)
	})
}

func TestBooleanLogic(t *testing.T) {
	tests := []struct {
		input    string
		expected int64
	}{
		{`if true || false { 1 } else { 0 }`, 1},
		{`if true or false { 1 } else { 0 }`, 1},
		{`if false || false { 1 } else { 0 }`, 0},
		{`if false or false { 1 } else { 0 }`, 0},
		{`if true && false { 1 } else { 0 }`, 0},
		{`if true and false { 1 } else { 0 }`, 0},
		{`if true && true { 1 } else { 0 }`, 1},
		{`if 0 || 5 { 1 } else { 0 }`, 1},
		{`if true && 1 { 1 } else { 0 }`, 1},
		{`if false || 3 { 1 } else { 0 }`, 1},
	}
	for _, tt := range tests {
		testIntegerObject(t, testEval(t, tt.input), tt.expected)
	}
}

func TestVirtualModuleFS(t *testing.T) {
	vfs := map[string]string{
		"lib.chpp": "let add = fn(a, b) { return a + b }\n",
	}
	SetModuleReader(func(p string) ([]byte, error) {
		for k, v := range vfs {
			if p == k || strings.HasSuffix(p, k) {
				return []byte(v), nil
			}
		}
		return nil, fmt.Errorf("not found: %s", p)
	})
	SetModuleExists(func(p string) bool {
		for k := range vfs {
			if p == k || strings.HasSuffix(p, k) {
				return true
			}
		}
		return false
	})
	defer func() {
		SetModuleReader(nil)
		SetModuleExists(nil)
	}()

	input := "import \"lib\" as L\nL.add(3, 4)"
	testIntegerObject(t, testEval(t, input), 7)
}

func TestCStyleFunction(t *testing.T) {
	tests := []struct {
		input    string
		expected int64
	}{
		// 基础：带类型参数
		{`int add(int a, int b) { return a + b }
add(3, 4)`, 7},
		// 递归
		{`int fact(int n) { if n <= 1 { return 1 } return n * fact(n - 1) }
fact(5)`, 120},
		// 无参数
		{`int noparam() { return 42 }
noparam()`, 42},
		// 混合：一个带类型一个不带
		{`int mixed(int a, b) { return a + b }
mixed(1, 2)`, 3},
		// 与旧 fn 语法共存
		{`let old = fn(x) { return x * 2 }
old(5)`, 10},
		// 类型隐式转换：字符串 "42" 转 int
		{`int conv(int a) { return a }
conv("42")`, 42},
	}
	for _, tt := range tests {
		testIntegerObject(t, testEval(t, tt.input), tt.expected)
	}
}

func TestStreamOutput(t *testing.T) {
	tests := []struct {
		input    string
		expected int64
	}{
		// prints << 字面量，最后返回整数验证链路没坏
		{`prints << "hi"
1`, 1},
		// prints << 表达式
		{`prints << 2 + 3
2`, 2},
		// 变量
		{`let x = 7
prints << x
3`, 3},
	}
	for _, tt := range tests {
		testIntegerObject(t, testEval(t, tt.input), tt.expected)
	}
}

func TestInferVarType(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{`let a = 1`, "int"},
		{`let b = 1.5`, "float"},
		{`let c = "s"`, "string"},
		{`let d = true`, "bool"},
	}
	for _, tt := range tests {
		env := object.NewEnvironment()
		l := lexer.New(tt.input)
		p := parser.New(l)
		prog := p.ParseProgram()
		if len(p.Errors()) > 0 {
			t.Fatalf("parse %q: %v", tt.input, p.Errors())
		}
		Eval(prog, env)
		// 取第一个 let 的变量名
		name := string(tt.input[4])
		if got := env.GetVarType(name); got != tt.want {
			t.Errorf("%q: GetVarType(%s) = %q, want %q", tt.input, name, got, tt.want)
		}
	}
}

func TestStartParallel(t *testing.T) {
	tests := []struct {
		input string
		want  []int64
	}{
		// 单函数
		{`fn one() { return 42 }
start(one)`, []int64{42}},
		// 多函数按顺序返回
		{`fn x() { return 1 }
fn y() { return 2 }
fn z() { return 3 }
start(x, y, z)`, []int64{1, 2, 3}},
		// 闭包传参
		{`fn add(a, b) { return a + b }
start(fn() { return add(3, 4) }, fn() { return add(10, 20) })`, []int64{7, 30}},
		// 直接带参数的调用
		{`fn add(a, b) { return a + b }
fn mul(a, b) { return a * b }
start(add(1, 2), mul(3, 4))`, []int64{3, 12}},
		// 函数对象与带参调用混合
		{`fn noparam() { return 99 }
fn add(a, b) { return a + b }
start(noparam, add(10, 20))`, []int64{99, 30}},
		// 参数是表达式
		{`fn calc(n) { return n * 2 }
start(calc(3 + 4), calc(10 / 2))`, []int64{14, 10}},
	}
	for _, tt := range tests {
		result := testEval(t, tt.input)
		arr, ok := result.(*object.Array)
		if !ok {
			t.Fatalf("%q: 期望数组，得到 %T (%s)", tt.input, result, result.Inspect())
		}
		if len(arr.Elements) != len(tt.want) {
			t.Fatalf("%q: 长度 %d，期望 %d", tt.input, len(arr.Elements), len(tt.want))
		}
		for i, w := range tt.want {
			testIntegerObject(t, arr.Elements[i], w)
		}
	}
}

func TestStartErrors(t *testing.T) {
	// 没有参数
	if result := testEval(t, `start()`); !isError(result) {
		t.Error("start() 无参数应该报错")
	}
	// 参数不是函数
	if result := testEval(t, `start(42)`); !isError(result) {
		t.Error("start(42) 应该报错")
	}
}

func TestEndlAndChaining(t *testing.T) {
	// endl 应该输出换行；链式 << 应该依次输出
	env := object.NewEnvironment()
	var buf strings.Builder
	old := SetOutput(&buf)
	defer SetOutput(old)

	src := `prints << "a" << endl
prints << "b"`
	l := lexer.New(src)
	p := parser.New(l)
	prog := p.ParseProgram()
	if len(p.Errors()) > 0 {
		t.Fatalf("parse: %v", p.Errors())
	}
	result := Eval(prog, env)
	if isError(result) {
		t.Fatalf("eval: %s", result.Inspect())
	}
	if buf.String() != "a\nb" {
		t.Errorf("输出为 %q，期望 %q", buf.String(), "a\nb")
	}
}

func TestStrongTyping(t *testing.T) {
	// 应该报错的场景
	errors := []struct {
		input string
		desc  string
	}{
		{`int add(int a, int b) { return a + b }
add(1)`, "参数个数不足"},
		{`int add(int a, int b) { return a + b }
add(1, 2, 3)`, "参数个数过多"},
		{`int add(int a, int b) { return a + b }
add("x", "y")`, "参数类型不符"},
		{`int bad() { return "hello" }
bad()`, "返回值类型不符"},
		{`void hello() { return 42 }
hello()`, "void 函数返回值"},
	}
	for _, tt := range errors {
		result := testEval(t, tt.input)
		if !isError(result) {
			t.Errorf("%s：期望报错，得到 %s", tt.desc, result.Inspect())
		}
	}

	// 应该正常的场景
	valid := []struct {
		input string
		want  int64
	}{
		{`int add(int a, int b) { return a + b }
add(1, 2)`, 3},
		{`int conv(int a) { return a }
conv("42")`, 42},
		{`float half(float x) { return x / 2 }
int(half(7))`, 3},
		{`any pass(any x) { return x }
pass(42)`, 42},
	}
	for _, tt := range valid {
		testIntegerObject(t, testEval(t, tt.input), tt.want)
	}
}

func TestPointer(t *testing.T) {
	// 基础：取地址、解引用
	tests := []struct {
		input    string
		expected int64
	}{
		{`let x = 42
let p = &x
*p`, 42},
		{`let x = 1
let p = &x
*p = 100
x`, 100},
		// 通过函数修改
		{`int inc(int *p) { *p = *p + 1; return *p }
let n = 10
inc(&n)
n`, 11},
		// 多个指针指向同一变量
		{`let x = 5
let p = &x
let q = &x
*p = 10
*q = 20
x`, 20},
	}
	for _, tt := range tests {
		testIntegerObject(t, testEval(t, tt.input), tt.expected)
	}
}

func TestPointerErrors(t *testing.T) {
	// 对非指针解引用
	if r := testEval(t, `*42`); !isError(r) {
		t.Error("*42 应该报错")
	}
	// 对未定义变量取地址
	if r := testEval(t, `&notdefined`); !isError(r) {
		t.Error("&notdefined 应该报错")
	}
	// 指针参数类型不符
	if r := testEval(t, `int f(int *p) { return *p }
f(42)`); !isError(r) {
		t.Error("传非指针给指针参数应该报错")
	}
}

func TestInterface(t *testing.T) {
	// 正确实现
	ok := `interface Speaker { fn speak() }
class Dog implements Speaker {
    fn speak() { return "woof" }
}
let d = new Dog()
d.speak()`
	if r := testEval(t, ok); r.Inspect() != "woof" {
		t.Errorf("期望 woof，得到 %s", r.Inspect())
	}

	// 缺少方法
	missing := `interface Speaker { fn speak() fn name() }
class Cat implements Speaker {
    fn speak() { return "meow" }
}`
	if r := testEval(t, missing); !isError(r) {
		t.Error("缺少方法应该报错")
	}

	// 接口不存在
	notExist := `class Bad implements NotExist { }`
	if r := testEval(t, notExist); !isError(r) {
		t.Error("接口不存在应该报错")
	}

	// 多接口
	multi := `interface A { fn a() }
interface B { fn b() }
class C implements A, B {
    fn a() { return 1 }
    fn b() { return 2 }
}
let c = new C()
c.a() + c.b()`
	testIntegerObject(t, testEval(t, multi), 3)

	// 继承 + 接口
	inherit := `interface S { fn s() }
class Base { fn h() { return 10 } }
class Sub extends Base implements S {
    fn s() { return 20 }
}
let x = new Sub()
x.h() + x.s()`
	testIntegerObject(t, testEval(t, inherit), 30)
}

func TestFunctionOverload(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		// 同名不同类型
		{`int add(int a, int b) { return a + b }
string add(string a, string b) { return a + b }
add(1, 2)`, "3"},
		{`int add(int a, int b) { return a + b }
string add(string a, string b) { return a + b }
add("x", "y")`, "xy"},
		// 参数个数不同
		{`int f(int a) { return a }
int f(int a, int b) { return a + b }
f(5)`, "5"},
		{`int f(int a) { return a }
int f(int a, int b) { return a + b }
f(5, 3)`, "8"},
		// int 传给 float 参数
		{`float g(float x) { return x * 2 }
g(3)`, "6"},
		// 三种类型
		{`int h(int x) { return 1 }
string h(string x) { return "two" }
int h(bool x) { return 3 }
h(true)`, "3"},
	}
	for _, tt := range tests {
		result := testEval(t, tt.input)
		if isError(result) {
			t.Errorf("%q 出错: %s", tt.input, result.Inspect())
			continue
		}
		if result.Inspect() != tt.expected {
			t.Errorf("%q = %s，期望 %s", tt.input, result.Inspect(), tt.expected)
		}
	}
}

func TestOverloadWithRecursion(t *testing.T) {
	src := `int fact(int n) {
    if n <= 1 { return 1 }
    return n * fact(n - 1)
}
fact(5)`
	testIntegerObject(t, testEval(t, src), 120)
}

func TestStruct(t *testing.T) {
	// 基础：字段零值 + 读写
	src := `struct Point {
    int x
    int y
}
let p = new Point()
p.x = 10
p.y = 20
p.x + p.y`
	testIntegerObject(t, testEval(t, src), 30)

	// 分号写法
	semi := `struct P { int a; int b }
let p = new P()
p.a = 1
p.b = 2
p.a + p.b`
	testIntegerObject(t, testEval(t, semi), 3)

	// 带方法
	method := `struct Person {
    string name
    fn init(n) { this.name = n }
    fn greet() { return "hi " + this.name }
}
let p = new Person("Tom")
p.greet()`
	if r := testEval(t, method); r.Inspect() != "hi Tom" {
		t.Errorf("期望 hi Tom，得到 %s", r.Inspect())
	}

	// struct 作为参数类型
	param := `struct Vec { int x }
int getX(Vec v) { return v.x }
let v = new Vec()
v.x = 42
getX(v)`
	testIntegerObject(t, testEval(t, param), 42)
}

func TestOperatorOverload(t *testing.T) {
	// 向量加法
	add := `class Vec {
    fn init(x, y) { this.x = x; this.y = y }
    fn operator+(o) { return new Vec(this.x + o.x, this.y + o.y) }
}
let a = new Vec(1, 2)
let b = new Vec(3, 4)
let c = a + b
c.x * 10 + c.y`
	testIntegerObject(t, testEval(t, add), 46)

	// 标量乘法
	mul := `class N {
    fn init(v) { this.v = v }
    fn operator*(k) { return new N(this.v * k) }
}
let a = new N(5)
(a * 3).v`
	testIntegerObject(t, testEval(t, mul), 15)

	// 比较运算符
	cmp := `class N {
    fn init(v) { this.v = v }
    fn operator<(o) { return this.v < o.v }
}
let a = new N(1)
let b = new N(2)
if a < b { 1 } else { 0 }`
	testIntegerObject(t, testEval(t, cmp), 1)

	// 减法
	sub := `class N {
    fn init(v) { this.v = v }
    fn operator-(o) { return new N(this.v - o.v) }
}
let a = new N(10)
let b = new N(3)
(a - b).v`
	testIntegerObject(t, testEval(t, sub), 7)
}

func TestGenerics(t *testing.T) {
	// 基础泛型
	basic := `class Box<T> {
    fn init(T v) { this.v = v }
    T get() { return this.v }
}
let a = new Box<int>(42)
a.get()`
	testIntegerObject(t, testEval(t, basic), 42)

	// 不同类型实例化
	multi := `class Box<T> {
    fn init(T v) { this.v = v }
    T get() { return this.v }
}
let s = new Box<string>("hi")
s.get()`
	if r := testEval(t, multi); r.Inspect() != "hi" {
		t.Errorf("期望 hi，得到 %s", r.Inspect())
	}

	// 多个类型参数
	pair := `class Pair<A, B> {
    fn init(A a, B b) { this.a = a; this.b = b }
    A first() { return this.a }
    B second() { return this.b }
}
let p = new Pair<int, string>(1, "one")
p.first() * 10`
	testIntegerObject(t, testEval(t, pair), 10)

	// 类型错误
	bad := `class Box<T> {
    fn init(T v) { this.v = v }
}
new Box<int>("not int")`
	if r := testEval(t, bad); !isError(r) {
		t.Error("类型不匹配应该报错")
	}

	// 缺少类型参数
	missing := `class Box<T> {
    fn init(T v) { this.v = v }
}
new Box(1)`
	if r := testEval(t, missing); !isError(r) {
		t.Error("泛型类缺类型参数应该报错")
	}
}

func TestArrayConcat(t *testing.T) {
	src := `let a = [1, 2]
let b = [3, 4]
let c = a + b
len(c)`
	testIntegerObject(t, testEval(t, src), 4)
}
