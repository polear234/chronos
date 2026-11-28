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
	"sort"
	"strings"
	"time"
)

// chmManifest .chm 包里的清单文件
type chmManifest struct {
	Name        string            `json:"name"`
	Version     string            `json:"version"`
	Description string            `json:"description"`
	Author      string            `json:"author"`
	Main        string            `json:"main"`   // 入口文件，默认 index.chpp
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

// cmdInstall 实现 chpp --install xxx.chm / chpp install <包名>
func cmdInstall(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(stderr, "用法: chpp install <包.chm | 包名>")
		fmt.Fprintln(stderr, "\n  包.chm  安装本地 .chm 文件")
		fmt.Fprintln(stderr, "  包名    从 GitHub 仓库拉取（需先有 chronos-packages 仓库）")
		fmt.Fprintln(stderr, "\n相关: chpp uninstall <包名>、chpp packages")
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

	// 检查依赖
	if len(mf.Requires) > 0 {
		missing := []string{}
		for depName, depVer := range mf.Requires {
			depDir := filepath.Join(packagesDir(), depName)
			depManifest := filepath.Join(depDir, "manifest.json")
			depMF, err := readManifest(depManifest)
			if err != nil {
				missing = append(missing, fmt.Sprintf("%s %s（未安装）", depName, depVer))
				continue
			}
			if !versionSatisfies(depMF.Version, depVer) {
				missing = append(missing, fmt.Sprintf("%s 需要 %s，当前 %s",
					depName, depVer, depMF.Version))
			}
		}
		if len(missing) > 0 {
			fmt.Fprintf(stderr, "error: %s 的依赖未满足：\n", mf.Name)
			for _, m := range missing {
				fmt.Fprintf(stderr, "  - %s\n", m)
			}
			fmt.Fprintf(stderr, "\n先安装依赖：\n")
			for depName := range mf.Requires {
				fmt.Fprintf(stderr, "  chpp install %s\n", depName)
			}
			return 1
		}
		fmt.Fprintf(stdout, "  依赖检查通过（%d 个）\n", len(mf.Requires))
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
	req.Header.Set("User-Agent", "chpp-installer")

	resp, err := client.Do(req)
	if err != nil {
		return nil, 0, err
	}
	return resp.Body, resp.StatusCode, nil
}

// cmdUninstall 卸载第三方包
func cmdUninstall(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(stderr, "用法: chpp uninstall <包名>")
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
		fmt.Fprintf(stdout, "\n安装: chpp install 包名\n")
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

// cmdSearch 搜索可用包（从 GitHub 仓库列表）
func cmdSearch(args []string, stdout, stderr io.Writer) int {
	keyword := ""
	if len(args) > 0 {
		keyword = strings.ToLower(args[0])
	}

	fmt.Fprintf(stdout, "正在查询 chronos-packages 仓库…\n")

	// 用 GitHub API 列出 packages 目录
	client := &http.Client{Timeout: 20 * time.Second}
	url := fmt.Sprintf("https://api.github.com/repos/%s/chronos-packages/contents/packages", repoOwner)
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "chpp")

	resp, err := client.Do(req)
	if err != nil {
		fmt.Fprintf(stderr, "error: 无法连接 GitHub: %v\n", err)
		return 1
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		fmt.Fprintf(stderr, "error: HTTP %d\n", resp.StatusCode)
		return 1
	}

	var items []struct {
		Name string `json:"name"`
		Type string `json:"type"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&items); err != nil {
		fmt.Fprintf(stderr, "error: 解析失败: %v\n", err)
		return 1
	}

	// 过滤已安装
	installed := map[string]bool{}
	if entries, err := os.ReadDir(packagesDir()); err == nil {
		for _, e := range entries {
			installed[e.Name()] = true
		}
	}

	count := 0
	fmt.Fprintf(stdout, "\n可用包：\n\n")
	for _, it := range items {
		if it.Type != "dir" {
			continue
		}
		if keyword != "" && !strings.Contains(strings.ToLower(it.Name), keyword) {
			continue
		}
		mark := " "
		if installed[it.Name] {
			mark = "✓"
		}
		fmt.Fprintf(stdout, "  %s %-20s\n", mark, it.Name)
		count++
	}
	if count == 0 {
		fmt.Fprintf(stdout, "  （没有匹配的包）\n")
	}
	fmt.Fprintf(stdout, "\n安装: chpp install <包名>\n")
	return 0
}

// cmdInfo 显示包详情
func cmdInfo(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(stderr, "用法: chpp info <包名>")
		return 2
	}
	name := args[0]

	// 先看本地
	localManifest := filepath.Join(packagesDir(), name, "manifest.json")
	if mf, err := readManifest(localManifest); err == nil {
		fmt.Fprintf(stdout, "已安装: %s %s\n", mf.Name, mf.Version)
		if mf.Description != "" {
			fmt.Fprintf(stdout, "  %s\n", mf.Description)
		}
		if mf.Author != "" {
			fmt.Fprintf(stdout, "  作者: %s\n", mf.Author)
		}
		if len(mf.Requires) > 0 {
			fmt.Fprintf(stdout, "  依赖:\n")
			for k, v := range mf.Requires {
				fmt.Fprintf(stdout, "    %s %s\n", k, v)
			}
		}
		fmt.Fprintf(stdout, "  位置: %s\n", filepath.Dir(localManifest))
		return 0
	}

	// 从 GitHub 查
	url := fmt.Sprintf("https://api.github.com/repos/%s/chronos-packages/contents/packages/%s/manifest.json", repoOwner, name)
	client := &http.Client{Timeout: 20 * time.Second}
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("Accept", "application/vnd.github.raw")
	req.Header.Set("User-Agent", "chpp")

	resp, err := client.Do(req)
	if err != nil {
		fmt.Fprintf(stderr, "error: %v\n", err)
		return 1
	}
	defer resp.Body.Close()

	if resp.StatusCode == 404 {
		fmt.Fprintf(stderr, "error: 找不到包 %q\n", name)
		return 1
	}

	var mf chmManifest
	if err := json.NewDecoder(resp.Body).Decode(&mf); err != nil {
		fmt.Fprintf(stderr, "error: 解析失败: %v\n", err)
		return 1
	}

	fmt.Fprintf(stdout, "%s %s（未安装）\n", mf.Name, mf.Version)
	if mf.Description != "" {
		fmt.Fprintf(stdout, "  %s\n", mf.Description)
	}
	if mf.Author != "" {
		fmt.Fprintf(stdout, "  作者: %s\n", mf.Author)
	}
	if len(mf.Requires) > 0 {
		fmt.Fprintf(stdout, "  依赖:\n")
		for k, v := range mf.Requires {
			fmt.Fprintf(stdout, "    %s %s\n", k, v)
		}
	}
	fmt.Fprintf(stdout, "\n安装: chpp install %s\n", name)
	return 0
}

// cmdFreeze 输出已安装包的精确版本（类似 pip freeze）
func cmdFreeze(args []string, stdout, stderr io.Writer) int {
	dir := packagesDir()
	entries, err := os.ReadDir(dir)
	if err != nil || len(entries) == 0 {
		fmt.Fprintln(stdout, "# 没有安装任何包")
		return 0
	}

	var lines []string
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		mf, err := readManifest(filepath.Join(dir, e.Name(), "manifest.json"))
		if err != nil {
			continue
		}
		lines = append(lines, fmt.Sprintf("%s==%s", mf.Name, mf.Version))
	}
	sort.Strings(lines)
	for _, l := range lines {
		fmt.Fprintln(stdout, l)
	}
	return 0
}

// cmdPack 把源码目录打包成 .chm（类似 pip 的 build）
//   chpp pack <目录> [-o 输出.chm]
func cmdPack(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(stderr, "用法: chpp pack <目录> [-o 输出.chm]")
		fmt.Fprintln(stderr, "\n目录里必须有 manifest.json")
		return 2
	}

	srcDir := args[0]
	outPath := ""

	// 解析 -o
	for i := 1; i < len(args); i++ {
		if (args[i] == "-o" || args[i] == "--output") && i+1 < len(args) {
			outPath = args[i+1]
			i++
		}
	}

	// 检查目录
	info, err := os.Stat(srcDir)
	if err != nil || !info.IsDir() {
		fmt.Fprintf(stderr, "error: 目录不存在: %s\n", srcDir)
		return 1
	}

	// 读 manifest
	mfPath := filepath.Join(srcDir, "manifest.json")
	mf, err := readManifest(mfPath)
	if err != nil {
		fmt.Fprintf(stderr, "error: 缺少或无效的 manifest.json: %v\n", err)
		return 1
	}
	if mf.Name == "" {
		fmt.Fprintln(stderr, "error: manifest.json 缺少 name")
		return 1
	}

	if outPath == "" {
		outPath = mf.Name + ".chm"
	}

	// 创建 tar.gz
	f, err := os.Create(outPath)
	if err != nil {
		fmt.Fprintf(stderr, "error: 无法创建 %s: %v\n", outPath, err)
		return 1
	}
	defer f.Close()

	gz := gzip.NewWriter(f)
	defer gz.Close()
	tw := tar.NewWriter(gz)
	defer tw.Close()

	err = filepath.Walk(srcDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		// 跳过输出文件自身和隐藏文件
		if path == outPath {
			return nil
		}
		base := filepath.Base(path)
		if strings.HasPrefix(base, ".") && path != srcDir {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		rel, err := filepath.Rel(srcDir, path)
		if err != nil {
			return err
		}
		if rel == "." {
			return nil
		}

		hdr, err := tar.FileInfoHeader(info, "")
		if err != nil {
			return err
		}
		hdr.Name = rel
		if info.IsDir() {
			hdr.Name += "/"
		}
		if err := tw.WriteHeader(hdr); err != nil {
			return err
		}
		if !info.IsDir() {
			data, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			if _, err := tw.Write(data); err != nil {
				return err
			}
		}
		return nil
	})

	if err != nil {
		fmt.Fprintf(stderr, "error: 打包失败: %v\n", err)
		return 1
	}

	if err := tw.Close(); err != nil {
		fmt.Fprintf(stderr, "error: %v\n", err)
		return 1
	}
	if err := gz.Close(); err != nil {
		fmt.Fprintf(stderr, "error: %v\n", err)
		return 1
	}

	stat, _ := os.Stat(outPath)
	size := int64(0)
	if stat != nil {
		size = stat.Size()
	}
	fmt.Fprintf(stdout, "✓ 已打包: %s (%d 字节)\n", outPath, size)
	fmt.Fprintf(stdout, "  包名: %s\n", mf.Name)
	fmt.Fprintf(stdout, "  版本: %s\n", mf.Version)
	fmt.Fprintf(stdout, "\n安装测试: chpp install %s\n", outPath)
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

// versionSatisfies 判断 version 是否满足约束 constraint
// 支持：">=1.0.0"、"^1.0.0"、"~1.0.0"、">1.0"、"<2.0"、"1.0.0"（精确）
func versionSatisfies(version, constraint string) bool {
	constraint = strings.TrimSpace(constraint)
	if constraint == "" || constraint == "*" {
		return true
	}

	// 精确匹配
	if !strings.ContainsAny(constraint, "<>=^~") {
		return version == constraint
	}

	// 解析操作符
	op := ""
	rest := constraint
	for _, prefix := range []string{">=", "<=", "^", "~", ">", "<", "="} {
		if strings.HasPrefix(constraint, prefix) {
			op = prefix
			rest = strings.TrimSpace(constraint[len(prefix):])
			break
		}
	}

	cmp := compareVersions(version, rest)
	switch op {
	case ">=":
		return cmp >= 0
	case "<=":
		return cmp <= 0
	case ">":
		return cmp > 0
	case "<":
		return cmp < 0
	case "=", "":
		return cmp == 0
	case "^":
		// ^1.2.3 表示 >=1.2.3 且 <2.0.0
		if cmp < 0 {
			return false
		}
		vMajor := strings.Split(version, ".")[0]
		cMajor := strings.Split(rest, ".")[0]
		return vMajor == cMajor
	case "~":
		// ~1.2.3 表示 >=1.2.3 且 <1.3.0
		if cmp < 0 {
			return false
		}
		vParts := strings.Split(version, ".")
		cParts := strings.Split(rest, ".")
		if len(vParts) < 2 || len(cParts) < 2 {
			return true
		}
		return vParts[0] == cParts[0] && vParts[1] == cParts[1]
	}
	return false
}

// compareVersions 比较两个版本号，返回 -1/0/1
func compareVersions(a, b string) int {
	parse := func(v string) []int {
		v = strings.TrimPrefix(v, "v")
		if i := strings.IndexAny(v, "-+"); i >= 0 {
			v = v[:i]
		}
		parts := strings.Split(v, ".")
		out := make([]int, len(parts))
		for i, p := range parts {
			n := 0
			for _, r := range p {
				if r < '0' || r > '9' {
					break
				}
				n = n*10 + int(r-'0')
			}
			out[i] = n
		}
		return out
	}
	av, bv := parse(a), parse(b)
	maxLen := len(av)
	if len(bv) > maxLen {
		maxLen = len(bv)
	}
	for i := 0; i < maxLen; i++ {
		var x, y int
		if i < len(av) {
			x = av[i]
		}
		if i < len(bv) {
			y = bv[i]
		}
		if x < y {
			return -1
		}
		if x > y {
			return 1
		}
	}
	return 0
}

// fileExists 判断文件是否存在且不是目录
func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}
