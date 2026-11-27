//go:build !js

package main

import (
	"flag"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

// cmdServe 实现 chpp serve [目录] [-p 端口] [-open]
// 内置静态网站服务器，类似 python -m http.server
func cmdServe(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("serve", flag.ContinueOnError)
	fs.SetOutput(stderr)
	port := fs.Int("p", 8000, "监听端口")
	portLong := fs.Int("port", 0, "监听端口（同 -p）")
	host := fs.String("host", "", "监听地址（默认全部网卡）")
	listen := fs.Bool("l", false, "只监听 127.0.0.1")
	noIndex := fs.Bool("no-index", false, "不生成目录列表")
	fs.Usage = func() {
		fmt.Fprintln(stderr, "用法: chpp serve [目录] [选项]")
		fmt.Fprintln(stderr, "\n选项:")
		fs.PrintDefaults()
	}
	// Go 的 flag 遇到第一个非选项参数就停止解析，
	// 而用户习惯写成 `chpp serve ./site -p 8080`（目录在前）。
	// 这里把非选项参数挪到末尾，让 -p 之类也能生效。
	var opts, positional []string
	for i := 0; i < len(args); i++ {
		a := args[i]
		if strings.HasPrefix(a, "-") && a != "-" {
			opts = append(opts, a)
			// 带值的选项：下一个参数不是选项就一起带上
			if !strings.Contains(a, "=") && i+1 < len(args) && !strings.HasPrefix(args[i+1], "-") {
				opts = append(opts, args[i+1])
				i++
			}
		} else {
			positional = append(positional, a)
		}
	}
	if err := fs.Parse(append(opts, positional...)); err != nil {
		return 2
	}

	// -port 优先于 -p（当它被显式指定时）
	p := *port
	if *portLong != 0 {
		p = *portLong
	}

	dir := "."
	if rest := fs.Args(); len(rest) > 0 {
		dir = rest[0]
	}
	abs, err := filepath.Abs(dir)
	if err != nil {
		fmt.Fprintf(stderr, "error: 无效目录 %s: %v\n", dir, err)
		return 1
	}
	info, err := os.Stat(abs)
	if err != nil || !info.IsDir() {
		fmt.Fprintf(stderr, "error: 目录不存在: %s\n", abs)
		return 1
	}

	addr := fmt.Sprintf("%s:%d", *host, p)
	if *listen {
		addr = fmt.Sprintf("127.0.0.1:%d", p)
	}

	// 目录列表（可关闭）
	var handler http.Handler
	if *noIndex {
		handler = http.FileServer(http.Dir(abs))
	} else {
		handler = dirLister(abs)
	}

	ln, err := net.Listen("tcp", addr)
	if err != nil {
		fmt.Fprintf(stderr, "error: 无法监听 %s: %v\n", addr, err)
		return 1
	}

	actual := ln.Addr().String()
	hostPart, portPart, _ := net.SplitHostPort(actual)
	if hostPart == "" || hostPart == "0.0.0.0" || hostPart == "::" {
		hostPart = "localhost"
	}
	fmt.Fprintf(stdout, "Chronospp 静态服务器已启动\n")
	fmt.Fprintf(stdout, "  目录: %s\n", abs)
	fmt.Fprintf(stdout, "  地址: http://%s:%s/\n", hostPart, portPart)
	if hostPart == "localhost" {
		// 打印局域网地址，方便手机/其它设备访问
		for _, ip := range localIPs() {
			fmt.Fprintf(stdout, "  局域网: http://%s:%s/\n", ip, portPart)
		}
	}
	fmt.Fprintln(stdout, "  按 Ctrl+C 停止")

	srv := &http.Server{Handler: handler}
	if err := srv.Serve(ln); err != nil && err != http.ErrServerClosed {
		fmt.Fprintf(stderr, "error: %v\n", err)
		return 1
	}
	return 0
}

// dirLister 返回一个带目录列表的处理器（比默认 FileServer 好看一些）
func dirLister(root string) http.Handler {
	fileServer := http.FileServer(http.Dir(root))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 只处理目录请求，其它交给 FileServer
		clean := filepath.Clean(r.URL.Path)
		if clean == "." {
			clean = "/"
		}
		full := filepath.Join(root, filepath.FromSlash(clean))
		info, err := os.Stat(full)
		if err != nil || !info.IsDir() {
			fileServer.ServeHTTP(w, r)
			return
		}

		entries, err := os.ReadDir(full)
		if err != nil {
			http.Error(w, "无法读取目录", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		var b strings.Builder
		b.WriteString("<!DOCTYPE html><html><head><meta charset=\"utf-8\">")
		b.WriteString("<meta name=\"viewport\" content=\"width=device-width,initial-scale=1\">")
		b.WriteString("<title>Index of " + htmlEscape(clean) + "</title>")
		b.WriteString("<style>body{font-family:system-ui,-apple-system,sans-serif;background:#0d1117;color:#c9d1d9;padding:24px;line-height:1.7}")
		b.WriteString("a{color:#58a6ff;text-decoration:none}a:hover{text-decoration:underline}")
		b.WriteString("h1{font-size:18px;font-weight:600;border-bottom:1px solid #30363d;padding-bottom:12px}")
		b.WriteString(".d{color:#7ee787}.f{color:#c9d1d9}.s{color:#8b949e;font-size:12px;margin-left:8px}</style></head><body>")
		b.WriteString("<h1>Index of " + htmlEscape(clean) + "</h1><ul>")

		if clean != "/" {
			b.WriteString("<li><a href=\"../\">../</a></li>")
		}
		for _, e := range entries {
			name := e.Name()
			display := name
			cls := "f"
			if e.IsDir() {
				display += "/"
				cls = "d"
			}
			href := "./" + urlPathEscape(name)
			if e.IsDir() {
				href += "/"
			}
			var size string
			if fi, err := e.Info(); err == nil && !e.IsDir() {
				size = humanSize(fi.Size())
			}
			b.WriteString("<li><a class=\"" + cls + "\" href=\"" + href + "\">" + htmlEscape(display) + "</a>")
			if size != "" {
				b.WriteString("<span class=\"s\">" + size + "</span>")
			}
			b.WriteString("</li>")
		}
		b.WriteString("</ul></body></html>")
		io.WriteString(w, b.String())
	})
}

func htmlEscape(s string) string {
	r := strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;", `"`, "&quot;")
	return r.Replace(s)
}

func urlPathEscape(s string) string {
	// 简单的路径转义：空格等常见字符
	r := strings.NewReplacer(" ", "%20", "#", "%23", "?", "%3F", "&", "&amp;")
	return r.Replace(s)
}

func humanSize(n int64) string {
	const unit = 1024
	if n < unit {
		return fmt.Sprintf("%d B", n)
	}
	div, exp := int64(unit), 0
	for m := n / unit; m >= unit; m /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(n)/float64(div), "KMGTPE"[exp])
}

// localIPs 返回本机非回环 IPv4 地址
func localIPs() []string {
	var out []string
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return out
	}
	for _, a := range addrs {
		if ipnet, ok := a.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ip4 := ipnet.IP.To4(); ip4 != nil {
				out = append(out, ip4.String())
			}
		}
	}
	return out
}
