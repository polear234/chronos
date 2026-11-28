package ast

import (
	"bytes"
	"chronospp/token"
	"strings"
)

// Node 所有AST节点的基础接口
type Node interface {
	TokenLiteral() string
	String() string
}

// Statement 语句接口
type Statement interface {
	Node
	statementNode()
}

// Expression 表达式接口
type Expression interface {
	Node
	expressionNode()
}

// Program 程序根节点
type Program struct {
	Statements []Statement
}

func (p *Program) TokenLiteral() string {
	if len(p.Statements) > 0 {
		return p.Statements[0].TokenLiteral()
	}
	return ""
}

func (p *Program) String() string {
	var out bytes.Buffer
	for _, s := range p.Statements {
		out.WriteString(s.String())
	}
	return out.String()
}

// LetStatement let 语句
type LetStatement struct {
	Token       token.Token // let 关键字
	Name        *Identifier
	IsTraveller bool   // 是否为 traveller 修饰变量（支持 rollback）
	VarType     string // 类型标注（"int"/"string"/"dict"/"array"/"float"/"bool"），空串表示无标注
	Value       Expression
}

func (ls *LetStatement) statementNode()       {}
func (ls *LetStatement) TokenLiteral() string { return ls.Token.Literal }
func (ls *LetStatement) String() string {
	var out bytes.Buffer
	out.WriteString(ls.TokenLiteral() + " ")
	if ls.IsTraveller {
		out.WriteString("traveller ")
	}
	if ls.VarType != "" {
		out.WriteString(ls.VarType + " ")
	}
	out.WriteString(ls.Name.String())
	out.WriteString(" = ")
	if ls.Value != nil {
		out.WriteString(ls.Value.String())
	}
	out.WriteString(";")
	return out.String()
}

// ConstStatement const 语句
type ConstStatement struct {
	Token token.Token
	Name  *Identifier
	Value Expression
}

func (cs *ConstStatement) statementNode()       {}
func (cs *ConstStatement) TokenLiteral() string { return cs.Token.Literal }
func (cs *ConstStatement) String() string {
	var out bytes.Buffer
	out.WriteString(cs.TokenLiteral() + " ")
	out.WriteString(cs.Name.String())
	out.WriteString(" = ")
	if cs.Value != nil {
		out.WriteString(cs.Value.String())
	}
	out.WriteString(";")
	return out.String()
}

// ReturnStatement return 语句
type ReturnStatement struct {
	Token       token.Token
	ReturnValue Expression
}

func (rs *ReturnStatement) statementNode()       {}
func (rs *ReturnStatement) TokenLiteral() string { return rs.Token.Literal }
func (rs *ReturnStatement) String() string {
	var out bytes.Buffer
	out.WriteString(rs.TokenLiteral() + " ")
	if rs.ReturnValue != nil {
		out.WriteString(rs.ReturnValue.String())
	}
	out.WriteString(";")
	return out.String()
}

// ExpressionStatement 表达式语句
type ExpressionStatement struct {
	Token      token.Token
	Expression Expression
}

func (es *ExpressionStatement) statementNode()       {}
func (es *ExpressionStatement) TokenLiteral() string { return es.Token.Literal }
func (es *ExpressionStatement) String() string {
	if es.Expression != nil {
		return es.Expression.String()
	}
	return ""
}

// BlockStatement 块语句 { ... }
type BlockStatement struct {
	Token      token.Token // {
	Statements []Statement
}

func (bs *BlockStatement) statementNode()       {}
func (bs *BlockStatement) TokenLiteral() string { return bs.Token.Literal }
func (bs *BlockStatement) String() string {
	var out bytes.Buffer
	out.WriteString("{\n")
	for _, s := range bs.Statements {
		out.WriteString(s.String())
		out.WriteString("\n")
	}
	out.WriteString("}")
	return out.String()
}

// IfExpression if 表达式
type IfExpression struct {
	Token       token.Token // if
	Condition   Expression
	Consequence *BlockStatement
	Alternative *BlockStatement // else 或 elif
}

func (ie *IfExpression) expressionNode()      {}
func (ie *IfExpression) TokenLiteral() string { return ie.Token.Literal }
func (ie *IfExpression) String() string {
	var out bytes.Buffer
	out.WriteString("if ")
	out.WriteString(ie.Condition.String())
	out.WriteString(" ")
	out.WriteString(ie.Consequence.String())
	if ie.Alternative != nil {
		out.WriteString(" else ")
		out.WriteString(ie.Alternative.String())
	}
	return out.String()
}

// WhileExpression while 循环
type WhileExpression struct {
	Token       token.Token
	Condition   Expression
	Body        *BlockStatement
	Alternative *BlockStatement
}

func (we *WhileExpression) expressionNode()      {}
func (we *WhileExpression) TokenLiteral() string { return we.Token.Literal }
func (we *WhileExpression) String() string {
	var out bytes.Buffer
	out.WriteString("while ")
	out.WriteString(we.Condition.String())
	out.WriteString(" ")
	out.WriteString(we.Body.String())
	if we.Alternative != nil {
		out.WriteString(" else ")
		out.WriteString(we.Alternative.String())
	}
	return out.String()
}

// ForExpression for 循环
type ForExpression struct {
	Token       token.Token
	Variable    *Identifier
	Iterable    Expression
	Body        *BlockStatement
	Alternative *BlockStatement
}

func (fe *ForExpression) expressionNode()      {}
func (fe *ForExpression) TokenLiteral() string { return fe.Token.Literal }
func (fe *ForExpression) String() string {
	var out bytes.Buffer
	out.WriteString("for ")
	out.WriteString(fe.Variable.String())
	out.WriteString(" in ")
	out.WriteString(fe.Iterable.String())
	out.WriteString(" ")
	out.WriteString(fe.Body.String())
	if fe.Alternative != nil {
		out.WriteString(" else ")
		out.WriteString(fe.Alternative.String())
	}
	return out.String()
}

// BreakStatement break 语句
type BreakStatement struct {
	Token token.Token
}

func (bs *BreakStatement) statementNode()       {}
func (bs *BreakStatement) TokenLiteral() string { return bs.Token.Literal }
func (bs *BreakStatement) String() string       { return "break;" }

// ContinueStatement continue 语句
type ContinueStatement struct {
	Token token.Token
}

func (cs *ContinueStatement) statementNode()       {}
func (cs *ContinueStatement) TokenLiteral() string { return cs.Token.Literal }
func (cs *ContinueStatement) String() string       { return "continue;" }

// FunctionLiteral 函数字面量
type FunctionLiteral struct {
	Token      token.Token // fn
	Parameters []*Identifier
	Body       *BlockStatement
	Name       string // 用于递归调用
	ReturnType string // C++ 风格返回类型：int add(...) 里的 "int"
}

func (fl *FunctionLiteral) expressionNode()      {}
func (fl *FunctionLiteral) TokenLiteral() string { return fl.Token.Literal }
func (fl *FunctionLiteral) String() string {
	var out bytes.Buffer
	params := []string{}
	for _, p := range fl.Parameters {
		params = append(params, p.String())
	}
	out.WriteString(fl.TokenLiteral())
	if fl.Name != "" {
		out.WriteString(" " + fl.Name)
	}
	out.WriteString("(")
	out.WriteString(strings.Join(params, ", "))
	out.WriteString(") ")
	out.WriteString(fl.Body.String())
	return out.String()
}

// CallExpression 函数调用
type CallExpression struct {
	Token     token.Token // (
	Function  Expression
	Arguments []Expression
}

func (ce *CallExpression) expressionNode()      {}
func (ce *CallExpression) TokenLiteral() string { return ce.Token.Literal }
func (ce *CallExpression) String() string {
	var out bytes.Buffer
	args := []string{}
	for _, a := range ce.Arguments {
		args = append(args, a.String())
	}
	out.WriteString(ce.Function.String())
	out.WriteString("(")
	out.WriteString(strings.Join(args, ", "))
	out.WriteString(")")
	return out.String()
}

// Identifier 标识符
type Identifier struct {
	Token    token.Token
	Value    string
	TypeHint string // C++ 风格参数类型：fn add(int a, int b) 里的 "int"
}

func (i *Identifier) expressionNode()      {}
func (i *Identifier) TokenLiteral() string { return i.Token.Literal }
func (i *Identifier) String() string       { return i.Value }

// IntegerLiteral 整数字面量
type IntegerLiteral struct {
	Token token.Token
	Value int64
}

func (il *IntegerLiteral) expressionNode()      {}
func (il *IntegerLiteral) TokenLiteral() string { return il.Token.Literal }
func (il *IntegerLiteral) String() string       { return il.Token.Literal }

// FloatLiteral 浮点数字面量
type FloatLiteral struct {
	Token token.Token
	Value float64
}

func (fl *FloatLiteral) expressionNode()      {}
func (fl *FloatLiteral) TokenLiteral() string { return fl.Token.Literal }
func (fl *FloatLiteral) String() string       { return fl.Token.Literal }

// StringLiteral 字符串字面量
type StringLiteral struct {
	Token token.Token
	Value string
}

func (sl *StringLiteral) expressionNode()      {}
func (sl *StringLiteral) TokenLiteral() string { return sl.Token.Literal }
func (sl *StringLiteral) String() string       { return sl.Token.Literal }

// BooleanLiteral 布尔字面量
type BooleanLiteral struct {
	Token token.Token
	Value bool
}

func (bl *BooleanLiteral) expressionNode()      {}
func (bl *BooleanLiteral) TokenLiteral() string { return bl.Token.Literal }
func (bl *BooleanLiteral) String() string       { return bl.Token.Literal }

// NilLiteral nil 字面量
type NilLiteral struct {
	Token token.Token
}

func (nl *NilLiteral) expressionNode()      {}
func (nl *NilLiteral) TokenLiteral() string { return nl.Token.Literal }
func (nl *NilLiteral) String() string       { return "nil" }

// PrefixExpression 前缀表达式 (-x, !x)
type PrefixExpression struct {
	Token    token.Token
	Operator string
	Right    Expression
}

func (pe *PrefixExpression) expressionNode()      {}
func (pe *PrefixExpression) TokenLiteral() string { return pe.Token.Literal }
func (pe *PrefixExpression) String() string {
	var out bytes.Buffer
	out.WriteString("(")
	out.WriteString(pe.Operator)
	out.WriteString(pe.Right.String())
	out.WriteString(")")
	return out.String()
}

// InfixExpression 中缀表达式 (x + y, x == y)
type InfixExpression struct {
	Token    token.Token
	Left     Expression
	Operator string
	Right    Expression
}

func (ie *InfixExpression) expressionNode()      {}
func (ie *InfixExpression) TokenLiteral() string { return ie.Token.Literal }
func (ie *InfixExpression) String() string {
	var out bytes.Buffer
	out.WriteString("(")
	out.WriteString(ie.Left.String())
	out.WriteString(" " + ie.Operator + " ")
	out.WriteString(ie.Right.String())
	out.WriteString(")")
	return out.String()
}

// IndexExpression 索引表达式 arr[index]
type IndexExpression struct {
	Token token.Token // [
	Left  Expression
	Index Expression
}

func (ie *IndexExpression) expressionNode()      {}
func (ie *IndexExpression) TokenLiteral() string { return ie.Token.Literal }
func (ie *IndexExpression) String() string {
	var out bytes.Buffer
	out.WriteString("(")
	out.WriteString(ie.Left.String())
	out.WriteString("[")
	out.WriteString(ie.Index.String())
	out.WriteString("])")
	return out.String()
}

// ArrayLiteral 数组字面量 [1, 2, 3]
type ArrayLiteral struct {
	Token    token.Token
	Elements []Expression
}

func (al *ArrayLiteral) expressionNode()      {}
func (al *ArrayLiteral) TokenLiteral() string { return al.Token.Literal }
func (al *ArrayLiteral) String() string {
	var out bytes.Buffer
	elements := []string{}
	for _, el := range al.Elements {
		elements = append(elements, el.String())
	}
	out.WriteString("[")
	out.WriteString(strings.Join(elements, ", "))
	out.WriteString("]")
	return out.String()
}

// HashLiteral 字典字面量 {key: value}
type HashLiteral struct {
	Token token.Token // {
	Pairs map[Expression]Expression
}

func (hl *HashLiteral) expressionNode()      {}
func (hl *HashLiteral) TokenLiteral() string { return hl.Token.Literal }
func (hl *HashLiteral) String() string {
	var out bytes.Buffer
	pairs := []string{}
	for k, v := range hl.Pairs {
		pairs = append(pairs, k.String()+": "+v.String())
	}
	out.WriteString("{")
	out.WriteString(strings.Join(pairs, ", "))
	out.WriteString("}")
	return out.String()
}

// AssignExpression 赋值表达式 x = y
type AssignExpression struct {
	Token token.Token
	Left  Expression
	Value Expression
}

func (ae *AssignExpression) expressionNode()      {}
func (ae *AssignExpression) TokenLiteral() string { return ae.Token.Literal }
func (ae *AssignExpression) String() string {
	var out bytes.Buffer
	out.WriteString(ae.Left.String())
	out.WriteString(" = ")
	out.WriteString(ae.Value.String())
	return out.String()
}

// PropertyExpression 属性访问 obj.prop
type PropertyExpression struct {
	Token   token.Token
	Object  Expression
	Property string
}

func (pe *PropertyExpression) expressionNode()      {}
func (pe *PropertyExpression) TokenLiteral() string { return pe.Token.Literal }
func (pe *PropertyExpression) String() string {
	var out bytes.Buffer
	out.WriteString(pe.Object.String())
	out.WriteString(".")
	out.WriteString(pe.Property)
	return out.String()
}

// MethodCallExpression 方法调用 obj.method()
type MethodCallExpression struct {
	Token     token.Token
	Object    Expression
	Method    string
	Arguments []Expression
}

func (mce *MethodCallExpression) expressionNode()      {}
func (mce *MethodCallExpression) TokenLiteral() string { return mce.Token.Literal }
func (mce *MethodCallExpression) String() string {
	var out bytes.Buffer
	args := []string{}
	for _, a := range mce.Arguments {
		args = append(args, a.String())
	}
	out.WriteString(mce.Object.String())
	out.WriteString(".")
	out.WriteString(mce.Method)
	out.WriteString("(")
	out.WriteString(strings.Join(args, ", "))
	out.WriteString(")")
	return out.String()
}

// ClassLiteral 类字面量
type ClassLiteral struct {
	Token       token.Token
	Name        *Identifier
	SuperClass  Expression
	Interfaces  []*Identifier   // implements A, B
	Methods     map[string]*FunctionLiteral
	Properties  map[string]Expression
	IsStruct    bool            // struct 声明的字段（不检查方法）
	StructFields []string       // struct 的字段名列表
}

// InterfaceLiteral 接口声明：interface Name { fn method(args) }
type InterfaceLiteral struct {
	Token   token.Token
	Name    *Identifier
	Methods []*InterfaceMethod
}

// InterfaceMethod 接口里的方法签名
type InterfaceMethod struct {
	Name   string
	Params []*Identifier
}

func (il *InterfaceLiteral) expressionNode()      {}
func (il *InterfaceLiteral) TokenLiteral() string { return il.Token.Literal }
func (il *InterfaceLiteral) String() string {
	var out bytes.Buffer
	out.WriteString("interface ")
	if il.Name != nil {
		out.WriteString(il.Name.Value)
	}
	out.WriteString(" {")
	for i, m := range il.Methods {
		if i > 0 {
			out.WriteString(",")
		}
		out.WriteString(" " + m.Name)
	}
	out.WriteString(" }")
	return out.String()
}

func (cl *ClassLiteral) expressionNode()      {}
func (cl *ClassLiteral) TokenLiteral() string { return cl.Token.Literal }
func (cl *ClassLiteral) String() string {
	var out bytes.Buffer
	out.WriteString("class ")
	out.WriteString(cl.Name.String())
	if cl.SuperClass != nil {
		out.WriteString(" extends ")
		out.WriteString(cl.SuperClass.String())
	}
	out.WriteString(" {")
	for name, method := range cl.Methods {
		out.WriteString("\n  " + name + ": " + method.String())
	}
	for name, prop := range cl.Properties {
		out.WriteString("\n  " + name + " = " + prop.String())
	}
	out.WriteString("\n}")
	return out.String()
}

// NewExpression new 表达式
type NewExpression struct {
	Token      token.Token
	Class      Expression
	Arguments  []Expression
}

func (ne *NewExpression) expressionNode()      {}
func (ne *NewExpression) TokenLiteral() string { return ne.Token.Literal }
func (ne *NewExpression) String() string {
	var out bytes.Buffer
	args := []string{}
	for _, a := range ne.Arguments {
		args = append(args, a.String())
	}
	out.WriteString("new ")
	out.WriteString(ne.Class.String())
	out.WriteString("(")
	out.WriteString(strings.Join(args, ", "))
	out.WriteString(")")
	return out.String()
}

// ThisExpression this 表达式
type ThisExpression struct {
	Token token.Token
}

func (te *ThisExpression) expressionNode()      {}
func (te *ThisExpression) TokenLiteral() string { return te.Token.Literal }
func (te *ThisExpression) String() string       { return "this" }

// SuperExpression super 表达式
type SuperExpression struct {
	Token  token.Token
	Method string
}

func (se *SuperExpression) expressionNode()      {}
func (se *SuperExpression) TokenLiteral() string { return se.Token.Literal }
func (se *SuperExpression) String() string {
	if se.Method != "" {
		return "super." + se.Method
	}
	return "super"
}

// ImportStatement import 语句
type ImportStatement struct {
	Token   token.Token
	Module  *StringLiteral
	Names   []*Identifier // 导入的名称
	Alias   *Identifier   // as 别名
}

func (is *ImportStatement) statementNode()       {}
func (is *ImportStatement) TokenLiteral() string { return is.Token.Literal }
func (is *ImportStatement) String() string {
	var out bytes.Buffer
	out.WriteString("import ")
	if len(is.Names) > 0 {
		names := []string{}
		for _, n := range is.Names {
			names = append(names, n.String())
		}
		out.WriteString("{ " + strings.Join(names, ", ") + " } from ")
	} else {
		out.WriteString(is.Module.String() + " ")
	}
	if is.Alias != nil {
		out.WriteString("as " + is.Alias.String())
	}
	out.WriteString(";")
	return out.String()
}

// ExportStatement export 语句
type ExportStatement struct {
	Token  token.Token
	Names  []*Identifier
	Value  Expression
}

func (es *ExportStatement) statementNode()       {}
func (es *ExportStatement) TokenLiteral() string { return es.Token.Literal }
func (es *ExportStatement) String() string {
	var out bytes.Buffer
	out.WriteString("export ")
	if es.Value != nil {
		out.WriteString(es.Value.String())
	} else {
		names := []string{}
		for _, n := range es.Names {
			names = append(names, n.String())
		}
		out.WriteString("{ " + strings.Join(names, ", ") + " }")
	}
	out.WriteString(";")
	return out.String()
}

// TryStatement try/catch/finally
type TryStatement struct {
	Token       token.Token
	Body        *BlockStatement
	CatchVar    *Identifier
	CatchBody   *BlockStatement
	FinallyBody *BlockStatement
}

func (ts *TryStatement) statementNode()       {}
func (ts *TryStatement) TokenLiteral() string { return ts.Token.Literal }
func (ts *TryStatement) String() string {
	var out bytes.Buffer
	out.WriteString("try ")
	out.WriteString(ts.Body.String())
	if ts.CatchBody != nil {
		out.WriteString(" catch (")
		out.WriteString(ts.CatchVar.String())
		out.WriteString(") ")
		out.WriteString(ts.CatchBody.String())
	}
	if ts.FinallyBody != nil {
		out.WriteString(" finally ")
		out.WriteString(ts.FinallyBody.String())
	}
	return out.String()
}

// ThrowStatement throw 语句
type ThrowStatement struct {
	Token token.Token
	Value Expression
}

func (ts *ThrowStatement) statementNode()       {}
func (ts *ThrowStatement) TokenLiteral() string { return ts.Token.Literal }
func (ts *ThrowStatement) String() string {
	return "throw " + ts.Value.String() + ";"
}

// MatchExpression match 表达式
type MatchExpression struct {
	Token      token.Token
	Value      Expression
	Cases      []*MatchCase
	Default    *BlockStatement
}

func (me *MatchExpression) expressionNode()      {}
func (me *MatchExpression) TokenLiteral() string { return me.Token.Literal }
func (me *MatchExpression) String() string {
	var out bytes.Buffer
	out.WriteString("match ")
	out.WriteString(me.Value.String())
	out.WriteString(" {")
	for _, c := range me.Cases {
		out.WriteString("\n  " + c.String())
	}
	if me.Default != nil {
		out.WriteString("\n  default: ")
		out.WriteString(me.Default.String())
	}
	out.WriteString("\n}")
	return out.String()
}

// MatchCase match 分支
type MatchCase struct {
	Condition Expression
	Body      *BlockStatement
}

func (mc *MatchCase) String() string {
	return "case " + mc.Condition.String() + ": " + mc.Body.String()
}