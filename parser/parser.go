package parser

import (
	"fmt"
	"strconv"

	"chronos/ast"
	"chronos/lexer"
	"chronos/token"
)

// Parser 语法分析器
type Parser struct {
	l         *lexer.Lexer
	curToken  token.Token
	peekToken token.Token
	errors    []string

	prefixParseFns map[token.TokenType]prefixParseFn
	infixParseFns  map[token.TokenType]infixParseFn
}

type prefixParseFn func() ast.Expression
type infixParseFn func(ast.Expression) ast.Expression

func New(l *lexer.Lexer) *Parser {
	p := &Parser{l: l, errors: []string{}}
	p.registerPrefixFns()
	p.registerInfixFns()
	p.nextToken()
	p.nextToken()
	return p
}

func (p *Parser) Errors() []string { return p.errors }

func (p *Parser) nextToken() {
	p.curToken = p.peekToken
	p.peekToken = p.l.NextToken()
}

func (p *Parser) curTokenIs(t token.TokenType) bool { return p.curToken.Type == t }
func (p *Parser) peekTokenIs(t token.TokenType) bool { return p.peekToken.Type == t }

func (p *Parser) expectPeek(t token.TokenType) bool {
	if p.peekTokenIs(t) {
		p.nextToken()
		return true
	}
	p.peekError(t)
	return false
}

func (p *Parser) peekError(t token.TokenType) {
	p.errors = append(p.errors,
		fmt.Sprintf("line %d:%d: expected %s, got %s (%q)",
			p.peekToken.Line, p.peekToken.Column, t, p.peekToken.Type, p.peekToken.Literal))
}

func (p *Parser) noPrefixParseFnError(t token.TokenType) {
	p.errors = append(p.errors,
		fmt.Sprintf("line %d:%d: unexpected token %s (%q)",
			p.curToken.Line, p.curToken.Column, t, p.curToken.Literal))
}

// ParseProgram 解析整个程序
func (p *Parser) ParseProgram() *ast.Program {
	program := &ast.Program{Statements: []ast.Statement{}}

	for !p.curTokenIs(token.EOF) {
		stmt := p.parseStatement()
		if stmt != nil {
			program.Statements = append(program.Statements, stmt)
		}
		p.nextToken()
	}
	return program
}

func (p *Parser) parseStatement() ast.Statement {
	switch p.curToken.Type {
	case token.LET:
		return p.parseLetStatement()
	case token.CONST:
		return p.parseConstStatement()
	case token.RETURN:
		return p.parseReturnStatement()
	case token.IMPORT:
		return p.parseImportStatement()
	case token.EXPORT:
		return p.parseExportStatement()
	case token.TRY:
		return p.parseTryStatement()
	case token.THROW:
		return p.parseThrowStatement()
	case token.BREAK:
		return &ast.BreakStatement{Token: p.curToken}
	case token.CONTINUE:
		return &ast.ContinueStatement{Token: p.curToken}
	default:
		// C++ 风格函数定义：int faction(参数) { ... }
		// 形如 <类型名> <函数名> ( 开头时按函数定义解析
		if p.curTokenIs(token.IDENT) && p.peekTokenIs(token.IDENT) && p.isTypeName(p.curToken.Literal) {
			return p.parseCStyleFunctionStatement()
		}
		return p.parseExpressionStatement()
	}
}

// isTypeName 判断是否是内置类型名（用于识别 C++ 风格函数定义）
func (p *Parser) isTypeName(s string) bool {
	switch s {
	case "int", "float", "string", "str", "bool", "void", "any":
		return true
	}
	return false
}

// parseCStyleFunctionStatement 解析 C++ 风格函数定义：
//   int add(int a, int b) { return a + b }
// 当前 curToken 是返回类型，peekToken 是函数名。
func (p *Parser) parseCStyleFunctionStatement() ast.Statement {
	retType := p.curToken.Literal
	p.nextToken() // curToken: 函数名
	nameTok := p.curToken

	if !p.expectPeek(token.LPAREN) {
		return nil
	}
	params := p.parseTypedFunctionParameters()
	if params == nil {
		return nil
	}
	if !p.expectPeek(token.LBRACE) {
		return nil
	}
	body := p.parseBlockStatement()

	fn := &ast.FunctionLiteral{
		Token:      nameTok,
		Name:       nameTok.Literal,
		Parameters: params,
		Body:       body,
		ReturnType: retType,
	}
	return &ast.ExpressionStatement{Token: nameTok, Expression: fn}
}

// parseTypedFunctionParameters 解析带类型的参数列表：int a, float b, c
func (p *Parser) parseTypedFunctionParameters() []*ast.Identifier {
	idents := []*ast.Identifier{}
	if p.peekTokenIs(token.RPAREN) {
		p.nextToken()
		return idents
	}
	for {
		p.nextToken()
		// 可能是 "int a" 或只是 "a"
		if p.curTokenIs(token.IDENT) && p.peekTokenIs(token.IDENT) && p.isTypeName(p.curToken.Literal) {
			typeHint := p.curToken.Literal
			p.nextToken() // 到参数名
			idents = append(idents, &ast.Identifier{
				Token:    p.curToken,
				Value:    p.curToken.Literal,
				TypeHint: typeHint,
			})
		} else if p.curTokenIs(token.IDENT) {
			idents = append(idents, &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal})
		} else {
			p.errors = append(p.errors, fmt.Sprintf("line %d: expected parameter name, got %s", p.curToken.Line, p.curToken.Type))
			return nil
		}
		if p.peekTokenIs(token.COMMA) {
			p.nextToken()
			continue
		}
		break
	}
	if !p.expectPeek(token.RPAREN) {
		return nil
	}
	return idents
}

func (p *Parser) parseLetStatement() *ast.LetStatement {
	stmt := &ast.LetStatement{Token: p.curToken}

	// 推进到第一个实际 token
	p.nextToken()

	// 可选修饰符：traveller
	if p.curTokenIs(token.IDENT) && p.curToken.Literal == "traveller" {
		stmt.IsTraveller = true
		p.nextToken() // 到类型名或变量名
	}

	// 现在 curToken 可能是类型名（IDENT）或变量名（IDENT）
	// 判据：若后面紧跟 IDENT，则当前是类型名；若后面紧跟 '='，则当前是变量名
	if !p.curTokenIs(token.IDENT) {
		p.errors = append(p.errors, fmt.Sprintf("line %d: expected identifier in let statement, got %s", p.curToken.Line, p.curToken.Type))
		return nil
	}

	if p.peekTokenIs(token.IDENT) {
		// 当前是类型名，下一个是变量名
		stmt.VarType = p.curToken.Literal
		p.nextToken() // 到变量名
		stmt.Name = &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal}
	} else if p.peekTokenIs(token.ASSIGN) {
		// 当前就是变量名（无类型标注）
		stmt.Name = &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal}
	} else {
		p.errors = append(p.errors, fmt.Sprintf("line %d: expected identifier or '=' in let statement", p.curToken.Line))
		return nil
	}

	if !p.expectPeek(token.ASSIGN) {
		return nil
	}

	p.nextToken()
	stmt.Value = p.parseExpression(LOWEST)
	p.consumeSemicolon()
	return stmt
}

func (p *Parser) parseConstStatement() *ast.ConstStatement {
	stmt := &ast.ConstStatement{Token: p.curToken}

	if !p.expectPeek(token.IDENT) {
		return nil
	}
	stmt.Name = &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal}

	if !p.expectPeek(token.ASSIGN) {
		return nil
	}

	p.nextToken()
	stmt.Value = p.parseExpression(LOWEST)
	p.consumeSemicolon()
	return stmt
}

func (p *Parser) parseReturnStatement() *ast.ReturnStatement {
	stmt := &ast.ReturnStatement{Token: p.curToken}

	p.nextToken()
	if !p.curTokenIs(token.SEMICOLON) && !p.curTokenIs(token.RBRACE) && !p.curTokenIs(token.EOF) {
		stmt.ReturnValue = p.parseExpression(LOWEST)
	}
	p.consumeSemicolon()
	return stmt
}

func (p *Parser) parseExpressionStatement() *ast.ExpressionStatement {
	stmt := &ast.ExpressionStatement{Token: p.curToken}
	stmt.Expression = p.parseExpression(LOWEST)
	p.consumeSemicolon()
	return stmt
}

func (p *Parser) consumeSemicolon() {
	if p.peekTokenIs(token.SEMICOLON) {
		p.nextToken()
	}
}

func (p *Parser) parseBlockStatement() *ast.BlockStatement {
	block := &ast.BlockStatement{Token: p.curToken, Statements: []ast.Statement{}}

	p.nextToken()
	for !p.curTokenIs(token.RBRACE) && !p.curTokenIs(token.EOF) {
		stmt := p.parseStatement()
		if stmt != nil {
			block.Statements = append(block.Statements, stmt)
		}
		p.nextToken()
	}
	if p.curTokenIs(token.EOF) {
		p.errors = append(p.errors, fmt.Sprintf("line %d: expected } but reached end of input", p.curToken.Line))
	}
	return block
}

// 优先级常量
const (
	_           = iota
	LOWEST      // 1
	STREAM      // 2  << >>（C++ 数据流，最低优先级）
	ASSIGN      // 3  =
	LOGICAL     // 3  && ||
	EQUALS      // 4  == !=
	LESSGREATER // 5  < > <= >=
	SUM         // 6  + -
	PRODUCT     // 7  * / % **
	PREFIX      // 8  -x !x not x
	CALL        // 9  fn() obj.m()
	INDEX       // 10 arr[i]
)

var precedences = map[token.TokenType]int{
	token.SHL:          STREAM,
	token.SHR:          STREAM,
	token.ASSIGN:       ASSIGN,
	token.PLUS_ASSIGN:  ASSIGN,
	token.MINUS_ASSIGN: ASSIGN,
	token.MUL_ASSIGN:   ASSIGN,
	token.DIV_ASSIGN:   ASSIGN,
	token.AND:          LOGICAL,
	token.OR:           LOGICAL,
	token.AND_KW:       LOGICAL,
	token.OR_KW:        LOGICAL,
	token.EQ:           EQUALS,
	token.NOT_EQ:       EQUALS,
	token.LT:           LESSGREATER,
	token.GT:           LESSGREATER,
	token.LTE:          LESSGREATER,
	token.GTE:          LESSGREATER,
	token.PLUS:         SUM,
	token.MINUS:        SUM,
	token.PIPE:         SUM,
	token.ASTERISK:     PRODUCT,
	token.SLASH:        PRODUCT,
	token.MOD:          PRODUCT,
	token.POW:          PRODUCT,
	token.LPAREN:       CALL,
	token.LBRACKET:     INDEX,
	token.DOT:          CALL,
}

func (p *Parser) peekPrecedence() int {
	if pr, ok := precedences[p.peekToken.Type]; ok {
		return pr
	}
	return LOWEST
}

func (p *Parser) curPrecedence() int {
	if pr, ok := precedences[p.curToken.Type]; ok {
		return pr
	}
	return LOWEST
}

// parseExpression 主表达式解析（Pratt 解析器）
func (p *Parser) parseExpression(precedence int) ast.Expression {
	prefix := p.prefixParseFns[p.curToken.Type]
	if prefix == nil {
		p.noPrefixParseFnError(p.curToken.Type)
		return nil
	}
	leftExp := prefix()

	for !p.peekTokenIs(token.SEMICOLON) && precedence < p.peekPrecedence() {
		infix := p.infixParseFns[p.peekToken.Type]
		if infix == nil {
			return leftExp
		}
		p.nextToken()
		leftExp = infix(leftExp)
	}
	return leftExp
}

// ---- 前缀解析 ----

func (p *Parser) parseIdentifier() ast.Expression {
	// C++ 风格强制转换：static_cast<int>(x)
	if p.curToken.Literal == "static_cast" && p.peekTokenIs(token.LT) {
		return p.parseStaticCast()
	}
	return &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal}
}

// parseStaticCast 解析 static_cast<T>(expr)，在语法树里降级成 T(expr) 调用。
// 这样转换逻辑直接复用 int/float/str 内置函数，无需单独的 AST 节点。
func (p *Parser) parseStaticCast() ast.Expression {
	castTok := p.curToken

	if !p.expectPeek(token.LT) {
		return nil
	}
	if !p.expectPeek(token.IDENT) {
		return nil
	}
	typeName := p.curToken.Literal
	if !p.expectPeek(token.GT) {
		return nil
	}
	if !p.expectPeek(token.LPAREN) {
		return nil
	}

	p.nextToken()
	expr := p.parseExpression(LOWEST)
	if expr == nil {
		return nil
	}
	if !p.expectPeek(token.RPAREN) {
		return nil
	}

	var fn string
	switch typeName {
	case "int", "integer", "int64":
		fn = "int"
	case "float", "double", "float64":
		fn = "float"
	case "str", "string":
		fn = "str"
	default:
		p.errors = append(p.errors,
			fmt.Sprintf("static_cast: 不支持的类型 %q（可用 int / float / str）", typeName))
		return nil
	}

	return &ast.CallExpression{
		Token: castTok,
		Function: &ast.Identifier{
			Token: token.Token{Type: token.IDENT, Literal: fn},
			Value: fn,
		},
		Arguments: []ast.Expression{expr},
	}
}

func (p *Parser) parseIntegerLiteral() ast.Expression {
	lit := &ast.IntegerLiteral{Token: p.curToken}
	v, err := strconv.ParseInt(p.curToken.Literal, 0, 64)
	if err != nil {
		p.errors = append(p.errors, fmt.Sprintf("could not parse %q as integer", p.curToken.Literal))
		return nil
	}
	lit.Value = v
	return lit
}

func (p *Parser) parseFloatLiteral() ast.Expression {
	lit := &ast.FloatLiteral{Token: p.curToken}
	v, err := strconv.ParseFloat(p.curToken.Literal, 64)
	if err != nil {
		p.errors = append(p.errors, fmt.Sprintf("could not parse %q as float", p.curToken.Literal))
		return nil
	}
	lit.Value = v
	return lit
}

func (p *Parser) parseStringLiteral() ast.Expression {
	return &ast.StringLiteral{Token: p.curToken, Value: p.curToken.Literal}
}

func (p *Parser) parseBooleanLiteral() ast.Expression {
	return &ast.BooleanLiteral{Token: p.curToken, Value: p.curTokenIs(token.TRUE)}
}

func (p *Parser) parseNilLiteral() ast.Expression {
	return &ast.NilLiteral{Token: p.curToken}
}

func (p *Parser) parsePrefixExpression() ast.Expression {
	exp := &ast.PrefixExpression{Token: p.curToken, Operator: p.curToken.Literal}
	p.nextToken()
	exp.Right = p.parseExpression(PREFIX)
	return exp
}

func (p *Parser) parseGroupedExpression() ast.Expression {
	p.nextToken()
	exp := p.parseExpression(LOWEST)
	if !p.expectPeek(token.RPAREN) {
		return nil
	}
	return exp
}

func (p *Parser) parseIfExpression() ast.Expression {
	exp := &ast.IfExpression{Token: p.curToken}

	// 可选括号：if (cond) 或 if cond
	if p.peekTokenIs(token.LPAREN) {
		p.nextToken() // 消费 (
		p.nextToken()
		exp.Condition = p.parseExpression(LOWEST)
		if !p.expectPeek(token.RPAREN) {
			return nil
		}
	} else {
		p.nextToken()
		exp.Condition = p.parseExpression(LOWEST)
	}

	if !p.expectPeek(token.LBRACE) {
		return nil
	}
	exp.Consequence = p.parseBlockStatement()

	if p.peekTokenIs(token.ELSE) {
		p.nextToken() // curToken: } → else
		p.nextToken() // curToken: else → { 或 if 或 elif
		switch {
		case p.curTokenIs(token.IF), p.curTokenIs(token.ELIF):
			// 当前 token 就是 if/elif，直接递归
			alt := p.parseIfExpression()
			if alt != nil {
				exp.Alternative = &ast.BlockStatement{
					Statements: []ast.Statement{
						&ast.ExpressionStatement{Token: alt.(*ast.IfExpression).Token, Expression: alt},
					},
				}
			}
		case p.curTokenIs(token.LBRACE):
			exp.Alternative = p.parseBlockStatement()
		default:
			p.peekError(token.LBRACE)
			return nil
		}
	}
	return exp
}

func (p *Parser) parseWhileExpression() ast.Expression {
	exp := &ast.WhileExpression{Token: p.curToken}

	// 可选括号
	if p.peekTokenIs(token.LPAREN) {
		p.nextToken()
		p.nextToken()
		exp.Condition = p.parseExpression(LOWEST)
		if !p.expectPeek(token.RPAREN) {
			return nil
		}
	} else {
		p.nextToken()
		exp.Condition = p.parseExpression(LOWEST)
	}

	if !p.expectPeek(token.LBRACE) {
		return nil
	}
	exp.Body = p.parseBlockStatement()
	exp.Alternative = p.parseLoopAlternative()
	return exp
}

// parseLoopAlternative 解析循环体后的可选 else 分支
func (p *Parser) parseLoopAlternative() *ast.BlockStatement {
	if !p.peekTokenIs(token.ELSE) {
		return nil
	}
	p.nextToken() // curToken: } → else
	if !p.expectPeek(token.LBRACE) {
		return nil
	}
	return p.parseBlockStatement()
}

func (p *Parser) parseForExpression() ast.Expression {
	exp := &ast.ForExpression{Token: p.curToken}

	// 可选括号
	hasParen := p.peekTokenIs(token.LPAREN)
	if hasParen {
		p.nextToken()
	}
	p.nextToken()

	if !p.curTokenIs(token.IDENT) {
		p.errors = append(p.errors, fmt.Sprintf("line %d: expected identifier in for loop", p.curToken.Line))
		return nil
	}
	exp.Variable = &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal}

	if !p.expectPeek(token.IN) {
		return nil
	}
	p.nextToken()
	exp.Iterable = p.parseExpression(LOWEST)

	if hasParen {
		if !p.expectPeek(token.RPAREN) {
			return nil
		}
	}
	if !p.expectPeek(token.LBRACE) {
		return nil
	}
	exp.Body = p.parseBlockStatement()
	exp.Alternative = p.parseLoopAlternative()
	return exp
}

func (p *Parser) parseFunctionLiteral() ast.Expression {
	lit := &ast.FunctionLiteral{Token: p.curToken}

	// 可选的函数名：fn name(params) {...}
	if p.peekTokenIs(token.IDENT) {
		p.nextToken()
		lit.Name = p.curToken.Literal
	}

	if !p.expectPeek(token.LPAREN) {
		return nil
	}
	lit.Parameters = p.parseFunctionParameters()
	if !p.expectPeek(token.LBRACE) {
		return nil
	}
	lit.Body = p.parseBlockStatement()
	return lit
}

func (p *Parser) parseFunctionParameters() []*ast.Identifier {
	idents := []*ast.Identifier{}
	if p.peekTokenIs(token.RPAREN) {
		p.nextToken()
		return idents
	}
	p.nextToken()
	idents = append(idents, &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal})
	for p.peekTokenIs(token.COMMA) {
		p.nextToken()
		p.nextToken()
		idents = append(idents, &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal})
	}
	if !p.expectPeek(token.RPAREN) {
		return nil
	}
	return idents
}

func (p *Parser) parseArrayLiteral() ast.Expression {
	arr := &ast.ArrayLiteral{Token: p.curToken}
	arr.Elements = p.parseExpressionList(token.RBRACKET)
	return arr
}

func (p *Parser) parseHashLiteral() ast.Expression {
	hash := &ast.HashLiteral{Token: p.curToken, Pairs: make(map[ast.Expression]ast.Expression)}

	for !p.peekTokenIs(token.RBRACE) {
		p.nextToken()
		key := p.parseExpression(LOWEST)
		if !p.expectPeek(token.COLON) {
			return nil
		}
		p.nextToken()
		value := p.parseExpression(LOWEST)
		hash.Pairs[key] = value
		if !p.peekTokenIs(token.RBRACE) && !p.expectPeek(token.COMMA) {
			return nil
		}
	}
	if !p.expectPeek(token.RBRACE) {
		return nil
	}
	return hash
}

func (p *Parser) parseClassLiteral() ast.Expression {
	cl := &ast.ClassLiteral{
		Token:      p.curToken,
		Methods:    make(map[string]*ast.FunctionLiteral),
		Properties: make(map[string]ast.Expression),
	}

	if !p.expectPeek(token.IDENT) {
		return nil
	}
	cl.Name = &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal}

	// extends 处理：extends 目前不是关键字，按 IDENT 处理
	if p.peekTokenIs(token.IDENT) && p.peekToken.Literal == "extends" {
		p.nextToken() // 消费 extends
		p.nextToken()
		cl.SuperClass = p.parseExpression(LOWEST)
	}

	if !p.expectPeek(token.LBRACE) {
		return nil
	}

	p.nextToken()
	for !p.curTokenIs(token.RBRACE) && !p.curTokenIs(token.EOF) {
		if p.curTokenIs(token.IDENT) {
			name := p.curToken.Literal
			nameTok := p.curToken
			switch {
			case p.peekTokenIs(token.LPAREN):
				// 方法定义：name(params) {...}
				cl.Methods[name] = p.parseMethod(name, nameTok)
			case p.peekTokenIs(token.ASSIGN):
				// 属性定义：name = value
				p.nextToken()
				p.nextToken()
				cl.Properties[name] = p.parseExpression(LOWEST)
			}
		}
		p.nextToken()
	}
	return cl
}

// parseMethod 解析类方法定义（当前 curToken 是方法名，peekToken 是 '('）
func (p *Parser) parseMethod(name string, nameTok token.Token) *ast.FunctionLiteral {
	lit := &ast.FunctionLiteral{Token: nameTok, Name: name}
	if !p.expectPeek(token.LPAREN) {
		return nil
	}
	lit.Parameters = p.parseFunctionParameters()
	if !p.expectPeek(token.LBRACE) {
		return nil
	}
	lit.Body = p.parseBlockStatement()
	return lit
}

func (p *Parser) parseNewExpression() ast.Expression {
	exp := &ast.NewExpression{Token: p.curToken}
	p.nextToken()
	exp.Class = p.parseExpression(CALL)
	// 若后面紧跟 '('，解析构造参数
	if p.peekTokenIs(token.LPAREN) {
		p.nextToken() // 消费 (
		exp.Arguments = p.parseExpressionList(token.RPAREN)
	}
	return exp
}

func (p *Parser) parseThisExpression() ast.Expression {
	return &ast.ThisExpression{Token: p.curToken}
}

func (p *Parser) parseSuperExpression() ast.Expression {
	// super 单独作为一个表达式；super.method(...) 由 parseDotExpression 处理
	return &ast.SuperExpression{Token: p.curToken}
}

func (p *Parser) parseMatchExpression() ast.Expression {
	exp := &ast.MatchExpression{Token: p.curToken}

	if !p.expectPeek(token.LPAREN) {
		return nil
	}
	p.nextToken()
	exp.Value = p.parseExpression(LOWEST)
	if !p.expectPeek(token.RPAREN) {
		return nil
	}
	if !p.expectPeek(token.LBRACE) {
		return nil
	}

	p.nextToken()
	for !p.curTokenIs(token.RBRACE) && !p.curTokenIs(token.EOF) {
		switch {
		case p.curTokenIs(token.CASE):
			p.nextToken()
			cond := p.parseExpression(LOWEST)
			if !p.expectPeek(token.COLON) {
				return nil
			}
			if !p.expectPeek(token.LBRACE) {
				return nil
			}
			body := p.parseBlockStatement()
			exp.Cases = append(exp.Cases, &ast.MatchCase{Condition: cond, Body: body})
		case p.curTokenIs(token.DEFAULT):
			if !p.expectPeek(token.COLON) {
				return nil
			}
			if !p.expectPeek(token.LBRACE) {
				return nil
			}
			exp.Default = p.parseBlockStatement()
		}
		p.nextToken()
	}
	return exp
}

// ---- 中缀解析 ----

func (p *Parser) parseInfixExpression(left ast.Expression) ast.Expression {
	exp := &ast.InfixExpression{
		Token:    p.curToken,
		Left:     left,
		Operator: p.curToken.Literal,
	}
	prec := p.curPrecedence()
	p.nextToken()
	exp.Right = p.parseExpression(prec)
	return exp
}

// parseStreamExpression 解析 C++ 风格数据流：
//   prints << x     输出
//   inputs >> name  读入
// 两者都按中缀表达式处理，求值时按左操作数决定方向。
func (p *Parser) parseStreamExpression(left ast.Expression) ast.Expression {
	exp := &ast.InfixExpression{
		Token:    p.curToken,
		Left:     left,
		Operator: p.curToken.Literal,
	}
	prec := p.curPrecedence()
	p.nextToken()
	exp.Right = p.parseExpression(prec)
	return exp
}

func (p *Parser) parseAssignExpression(left ast.Expression) ast.Expression {	exp := &ast.AssignExpression{Token: p.curToken, Left: left}
	p.nextToken()
	exp.Value = p.parseExpression(LOWEST)
	return exp
}

func (p *Parser) parseCallExpression(function ast.Expression) ast.Expression {
	exp := &ast.CallExpression{Token: p.curToken, Function: function}
	exp.Arguments = p.parseExpressionList(token.RPAREN)
	return exp
}

func (p *Parser) parseExpressionList(end token.TokenType) []ast.Expression {
	list := []ast.Expression{}
	if p.peekTokenIs(end) {
		p.nextToken()
		return list
	}
	p.nextToken()
	list = append(list, p.parseExpression(LOWEST))
	for p.peekTokenIs(token.COMMA) {
		p.nextToken()
		p.nextToken()
		list = append(list, p.parseExpression(LOWEST))
	}
	if !p.expectPeek(end) {
		return nil
	}
	return list
}

func (p *Parser) parseIndexExpression(left ast.Expression) ast.Expression {
	exp := &ast.IndexExpression{Token: p.curToken, Left: left}
	p.nextToken()
	exp.Index = p.parseExpression(LOWEST)
	if !p.expectPeek(token.RBRACKET) {
		return nil
	}
	return exp
}

// parseDotExpression 处理 obj.prop 与 obj.method(args) 的链式访问
func (p *Parser) parseDotExpression(left ast.Expression) ast.Expression {
	if !p.expectPeek(token.IDENT) {
		return nil
	}
	name := p.curToken.Literal

	// 方法调用 obj.method(...)
	if p.peekTokenIs(token.LPAREN) {
		p.nextToken()
		return &ast.MethodCallExpression{
			Token:     p.curToken,
			Object:    left,
			Method:    name,
			Arguments: p.parseExpressionList(token.RPAREN),
		}
	}
	// 属性访问 obj.prop
	return &ast.PropertyExpression{
		Token:    p.curToken,
		Object:   left,
		Property: name,
	}
}

// parsePipeExpression 处理管道操作符 |>
// a |> f(b)  等价于  f(a, b)
// a |> f     等价于  f(a)
func (p *Parser) parsePipeExpression(left ast.Expression) ast.Expression {
	pipeToken := p.curToken // 当前是 |>

	// 推进到右侧表达式
	p.nextToken()

	// 右侧通常是函数调用 f(b) 或函数标识符 f
	// 用 PIPE 优先级之上的优先级解析，确保 f(b) 的调用结构完整
	right := p.parseExpression(SUM)

	if right == nil {
		return nil
	}

	switch r := right.(type) {
	case *ast.CallExpression:
		// 把 left 插入到参数列表第 0 位
		newArgs := make([]ast.Expression, 0, len(r.Arguments)+1)
		newArgs = append(newArgs, left)
		newArgs = append(newArgs, r.Arguments...)
		r.Arguments = newArgs
		return r

	case *ast.Identifier:
		// 普通标识符：构造 f(left)
		return &ast.CallExpression{
			Token:     pipeToken,
			Function:  r,
			Arguments: []ast.Expression{left},
		}

	case *ast.MethodCallExpression:
		// 右侧是方法调用 obj.m(x)，也把 left 插到参数首位
		newArgs := make([]ast.Expression, 0, len(r.Arguments)+1)
		newArgs = append(newArgs, left)
		newArgs = append(newArgs, r.Arguments...)
		r.Arguments = newArgs
		return r

	default:
		// 其他情况（如匿名函数、括号表达式等），按 f(left) 处理
		return &ast.CallExpression{
			Token:     pipeToken,
			Function:  right,
			Arguments: []ast.Expression{left},
		}
	}
}

// ---- 语句解析 ----

func (p *Parser) parseImportStatement() *ast.ImportStatement {
	stmt := &ast.ImportStatement{Token: p.curToken}

	p.nextToken()
	if p.curTokenIs(token.LBRACE) {
		p.nextToken()
		for !p.curTokenIs(token.RBRACE) && !p.curTokenIs(token.EOF) {
			if p.curTokenIs(token.IDENT) {
				stmt.Names = append(stmt.Names, &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal})
			}
			if p.peekTokenIs(token.COMMA) {
				p.nextToken()
			}
			p.nextToken()
		}
		if !p.expectPeek(token.FROM) {
			return nil
		}
		p.nextToken()
		if !p.curTokenIs(token.STRING) {
			return nil
		}
		stmt.Module = &ast.StringLiteral{Token: p.curToken, Value: p.curToken.Literal}
	} else if p.curTokenIs(token.STRING) {
		stmt.Module = &ast.StringLiteral{Token: p.curToken, Value: p.curToken.Literal}
		if p.peekTokenIs(token.AS) {
			p.nextToken()
			p.nextToken()
			if p.curTokenIs(token.IDENT) {
				stmt.Alias = &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal}
			}
		}
	} else {
		p.errors = append(p.errors, fmt.Sprintf("line %d: invalid import syntax", p.curToken.Line))
		return nil
	}
	p.consumeSemicolon()
	return stmt
}

func (p *Parser) parseExportStatement() *ast.ExportStatement {
	stmt := &ast.ExportStatement{Token: p.curToken}

	p.nextToken()
	if p.curTokenIs(token.LBRACE) {
		p.nextToken()
		for !p.curTokenIs(token.RBRACE) && !p.curTokenIs(token.EOF) {
			if p.curTokenIs(token.IDENT) {
				stmt.Names = append(stmt.Names, &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal})
			}
			if p.peekTokenIs(token.COMMA) {
				p.nextToken()
			}
			p.nextToken()
		}
	} else {
		// 简化：export 后跟一个语句/表达式
		stmt.Value = p.parseExpression(LOWEST)
	}
	p.consumeSemicolon()
	return stmt
}

func (p *Parser) parseTryStatement() *ast.TryStatement {
	stmt := &ast.TryStatement{Token: p.curToken}

	if !p.expectPeek(token.LBRACE) {
		return nil
	}
	stmt.Body = p.parseBlockStatement()

	if p.peekTokenIs(token.CATCH) {
		p.nextToken()
		if !p.expectPeek(token.LPAREN) {
			return nil
		}
		p.nextToken()
		if p.curTokenIs(token.IDENT) {
			stmt.CatchVar = &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal}
		}
		if !p.expectPeek(token.RPAREN) {
			return nil
		}
		if !p.expectPeek(token.LBRACE) {
			return nil
		}
		stmt.CatchBody = p.parseBlockStatement()
	}

	if p.peekTokenIs(token.FINALLY) {
		p.nextToken()
		if !p.expectPeek(token.LBRACE) {
			return nil
		}
		stmt.FinallyBody = p.parseBlockStatement()
	}
	return stmt
}

func (p *Parser) parseThrowStatement() *ast.ThrowStatement {
	stmt := &ast.ThrowStatement{Token: p.curToken}
	p.nextToken()
	stmt.Value = p.parseExpression(LOWEST)
	p.consumeSemicolon()
	return stmt
}

// ---- 注册解析函数 ----

func (p *Parser) registerPrefixFns() {
	p.prefixParseFns = map[token.TokenType]prefixParseFn{
		token.IDENT:    p.parseIdentifier,
		token.INT:      p.parseIntegerLiteral,
		token.FLOAT:    p.parseFloatLiteral,
		token.STRING:   p.parseStringLiteral,
		token.TRUE:     p.parseBooleanLiteral,
		token.FALSE:    p.parseBooleanLiteral,
		token.NIL:      p.parseNilLiteral,
		token.NOT:      p.parsePrefixExpression,
		token.NOT_KW:   p.parsePrefixExpression,
		token.MINUS:    p.parsePrefixExpression,
		token.LPAREN:   p.parseGroupedExpression,
		token.IF:       p.parseIfExpression,
		token.ELIF:     p.parseIfExpression,
		token.WHILE:    p.parseWhileExpression,
		token.FOR:      p.parseForExpression,
		token.FN:       p.parseFunctionLiteral,
		token.LBRACKET: p.parseArrayLiteral,
		token.LBRACE:   p.parseHashLiteral,
		token.CLASS:    p.parseClassLiteral,
		token.NEW:      p.parseNewExpression,
		token.THIS:     p.parseThisExpression,
		token.SUPER:    p.parseSuperExpression,
		token.MATCH:    p.parseMatchExpression,
	}
}

func (p *Parser) registerInfixFns() {
	p.infixParseFns = map[token.TokenType]infixParseFn{
		token.PLUS:         p.parseInfixExpression,
		token.MINUS:        p.parseInfixExpression,
		token.ASTERISK:     p.parseInfixExpression,
		token.SLASH:        p.parseInfixExpression,
		token.MOD:          p.parseInfixExpression,
		token.POW:          p.parseInfixExpression,
		token.EQ:           p.parseInfixExpression,
		token.NOT_EQ:       p.parseInfixExpression,
		token.LT:           p.parseInfixExpression,
		token.GT:           p.parseInfixExpression,
		token.LTE:          p.parseInfixExpression,
		token.GTE:          p.parseInfixExpression,
		token.AND:          p.parseInfixExpression,
		token.OR:           p.parseInfixExpression,
		token.AND_KW:       p.parseInfixExpression,
		token.OR_KW:        p.parseInfixExpression,
		token.ASSIGN:       p.parseAssignExpression,
		token.PLUS_ASSIGN:  p.parseAssignExpression,
		token.MINUS_ASSIGN: p.parseAssignExpression,
		token.MUL_ASSIGN:   p.parseAssignExpression,
		token.DIV_ASSIGN:   p.parseAssignExpression,
		token.SHL:          p.parseStreamExpression,
		token.SHR:          p.parseStreamExpression,
		token.LPAREN:       p.parseCallExpression,
		token.LBRACKET:     p.parseIndexExpression,
		token.DOT:          p.parseDotExpression,
		token.PIPE:         p.parsePipeExpression,
	}
}
