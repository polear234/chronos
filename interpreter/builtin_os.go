//go:build !js

package interpreter

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"chronos/object"
)

// builtinModules 内置模块：import "os" 直接拿到，不走文件系统
var builtinModules = map[string]func() *object.Module{
	"os":   newOSModule,
	"math": newMathModule,
}

// lookupBuiltinModule 按名字返回内置模块，没有则返回 nil
func lookupBuiltinModule(name string) *object.Module {
	// 允许 "os" / "os.chr" / "./os" 这类写法都命中
	base := strings.TrimSuffix(filepath.Base(name), filepath.Ext(name))
	if fn, ok := builtinModules[base]; ok {
		return fn()
	}
	return nil
}

// newOSModule 构造 Python 风格的 os 模块
func newOSModule() *object.Module {
	env := object.NewEnvironment()

	def := func(name string, fn func(args ...object.Object) object.Object) {
		env.Define(name, &object.Builtin{Fn: fn})
	}

	// ── 文件读写 ──────────────────────────────
	def("readFile", func(args ...object.Object) object.Object {
		if len(args) < 1 {
			return newError("os.readFile 需要一个文件路径")
		}
		path := args[0].Inspect()
		data, err := os.ReadFile(path)
		if err != nil {
			return newError("os.readFile: %v", err)
		}
		return &object.String{Value: string(data)}
	})

	def("writeFile", func(args ...object.Object) object.Object {
		if len(args) < 2 {
			return newError("os.writeFile 需要路径和内容")
		}
		path := args[0].Inspect()
		content := args[1].Inspect()
		// 自动建父目录
		if dir := filepath.Dir(path); dir != "" && dir != "." {
			os.MkdirAll(dir, 0o755)
		}
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			return newError("os.writeFile: %v", err)
		}
		return NIL
	})

	def("appendFile", func(args ...object.Object) object.Object {
		if len(args) < 2 {
			return newError("os.appendFile 需要路径和内容")
		}
		path := args[0].Inspect()
		content := args[1].Inspect()
		if dir := filepath.Dir(path); dir != "" && dir != "." {
			os.MkdirAll(dir, 0o755)
		}
		f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
		if err != nil {
			return newError("os.appendFile: %v", err)
		}
		defer f.Close()
		if _, err := f.WriteString(content); err != nil {
			return newError("os.appendFile: %v", err)
		}
		return NIL
	})

	def("exists", func(args ...object.Object) object.Object {
		if len(args) < 1 {
			return newError("os.exists 需要一个路径")
		}
		_, err := os.Stat(args[0].Inspect())
		return nativeBool(err == nil)
	})

	def("remove", func(args ...object.Object) object.Object {
		if len(args) < 1 {
			return newError("os.remove 需要一个路径")
		}
		if err := os.Remove(args[0].Inspect()); err != nil {
			return newError("os.remove: %v", err)
		}
		return NIL
	})

	def("mkdir", func(args ...object.Object) object.Object {
		if len(args) < 1 {
			return newError("os.mkdir 需要一个路径")
		}
		if err := os.MkdirAll(args[0].Inspect(), 0o755); err != nil {
			return newError("os.mkdir: %v", err)
		}
		return NIL
	})

	def("listDir", func(args ...object.Object) object.Object {
		dir := "."
		if len(args) > 0 {
			dir = args[0].Inspect()
		}
		entries, err := os.ReadDir(dir)
		if err != nil {
			return newError("os.listDir: %v", err)
		}
		names := make([]object.Object, 0, len(entries))
		for _, e := range entries {
			names = append(names, &object.String{Value: e.Name()})
		}
		return &object.Array{Elements: names}
	})

	// ── 路径 ──────────────────────────────
	def("join", func(args ...object.Object) object.Object {
		parts := make([]string, 0, len(args))
		for _, a := range args {
			parts = append(parts, a.Inspect())
		}
		return &object.String{Value: filepath.Join(parts...)}
	})

	def("abs", func(args ...object.Object) object.Object {
		if len(args) < 1 {
			return newError("os.abs 需要一个路径")
		}
		p, err := filepath.Abs(args[0].Inspect())
		if err != nil {
			return newError("os.abs: %v", err)
		}
		return &object.String{Value: p}
	})

	def("basename", func(args ...object.Object) object.Object {
		if len(args) < 1 {
			return newError("os.basename 需要一个路径")
		}
		return &object.String{Value: filepath.Base(args[0].Inspect())}
	})

	def("dirname", func(args ...object.Object) object.Object {
		if len(args) < 1 {
			return newError("os.dirname 需要一个路径")
		}
		return &object.String{Value: filepath.Dir(args[0].Inspect())}
	})

	// ── 系统信息 ──────────────────────────────
	def("getenv", func(args ...object.Object) object.Object {
		if len(args) < 1 {
			return newError("os.getenv 需要一个变量名")
		}
		return &object.String{Value: os.Getenv(args[0].Inspect())}
	})

	def("setenv", func(args ...object.Object) object.Object {
		if len(args) < 2 {
			return newError("os.setenv 需要变量名和值")
		}
		if err := os.Setenv(args[0].Inspect(), args[1].Inspect()); err != nil {
			return newError("os.setenv: %v", err)
		}
		return NIL
	})

	def("cwd", func(args ...object.Object) object.Object {
		dir, err := os.Getwd()
		if err != nil {
			return newError("os.cwd: %v", err)
		}
		return &object.String{Value: dir}
	})

	def("chdir", func(args ...object.Object) object.Object {
		if len(args) < 1 {
			return newError("os.chdir 需要一个路径")
		}
		if err := os.Chdir(args[0].Inspect()); err != nil {
			return newError("os.chdir: %v", err)
		}
		return NIL
	})

	// ── 执行命令 ──────────────────────────────
	// os.run("ls", "-l") → { code, out, err }
	def("run", func(args ...object.Object) object.Object {
		if len(args) < 1 {
			return newError("os.run 需要命令名")
		}
		name := args[0].Inspect()
		cmdArgs := make([]string, 0, len(args)-1)
		for _, a := range args[1:] {
			cmdArgs = append(cmdArgs, a.Inspect())
		}
		cmd := exec.Command(name, cmdArgs...)
		var stdout, stderr strings.Builder
		cmd.Stdout = &stdout
		cmd.Stderr = &stderr
		err := cmd.Run()

		code := 0
		if err != nil {
			if ee, ok := err.(*exec.ExitError); ok {
				code = ee.ExitCode()
			} else {
				return newError("os.run: %v", err)
			}
		}

		pairs := map[object.HashKey]object.HashPair{}
		put := func(k string, v object.Object) {
			key := &object.String{Value: k}
			pairs[key.HashKey()] = object.HashPair{Key: key, Value: v}
		}
		put("code", &object.Integer{Value: int64(code)})
		put("out", &object.String{Value: stdout.String()})
		put("err", &object.String{Value: stderr.String()})
		return &object.Hash{Pairs: pairs}
	})

	// os.exec("ls -l") 简写：整串交给 shell
	def("exec", func(args ...object.Object) object.Object {
		if len(args) < 1 {
			return newError("os.exec 需要一条命令")
		}
		command := args[0].Inspect()
		var cmd *exec.Cmd
		if isWindows() {
			cmd = exec.Command("cmd", "/c", command)
		} else {
			cmd = exec.Command("sh", "-c", command)
		}
		out, err := cmd.Output()
		if err != nil {
			if ee, ok := err.(*exec.ExitError); ok {
				return &object.String{Value: string(ee.Stderr)}
			}
			return newError("os.exec: %v", err)
		}
		return &object.String{Value: string(out)}
	})

	return &object.Module{
		Name:    "os",
		Env:     env,
		Exports: env.Snapshot(),
	}
}

func isWindows() bool {
	return filepath.Separator == '\\'
}
