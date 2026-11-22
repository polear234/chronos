<div align="center">

<img src="docs/logo.png" width="180" alt="Chronos logo">

# Chronos

**一门把「时间」做成一等公民的脚本语言**

*变量可回滚 · 数据可快照 · 一行编译成 APK / SH*

<br/>

[![License: MIT](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)
[![Build Status](https://img.shields.io/github/actions/workflow/status/polear234/chronos/release.yml?branch=main)](https://github.com/polear234/chronos/actions)
[![Go Report Card](https://goreportcard.com/badge/github.com/polear234/chronos)](https://goreportcard.com/report/github.com/polear234/chronos)
[![Version](https://img.shields.io/badge/version-0.3.1-3ddcae.svg)]()
[![Go Version](https://img.shields.io/badge/go-1.21%2B-00ADD8.svg)](https://go.dev/)
[![Platform](https://img.shields.io/badge/platform-android%20%7C%20linux%20%7C%20macos%20%7C%20windows-lightgrey.svg)]()

[![Lines](https://img.shields.io/badge/code-7k%20lines-informational.svg)]()
[![Deps](https://img.shields.io/badge/dependencies-zero-success.svg)]()

</div>

---

## 🕐 核心：变量时间旅行

这是 Chronos 与其他脚本语言最大的不同 —— **每一次赋值都会被记录，你可以随时回到过去**。

```chronos
let traveller int hp = 100

hp = 80        // 受伤
hp = 30        // 重伤
print(hp)      // 30

hp.rollback()  // 时光倒流
print(hp)      // 80

hp.rollback()
print(hp)      // 100
```

不是撤销一次操作，而是**每个变量自带一条历史栈**。调试、模拟、状态回退，不用再手写备份逻辑。

---

## 🔄 事务性快照

改坏了不怕，`withSnapshot` 在回调结束后自动还原：

```chronos
let arr = [1, 2, 3]

withSnapshot([arr], fn() {
    arr.push(99)
    arr.push(100)
    print(arr)   // [1, 2, 3, 99, 100]
})

print(arr)       // [1, 2, 3]  ← 自动恢复
```

需要手动控制提交时机时，用 `Timeline`：

```chronos
let t = Timeline(arr)
arr.push(4)
t.commit()       // 提交，丢弃历史
```

---

## ⛓️ 管道操作符

把嵌套调用写成从左到右的数据流：

```chronos
// 传统写法
let evens = filter(map([1,2,3,4,5], fn(x){ x * 2 }), fn(x){ x % 4 == 0 })

// Chronos
let evens = [1,2,3,4,5]
    |> map(fn(x) { x * 2 })
    |> filter(fn(x) { x % 4 == 0 })
```

`a |> f(b)` 完全等价于 `f(b, a)` —— 值作为**最后一个参数**传入。

---

## 📱 编译成安卓应用

这是 0.2 版本的新能力：**把脚本变成真正的安卓应用**。

> APK / SH 编译由手机端的 **Chronos IDE** 提供；CLI 的 `chronos build` 只生成桌面可执行文件。

在 Chronos IDE 里写好代码，点按钮即可：

| 按钮 | 产物 | 用途 |
|------|------|------|
| 📦 打包 APK | `xxx.apk` | 独立安卓应用，可分享安装 |
| 🐚 编译 SH | `xxx.sh` | Termux / Linux 直接运行 |

编译出的 APK 是**纯运行模式** —— 打开只显示输出，没有编辑器、没有按钮，就像 C++ 编译出的可执行文件。

<details>
<summary>技术细节：APK 是怎么生成的</summary>

1. App 内预置一个「模板 APK」（裸的 WebView 宿主）
2. 打包时读模板，用 `ZipOutputStream` 重写：剔除旧签名、注入 `assets/script.chr`
3. 关键约束：Android 11+ 要求 `resources.arsc` **不压缩且 4 字节对齐** —— 通过「首个条目 + STORED + 无 extra」让它天然落在偏移 44
4. 用内置 keystore + `apksig` 重新签名（v1 + v2）
5. 自定义包名：解析二进制 AXML 字符串池，替换 `package` 属性的字符串（支持 UTF-16 / UTF-8 两种编码）
6. 通过自实现的 `ContentProvider` 暴露 `content://` URI，拉起系统安装器

</details>

---

## 📦 安装

### Windows

| 方式 | 文件 | 说明 |
|------|------|------|
| **MSI 安装包** | `chronos-0.3.1-windows-amd64.msi` | 双击安装到 `Program Files\Chronos` |
| **便携版 ZIP** | `chronos-0.3.1-windows-amd64.zip` | 解压即用，无需安装 |
| **PowerShell** | `install.ps1` | `.\install.ps1` 用户级安装 |

> ARM64 Windows（如 Surface Pro X）把文件名里的 `amd64` 换成 `arm64`。

### Linux

```bash
# deb 包（Ubuntu / Debian）
sudo dpkg -i chronos_0.3.1_amd64.deb     # x86_64
sudo dpkg -i chronos_0.3.1_arm64.deb     # arm64（树莓派 / 手机）

# 或安装脚本
sudo ./install.sh          # 系统级 → /usr/local/bin
./install.sh --user        # 用户级 → ~/.local/bin
```

### macOS

```bash
# Apple Silicon
chmod +x chronos-darwin-arm64 && sudo mv chronos-darwin-arm64 /usr/local/bin/chronos

# Intel
chmod +x chronos-darwin-amd64 && sudo mv chronos-darwin-amd64 /usr/local/bin/chronos
```

### Android

安装 `chronos-ide-0.3.1.apk` —— 内置编辑器和编译器，可离线运行、打包 APK / SH。

### 源码编译

```bash
git clone https://github.com/polear234/chronos.git
cd chronos
go build -o chronos .      # 需要 Go 1.21+，零第三方依赖
```

---

## 🚀 第一个程序

```chronos
// hello.chr
print("Hello, Chronos!")

fn fib(n) {
    if n <= 1 { return n }
    return fib(n - 1) + fib(n - 2)
}
print("fib(10) = " + str(fib(10)))   // 55
```

```bash
chronos run hello.chr
# Hello, Chronos!
# fib(10) = 55
```

---

## 🌟 完整特性

| 类别 | 能力 |
|------|------|
| 🕐 **时间特性** | `traveller` 回滚变量 · `withSnapshot` 事务快照 · `Timeline` 提交/撤销 |
| 🧠 **语言核心** | 动态类型 · 一等函数 · 闭包 · 递归 · 高阶函数 |
| 🏛️ **面向对象** | 类 · 单继承 · 多态 · `this` / `super` · 方法重写 · 继承环检测 |
| ⛓️ **语法糖** | 管道 `\|>` · 方法链 · 类型标注 · 复合赋值 `+= -= *= /=` |
| 🎯 **控制流** | `if`/`elif`/`else` · `while` · `for-in` · `break`/`continue` |
| 🧩 **模式匹配** | `match` / `case` / `default` |
| 🛡️ **错误处理** | `try` / `catch` / `finally` / `throw` |
| 📦 **模块系统** | `import` / `as` · 独立环境 · 循环引用检测 |
| 🌏 **国际化** | Unicode 标识符：中文 / 日文 / 俄文均可命名 |
| 📱 **分发** | 编译为 APK · 编译为自包含 SH 脚本 |

---

## 📖 语法速览

### 变量与类型

```chronos
let x = 10                    // 动态类型
let int n = 42                // 类型标注，赋值时会校验
let string s = "你好"
let traveller int hp = 100    // 带历史栈的变量
```

### 控制流

```chronos
if x > 5 {
    print("big")
} elif x > 0 {
    print("small")
} else {
    print("zero")
}

for i in range(0, 3) {
    print(i)
}

let i = 0
while i < 3 {
    i += 1
}
```

### 函数与闭包

```chronos
fn add(a, b) { return a + b }

let mul = fn(a, b) { a * b }   // 匿名函数
print(add(1, 2))               // 3
print(mul(3, 4))               // 12
```

### 类与继承

```chronos
class Animal {
    name = ""
    init(name) { this.name = name }
    speak() { return this.name + " makes a sound" }
}

class Dog extends Animal {
    speak() { return this.name + " barks!" }
    info() { return super.speak() }   // 调用父类
}

let d = new Dog("Buddy")
print(d.speak())   // "Buddy barks!"
```

### 类型转换

```chronos
// C++ 风格强制转换
let n = static_cast<int>("42")      // 42
let f = static_cast<float>(3)       // 3.0
let s = static_cast<str>(100)       // "100"
let t = static_cast<int>(3.99)      // 3（截断）

// 也支持函数式写法（完全等价）
let m = int("42")
```

类型别名：`int`/`integer`/`int64`、`float`/`double`/`float64`、`str`/`string`。

### 输入

```chronos
// 推荐：先声明类型，input 赋值时自动转换
let int age = 0
age = input("年龄: ")          // "25" → 25
print("明年 " + str(age + 1) + " 岁")

let float pi = 0.0
pi = input("圆周率: ")         // "3.14" → 3.14

let string name = ""
name = input("名字: ")
print("你好，" + name)
```

也可以用 `inputInt()` / `inputFloat()` 直接读成数字，或 `int(input(...))` 手动转换。

> CLI 从 stdin 读；Android IDE 里会弹出输入框。

### 类型化变量与隐式转换

声明了类型的变量，赋值时会把**字符串自动转成数字**：

```chronos
let int n = 0
n = "42"        // ✓ 自动转成 42

let float f = 0
f = "3.14"      // ✓ 自动转成 3.14

let string s = "a"
s = 42          // ✗ 报错：数字不会隐式转成字符串
```

只做「字符串 → 数字」的单向转换 —— 反向转换会掩盖类型错误。

### 模块

```chronos
import "math_utils.chr" as math
print(math.add(1, 2))
```

### 中文标识符

```chronos
let 名字 = "世界"
fn 打招呼(谁) {
    return "你好，" + 谁
}
print(打招呼(名字))   // 你好，世界
```

---

## 🧰 内置函数

| 分类 | 函数 |
|------|------|
| 输入输出 | `print` `println` `input` `inputInt` `inputFloat` |
| 类型 | `type` `int` `float` `str` |
| 集合 | `len` `push` `pop` `shift` `unshift` `keys` `values` `range` |
| 高阶 | `map` `filter` `forEach` `reduce` `find`（可被用户函数覆盖） |

### 方法

```chronos
// 字符串
s.length()  s.upper()  s.lower()  s.trim()
s.contains("x")  s.split(",")  s.replace("a", "b")

// 数组方法
arr.push(x)  arr.pop()  arr.shift()  arr.unshift(x)
arr.insert(i, x)  arr.contains(x)  arr.concat(other)  arr.clear()
arr.slice(a, b)  arr.join(",")  arr.reverse()  arr.length()

// 高阶函数（全局形式，也推荐用管道）
map(arr, fn)  filter(arr, fn)  forEach(arr, fn)
reduce(arr, fn, init)  find(arr, fn)
arr |> map(fn)      // 等价于 map(arr, fn)

// 字典
h.length()  h.keys()  h.values()  h.has("k")
```

---

## 💻 命令行

```bash
chronos run app.chr          # 运行脚本
chronos build app.chr        # 编译成独立可执行文件
chronos build app.chr -p     # 明文模式（默认加密混淆）
chronos build app.chr -o out # 指定输出名
chronos repl                # 交互式 REPL
chronos version             # 查看版本
chronos help                # 帮助
```

### 编译产物

```bash
chronos build examples/hello.chr -o hello
./hello        # 独立运行，不依赖源码
```

编译采用「解释器 + 内嵌脚本」模式：
- 默认 **AES-256-CTR 加密**脚本，防止明文泄露
- `-p / --plain` 保留明文便于调试
- 产物约 2 MB（含完整解释器）

> ⚠️ 这是**混淆**而非真正的代码保护 —— 密钥在二进制内，有能力的逆向者仍可提取。

---

## 📁 项目结构

```
chronos/
├── main.go              CLI 入口
├── embed.go             编译/加密/内嵌 payload
├── wasm_main.go         WASM 入口（浏览器 / 安卓）
├── token/               词法单元
├── lexer/               词法分析（Unicode 标识符）
├── ast/                 抽象语法树
├── parser/              Pratt 解析器
├── object/              运行时对象 + 环境 + Traveller
├── interpreter/         求值器 + 内置函数
├── repl/                交互式解释器
├── examples/            示例程序
├── android/             安卓 IDE（WebView + WASM）
└── scripts/             安装脚本
```

---

## ⚠️ 已知限制

- **标准库较薄**：无文件 IO / 网络 / 正则 / 数学库
- **`import` 无缓存**：每次导入都会重新执行模块
- **`match` 仅值匹配**：不支持解构或守卫
- **无调试器**：只有 `print` 和错误堆栈
- **编译产物 ≈ 解释器体积**：打包而非 AOT 编译
- **快照是浅拷贝**：`withSnapshot` 的数组快照只记录长度，元素覆写不会恢复

---

## 🤖 自动发布

推 `v*` tag 会触发 GitHub Actions，自动产出全套发行物：

| Job | 产物 |
|-----|------|
| `build` | 六平台二进制（linux / darwin / windows × amd64 / arm64） |
| `package-windows` | **MSI 安装包**（WiX，自动写入 PATH）+ 便携 ZIP |
| `package-deb` | Debian / Ubuntu deb 包 |
| `package-android` | **Android IDE APK**（WebView + WASM） |

### APK 签名配置

在仓库 **Settings → Secrets and variables → Actions** 添加：

```
ANDROID_KEYSTORE_BASE64 = <你的 keystore 的 base64>
```

```bash
# 本地生成（只需一次）
keytool -genkeypair -keystore chronos.keystore -alias chronos \
        -keyalg RSA -keysize 2048 -validity 10000 \
        -storepass android -keypass android \
        -dname "CN=Chronos, OU=Dev, O=Chronos, C=CN"
base64 -w0 chronos.keystore    # 复制输出到 Secret
```

> 不配置也能构建，但会生成临时密钥 —— 产出的 APK **无法覆盖升级**已安装的版本。

---

## 🤝 贡献

欢迎 Issue 和 PR。核心原则：

- **零第三方依赖** —— 只用 Go 标准库
- **代码可读性优先** —— 这是教学向项目
- **新特性需带测试**

```bash
go test ./...      # 跑全部测试
go vet ./...       # 静态检查
make test          # 同上
```

---

## 📄 许可证

[MIT](LICENSE) © 2026 Chronos Contributors
