//go:build (linux || darwin) && !js

package interpreter

import (
	"github.com/ebitengine/purego"
)

// ffiDlopen 加载动态库
func ffiDlopen(name string) (uintptr, error) {
	return purego.Dlopen(name, purego.RTLD_NOW|purego.RTLD_GLOBAL)
}

// ffiDlsym 查找符号
func ffiDlsym(handle uintptr, name string) (uintptr, error) {
	return purego.Dlsym(handle, name)
}

// ffiRegisterLibFunc 注册浮点函数（Unix 用 purego）
func ffiRegisterLibFunc(fptr interface{}, handle uintptr, name string) {
	purego.RegisterLibFunc(fptr, handle, name)
}

// ffiSyscallN 整数 ABI 调用
func ffiSyscallN(fptr uintptr, args ...uintptr) (uintptr, uintptr, uintptr) {
	return purego.SyscallN(fptr, args...)
}
