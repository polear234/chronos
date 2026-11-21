<div align="center">

<img src="docs/logo.png" width="180" alt="Chronos logo">

# 🚀 Chronos

**一个用 Go 手写的轻量级动态类型脚本语言**

*变量时间旅行 · 事务性快照 · 管道操作符 · 类继承*

<br/>

[![License: MIT](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)
[![Build Status](https://img.shields.io/github/actions/workflow/status/polear234/chronos/release.yml?branch=main)](https://github.com/polear234/chronos/actions)
[![Go Report Card](https://goreportcard.com/badge/github.com/polear234/chronos)](https://goreportcard.com/report/github.com/polear234/chronos)
[![Go Version](https://img.shields.io/badge/go-1.21%2B-00ADD8.svg)](https://go.dev/)
[![Platform](https://img.shields.io/badge/platform-linux%20%7C%20macos%20%7C%20windows-lightgrey.svg)]()
[![Lines](https://img.shields.io/badge/code-6.5k%20lines-informational.svg)]()

<br/>

</div>

---

## ✨ 为什么是 Chronos？

大多数脚本语言把「时间」留给了程序员自己管理。Chronos 把它做成了语言的一等概念 —— **变量能回滚、数据能快照、操作能撤销**。

```chronos
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

- [✨ 为什么是 Chronos？](#-为什么是-chronos)
- [🌟 特性](#-特性)
- [📦 安装](#-安装)
- [🚀 快速开始](#-快速开始)
- [🕐 时间特性](#-时间特性)
- [📖 语法速览](#-语法速览)
- [🧰 内置函数与方法](#-内置函数与方法)
- [💻 命令行](#-命令行)
- [📦 编译成独立可执行文件](#-编译成独立可执行文件)
- [📁 示例程序](#-示例程序)
- [🏗️ 架构](#️-架构)
- [⚠️ 已知限制](#️-已知限制)
- [🛣️ 路线图](#️-路线图)
- [🤝 贡献](#-贡献)
- [📄 许可证](#-许可证)

---

## 🌟 特性

| 类别 | 特性 |
|------|------|
| 🧠 **语言核心** | 动态类型 · 一等函数 · 闭包 · 递归 · 高阶函数 |
| 🏛️ **面向对象** | 类 · 单继承 · 多态 · `this`/`super` · 方法重写 |
| 🕐 **时间特性** | traveller 回滚变量 · `withSnapshot` 事务快照 · `Timeline` 版本历史 |
| 🎯 **控制流** | `if`/`elif`/`else` · `while` · `for-in` · `break`/`continue` |
| 🧩 **模式匹配** | `match`/`case`/`default`（值匹配） |
| 🛡️ **错误处理** | `try`/`catch`/`finally`/`throw` |
| ⛓️ **语法糖** | 管道操作符 `\|>` · 数组/字符串/字典方法链 · 类型标注 |
| 📦 **模块系统** | `import`/`as` · 独立环境 · 循环引用检测 |
| 🧰 **标准库** | 字符串 / 数组 / 字典方法 · 类型转换 · `range` |
| 🌏 **国际化** | Unicode 标识符：中文 / 日文 / 俄文均可作变量名、函数名、类名 |

---

## 📦 安装

### 🐧 方式一：deb 安装包（Ubuntu / Debian，推荐）

```bash
# amd64 机器
sudo dpkg -i chronos_0.1.0_amd64.deb

# arm64 机器（手机 / 树莓派 / 苹果 M 系列）
sudo dpkg -i chronos_0.1.0_arm64.deb

# 验证
chronos --version     # Chronos 0.1.0
```

### 📜 方式二：安装脚本

```bash
sudo ./install.sh          # 系统级安装到 /usr/local/bin
./install.sh --user        # 用户级安装到 ~/.local/bin
```

### 🔨 方式三：源码编译

```bash
go build -o chronos .        # 编译单文件二进制
make dist                  # 交叉编译六平台（linux/macos/windows × amd64/arm64）
```

> 💡 编译产物是**单文件静态二进制**（`CGO_ENABLED=0` + `-s -w`），无任何运行时依赖，直接拷贝到同架构机器即可运行。

---

## 🚀 快速开始

```bash
$ chronos
Chronos v0.1.0 — a lightweight scripting language (Go implementation)
Type 'exit' / 'quit' to leave, 'clear' to reset the environment.

chronos> let x = 40 + 2
chronos> x
42
chronos> fn fib(n) { if n <= 1 { return n } return fib(n-1) + fib(n-2) }
chronos> fib(10)
55
```

运行脚本：

```bash
chronos -run hello.px            # 运行文件
chronos run -                    # 从 stdin 读取
echo 'print("Hello World!")' | chronos run -
```

---

## 🕐 时间特性

这是 Chronos 与其他脚本语言最大的不同。

### traveller 变量（可回滚）

```chronos
let traveller int x = 1
x = 2
x = 3
x.rollback()   // → 2
x.rollback()   // → 1
x.rollback()   // 💥 ERROR: Traveller history is empty
```

- 每次赋值把旧值压入历史栈（默认上限 100 条，防止内存膨胀）
- 读取时透明解包，`x + 1` 直接当普通变量用
- 支持类型标注，赋值时校验类型

### withSnapshot（事务性回滚）

```chronos
let arr = [1, 2, 3]
withSnapshot([arr], fn() {
    arr.push(4)
    arr.push(5)
})
arr          // → [1, 2, 3]（自动恢复）
```

数组按长度快照、字典按浅拷贝快照，执行完**原地恢复**，引用不变。

### Timeline（版本历史）

```chronos
let data = [1]
let tl = Timeline(data)

tl.commit()      // 快照 [1]
data.push(2)
tl.commit()      // 快照 [1, 2]
data.push(3)

tl.undo()        // → [1, 2]
tl.undo()        // → [1]
```

---

## 📖 语法速览

### 📌 变量与类型

```chronos
let x = 10            // 可变变量
const PI = 3.14159    // 常量（重新赋值会报错）

let num = 42          // 整数
let pi = 3.14         // 浮点数
let str = "Hello"     // 字符串
let flag = true       // 布尔值
let nothing = nil     // 空值

// 类型标注
let int count = 0
let string name = "Chronos"
```

### 📚 数组与字典

```chronos
let lst = [1, 2, 3, "hello"]

lst.push(4)                            // 追加
lst.pop()                              // 移除末尾
lst.length()                           // 长度
lst.map(fn(x) { x * 2 })               // 映射
lst.filter(fn(x) { x > 1 })            // 过滤
lst.reduce(fn(acc, x) { acc + x }, 0)  // 归约
lst[0]                                 // 索引访问

let dict = {"name": "Alice", "age": 25}
dict["name"]                           // 访问
dict["age"] = 26                       // 修改
dict.has("name")                       // 是否存在
```

### 🎯 函数与闭包

```chronos
fn add(a, b) {
    return a + b
}

// 闭包捕获外层变量
fn makeCounter() {
    let count = 0
    return fn() {
        count = count + 1
        return count
    }
}

let counter = makeCounter()
counter()   // 1
counter()   // 2
```

### 🔀 控制流

```chronos
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

for i in range(10) {
    print(i)          // 0 到 9
}

while condition {
    // ...
}
```

### 🏛️ 类与继承

```chronos
class Animal {
    name = ""

    init(name) {
        this.name = name
    }

    speak() {
        return this.name + " makes a sound"
    }
}

class Dog extends Animal {
    speak() {
        return this.name + " barks!"
    }
}

let dog = new Dog("Buddy")
dog.speak()    // → "Buddy barks!"
```

支持多层继承、`super.init()` 调用父类构造、`super.method()` 调用父类方法、方法重写。

### 🧩 模式匹配

```chronos
match (value) {
    case 1:       { print("one") }
    case "hello": { print("greeting") }
    default:      { print("other: " + str(value)) }
}
```

### 🛡️ 错误处理

```chronos
try {
    let result = 10 / 0
} catch (e) {
    print("Error: " + e)     // e 是错误消息字符串
} finally {
    print("cleanup")
}

throw("Something went wrong")
```

### 📦 模块系统

```chronos
// math.px
let PI = 3.14159
fn square(x) { return x * x }

// main.px
import "math" as m
print(m.square(5))    // → 25

import "math"         // 默认用文件名做别名
print(math.PI)        // → 3.14159
```

- 模块在**独立环境**中执行，不污染调用方作用域
- 自动检测循环 import（A↔B 报错）
- 支持 `import "dir/xxx"` 子目录，自动补 `.px`

---

## 🧰 内置函数与方法

### 全局函数

| 函数 | 说明 |
|------|------|
| `print(...)` / `println(...)` | 打印到控制台 |
| `len(x)` | 长度（字符串 / 数组 / 字典） |
| `type(x)` | 返回类型名 |
| `int(x)` / `float(x)` / `str(x)` | 类型转换（失败会报错） |
| `range(a, b?, step?)` | 生成数字数组 |
| `push/pop/shift/unshift(arr, ...)` | 数组操作函数风格 |

### 字符串方法

| 方法 | 说明 |
|------|------|
| `s.length()` | 字符数（按 rune 计） |
| `s.upper()` / `s.lower()` | 大小写转换 |
| `s.trim()` | 去除首尾空白 |
| `s.contains(sub)` | 是否包含子串 |
| `s.split(sep)` | 切分为数组 |
| `s.replace(old, new)` | 替换全部 |

### 数组方法（14 个）

| 类别 | 方法 |
|------|------|
| 增删 | `push` `pop` `shift` `unshift` `insert` `removeAt` `clear` |
| 查询 | `length` `indexOf` `contains` |
| 变换 | `slice` `concat` `reverse` `join` |
| 高阶 | `map` `filter` `forEach` `reduce` `find` |

### 字典方法

| 方法 | 说明 |
|------|------|
| `h.length()` | 键值对数量 |
| `h.keys()` / `h.values()` | 键数组 / 值数组 |
| `h.has(key)` | 键是否存在 |

---

## 💻 命令行

| 命令 | 说明 |
|------|------|
| `chronos` | 启动交互式 REPL |
| `chronos run <file.px>` | 运行脚本 |
| `chronos -run <file.px>` | 运行脚本（简写） |
| `chronos build <file.px>` | **编译成独立可执行文件** |
| `chronos build <file.px> -o <name>` | 编译并指定输出名 |
| `chronos <file.px>` | 运行脚本（识别 `.px` 后缀） |
| `chronos run -` | 从 stdin 读取并执行 |
| `chronos version` | 查看版本 |
| `chronos help` | 查看帮助 |

退出码：`0` 成功，`1` 运行时/编译错误，`2` 用法错误。

REPL 内置命令：`exit` / `quit` 退出，`clear` 清空环境。

---

## 📦 编译成独立可执行文件

`chronos build` 把 `.px` 脚本**嵌入解释器副本**，产出一个单文件可执行程序 —— 双击就能跑，不需要源文件、不需要装 Chronos。

```bash
$ chronos build examples/hello.px
✓ 已编译: hello
  源文件: examples/hello.px (2518 字节)
  产物:   /home/me/chronos/hello (2737564 字节)
  运行:   ./hello

$ ./hello
Hello, Chronos v0.1
Is awesome: true
...
```

自定义输出名：

```bash
chronos build app.px -o /usr/local/bin/app
```

**实现原理**：在解释器二进制尾部追加 `[payload][8字节长度][16字节 magic]`，启动时检测到 magic 就直接执行内嵌脚本。

**默认开启混淆加密**（AES-256-CTR，每份产物随机盐）：

```bash
chronos build app.px              # 默认：加密混淆
chronos build app.px --plain      # 关闭加密，嵌入明文（便于调试）
```

- 加密后源码在二进制里是密文，`strings` / `hexdump` / `grep -a` 都读不出脚本内容
- 同一脚本两次编译产物**不同**（随机盐），无法用哈希比对或差分推断
- ⚠️ **这是混淆不是密码学安全**：解密密钥必须随二进制分发，有决心的人仍可逆向还原。它的目标是阻止"随手翻出源码"，不是防专业破解。真正的保密需要运行时输入密码解密，那是另一套设计。

其他特性：

- 产物与原解释器体积几乎相同（只多出脚本本身的大小）
- **import 的相对路径以可执行文件所在目录为基准**，可把程序和模块一起分发
- 编译前做语法检查，有错误直接中止（不会产出「一跑就炸」的二进制）
- 支持对已 build 的二进制再次 build（自动剥离旧 payload，不叠加）

---

## 📁 示例程序

| 示例 | 说明 |
|------|------|
| [`hello.px`](examples/hello.px) | 基础语法完整演示（约 140 行） |
| [`fibonacci.px`](examples/fibonacci.px) | 斐波那契三种实现：递归 / 迭代 / 备忘录 |
| [`oop.px`](examples/oop.px) | 面向对象：类、继承、多态、静态方法、动态方法 |
| [`math.px`](examples/math.px) | 可被 import 的数学模块 |
| [`import_demo.px`](examples/import_demo.px) | 模块系统演示 |
| [`i18n.px`](examples/i18n.px) | 中文变量/类/管道/回滚 全方位国际化测试 |

```bash
chronos -run examples/hello.px
chronos -run examples/oop.px
```

---

## 🏗️ 架构

经典的「词法 → 语法 → 树 → 求值」四段式，无第三方依赖，纯标准库实现。

```
chronos/                          ~6.5k 行 Go 代码
├── main.go            164 行    CLI 入口（run / repl / version / help）
├── token/             143 行    词法标记定义
├── lexer/             647 行    词法分析器（含转义、注释、多字符运算符）
├── parser/           1271 行    语法分析器（Pratt 解析，40+ 优先级规则）
├── ast/               697 行    抽象语法树节点
├── object/            508 行    运行时对象（含 Traveller / Timeline / Module）
├── interpreter/      3011 行    求值器 + 内置函数 + 模块加载
├── repl/              134 行    REPL 交互环境（多行输入检测）
├── scripts/                     构建与安装脚本
├── docs/                        Logo 等资源
└── examples/                    示例程序
```

---

## ⚠️ 已知限制

诚实说明当前版本还没做到的事：

- **标准库较薄**：没有文件 IO、网络、正则、数学函数（`sqrt`/`sin` 等）
- **`import` 不缓存**：同一个模块被多处导入会重复执行
- **`match` 只做值匹配**：不支持 `case [a, b]:` 这类结构解构
- **无 GC 调优**：解释器直接用 Go 的 GC，长循环会积累临时对象
- **无调试器**：只能靠 `print` 和 REPL 排查

---

## 🛣️ 路线图

- [x] 模块系统（import）
- [x] traveller 变量回滚
- [x] 管道操作符 `|>`
- [x] withSnapshot / Timeline
- [x] 类继承与多态
- [x] 字符串 / 数组 / 字典方法
- [ ] 标准库扩展（文件 IO、网络、正则、数学）
- [ ] 模块缓存与包管理器
- [ ] 编译器优化（字节码 VM）
- [ ] 调试器支持
- [ ] LSP 语言服务器

---

## 🤝 贡献

欢迎 Issue 和 PR。本地开发：

```bash
git clone git@github.com:polear234/chronos.git
cd chronos
go test ./...        # 跑全部测试（词法 / 语法 / 求值）
go build -o chronos .  # 编译
```

提交前请确保 `go vet ./...` 和 `go test ./...` 通过。

---

## 📄 许可证

[MIT License](LICENSE) © 2026 Chronos Contributors

---

<div align="center">

**Made with ❤️ and Go**

</div>
