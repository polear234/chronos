package repl

import (
	"bufio"
	"fmt"
	"io"
	"strings"

	"chronos/interpreter"
	"chronos/lexer"
	"chronos/object"
	"chronos/parser"
)

const PROMPT = "chronos> "
const CONTINUATION_PROMPT = ".... "

// Start 启动 REPL
func Start(in io.Reader, out io.Writer) {
	scanner := bufio.NewScanner(in)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	env := object.NewEnvironment()

	fmt.Fprintln(out, "Chronos — a lightweight scripting language (Go implementation)")
	fmt.Fprintln(out, "Type 'exit' / 'quit' to leave, 'clear' to reset the environment.")
	fmt.Fprintln(out)

	var buffer strings.Builder

	for {
		if buffer.Len() > 0 {
			fmt.Fprint(out, CONTINUATION_PROMPT)
		} else {
			fmt.Fprint(out, PROMPT)
		}

		if !scanner.Scan() {
			fmt.Fprintln(out)
			return
		}

		line := scanner.Text()

		if buffer.Len() == 0 {
			trimmed := strings.TrimSpace(line)
			switch trimmed {
			case "exit", "quit":
				fmt.Fprintln(out, "Goodbye!")
				return
			case "clear":
				env = object.NewEnvironment()
				fmt.Fprintln(out, "Environment cleared.")
				continue
			case "":
				continue
			}
		}

		buffer.WriteString(line)
		buffer.WriteString("\n")

		// 判断是否需要继续读取：检查大括号、括号、方括号是否平衡
		if !isBalanced(buffer.String()) {
			continue
		}

		input := buffer.String()
		buffer.Reset()

		l := lexer.New(input)
		p := parser.New(l)
		program := p.ParseProgram()

		if len(p.Errors()) > 0 {
			for _, err := range p.Errors() {
				fmt.Fprintf(out, "Parse error: %s\n", err)
			}
			continue
		}

		result := interpreter.Eval(program, env)
		if result != nil && result.Type() != object.NIL_OBJ {
			fmt.Fprintf(out, "%s\n", result.Inspect())
		}
	}
}

// isBalanced 检查输入中的括号是否平衡（忽略字符串字面量）
func isBalanced(input string) bool {
	stack := []rune{}
	pairs := map[rune]rune{')': '(', ']': '[', '}': '{'}
	inStr := byte(0) // 0 = 不在字符串中
	escaped := false

	for i := 0; i < len(input); i++ {
		ch := input[i]

		// 处理字符串
		if inStr != 0 {
			if escaped {
				escaped = false
				continue
			}
			if ch == '\\' {
				escaped = true
				continue
			}
			if ch == inStr {
				inStr = 0
			}
			continue
		}
		if ch == '"' || ch == '\'' {
			inStr = ch
			continue
		}

		r := rune(ch)
		switch r {
		case '(', '[', '{':
			stack = append(stack, r)
		case ')', ']', '}':
			if len(stack) == 0 {
				return false
			}
			top := stack[len(stack)-1]
			if top != pairs[r] {
				return false
			}
			stack = stack[:len(stack)-1]
		}
	}
	return len(stack) == 0
}
