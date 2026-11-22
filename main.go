//go:build !js

package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"chronos/interpreter"
	"chronos/lexer"
	"chronos/object"
	"chronos/parser"
	"chronos/repl"
)

// version 由 ldflags 在编译期注入（-X main.version=...），必须是 var 不能是 const
var version = "0.2.2"

func main() {
	// 被 chronos build 编译过的程序：直接执行内嵌脚本
	if script, ok := selfPayload(); ok {
		os.Exit(runEmbedded(script))
	}
	os.Exit(run(os.Args[1:], os.Stdin, os.Stdout, os.Stderr))
}

// run 返回进程退出码
func run(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	// 无参数：默认启动 REPL
	if len(args) == 0 {
		repl.Start(stdin, stdout)
		return 0
	}

	switch args[0] {
	case "run", "-run", "--run":
		return cmdRun(args[1:], stdin, stdout, stderr)

	case "build", "-build", "--build":
		return cmdBuild(args[1:], stdout, stderr)

	case "repl", "-repl", "--repl":
		repl.Start(stdin, stdout)
		return 0

	case "version", "-v", "-version", "--version":
		fmt.Fprintf(stdout, "Chronos %s\n", version)
		return 0

	case "help", "-h", "-help", "--help":
		printUsage(stdout)
		return 0

	default:
		// 兼容旧用法：chronos foo.px 直接运行（视为 run 的简写）
		if isScriptArg(args[0]) {
			return cmdRun(args, stdin, stdout, stderr)
		}
		fmt.Fprintf(stderr, "unknown command: %s\n\n", args[0])
		printUsage(stderr)
		return 2
	}
}

func cmdRun(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("run", flag.ContinueOnError)
	fs.SetOutput(stderr)
	fs.Usage = func() {
		fmt.Fprintln(stderr, "Usage: chronos run [options] <file.px>")
		fmt.Fprintln(stderr, "       chronos -run <file.px>")
		fmt.Fprintln(stderr, "\nOptions:")
		fs.PrintDefaults()
	}
	_ = fs.Parse(args)

	files := fs.Args()
	if len(files) == 0 {
		fmt.Fprintln(stderr, "error: no input file (use 'chronos run file.px', or '-' to read from stdin)")
		return 2
	}

	exitCode := 0
	for _, file := range files {
		if file == "-" {
			if err := runReader(stdin, stdout); err != nil {
				fmt.Fprintf(stderr, "error: %v\n", err)
				exitCode = 1
			}
			continue
		}
		if err := runFile(file, stdout); err != nil {
			fmt.Fprintf(stderr, "error: %v\n", err)
			exitCode = 1
		}
	}
	return exitCode
}

// isScriptArg 判断参数是否像脚本路径（含 .px 或路径分隔符）
func isScriptArg(arg string) bool {
	if len(arg) >= 3 && arg[len(arg)-3:] == ".px" {
		return true
	}
	for _, r := range arg {
		if r == '/' || r == '\\' || r == '.' {
			return true
		}
	}
	return false
}

func printUsage(w io.Writer) {
	fmt.Fprintf(w, `Chronos %s — a lightweight scripting language

Usage:
  chronos                     Start the interactive REPL
  chronos run <file.px>       Run a script file
  chronos -run <file.px>      Run a script file (shorthand)
  chronos build <file.px>     Compile a script into a standalone executable
  chronos build <file.px> -o <name>   Compile with a custom output name
  chronos run -               Read and run source from stdin
  chronos version             Print version
  chronos help                Show this help

Examples:
  chronos
  chronos run examples/hello.px
  chronos build examples/hello.px        # 产出 ./hello 可执行文件
  chronos build app.px -o /usr/local/bin/app
  echo 'print("hi")' | chronos run -
`, version)
}

func runFile(path string, out io.Writer) error {
	src, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read file %s: %w", path, err)
	}
	// 注入脚本所在目录，供 import 相对路径解析
	return runSource(string(src), out, filepath.Dir(path))
}

func runReader(r io.Reader, out io.Writer) error {
	src, err := io.ReadAll(r)
	if err != nil {
		return fmt.Errorf("read stdin: %w", err)
	}
	// stdin 模式：import 相对当前工作目录
	return runSource(string(src), out, ".")
}

func runSource(src string, out io.Writer, importDirs ...string) error {
	l := lexer.New(src)
	p := parser.New(l)
	program := p.ParseProgram()

	if len(p.Errors()) > 0 {
		for _, e := range p.Errors() {
			fmt.Fprintf(out, "Parse error: %s\n", e)
		}
		return fmt.Errorf("parse failed with %d error(s)", len(p.Errors()))
	}

	env := object.NewEnvironment()
	if len(importDirs) > 0 && importDirs[0] != "" {
		env.ImportDir = importDirs[0]
	}
	result := interpreter.Eval(program, env)
	if result != nil && result.Type() == object.ERROR_OBJ {
		fmt.Fprintf(out, "%s\n", result.Inspect())
		return fmt.Errorf("runtime error")
	}
	return nil
}
