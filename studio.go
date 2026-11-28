//go:build !js

package main

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"os/exec"

	"chronospp/interpreter"
	"chronospp/lexer"
	"chronospp/object"
	"chronospp/parser"
)

//go:embed studio/index.html
var studioHTML string

// cmdStudio 实现 chpp studio [--port N] [--no-open]
// 启动一个本地 Web 编辑器（VS Code 风格），在浏览器里写代码、运行、编译。
func cmdStudio(args []string, stdout, stderr io.Writer) int {
	port := 8420
	noOpen := false
	for i := 0; i < len(args); i++ {
		a := args[i]
		switch a {
		case "-p", "--port", "-port":
			if i+1 < len(args) {
				fmt.Sscanf(args[i+1], "%d", &port)
				i++
			}
		case "--no-open", "-no-open":
			noOpen = true
		case "-h", "--help", "-help":
			fmt.Fprintln(stdout, "用法: chpp studio [选项]")
			fmt.Fprintln(stdout, "\n选项:")
			fmt.Fprintln(stdout, "  -p, --port N   监听端口（默认 8420）")
			fmt.Fprintln(stdout, "  --no-open      不自动打开浏览器")
			fmt.Fprintln(stdout, "\n在浏览器里写 Chronospp 代码，按 F5 运行。")
			return 0
		}
	}

	mux := http.NewServeMux()

	// 编辑器页面
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		io.WriteString(w, studioHTML)
	})

	// 运行代码
	mux.HandleFunc("/api/run", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "POST only", http.StatusMethodNotAllowed)
			return
		}
		var req struct {
			Code string `json:"code"`
			Name string `json:"name"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, map[string]any{"error": "请求格式错误: " + err.Error()})
			return
		}
		writeJSON(w, runStudioCode(req.Code, req.Name))
	})

	// 编译成可执行文件
	mux.HandleFunc("/api/build", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "POST only", http.StatusMethodNotAllowed)
			return
		}
		var req struct {
			Code string `json:"code"`
			Name string `json:"name"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, map[string]any{"error": "请求格式错误: " + err.Error()})
			return
		}
		writeJSON(w, buildStudioCode(req.Code, req.Name))
	})

	// 语法检查
	mux.HandleFunc("/api/check", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Code string `json:"code"`
		}
		json.NewDecoder(r.Body).Decode(&req)
		l := lexer.New(req.Code)
		p := parser.New(l)
		p.ParseProgram()
		errs := p.Errors()
		if len(errs) == 0 {
			writeJSON(w, map[string]any{"ok": true})
			return
		}
		writeJSON(w, map[string]any{"ok": false, "errors": errs})
	})

	addr := fmt.Sprintf("127.0.0.1:%d", port)
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		// 端口被占用时自动换一个
		for p := port + 1; p < port+20; p++ {
			addr = fmt.Sprintf("127.0.0.1:%d", p)
			if l2, e := net.Listen("tcp", addr); e == nil {
				ln = l2
				err = nil
				port = p
				break
			}
		}
		if err != nil {
			fmt.Fprintf(stderr, "error: 无法启动服务: %v\n", err)
			return 1
		}
	}

	url := fmt.Sprintf("http://%s/", addr)
	fmt.Fprintf(stdout, "Chronospp Studio 已启动\n")
	fmt.Fprintf(stdout, "  地址: %s\n", url)
	fmt.Fprintf(stdout, "  按 Ctrl+C 停止\n\n")

	if !noOpen {
		openBrowser(url)
	}

	srv := &http.Server{Handler: mux}
	if err := srv.Serve(ln); err != nil && err != http.ErrServerClosed {
		fmt.Fprintf(stderr, "error: %v\n", err)
		return 1
	}
	return 0
}

// runStudioCode 在内存里执行一段代码，捕获输出
func runStudioCode(code, name string) map[string]any {
	start := time.Now()

	// 语法检查先行
	l := lexer.New(code)
	p := parser.New(l)
	program := p.ParseProgram()
	if len(p.Errors()) > 0 {
		return map[string]any{
			"error": "语法错误:\n" + strings.Join(p.Errors(), "\n"),
			"ms":    time.Since(start).Milliseconds(),
		}
	}

	var buf strings.Builder
	old := interpreter.SetOutput(&buf)
	env := newStudioEnv()
	result := interpreter.Eval(program, env)
	interpreter.SetOutput(old)

	out := buf.String()
	resp := map[string]any{"output": out, "ms": time.Since(start).Milliseconds()}
	if result != nil && result.Type() == "ERROR" {
		resp["error"] = result.Inspect()
	}
	return resp
}

// buildStudioCode 把代码写成临时文件再编译
func buildStudioCode(code, name string) map[string]any {
	// 语法检查
	l := lexer.New(code)
	p := parser.New(l)
	p.ParseProgram()
	if len(p.Errors()) > 0 {
		return map[string]any{"error": "语法错误:\n" + strings.Join(p.Errors(), "\n")}
	}

	dir, err := os.MkdirTemp("", "chpp-studio-*")
	if err != nil {
		return map[string]any{"error": "无法创建临时目录: " + err.Error()}
	}
	defer os.RemoveAll(dir)

	base := strings.TrimSuffix(name, filepath.Ext(name))
	if base == "" {
		base = "app"
	}
	srcPath := filepath.Join(dir, base+".chpp")
	if err := os.WriteFile(srcPath, []byte(code), 0o644); err != nil {
		return map[string]any{"error": "写入失败: " + err.Error()}
	}

	outPath := filepath.Join(dir, base)
	var buf strings.Builder
	code2 := cmdBuild([]string{srcPath, "-o", outPath}, &buf, &buf)
	if code2 != 0 {
		return map[string]any{"error": "编译失败:\n" + buf.String()}
	}

	info, _ := os.Stat(outPath)
	size := int64(0)
	if info != nil {
		size = info.Size()
	}
	return map[string]any{
		"output": fmt.Sprintf("编译成功\n  产物: %s (%d 字节)\n  位置: %s", base, size, outPath),
	}
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	json.NewEncoder(w).Encode(v)
}

// openBrowser 尽力打开系统浏览器，失败不影响服务
func openBrowser(url string) {
	for _, cmd := range [][]string{
		{"xdg-open", url},
		{"open", url},
		{"cmd", "/c", "start", url},
	} {
		if _, err := os.Stat("/usr/bin/" + cmd[0]); err == nil || cmd[0] == "cmd" {
			if startErr := startDetached(cmd); startErr == nil {
				return
			}
		}
	}
}

// newStudioEnv 创建一个用于 Studio 的顶层环境
func newStudioEnv() *object.Environment {
	env := object.NewEnvironment()
	env.ImportChain = make(map[string]bool)
	return env
}

// startDetached 启动外部程序，不等待
func startDetached(cmd []string) error {
	c := exec.Command(cmd[0], cmd[1:]...)
	c.Stdout = nil
	c.Stderr = nil
	return c.Start()
}
