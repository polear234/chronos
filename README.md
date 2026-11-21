<div align="center">

# 🚀 PicoX

**一个用 Go 手写的轻量级动态类型脚本语言**

*变量时间旅行 · 事务性快照 · 管道操作符 · 类继承*

<br/>

[![License: MIT](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)
[![Build Status](https://img.shields.io/github/actions/workflow/status/polear234/picox/release.yml?branch=main)](https://github.com/polear234/picox/actions)
[![Go Report Card](https://goreportcard.com/badge/github.com/polear234/picox)](https://goreportcard.com/report/github.com/polear234/picox)
[![Go Version](https://img.shields.io/badge/go-1.21%2B-00ADD8.svg)](https://go.dev/)
[![Platform](https://img.shields.io/badge/platform-linux%20%7C%20macos%20%7C%20windows-lightgrey.svg)]()

<br/>

</div>

---

## ✨ 为什么是 PicoX？

PicoX 的独特之处在于它把 **"时间"** 变成了语言的一等概念 —— 变量能回滚、数据能快照、操作能撤销。

```picox
// 🕐 变量时间旅行：每次赋值自动压栈，可逐步回滚
let traveller int b = 1
b = 2
b = 3
b.rollback()    // → 2
b.rollback()    // → 1

// 🔄 事务性快照：callback 内随便改，执行完自动恢复
let arr = [1, 2, 3]
withSnapshot([arr], fn() {
    arr.push(99)
})
// arr 仍是 [1, 2, 3]

// ⛓️ 管道操作符：消除嵌套函数调用
let n = [1, 2, 3] |> len()    // → 3

// 🧬 类继承 + 多态
class Dog extends Animal {
    speak() { return this.name + " barks!" }
}
let dog = new Dog("Buddy")
print(dog.speak())            // "Buddy barks!"
```

---

## 📑 目录

- [✨ 为什么是 PicoX？](#-为什么是-picox)
- [🌟 特性](#-特性)
- [📦 安装](#-安装)
- [🚀 快速开始](#-快速开始)
- [📖 语法速览](#-语法速览)
- [🔧 内置函数](#-内置函数)
- [📁 示例程序](#-示例程序)
- [🏗️ 架构](#️-架构)
- [🛣️ 路线图](#️-路线图)
- [📄 许可证](#-许可证)

---

## 🌟 特性

| 类别 | 特性 |
|------|------|
| 🧠 **语言核心** | 动态类型 · 一等函数 · 闭包 · 递归 · 高阶函数 |
| 🏛️ **面向对象** | 类 · 单继承 · 多态 · `this`/`super` · 方法重写 · 静态方法 |
| 🕐 **时间特性** | traveller 回滚变量 · `withSnapshot` 事务快照 · `Timeline` 版本历史 |
| 🎯 **控制流** | `if`/`elif`/`else` · `while` · `for-in` · `break`/`continue` |
| 🧩 **模式匹配** | `match`/`case`/`default` |
| 🛡️ **错误处理** | `try`/`catch`/`finally`/`throw` |
| ⛓️ **语法糖** | 管道操作符 `\|>` · 数组方法链 · 类型标注 |
| 📦 **模块系统** | `import`/`as` · 独立环境 · 循环引用检测 |
| 🧰 **标准库** | 字符串 · 数组 · 字典 · 数学 · 类型转换 |

---

## 📦 安装

### 🐧 方式一：deb 安装包（Ubuntu / Debian，推荐）

```bash
# amd64 机器
sudo dpkg -i picox_0.1.0_amd64.deb

# arm64 机器（手机 / 树莓派 / 苹果 M 系列）
sudo dpkg -i picox_0.1.0_arm64.deb
```

### 📜 方式二：安装脚本

```bash
sudo ./install.sh          # 系统级安装到 /usr/local/bin
./install.sh --user        # 用户级安装到 ~/.local/bin
```

### 🔨 方式三：源码编译

```bash
go build -o picox .        # 编译单文件二进制
make dist                  # 交叉编译多平台
```

> 💡 编译产物是**单文件静态二进制**（`CGO_ENABLED=0`），拷贝到任何同架构机器即可直接运行，无需 Go 或任何依赖。

---

## 🚀 快速开始

```bash
picox                          # 启动交互式 REPL
picox -run hello.px            # 运行脚本
picox run -                    # 从 stdin 读取执行
```

```bash
$ echo 'print("Hello World!")' | picox run -
Hello World!

$ picox
picox> let x = 40 + 2
picox> x
42
```

---

## 📖 语法速览

### 📌 变量与类型

```picox
let x = 10            // 可变变量
const PI = 3.14159    // 常量（不可重新赋值）

let num = 42          // 整数
let pi = 3.14         // 浮点数
let str = "Hello"     // 字符串
let flag = true       // 布尔值
let nil = nil         // 空值
```

### 🕐 traveller 变量（可回滚）

```picox
let traveller int b = 1
b = 2
b = 3
b.rollback()    // → 2
b.rollback()    // → 1
b.rollback()    // 💥 ERROR: Traveller history is empty
```

### 📚 数组与字典

```picox
let lst = [1, 2, 3]

lst.push(4)                          // 追加
lst.pop()                            // 移除末尾
lst.length()                         // 长度
lst.map(fn(x) { x * 2 })             // 映射
lst.filter(fn(x) { x > 1 })          // 过滤
lst.reduce(fn(acc, x) { acc + x }, 0) // 归约

let dict = {"name": "Alice", "age": 25}
dict["name"]       // 访问
dict["age"] = 26   // 修改
```

### 🎯 函数与控制流

```picox
fn add(a, b) {
    return a + b
}

if x > 0 {
    print("positive")
} elif x < 0 {
    print("negative")
} else {
    print("zero")
}

for item in array {
    print(item)
}

while condition {
    // ...
}
```

### 🏛️ 类与继承

```picox
class Animal {
    name = ""
    init(name) { this.name = name }
    speak() { return this.name + " makes a sound" }
}

class Dog extends Animal {
    speak() { return this.name + " barks!" }
}

let dog = new Dog("Buddy")
print(dog.speak())   // "Buddy barks!"
```

### ⛓️ 管道操作符

```picox
// a |> f(b) 等价于 f(a, b)
let arr = [1, 2, 3]
let n = arr |> len()    // → 3
```

### 🔄 快照与时间线

```picox
// withSnapshot：事务性回滚
let arr = [1, 2, 3]
withSnapshot([arr], fn() { arr.push(99) })
// arr 恢复为 [1, 2, 3]

// Timeline：版本历史
let data = [1]
let tl = Timeline(data)
tl.commit()      // 记录当前状态
data.push(2)
tl.undo()        // 回到 [1]
```

### 📦 模块系统

```picox
// math.px
let PI = 3.14159
fn square(x) { return x * x }

// main.px
import "math" as m
print(m.square(5))    // 25
```

### 🛡️ 错误处理

```picox
try {
    riskyOperation()
} catch (e) {
    print("Error: " + e)
} finally {
    cleanup()
}
```

---

## 🔧 内置函数

| 函数 | 说明 | 函数 | 说明 |
|------|------|------|------|
| `print(...)` | 打印到控制台 | `len(x)` | 长度（字符串/数组/字典） |
| `push(arr, x)` | 数组末尾添加 | `pop(arr)` | 移除并返回末尾元素 |
| `shift(arr)` | 移除并返回首元素 | `unshift(arr, x)` | 数组开头添加 |
| `keys(dict)` | 字典所有键 | `values(dict)` | 字典所有值 |
| `type(x)` | 类型字符串 | `str(x)` | 转字符串 |
| `int(x)` | 转整数 | `float(x)` | 转浮点数 |
| `range(a, b, s?)` | 生成数字数组 | | |

---

## 📁 示例程序

| 示例 | 说明 |
|------|------|
| [`hello.px`](examples/hello.px) | 基础语法完整演示 |
| [`fibonacci.px`](examples/fibonacci.px) | 斐波那契三种实现 |
| [`oop.px`](examples/oop.px) | 面向对象编程 |
| [`math.px`](examples/math.px) | 可导入的数学模块 |
| [`import_demo.px`](examples/import_demo.px) | import 模块演示 |

---

## 🏗️ 架构

```
picox/
├── main.go           # 入口（CLI：run / repl / version / help）
├── token/            # 词法标记定义
├── lexer/            # 词法分析器
├── parser/           # 语法分析器（Pratt 解析）
├── ast/              # 抽象语法树节点
├── object/           # 运行时对象（含 Traveller / Timeline）
├── interpreter/      # 解释器 + 内置函数
├── repl/             # REPL 交互环境
├── scripts/          # 构建与安装脚本
├── bin/              # 预编译多平台二进制
└── examples/         # 示例程序
```

---

## 🛣️ 路线图

- [x] 模块系统（import）
- [x] traveller 变量回滚
- [x] 管道操作符 `|>`
- [x] withSnapshot / Timeline
- [x] 类继承与多态
- [ ] 标准库扩展（文件 IO、网络、正则）
- [ ] 编译器优化（字节码 VM）
- [ ] 调试器支持
- [ ] 包管理器
- [ ] LSP 语言服务器

---

## 📄 许可证

[MIT License](LICENSE) © 2026 PicoX Contributors

---

<div align="center">

**Made with ❤️ and Go**

</div>
