//go:build linux && !js

package interpreter

import (
	"fmt"
	"os"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"unsafe"

	"chronospp/object"
)

// newProcessModule 构造进程内存读写模块（Linux 版）
//   import "process"
//   let pid = process.find("gtavc")
//   let maps = process.maps(pid)
//   let val = process.readInt(pid, 0x7f0000)
//   process.writeInt(pid, 0x7f0000, 999999)
func newProcessModule() *object.Module {
	env := object.NewEnvironment()

	// process.find(name) → PID，找不到返回 -1
	env.Define("find", &object.Builtin{
		Fn: func(args ...object.Object) object.Object {
			if len(args) < 1 {
				return newError("process.find 需要进程名")
			}
			name := strings.ToLower(args[0].Inspect())
			entries, err := os.ReadDir("/proc")
			if err != nil {
				return newError("process.find: 无法读取 /proc: %v", err)
			}
			for _, e := range entries {
				pid, err := strconv.Atoi(e.Name())
				if err != nil {
					continue
				}
				comm, err := os.ReadFile(fmt.Sprintf("/proc/%d/comm", pid))
				if err != nil {
					continue
				}
				if strings.Contains(strings.ToLower(strings.TrimSpace(string(comm))), name) {
					return &object.Integer{Value: int64(pid)}
				}
			}
			return &object.Integer{Value: -1}
		},
	})

	// process.list() → 数组 [{pid, name}, ...]
	env.Define("list", &object.Builtin{
		Fn: func(args ...object.Object) object.Object {
			entries, err := os.ReadDir("/proc")
			if err != nil {
				return newError("process.list: %v", err)
			}
			out := []object.Object{}
			for _, e := range entries {
				pid, err := strconv.Atoi(e.Name())
				if err != nil {
					continue
				}
				comm, err := os.ReadFile(fmt.Sprintf("/proc/%d/comm", pid))
				if err != nil {
					continue
				}
				pairs := map[object.HashKey]object.HashPair{}
				k1 := &object.String{Value: "pid"}
				k2 := &object.String{Value: "name"}
				pairs[k1.HashKey()] = object.HashPair{Key: k1, Value: &object.Integer{Value: int64(pid)}}
				pairs[k2.HashKey()] = object.HashPair{Key: k2, Value: &object.String{Value: strings.TrimSpace(string(comm))}}
				out = append(out, &object.Hash{Pairs: pairs})
			}
			return &object.Array{Elements: out}
		},
	})

	// process.maps(pid) → 数组 [{start, end, perms, path}, ...]
	env.Define("maps", &object.Builtin{
		Fn: func(args ...object.Object) object.Object {
			if len(args) < 1 {
				return newError("process.maps 需要 PID")
			}
			pid, ok := args[0].(*object.Integer)
			if !ok {
				return newError("process.maps 的 PID 要是整数")
			}
			data, err := os.ReadFile(fmt.Sprintf("/proc/%d/maps", pid.Value))
			if err != nil {
				return newError("process.maps: %v", err)
			}
			out := []object.Object{}
			for _, line := range strings.Split(string(data), "\n") {
				if line == "" {
					continue
				}
				// 格式：7f0000000000-7f0000001000 r-xp 00000000 08:01 12345 /path
				fields := strings.Fields(line)
				if len(fields) < 2 {
					continue
				}
				addrs := strings.Split(fields[0], "-")
				if len(addrs) != 2 {
					continue
				}
				start, err1 := strconv.ParseInt(addrs[0], 16, 64)
				end, err2 := strconv.ParseInt(addrs[1], 16, 64)
				if err1 != nil || err2 != nil {
					continue
				}
				path := ""
				if len(fields) >= 6 {
					path = fields[5]
				}
				pairs := map[object.HashKey]object.HashPair{}
				put := func(k string, v object.Object) {
					key := &object.String{Value: k}
					pairs[key.HashKey()] = object.HashPair{Key: key, Value: v}
				}
				put("start", &object.Integer{Value: start})
				put("end", &object.Integer{Value: end})
				put("size", &object.Integer{Value: end - start})
				put("perms", &object.String{Value: fields[1]})
				put("path", &object.String{Value: path})
				out = append(out, &object.Hash{Pairs: pairs})
			}
			return &object.Array{Elements: out}
		},
	})

	// process.readBytes(pid, addr, size) → 字节数组
	env.Define("readBytes", &object.Builtin{
		Fn: func(args ...object.Object) object.Object {
			if len(args) < 3 {
				return newError("process.readBytes 需要 PID、地址、长度")
			}
			pid := toInt(args[0])
			addr := toInt(args[1])
			size := toInt(args[2])
			if pid < 0 || addr < 0 || size <= 0 || size > 65536 {
				return newError("process.readBytes 参数无效")
			}
			data, err := readProcessMemory(int(pid), uintptr(addr), int(size))
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

	// process.writeBytes(pid, addr, bytes) → 写入字节数
	env.Define("writeBytes", &object.Builtin{
		Fn: func(args ...object.Object) object.Object {
			if len(args) < 3 {
				return newError("process.writeBytes 需要 PID、地址、字节数组")
			}
			pid := toInt(args[0])
			addr := toInt(args[1])
			arr, ok := args[2].(*object.Array)
			if !ok {
				return newError("process.writeBytes 第三个参数要是字节数组")
			}
			data := make([]byte, len(arr.Elements))
			for i, e := range arr.Elements {
				data[i] = byte(toInt(e))
			}
			if err := writeProcessMemory(int(pid), uintptr(addr), data); err != nil {
				return newError("process.writeBytes: %v", err)
			}
			return &object.Integer{Value: int64(len(data))}
		},
	})

	// process.readInt(pid, addr) → int32（小端）
	env.Define("readInt", &object.Builtin{
		Fn: func(args ...object.Object) object.Object {
			if len(args) < 2 {
				return newError("process.readInt 需要 PID 和地址")
			}
			pid := toInt(args[0])
			addr := toInt(args[1])
			data, err := readProcessMemory(int(pid), uintptr(addr), 4)
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
			pid := toInt(args[0])
			addr := toInt(args[1])
			val := int32(toInt(args[2]))
			data := []byte{
				byte(val), byte(val >> 8), byte(val >> 16), byte(val >> 24),
			}
			if err := writeProcessMemory(int(pid), uintptr(addr), data); err != nil {
				return newError("process.writeInt: %v", err)
			}
			return NIL
		},
	})

	// process.readFloat(pid, addr) → float32
	env.Define("readFloat", &object.Builtin{
		Fn: func(args ...object.Object) object.Object {
			if len(args) < 2 {
				return newError("process.readFloat 需要 PID 和地址")
			}
			pid := toInt(args[0])
			addr := toInt(args[1])
			data, err := readProcessMemory(int(pid), uintptr(addr), 4)
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
			pid := toInt(args[0])
			addr := toInt(args[1])
			var f float64
			if fl, ok := args[2].(*object.Float); ok {
				f = fl.Value
			} else {
				f = float64(toInt(args[2]))
			}
			bits := float32bits(float32(f))
			data := []byte{
				byte(bits), byte(bits >> 8), byte(bits >> 16), byte(bits >> 24),
			}
			if err := writeProcessMemory(int(pid), uintptr(addr), data); err != nil {
				return newError("process.writeFloat: %v", err)
			}
			return NIL
		},
	})

	// process.scanInt(pid, value) → 数组 [{addr, current}, ...]
	// 在可写内存区扫描指定整数值
	env.Define("scanInt", &object.Builtin{
		Fn: func(args ...object.Object) object.Object {
			if len(args) < 2 {
				return newError("process.scanInt 需要 PID 和要查找的值")
			}
			pid := toInt(args[0])
			target := int32(toInt(args[1]))
			return scanMemory(int(pid), func(data []byte, base uintptr) []object.Object {
				var found []object.Object
				for i := 0; i+4 <= len(data); i++ {
					val := int32(uint32(data[i]) | uint32(data[i+1])<<8 |
						uint32(data[i+2])<<16 | uint32(data[i+3])<<24)
					if val == target {
						pairs := map[object.HashKey]object.HashPair{}
						k1 := &object.String{Value: "addr"}
						k2 := &object.String{Value: "value"}
						pairs[k1.HashKey()] = object.HashPair{Key: k1, Value: &object.Integer{Value: int64(base) + int64(i)}}
						pairs[k2.HashKey()] = object.HashPair{Key: k2, Value: &object.Integer{Value: int64(val)}}
						found = append(found, &object.Hash{Pairs: pairs})
					}
				}
				return found
			})
		},
	})

	return &object.Module{
		Name:    "process",
		Env:     env,
		Exports: env.Snapshot(),
	}
}

// ── 底层实现 ──────────────────────────────

// readProcessMemory 用 process_vm_readv 读目标进程内存
func readProcessMemory(pid int, addr uintptr, size int) ([]byte, error) {
	// 先用 /proc/pid/mem 尝试（需要权限）
	f, err := os.OpenFile(fmt.Sprintf("/proc/%d/mem", pid), os.O_RDONLY, 0)
	if err == nil {
		defer f.Close()
		buf := make([]byte, size)
		if _, err := f.ReadAt(buf, int64(addr)); err == nil {
			return buf, nil
		}
	}

	// 回退：process_vm_readv（系统调用号按架构区分）
	var sysReadv uintptr
	switch runtime.GOARCH {
	case "amd64":
		sysReadv = 310
	case "arm64":
		sysReadv = 270
	case "386":
		sysReadv = 347
	case "arm":
		sysReadv = 376
	default:
		return nil, fmt.Errorf("读取失败（不支持的架构 %s）", runtime.GOARCH)
	}

	local := make([]byte, size)
	localIov := syscall.Iovec{Base: &local[0], Len: uint64(size)}
	remoteIov := syscall.Iovec{Base: (*byte)(unsafe.Pointer(addr)), Len: uint64(size)}

	_, _, errno := syscall.Syscall6(
		sysReadv,
		uintptr(pid),
		uintptr(unsafe.Pointer(&localIov)), 1,
		uintptr(unsafe.Pointer(&remoteIov)), 1,
		0,
	)
	if errno != 0 {
		return nil, fmt.Errorf("读取失败（需要 root 或 ptrace 权限）: %v", errno)
	}
	return local, nil
}

// writeProcessMemory 用 process_vm_writev 写目标进程内存
func writeProcessMemory(pid int, addr uintptr, data []byte) error {
	f, err := os.OpenFile(fmt.Sprintf("/proc/%d/mem", pid), os.O_WRONLY, 0)
	if err == nil {
		defer f.Close()
		if _, err := f.WriteAt(data, int64(addr)); err == nil {
			return nil
		}
	}

	var sysWritev uintptr
	switch runtime.GOARCH {
	case "amd64":
		sysWritev = 311
	case "arm64":
		sysWritev = 271
	case "386":
		sysWritev = 348
	case "arm":
		sysWritev = 377
	default:
		return fmt.Errorf("写入失败（不支持的架构 %s）", runtime.GOARCH)
	}

	localIov := syscall.Iovec{Base: &data[0], Len: uint64(len(data))}
	remoteIov := syscall.Iovec{Base: (*byte)(unsafe.Pointer(addr)), Len: uint64(len(data))}

	_, _, errno := syscall.Syscall6(
		sysWritev,
		uintptr(pid),
		uintptr(unsafe.Pointer(&localIov)), 1,
		uintptr(unsafe.Pointer(&remoteIov)), 1,
		0,
	)
	if errno != 0 {
		return fmt.Errorf("写入失败（需要 root 或 ptrace 权限）: %v", errno)
	}
	return nil
}

// scanMemory 遍历可写内存区执行扫描
func scanMemory(pid int, fn func(data []byte, base uintptr) []object.Object) object.Object {
	mapsPath := fmt.Sprintf("/proc/%d/maps", pid)
	data, err := os.ReadFile(mapsPath)
	if err != nil {
		return newError("scanInt: 无法读取内存映射: %v", err)
	}

	results := []object.Object{}
	scanned := 0

	for _, line := range strings.Split(string(data), "\n") {
		if line == "" {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		// 只扫可读可写的匿名区（堆、栈、数据段）
		perms := fields[1]
		if !strings.HasPrefix(perms, "rw") {
			continue
		}
		addrs := strings.Split(fields[0], "-")
		if len(addrs) != 2 {
			continue
		}
		start, err1 := strconv.ParseInt(addrs[0], 16, 64)
		end, err2 := strconv.ParseInt(addrs[1], 16, 64)
		if err1 != nil || err2 != nil {
			continue
		}
		size := end - start
		// 跳过超大区域（>64MB）
		if size <= 0 || size > 64*1024*1024 {
			continue
		}
		// 每次读 1MB 分块
		const chunk = 1024 * 1024
		for off := int64(0); off < size; off += chunk {
			readSize := int64(chunk)
			if off+readSize > size {
				readSize = size - off
			}
			buf, err := readProcessMemory(pid, uintptr(start+off), int(readSize))
			if err != nil {
				continue
			}
			scanned += len(buf)
			results = append(results, fn(buf, uintptr(start+off))...)
			if len(results) > 10000 {
				return &object.Array{Elements: results}
			}
		}
	}

	_ = scanned
	return &object.Array{Elements: results}
}

// toInt 把对象转成 int64
func toInt(o object.Object) int64 {
	switch v := o.(type) {
	case *object.Integer:
		return v.Value
	case *object.Float:
		return int64(v.Value)
	case *object.String:
		n, _ := strconv.ParseInt(strings.TrimSpace(v.Value), 0, 64)
		return n
	}
	return 0
}

// float32 位操作
func float32frombits(b uint32) float32 {
	return *(*float32)(unsafe.Pointer(&b))
}

func float32bits(f float32) uint32 {
	return *(*uint32)(unsafe.Pointer(&f))
}
