//go:build windows && !js

package interpreter

import (
	"fmt"
	"syscall"
	"unsafe"
)

// ffiDlopen 加载 DLL
func ffiDlopen(name string) (uintptr, error) {
	h, err := syscall.LoadDLL(name)
	if err != nil {
		return 0, fmt.Errorf("无法加载 %q: %v", name, err)
	}
	return uintptr(h.Handle), nil
}

// ffiDlsym 查找导出函数
func ffiDlsym(handle uintptr, name string) (uintptr, error) {
	dll := &syscall.DLL{Handle: syscall.Handle(handle)}
	proc, err := dll.FindProc(name)
	if err != nil {
		return 0, fmt.Errorf("符号 %q 未找到: %v", name, err)
	}
	return proc.Addr(), nil
}

// ffiRegisterLibFunc Windows 不支持 purego.RegisterLibFunc，用 SyscallN 代替
func ffiRegisterLibFunc(fptr interface{}, handle uintptr, name string) {
	// Windows 上暂不支持浮点 ABI 优化
}

// ffiSyscallN 整数 ABI 调用（Windows x64 calling convention）
func ffiSyscallN(fptr uintptr, args ...uintptr) (uintptr, uintptr, uintptr) {
	var r1, r2 uintptr
	// 简单调用：最多 4 个参数
	switch len(args) {
	case 0:
		r, _, _ := syscall.SyscallN(fptr)
		return r, 0, 0
	case 1:
		r, _, _ := syscall.SyscallN(fptr, args[0])
		return r, 0, 0
	case 2:
		r, _, _ := syscall.SyscallN(fptr, args[0], args[1])
		return r, 0, 0
	case 3:
		r, _, _ := syscall.SyscallN(fptr, args[0], args[1], args[2])
		return r, 0, 0
	default:
		r, _, _ := syscall.SyscallN(fptr, args[:4]...)
		return r, 0, 0
	}
	_ = r1
	_ = r2
	_ = unsafe.Sizeof(args)
}
