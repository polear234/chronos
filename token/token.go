package token

type TokenType string

const (
	// 特殊标记
	ILLEGAL = "ILLEGAL"
	EOF     = "EOF"

	// 标识符和字面量
	IDENT  = "IDENT"  // 变量名、函数名等
	INT    = "INT"    // 整数
	FLOAT  = "FLOAT"  // 浮点数
	STRING = "STRING" // 字符串

	// 运算符
	ASSIGN   = "="
	PLUS     = "+"
	MINUS    = "-"
	ASTERISK = "*"
	SLASH    = "/"
	MOD      = "%"
	POW      = "**"

	EQ     = "=="
	NOT_EQ = "!="
	LT     = "<"
	GT     = ">"
	LTE    = "<="
	GTE    = ">="

	// C++ 风格数据流：cin >> x、cout << x
	SHL = "<<"
	SHR = ">>"

	AND = "&&"
	OR  = "||"
	NOT = "!"

	PIPE = "|>"

	PLUS_ASSIGN  = "+="
	MINUS_ASSIGN = "-="
	MUL_ASSIGN   = "*="
	DIV_ASSIGN   = "/="

	INCREMENT = "++"
	DECREMENT = "--"

	// 分隔符
	COMMA     = ","
	SEMICOLON = ";"
	COLON     = ":"
	DOT       = "."
	ARROW     = "=>"

	LPAREN   = "("
	RPAREN   = ")"
	LBRACE   = "{"
	RBRACE   = "}"
	LBRACKET = "["
	RBRACKET = "]"

	// 关键字
	LET      = "LET"
	CONST    = "CONST"
	FN       = "FN"
	RETURN   = "RETURN"
	IF       = "IF"
	ELSE     = "ELSE"
	ELIF     = "ELIF"
	WHILE    = "WHILE"
	FOR      = "FOR"
	IN       = "IN"
	BREAK    = "BREAK"
	CONTINUE = "CONTINUE"
	TRUE     = "TRUE"
	FALSE    = "FALSE"
	NIL      = "NIL"
	AND_KW   = "AND"
	OR_KW    = "OR"
	NOT_KW   = "NOT"
	CLASS    = "CLASS"
	NEW      = "NEW"
	THIS     = "THIS"
	SUPER    = "SUPER"
	IMPORT   = "IMPORT"
	EXPORT   = "EXPORT"
	FROM     = "FROM"
	AS       = "AS"
	TRY      = "TRY"
	CATCH    = "CATCH"
	FINALLY  = "FINALLY"
	THROW    = "THROW"
	MATCH    = "MATCH"
	CASE     = "CASE"
	DEFAULT  = "DEFAULT"
)

var keywords = map[string]TokenType{
	"let":      LET,
	"const":    CONST,
	"fn":       FN,
	"return":   RETURN,
	"if":       IF,
	"else":     ELSE,
	"elif":     ELIF,
	"while":    WHILE,
	"for":      FOR,
	"in":       IN,
	"break":    BREAK,
	"continue": CONTINUE,
	"true":     TRUE,
	"false":    FALSE,
	"nil":      NIL,
	"and":      AND_KW,
	"or":       OR_KW,
	"not":      NOT_KW,
	"class":    CLASS,
	"new":      NEW,
	"this":     THIS,
	"super":    SUPER,
	"import":   IMPORT,
	"export":   EXPORT,
	"from":     FROM,
	"as":       AS,
	"try":      TRY,
	"catch":    CATCH,
	"finally":  FINALLY,
	"throw":    THROW,
	"match":    MATCH,
	"case":     CASE,
	"default":  DEFAULT,
}

func LookupIdent(ident string) TokenType {
	if tok, ok := keywords[ident]; ok {
		return tok
	}
	return IDENT
}

type Token struct {
	Type    TokenType
	Literal string
	Line    int
	Column  int
}