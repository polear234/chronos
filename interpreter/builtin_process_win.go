//go:build windows && !js

package interpreter

import (
	"fmt"
	"strings"
	"syscall"
	"unsafe"

	"chronospp/object"
)

// Windows 内存访问权限
const (
	PROCESS_VM_READ       = 0x0010
	PROCESS_VM_WRITE      = 0x0020
	PROCESS_VM_OPERATION  = 0x0008
	PROCESS_QUERY_INFO    = 0x0400
	MEM_COMMIT            = 0x1000
	PAGE_NOACCESS         = 0x01
	PAGE_READONLY         = 0x02
	PAGE_READWRITE        = 0x04
	PAGE_WRITECOPY        = 0x08
	PAGE_EXECUTE          = 0x10
	PAGE_EXECUTE_READ     = 0x20
	PAGE_EXECUTE_READWRITE = 0x40
	PAGE_GUARD            = 0x100
	TH32CS_SNAPPROCESS    = 0x00000002
	MAX_PATH              = 260
)

var (
	kernel32            = syscall.NewLazyDLL("kernel32.dll")
	procOpenProcess     = kernel32.NewProc("OpenProcess")
	procCloseHandle     = kernel32.NewProc("CloseHandle")
	procReadProcessMem  = kernel32.NewProc("ReadProcessMemory")
	procWriteProcessMem = kernel32.NewProc("WriteProcessMemory")
	procVirtualQueryEx  = kernel32.NewProc("VirtualQueryEx")
	procCreateSnapshot  = kernel32.NewProc("CreateToolhelp32Snapshot")
	procProcess32First  = kernel32.NewProc("Process32FirstW")
	procProcess32Next   = kernel32.NewProc("Process32NextW")
)

// PROCESSENTRY32W 进程快照结构
type processEntry32W struct {
	Size              uint32
	CntUsage          uint32
	ProcessID         uint32
	DefaultHeapID     uintptr
	ModuleID          uint32
	CntThreads        uint32
	ParentProcessID   uint32
	PriClassBase      int32
	Flags             uint32
	ExeFile           [MAX_PATH]uint16
}

// MEMORY_BASIC_INFORMATION 内存信息结构
type memoryBasicInformation struct {
	BaseAddress       uintptr
	AllocationBase    uintptr
	AllocationProtect uint32
	RegionSize        uintptr
	State             uint32
	Protect           uint32
	Type              uint32
}

// newProcessModule 构造进程内存读写模块（Windows 版）
func newProcessModule() *object.Module {
	env := object.NewEnvironment()

	// process.find(name) → PID
	env.Define("find", &object.Builtin{
		Fn: func(args ...object.Object) object.Object {
			if len(args) < 1 {
				return newError("process.find 需要进程名")
			}
			target := strings.ToLower(args[0].Inspect())
			// 去掉 .exe 后缀方便匹配
			target = strings.TrimSuffix(target, ".exe")

			snapshot, _, _ := procCreateSnapshot.Call(TH32CS_SNAPPROCESS, 0)
			if snapshot == uintptr(syscall.InvalidHandle) || snapshot == 0 {
				return newError("process.find: 无法创建进程快照")
			}
			defer procCloseHandle.Call(snapshot)

			var entry processEntry32W
			entry.Size = uint32(unsafe.Sizeof(entry))

			ret, _, _ := procProcess32First.Call(snapshot, uintptr(unsafe.Pointer(&entry)))
			for ret != 0 {
				name := syscall.UTF16ToString(entry.ExeFile[:])
				lower := strings.ToLower(name)
				lowerNoExt := strings.TrimSuffix(lower, ".exe")
				if lower == target || lowerNoExt == target {
					return &object.Integer{Value: int64(entry.ProcessID)}
				}
				entry.Size = uint32(unsafe.Sizeof(entry))
				ret, _, _ = procProcess32Next.Call(snapshot, uintptr(unsafe.Pointer(&entry)))
			}
			return &object.Integer{Value: -1}
		},
	})

	// process.list() → [{pid, name}, ...]
	env.Define("list", &object.Builtin{
		Fn: func(args ...object.Object) object.Object {
			snapshot, _, _ := procCreateSnapshot.Call(TH32CS_SNAPPROCESS, 0)
			if snapshot == uintptr(syscall.InvalidHandle) || snapshot == 0 {
				return newError("process.list: 无法创建进程快照")
			}
			defer procCloseHandle.Call(snapshot)

			out := []object.Object{}
			var entry processEntry32W
			entry.Size = uint32(unsafe.Sizeof(entry))

			ret, _, _ := procProcess32First.Call(snapshot, uintptr(unsafe.Pointer(&entry)))
			for ret != 0 {
				pairs := map[object.HashKey]object.HashPair{}
				k1 := &object.String{Value: "pid"}
				k2 := &object.String{Value: "name"}
				pairs[k1.HashKey()] = object.HashPair{
					Key: k1, Value: &object.Integer{Value: int64(entry.ProcessID)}}
				pairs[k2.HashKey()] = object.HashPair{
					Key: k2, Value: &object.String{Value: syscall.UTF16ToString(entry.ExeFile[:])}}
				out = append(out, &object.Hash{Pairs: pairs})

				entry.Size = uint32(unsafe.Sizeof(entry))
				ret, _, _ = procProcess32Next.Call(snapshot, uintptr(unsafe.Pointer(&entry)))
			}
			return &object.Array{Elements: out}
		},
	})

	// process.maps(pid) → 内存区域列表
	env.Define("maps", &object.Builtin{
		Fn: func(args ...object.Object) object.Object {
			if len(args) < 1 {
				return newError("process.maps 需要 PID")
			}
			pid := uint32(toInt(args[0]))
			h, err := openProcess(pid)
			if err != nil {
				return newError("process.maps: %v", err)
			}
			defer procCloseHandle.Call(h)

			out := []object.Object{}
			var addr uintptr = 0
			for {
				var mbi memoryBasicInformation
				ret, _, _ := procVirtualQueryEx.Call(h, addr,
					uintptr(unsafe.Pointer(&mbi)), unsafe.Sizeof(mbi))
				if ret == 0 {
					break
				}
				if mbi.State == MEM_COMMIT {
					pairs := map[object.HashKey]object.HashPair{}
					put := func(k string, v object.Object) {
						key := &object.String{Value: k}
						pairs[key.HashKey()] = object.HashPair{Key: key, Value: v}
					}
					put("start", &object.Integer{Value: int64(mbi.BaseAddress)})
					put("end", &object.Integer{Value: int64(mbi.BaseAddress + mbi.RegionSize)})
					put("size", &object.Integer{Value: int64(mbi.RegionSize)})
					put("protect", &object.Integer{Value: int64(mbi.Protect)})
					put("writable", &object.Boolean{
						Value: mbi.Protect&PAGE_READWRITE != 0 || mbi.Protect&PAGE_EXECUTE_READWRITE != 0})
					out = append(out, &object.Hash{Pairs: pairs})
				}
				next := mbi.BaseAddress + mbi.RegionSize
				if next <= addr {
					break
				}
				addr = next
			}
			return &object.Array{Elements: out}
		},
	})

	// process.readBytes(pid, addr, size)
	env.Define("readBytes", &object.Builtin{
		Fn: func(args ...object.Object) object.Object {
			if len(args) < 3 {
				return newError("process.readBytes 需要 PID、地址、长度")
			}
			pid := uint32(toInt(args[0]))
			addr := uintptr(toInt(args[1]))
			size := int(toInt(args[2]))
			if size <= 0 || size > 65536 {
				return newError("process.readBytes 长度无效")
			}
			data, err := readMem(pid, addr, size)
			if err != nil {
				return newError("process.readBytes: %v", err)
			}
			elems := make([]object.Object, len(data))
			for i, b := range data {
				elems[i] = &object.Integer{Value: int64(b)}
			}
			return &object.Array{Elements: elems}
		},
	})

	// process.writeBytes(pid, addr, bytes)
	env.Define("writeBytes", &object.Builtin{
		Fn: func(args ...object.Object) object.Object {
			if len(args) < 3 {
				return newError("process.writeBytes 需要 PID、地址、字节数组")
			}
			pid := uint32(toInt(args[0]))
			addr := uintptr(toInt(args[1]))
			arr, ok := args[2].(*object.Array)
			if !ok {
				return newError("process.writeBytes 第三个参数要是字节数组")
			}
			data := make([]byte, len(arr.Elements))
			for i, e := range arr.Elements {
				data[i] = byte(toInt(e))
			}
			if err := writeMem(pid, addr, data); err != nil {
				return newError("process.writeBytes: %v", err)
			}
			return &object.Integer{Value: int64(len(data))}
		},
	})

	// process.readInt(pid, addr)
	env.Define("readInt", &object.Builtin{
		Fn: func(args ...object.Object) object.Object {
			if len(args) < 2 {
				return newError("process.readInt 需要 PID 和地址")
			}
			pid := uint32(toInt(args[0]))
			addr := uintptr(toInt(args[1]))
			data, err := readMem(pid, addr, 4)
			if err != nil {
				return newError("process.readInt: %v", err)
			}
			val := int32(uint32(data[0]) | uint32(data[1])<<8 |
				uint32(data[2])<<16 | uint32(data[3])<<24)
			return &object.Integer{Value: int64(val)}
		},
	})

	// process.writeInt(pid, addr, value)
	env.Define("writeInt", &object.Builtin{
		Fn: func(args ...object.Object) object.Object {
			if len(args) < 3 {
				return newError("process.writeInt 需要 PID、地址、值")
			}
			pid := uint32(toInt(args[0]))
			addr := uintptr(toInt(args[1]))
			val := int32(toInt(args[2]))
			data := []byte{byte(val), byte(val >> 8), byte(val >> 16), byte(val >> 24)}
			if err := writeMem(pid, addr, data); err != nil {
				return newError("process.writeInt: %v", err)
			}
			return NIL
		},
	})

	// process.readFloat(pid, addr)
	env.Define("readFloat", &object.Builtin{
		Fn: func(args ...object.Object) object.Object {
			if len(args) < 2 {
				return newError("process.readFloat 需要 PID 和地址")
			}
			pid := uint32(toInt(args[0]))
			addr := uintptr(toInt(args[1]))
			data, err := readMem(pid, addr, 4)
			if err != nil {
				return newError("process.readFloat: %v", err)
			}
			bits := uint32(data[0]) | uint32(data[1])<<8 |
				uint32(data[2])<<16 | uint32(data[3])<<24
			return &object.Float{Value: float64(float32frombits(bits))}
		},
	})

	// process.writeFloat(pid, addr, value)
	env.Define("writeFloat", &object.Builtin{
		Fn: func(args ...object.Object) object.Object {
			if len(args) < 3 {
				return newError("process.writeFloat 需要 PID、地址、值")
			}
			pid := uint32(toInt(args[0]))
			addr := uintptr(toInt(args[1]))
			var f float64
			if fl, ok := args[2].(*object.Float); ok {
				f = fl.Value
			} else {
				f = float64(toInt(args[2]))
			}
			bits := float32bits(float32(f))
			data := []byte{byte(bits), byte(bits >> 8), byte(bits >> 16), byte(bits >> 24)}
			if err := writeMem(pid, addr, data); err != nil {
				return newError("process.writeFloat: %v", err)
			}
			return NIL
		},
	})

	// process.scanInt(pid, value)
	env.Define("scanInt", &object.Builtin{
		Fn: func(args ...object.Object) object.Object {
			if len(args) < 2 {
				return newError("process.scanInt 需要 PID 和要查找的值")
			}
			pid := uint32(toInt(args[0]))
			target := int32(toInt(args[1]))

			h, err := openProcess(pid)
			if err != nil {
				return newError("process.scanInt: %v", err)
			}
			defer procCloseHandle.Call(h)

			results := []object.Object{}
			var addr uintptr = 0
			for {
				var mbi memoryBasicInformation
				ret, _, _ := procVirtualQueryEx.Call(h, addr,
					uintptr(unsafe.Pointer(&mbi)), unsafe.Sizeof(mbi))
				if ret == 0 {
					break
				}
				// 只扫可读可写、已提交、非保护页
				readable := mbi.Protect&PAGE_READWRITE != 0 ||
					mbi.Protect&PAGE_EXECUTE_READWRITE != 0 ||
					mbi.Protect&PAGE_WRITECOPY != 0
				if mbi.State == MEM_COMMIT && readable && mbi.Protect&PAGE_GUARD == 0 {
					size := int(mbi.RegionSize)
					if size > 0 && size <= 64*1024*1024 {
						buf, err := readMem(pid, mbi.BaseAddress, size)
						if err == nil {
							for i := 0; i+4 <= len(buf); i++ {
								val := int32(uint32(buf[i]) | uint32(buf[i+1])<<8 |
									uint32(buf[i+2])<<16 | uint32(buf[i+3])<<24)
								if val == target {
									pairs := map[object.HashKey]object.HashPair{}
									k1 := &object.String{Value: "addr"}
									k2 := &object.String{Value: "value"}
									pairs[k1.HashKey()] = object.HashPair{
										Key: k1, Value: &object.Integer{
											Value: int64(mbi.BaseAddress) + int64(i)}}
									pairs[k2.HashKey()] = object.HashPair{
										Key: k2, Value: &object.Integer{Value: int64(val)}}
									results = append(results, &object.Hash{Pairs: pairs})
									if len(results) > 10000 {
										return &object.Array{Elements: results}
									}
								}
							}
						}
					}
				}
				next := mbi.BaseAddress + mbi.RegionSize
				if next <= addr {
					break
				}
				addr = next
			}
			return &object.Array{Elements: results}
		},
	})

	return &object.Module{
		Name:    "process",
		Env:     env,
		Exports: env.Snapshot(),
	}
}

// openProcess 打开进程句柄
func openProcess(pid uint32) (uintptr, error) {
	h, _, err := procOpenProcess.Call(
		PROCESS_VM_READ|PROCESS_VM_WRITE|PROCESS_VM_OPERATION|PROCESS_QUERY_INFO,
		0, uintptr(pid))
	if h == 0 {
		return 0, fmt.Errorf("无法打开进程 %d（需要管理员权限）: %v", pid, err)
	}
	return h, nil
}

// readMem 读目标进程内存
func readMem(pid uint32, addr uintptr, size int) ([]byte, error) {
	h, err := openProcess(pid)
	if err != nil {
		return nil, err
	}
	defer procCloseHandle.Call(h)

	buf := make([]byte, size)
	var read uintptr
	ret, _, _ := procReadProcessMem.Call(h, addr,
		uintptr(unsafe.Pointer(&buf[0])), uintptr(size), uintptr(unsafe.Pointer(&read)))
	if ret == 0 {
		return nil, fmt.Errorf("读取失败（地址 0x%x 可能无效）", addr)
	}
	return buf[:read], nil
}

// writeMem 写目标进程内存
func writeMem(pid uint32, addr uintptr, data []byte) error {
	h, err := openProcess(pid)
	if err != nil {
		return err
	}
	defer procCloseHandle.Call(h)

	var written uintptr
	ret, _, _ := procWriteProcessMem.Call(h, addr,
		uintptr(unsafe.Pointer(&data[0])), uintptr(len(data)),
		uintptr(unsafe.Pointer(&written)))
	if ret == 0 {
		return fmt.Errorf("写入失败（地址 0x%x 可能无效）", addr)
	}
	return nil
}


