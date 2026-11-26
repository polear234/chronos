//go:build !js

package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

const (
	repoOwner = "polear234"
	repoName  = "chronos"
	apiLatest = "https://api.github.com/repos/" + repoOwner + "/" + repoName + "/releases/latest"
)

type ghRelease struct {
	TagName string `json:"tag_name"`
	Assets  []struct {
		Name        string `json:"name"`
		DownloadURL string `json:"browser_download_url"`
		Size        int64  `json:"size"`
	} `json:"assets"`
}

// cmdUpgrade 实现 chr --upgrade [--check] [--force]
// 从 GitHub 拉取最新版本并替换当前可执行文件。
func cmdUpgrade(args []string, stdout, stderr io.Writer) int {
	checkOnly := false
	force := false
	for _, a := range args {
		switch a {
		case "--check", "-check":
			checkOnly = true
		case "--force", "-force", "-f":
			force = true
		case "--help", "-h":
			fmt.Fprintln(stdout, "用法: chr --upgrade [--check] [--force]")
			fmt.Fprintln(stdout, "\n  --check   只检查是否有新版本，不下载")
			fmt.Fprintln(stdout, "  --force   版本相同也强制重新下载")
			return 0
		default:
			fmt.Fprintf(stderr, "error: 未知参数 %s\n", a)
			return 2
		}
	}

	fmt.Fprintf(stdout, "当前版本: %s\n", version)
	fmt.Fprintf(stdout, "正在检查更新…\n")

	rel, err := fetchLatestRelease()
	if err != nil {
		fmt.Fprintf(stderr, "error: 检查更新失败: %v\n", err)
		return 1
	}

	latest := strings.TrimPrefix(rel.TagName, "v")
	fmt.Fprintf(stdout, "最新版本: %s\n", latest)

	if !force && !isNewer(latest, version) {
		fmt.Fprintf(stdout, "已经是最新版本。\n")
		return 0
	}

	if checkOnly {
		fmt.Fprintf(stdout, "\n有新版本可用: %s → %s\n", version, latest)
		fmt.Fprintf(stdout, "运行 chr --upgrade 更新\n")
		return 0
	}

	// 找当前平台的产物
	assetName := assetNameFor(runtime.GOOS, runtime.GOARCH)
	var url string
	var size int64
	for _, a := range rel.Assets {
		if a.Name == assetName {
			url = a.DownloadURL
			size = a.Size
			break
		}
	}
	if url == "" {
		fmt.Fprintf(stderr, "error: 该版本没有 %s/%s 的构建（找的是 %s）\n",
			runtime.GOOS, runtime.GOARCH, assetName)
		return 1
	}

	fmt.Fprintf(stdout, "正在下载 %s（%.1f MB）…\n", assetName, float64(size)/1024/1024)
	tmp, err := downloadToTemp(url)
	if err != nil {
		fmt.Fprintf(stderr, "error: 下载失败: %v\n", err)
		return 1
	}
	defer os.Remove(tmp)

	// 替换自身
	self, err := os.Executable()
	if err != nil {
		fmt.Fprintf(stderr, "error: 无法定位当前程序: %v\n", err)
		return 1
	}
	self, _ = filepath.EvalSymlinks(self)

	if err := replaceSelf(tmp, self); err != nil {
		fmt.Fprintf(stderr, "error: 替换失败: %v\n", err)
		fmt.Fprintf(stderr, "提示: 可能需要管理员权限，或手动下载: %s\n", url)
		return 1
	}

	fmt.Fprintf(stdout, "✓ 已更新到 %s\n", latest)
	return 0
}

func fetchLatestRelease() (*ghRelease, error) {
	client := &http.Client{Timeout: 20 * time.Second}
	req, err := http.NewRequest("GET", apiLatest, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "chr-upgrader")

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == 404 {
		return nil, fmt.Errorf("仓库不存在或没有 release")
	}
	if resp.StatusCode == 403 {
		return nil, fmt.Errorf("GitHub API 限流，稍后再试")
	}
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	var rel ghRelease
	if err := json.NewDecoder(resp.Body).Decode(&rel); err != nil {
		return nil, fmt.Errorf("解析响应失败: %w", err)
	}
	return &rel, nil
}

// assetNameFor 按平台拼出发布产物名，例如 chr-linux-amd64
func assetNameFor(goos, goarch string) string {
	name := fmt.Sprintf("chr-%s-%s", goos, goarch)
	if goos == "windows" {
		name += ".exe"
	}
	return name
}

// isNewer 判断 latest 是否比 current 新（按 x.y.z 逐段比较）
func isNewer(latest, current string) bool {
	l := parseVer(latest)
	c := parseVer(current)
	for i := 0; i < len(l) && i < len(c); i++ {
		if l[i] != c[i] {
			return l[i] > c[i]
		}
	}
	return len(l) > len(c)
}

func parseVer(s string) []int {
	s = strings.TrimPrefix(s, "v")
	// 去掉 -rc.1 之类的后缀
	if i := strings.IndexAny(s, "-+"); i >= 0 {
		s = s[:i]
	}
	parts := strings.Split(s, ".")
	out := make([]int, 0, len(parts))
	for _, p := range parts {
		n := 0
		for _, r := range p {
			if r < '0' || r > '9' {
				break
			}
			n = n*10 + int(r-'0')
		}
		out = append(out, n)
	}
	return out
}

func downloadToTemp(url string) (string, error) {
	client := &http.Client{Timeout: 5 * time.Minute}
	resp, err := client.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return "", fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	f, err := os.CreateTemp("", "chr-upgrade-*")
	if err != nil {
		return "", err
	}
	if _, err := io.Copy(f, resp.Body); err != nil {
		f.Close()
		os.Remove(f.Name())
		return "", err
	}
	if err := f.Close(); err != nil {
		os.Remove(f.Name())
		return "", err
	}
	return f.Name(), nil
}

// replaceSelf 用新文件替换当前可执行文件。
// Windows 上正在运行的程序不能直接覆盖，先改名旧文件再放新的。
func replaceSelf(newPath, selfPath string) error {
	if err := os.Chmod(newPath, 0o755); err != nil && runtime.GOOS != "windows" {
		return err
	}

	if runtime.GOOS == "windows" {
		old := selfPath + ".old"
		os.Remove(old)
		if err := os.Rename(selfPath, old); err != nil {
			return fmt.Errorf("无法重命名旧版本（程序可能在运行）: %w", err)
		}
		if err := copyFile(newPath, selfPath); err != nil {
			// 回滚
			os.Rename(old, selfPath)
			return err
		}
		// 下次启动时清理；现在删不掉也无妨
		os.Remove(old)
		return nil
	}

	// Unix：先写临时文件到同目录再原子替换
	dir := filepath.Dir(selfPath)
	tmpInDir, err := os.CreateTemp(dir, ".chr-upgrade-*")
	if err != nil {
		return err
	}
	tmpName := tmpInDir.Name()
	tmpInDir.Close()
	defer os.Remove(tmpName)

	if err := copyFile(newPath, tmpName); err != nil {
		return err
	}
	if err := os.Chmod(tmpName, 0o755); err != nil {
		return err
	}
	return os.Rename(tmpName, selfPath)
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o755)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		return err
	}
	return out.Close()
}
