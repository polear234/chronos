//go:build !js

package main

import (
	"archive/tar"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// chmManifest .chm 包里的清单文件
type chmManifest struct {
	Name        string            `json:"name"`
	Version     string            `json:"version"`
	Description string            `json:"description"`
	Author      string            `json:"author"`
	Main        string            `json:"main"`   // 入口文件，默认 index.chr
	Files       []string          `json:"files"`  // 包含的文件列表
	Requires    map[string]string `json:"requires"` // 依赖的其它包
}

// packagesDir 返回第三方包安装目录：~/.chronos/packages
func packagesDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		home = "."
	}
	return filepath.Join(home, ".chronos", "packages")
}

// cmdInstall 实现 chr --install xxx.chm / chr install <包名>
func cmdInstall(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(stderr, "用法: chr install <包.chm | 包名>")
		fmt.Fprintln(stderr, "\n  包.chm  安装本地 .chm 文件")
		fmt.Fprintln(stderr, "  包名    从 GitHub 仓库拉取（需先有 chronos-packages 仓库）")
		fmt.Fprintln(stderr, "\n相关: chr uninstall <包名>、chr packages")
		return 2
	}

	target := args[0]

	// 本地 .chm 文件
	if strings.HasSuffix(strings.ToLower(target), ".chm") || fileExists(target) {
		return installFromFile(target, stdout, stderr)
	}

	// 包名：从 GitHub 拉
	return installFromGitHub(target, stdout, stderr)
}

func installFromFile(path string, stdout, stderr io.Writer) int {
	if !fileExists(path) {
		fmt.Fprintf(stderr, "error: 找不到文件 %s\n", path)
		return 1
	}

	f, err := os.Open(path)
	if err != nil {
		fmt.Fprintf(stderr, "error: 无法打开 %s: %v\n", path, err)
		return 1
	}
	defer f.Close()

	gz, err := gzip.NewReader(f)
	if err != nil {
		fmt.Fprintf(stderr, "error: %s 不是有效的 .chm 包（gzip 解压失败）\n", path)
		return 1
	}
	defer gz.Close()

	// 先读到内存解析 manifest，再决定装到哪
	tmpDir, err := os.MkdirTemp("", "chm-install-*")
	if err != nil {
		fmt.Fprintf(stderr, "error: 无法创建临时目录: %v\n", err)
		return 1
	}
	defer os.RemoveAll(tmpDir)

	if err := extractTar(gz, tmpDir); err != nil {
		fmt.Fprintf(stderr, "error: 解包失败: %v\n", err)
		return 1
	}

	manifestPath := filepath.Join(tmpDir, "manifest.json")
	mf, err := readManifest(manifestPath)
	if err != nil {
		fmt.Fprintf(stderr, "error: 包里缺少有效的 manifest.json: %v\n", err)
		return 1
	}
	if mf.Name == "" {
		fmt.Fprintln(stderr, "error: manifest.json 缺少 name 字段")
		return 1
	}

	dest := filepath.Join(packagesDir(), mf.Name)
	fmt.Fprintf(stdout, "正在安装 %s %s…\n", mf.Name, mf.Version)

	// 已存在则先删（覆盖安装）
	if _, err := os.Stat(dest); err == nil {
		if err := os.RemoveAll(dest); err != nil {
			fmt.Fprintf(stderr, "error: 无法覆盖已有版本: %v\n", err)
			return 1
		}
	}

	if err := os.MkdirAll(dest, 0o755); err != nil {
		fmt.Fprintf(stderr, "error: 无法创建安装目录: %v\n", err)
		return 1
	}
	if err := copyDir(tmpDir, dest); err != nil {
		fmt.Fprintf(stderr, "error: 复制文件失败: %v\n", err)
		return 1
	}

	fmt.Fprintf(stdout, "✓ 已安装到 %s\n", dest)
	if mf.Description != "" {
		fmt.Fprintf(stdout, "  %s\n", mf.Description)
	}
	fmt.Fprintf(stdout, "  用法: import \"%s\"\n", mf.Name)
	return 0
}

func installFromGitHub(name string, stdout, stderr io.Writer) int {
	fmt.Fprintf(stdout, "正在从 GitHub 获取 %s…\n", name)

	client := &http.Client{Timeout: 60 * time.Second}

	// 优先用 GitHub API 的内容接口（api.github.com 在国内通常可达，
	// raw.githubusercontent.com 经常被重置）
	body, status, err := fetchPackageViaAPI(client, name)
	if body != nil {
		defer body.Close()
	}
	if err != nil || status != 200 {
		// 回退到 raw 域名
		rawURL := fmt.Sprintf("https://raw.githubusercontent.com/%s/chronos-packages/main/packages/%s/%s.chm",
			repoOwner, name, name)
		resp, rerr := client.Get(rawURL)
		if rerr != nil {
			fmt.Fprintf(stderr, "error: 下载失败: %v\n", rerr)
			return 1
		}
		if resp.StatusCode == 404 {
			fmt.Fprintf(stderr, "error: 找不到包 %q\n", name)
			fmt.Fprintf(stderr, "提示: 检查包名，或到 https://github.com/%s/chronos-packages 看看有哪些\n", repoOwner)
			return 1
		}
		if resp.StatusCode != 200 {
			fmt.Fprintf(stderr, "error: HTTP %d\n", resp.StatusCode)
			return 1
		}
		body = resp.Body
		defer body.Close()
	}

	tmp, err := os.CreateTemp("", "chm-download-*.chm")
	if err != nil {
		fmt.Fprintf(stderr, "error: %v\n", err)
		return 1
	}
	tmpPath := tmp.Name()
	defer os.Remove(tmpPath)

	if _, err := io.Copy(tmp, body); err != nil {
		tmp.Close()
		fmt.Fprintf(stderr, "error: 下载中断: %v\n", err)
		return 1
	}
	tmp.Close()

	return installFromFile(tmpPath, stdout, stderr)
}

// fetchPackageViaAPI 用 GitHub Contents API 拿包内容（走 api.github.com）
func fetchPackageViaAPI(client *http.Client, name string) (io.ReadCloser, int, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/chronos-packages/contents/packages/%s/%s.chm",
		repoOwner, name, name)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, 0, err
	}
	req.Header.Set("Accept", "application/vnd.github.raw")
	req.Header.Set("User-Agent", "chr-installer")

	resp, err := client.Do(req)
	if err != nil {
		return nil, 0, err
	}
	return resp.Body, resp.StatusCode, nil
}

// cmdUninstall 卸载第三方包
func cmdUninstall(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(stderr, "用法: chr uninstall <包名>")
		return 2
	}
	name := args[0]
	dest := filepath.Join(packagesDir(), name)

	if _, err := os.Stat(dest); os.IsNotExist(err) {
		fmt.Fprintf(stderr, "error: 没有安装 %q\n", name)
		return 1
	}
	if err := os.RemoveAll(dest); err != nil {
		fmt.Fprintf(stderr, "error: 卸载失败: %v\n", err)
		return 1
	}
	fmt.Fprintf(stdout, "✓ 已卸载 %s\n", name)
	return 0
}

// cmdPackages 列出已安装的包
func cmdPackages(args []string, stdout, stderr io.Writer) int {
	dir := packagesDir()
	entries, err := os.ReadDir(dir)
	if err != nil || len(entries) == 0 {
		fmt.Fprintln(stdout, "还没有安装任何第三方包。")
		fmt.Fprintf(stdout, "\n安装: chr install 包名\n")
		fmt.Fprintf(stdout, "目录: %s\n", dir)
		return 0
	}

	fmt.Fprintf(stdout, "已安装 %d 个包（%s）：\n\n", len(entries), dir)
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		mf, err := readManifest(filepath.Join(dir, e.Name(), "manifest.json"))
		if err != nil {
			fmt.Fprintf(stdout, "  %-20s (清单损坏)\n", e.Name())
			continue
		}
		ver := mf.Version
		if ver == "" {
			ver = "?"
		}
		fmt.Fprintf(stdout, "  %-20s %-10s %s\n", mf.Name, ver, mf.Description)
	}
	return 0
}

// ── 辅助 ──────────────────────────────

func readManifest(path string) (*chmManifest, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var mf chmManifest
	if err := json.Unmarshal(data, &mf); err != nil {
		return nil, err
	}
	return &mf, nil
}

// extractTar 把 tar 流解到 destDir，拒绝路径穿越
func extractTar(r io.Reader, destDir string) error {
	tr := tar.NewReader(r)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}

		// 防路径穿越
		clean := filepath.Clean(hdr.Name)
		if strings.HasPrefix(clean, "..") || filepath.IsAbs(clean) {
			return fmt.Errorf("包内含非法路径: %s", hdr.Name)
		}
		target := filepath.Join(destDir, clean)

		switch hdr.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, 0o755); err != nil {
				return err
			}
		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
				return err
			}
			out, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
			if err != nil {
				return err
			}
			if _, err := io.Copy(out, tr); err != nil {
				out.Close()
				return err
			}
			out.Close()
		}
	}
}

func copyDir(src, dst string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		if rel == "." {
			return nil
		}
		target := filepath.Join(dst, rel)
		if info.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		return copyFile(path, target)
	})
}

// fileExists 判断文件是否存在且不是目录
func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}
