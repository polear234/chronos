package parser

import (
	"testing"

	"picox/ast"
	"picox/lexer"
)

func TestLetStatements(t *testing.T) {
	input := `
let x = 5;
let y = 10;
let foobar = 838383;
`

	l := lexer.New(input)
	p := New(l)
	program := p.ParseProgram()
	checkParserErrors(t, p)

	if program == nil {
		t.Fatalf("ParseProgram() returned nil")
	}
	if len(program.Statements) != 3 {
		t.Fatalf("program.Statements does not contain 3 statements. got=%d", len(program.Statements))
	}

	tests := []struct {
		expectedIdentifier string
	}{
		{"x"},
		{"y"},
		{"foobar"},
	}

	for i, tt := range tests {
		stmt := program.Statements[i]
		if !testLetStatement(t, stmt, tt.expectedIdentifier) {
			return
		}
	}
}

func testLetStatement(t *testing.T, s ast.Statement, name string) bool {
	if s.TokenLiteral() != "let" {
		t.Errorf("s.TokenLiteral not 'let'. got=%q", s.TokenLiteral())
		return false
	}
	letStmt, ok := s.(*ast.LetStatement)
	if !ok {
		t.Errorf("s not *ast.LetStatement. got=%T", s)
		return false
	}
	if letStmt.Name.Value != name {
		t.Errorf("letStmt.Name.Value not '%s'. got=%s", name, letStmt.Name.Value)
		return false
	}
	return true
}

func TestReturnStatements(t *testing.T) {
	input := `
return 5;
return 10;
return 993322;
`
	l := lexer.New(input)
	p := New(l)
	program := p.ParseProgram()
	checkParserErrors(t, p)

	if len(program.Statements) != 3 {
		t.Fatalf("program.Statements does not contain 3 statements. got=%d", len(program.Statements))
	}

	for _, stmt := range program.Statements {
		returnStmt, ok := stmt.(*ast.ReturnStatement)
		if !ok {
			t.Errorf("stmt not *ast.ReturnStatement. got=%T", stmt)
			continue
		}
		if returnStmt.TokenLiteral() != "return" {
			t.Errorf("returnStmt.TokenLiteral not 'return', got %q", returnStmt.TokenLiteral())
		}
	}
}

func TestIfExpression(t *testing.T) {
	input := `if (x < y) { x }`
	l := lexer.New(input)
	p := New(l)
	program := p.ParseProgram()
	checkParserErrors(t, p)

	if len(program.Statements) != 1 {
		t.Fatalf("program.Statements does not contain 1 statement. got=%d", len(program.Statements))
	}
	stmt, ok := program.Statements[0].(*ast.ExpressionStatement)
	if !ok {
		t.Fatalf("stmt not *ast.ExpressionStatement. got=%T", program.Statements[0])
	}
	exp, ok := stmt.Expression.(*ast.IfExpression)
	if !ok {
		t.Fatalf("exp not *ast.IfExpression. got=%T", stmt.Expression)
	}
	if exp.Condition == nil {
		t.Fatal("if condition is nil")
	}
	if exp.Consequence == nil {
		t.Fatal("if consequence is nil")
	}
}

func TestIfWithoutParens(t *testing.T) {
	input := `if x < y { x }`
	l := lexer.New(input)
	p := New(l)
	program := p.ParseProgram()
	checkParserErrors(t, p)

	if len(program.Statements) != 1 {
		t.Fatalf("program.Statements does not contain 1 statement. got=%d", len(program.Statements))
	}
}

func TestFunctionLiteral(t *testing.T) {
	input := `fn add(x, y) { x + y; }`
	l := lexer.New(input)
	p := New(l)
	program := p.ParseProgram()
	checkParserErrors(t, p)

	if len(program.Statements) != 1 {
		t.Fatalf("program.Statements does not contain 1 statement. got=%d", len(program.Statements))
	}
	stmt, ok := program.Statements[0].(*ast.ExpressionStatement)
	if !ok {
		t.Fatalf("stmt not *ast.ExpressionStatement. got=%T", program.Statements[0])
	}
	fn, ok := stmt.Expression.(*ast.FunctionLiteral)
	if !ok {
		t.Fatalf("exp not *ast.FunctionLiteral. got=%T", stmt.Expression)
	}
	if fn.Name != "add" {
		t.Errorf("fn.Name not 'add'. got=%q", fn.Name)
	}
	if len(fn.Parameters) != 2 {
		t.Errorf("fn.Parameters wrong. want=2, got=%d", len(fn.Parameters))
	}
}

func TestClassLiteral(t *testing.T) {
	input := `
class Dog extends Animal {
    breed = ""
    init(name, breed) {
        super.init(name)
        this.breed = breed
    }
    speak() {
        return "bark"
    }
}`
	l := lexer.New(input)
	p := New(l)
	program := p.ParseProgram()
	checkParserErrors(t, p)

	if len(program.Statements) != 1 {
		t.Fatalf("program.Statements does not contain 1 statement. got=%d", len(program.Statements))
	}
	stmt, ok := program.Statements[0].(*ast.ExpressionStatement)
	if !ok {
		t.Fatalf("stmt not *ast.ExpressionStatement. got=%T", program.Statements[0])
	}
	cl, ok := stmt.Expression.(*ast.ClassLiteral)
	if !ok {
		t.Fatalf("exp not *ast.ClassLiteral. got=%T", stmt.Expression)
	}
	if cl.Name.Value != "Dog" {
		t.Errorf("class name wrong. got=%q", cl.Name.Value)
	}
	if cl.SuperClass == nil {
		t.Error("superclass is nil")
	}
	if len(cl.Methods) != 2 {
		t.Errorf("methods wrong. want=2, got=%d", len(cl.Methods))
	}
	if len(cl.Properties) != 1 {
		t.Errorf("properties wrong. want=1, got=%d", len(cl.Properties))
	}
}

func TestForLoop(t *testing.T) {
	input := `for x in [1, 2, 3] { print(x); }`
	l := lexer.New(input)
	p := New(l)
	program := p.ParseProgram()
	checkParserErrors(t, p)

	if len(program.Statements) != 1 {
		t.Fatalf("program.Statements does not contain 1 statement. got=%d", len(program.Statements))
	}
}

func TestPipeOperatorToCall(t *testing.T) {
	// let res = [1, 2] |> len() 应等价于 len([1, 2])
	input := `let res = [1, 2] |> len()`
	l := lexer.New(input)
	p := New(l)
	program := p.ParseProgram()
	checkParserErrors(t, p)

	if len(program.Statements) != 1 {
		t.Fatalf("expected 1 statement, got %d", len(program.Statements))
	}
	letStmt, ok := program.Statements[0].(*ast.LetStatement)
	if !ok {
		t.Fatalf("expected LetStatement, got %T", program.Statements[0])
	}
	call, ok := letStmt.Value.(*ast.CallExpression)
	if !ok {
		t.Fatalf("expected CallExpression, got %T", letStmt.Value)
	}
	// 函数名应是 len
	fnIdent, ok := call.Function.(*ast.Identifier)
	if !ok || fnIdent.Value != "len" {
		t.Fatalf("expected function 'len', got %v", call.Function)
	}
	// 参数应有 1 个（[1,2] 被插入到第 0 位）
	if len(call.Arguments) != 1 {
		t.Fatalf("expected 1 argument, got %d", len(call.Arguments))
	}
	// 参数应是数组字面量
	if _, ok := call.Arguments[0].(*ast.ArrayLiteral); !ok {
		t.Fatalf("expected ArrayLiteral as first arg, got %T", call.Arguments[0])
	}
}

func TestPipeOperatorTwoArgs(t *testing.T) {
	// a |> f(b) 等价于 f(a, b)
	input := `arr |> filter(x)`
	l := lexer.New(input)
	p := New(l)
	program := p.ParseProgram()
	checkParserErrors(t, p)

	stmt, ok := program.Statements[0].(*ast.ExpressionStatement)
	if !ok {
		t.Fatalf("expected ExpressionStatement, got %T", program.Statements[0])
	}
	call, ok := stmt.Expression.(*ast.CallExpression)
	if !ok {
		t.Fatalf("expected CallExpression, got %T", stmt.Expression)
	}
	fnIdent, ok := call.Function.(*ast.Identifier)
	if !ok || fnIdent.Value != "filter" {
		t.Fatalf("expected function 'filter', got %v", call.Function)
	}
	if len(call.Arguments) != 2 {
		t.Fatalf("expected 2 arguments, got %d", len(call.Arguments))
	}
	// 第 0 位是 arr（Identifier）
	if id, ok := call.Arguments[0].(*ast.Identifier); !ok || id.Value != "arr" {
		t.Fatalf("expected 'arr' as first arg, got %v", call.Arguments[0])
	}
	// 第 1 位是 x（Identifier）
	if id, ok := call.Arguments[1].(*ast.Identifier); !ok || id.Value != "x" {
		t.Fatalf("expected 'x' as second arg, got %v", call.Arguments[1])
	}
}

func TestPipeOperatorToIdentifier(t *testing.T) {
	// arr |> print 等价于 print(arr)
	input := `arr |> print`
	l := lexer.New(input)
	p := New(l)
	program := p.ParseProgram()
	checkParserErrors(t, p)

	stmt, ok := program.Statements[0].(*ast.ExpressionStatement)
	if !ok {
		t.Fatalf("expected ExpressionStatement, got %T", program.Statements[0])
	}
	call, ok := stmt.Expression.(*ast.CallExpression)
	if !ok {
		t.Fatalf("expected CallExpression, got %T", stmt.Expression)
	}
	fnIdent, ok := call.Function.(*ast.Identifier)
	if !ok || fnIdent.Value != "print" {
		t.Fatalf("expected function 'print', got %v", call.Function)
	}
	if len(call.Arguments) != 1 {
		t.Fatalf("expected 1 argument, got %d", len(call.Arguments))
	}
}

func TestPipeChain(t *testing.T) {
	// 链式管道：arr |> filter(x) |> map(y) 应等价于 map(filter(arr, x), y)
	input := `arr |> filter(x) |> map(y)`
	l := lexer.New(input)
	p := New(l)
	program := p.ParseProgram()
	checkParserErrors(t, p)

	stmt, ok := program.Statements[0].(*ast.ExpressionStatement)
	if !ok {
		t.Fatalf("expected ExpressionStatement, got %T", program.Statements[0])
	}
	// 最外层应是 map 调用
	call, ok := stmt.Expression.(*ast.CallExpression)
	if !ok {
		t.Fatalf("expected CallExpression, got %T", stmt.Expression)
	}
	outerFn, ok := call.Function.(*ast.Identifier)
	if !ok || outerFn.Value != "map" {
		t.Fatalf("expected outer function 'map', got %v", call.Function)
	}
	if len(call.Arguments) != 2 {
		t.Fatalf("expected 2 arguments, got %d", len(call.Arguments))
	}
	// 第 0 位应是内层 filter 调用
	inner, ok := call.Arguments[0].(*ast.CallExpression)
	if !ok {
		t.Fatalf("expected inner CallExpression, got %T", call.Arguments[0])
	}
	innerFn, ok := inner.Function.(*ast.Identifier)
	if !ok || innerFn.Value != "filter" {
		t.Fatalf("expected inner function 'filter', got %v", inner.Function)
	}
}

func checkParserErrors(t *testing.T, p *Parser) {
	errors := p.Errors()
	if len(errors) == 0 {
		return
	}
	t.Errorf("parser has %d errors", len(errors))
	for _, msg := range errors {
		t.Errorf("parser error: %q", msg)
	}
	t.FailNow()
}
