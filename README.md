<div align="center">

<img src="docs/logo.png" width="180" alt="Chronospp logo">

# Chronospp

**一门能回退时间的编程语言**

*变量记住自己的历史 · 事务自动回滚 · 时间点快照*

*C++ 的语法 · Python 的表达力 · C++ 级的类型系统*

<br/>

[![License: MIT](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)
[![Build Status](https://img.shields.io/github/actions/workflow/status/polear234/chronospp/release.yml?branch=main)](https://github.com/polear234/chronospp/actions)
[![Go Report Card](https://goreportcard.com/badge/github.com/polear234/chronospp)](https://goreportcard.com/report/github.com/polear234/chronospp)
[![Version](https://img.shields.io/badge/version-1.1.0-3ddcae.svg)]()
[![Go Version](https://img.shields.io/badge/go-1.21%2B-00ADD8.svg)](https://go.dev/)
[![Platform](https://img.shields.io/badge/platform-android%20%7C%20linux%20%7C%20macos%20%7C%20windows-lightgrey.svg)]()

[![Lines](https://img.shields.io/badge/code-11k%20lines-informational.svg)]()
[![Deps](https://img.shields.io/badge/dependencies-zero-success.svg)]()

</div>

---

## 🕐 核心：变量时间旅行

这是 Chronospp 与其他脚本语言最大的不同 —— **每一次赋值都会被记录，你可以随时回到过去**。

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

// Chronospp
let evens = [1,2,3,4,5]
    |> map(fn(x) { x * 2 })
    |> filter(fn(x) { x % 4 == 0 })
```

`a |> f(b)` 完全等价于 `f(b, a)` —— 值作为**最后一个参数**传入。

---

## 📱 编译成安卓应用

这是 0.2 版本的新能力：**把脚本变成真正的安卓应用**。

> APK / SH 编译由手机端的 **Chronospp IDE** 提供；CLI 的 `chpp build` 只生成桌面可执行文件。

在 Chronospp IDE 里写好代码，点按钮即可：

| 按钮 | 产物 | 用途 |
|------|------|------|
| 📦 打包 APK | `xxx.apk` | 独立安卓应用，可分享安装 |
| 🐚 编译 SH | `xxx.sh` | Termux / Linux 直接运行 |

编译出的 APK 是**纯运行模式** —— 打开只显示输出，没有编辑器、没有按钮，就像 C++ 编译出的可执行文件。

<details>
<summary>技术细节：APK 是怎么生成的</summary>

1. App 内预置一个「模板 APK」（裸的 WebView 宿主）
2. 打包时读模板，用 `ZipOutputStream` 重写：剔除旧签名、注入 `assets/script.chpp`
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
| **MSI 安装包** | `chpp-1.0.0-windows-amd64.msi` | 双击安装到 `Program Files\Chronospp` |
| **便携版 ZIP** | `chpp-1.0.0-windows-amd64.zip` | 解压即用，无需安装 |
| **PowerShell** | `install.ps1` | `.\install.ps1` 用户级安装 |

> ARM64 Windows（如 Surface Pro X）把文件名里的 `amd64` 换成 `arm64`。

### Linux

```bash
# deb 包（Ubuntu / Debian）
sudo dpkg -i chpp_1.0.0_amd64.deb     # x86_64
sudo dpkg -i chpp_1.0.0_arm64.deb     # arm64（树莓派 / 手机）

# 或安装脚本
sudo ./install.sh          # 系统级 → /usr/local/bin
./install.sh --user        # 用户级 → ~/.local/bin
```

### macOS

```bash
# Apple Silicon
chmod +x chpp-darwin-arm64 && sudo mv chpp-darwin-arm64 /usr/local/bin/chpp

# Intel
chmod +x chpp-darwin-amd64 && sudo mv chpp-darwin-amd64 /usr/local/bin/chpp
```

### Android

安装 `chronospp-ide-1.0.0.apk` —— 内置编辑器和编译器，可离线运行、打包 APK / SH。

### 源码编译

```bash
git clone https://github.com/polear234/chronospp.git
cd chronos
go build -o chpp .      # 需要 Go 1.21+，零第三方依赖
```

---

## 🚀 第一个程序

```chronos
// hello.chpp
print("Hello, Chronospp!")

fn fib(n) {
    if n <= 1 { return n }
    return fib(n - 1) + fib(n - 2)
}
print("fib(10) = " + str(fib(10)))   // 55
```

```bash
chpp run hello.chpp
# Hello, Chronospp!
# fib(10) = 55
```

---

## 🌟 完整特性

| 类别 | 能力 |
|------|------|
| 🕐 **时间旅行** | `traveller` 历史栈 · `mark()`/`restore()` 时间点 · `transaction` 自动回滚 · `diff()` 变化对比 |
| 🧠 **语言核心** | 一等函数 · 闭包 · 递归 · 高阶函数 · 可选静态类型 |
| 🏛️ **面向对象** | 类 · 单继承 · **接口** `interface`/`implements` · 多态 · `this`/`super` |
| 🔧 **C++ 风格** | **指针** `&x`/`*p` · 强类型函数 `int add(int a, int b)` · 数据流 `prints << x << endl` |
| ⚡ **并发** | `start(fn1, fn2)` 真并行执行，实测快 3 倍 |
| ⛓️ **语法糖** | 管道 `\|>` · 方法链 · 类型标注 · 复合赋值 `+= -= *= /=` |
| 🎯 **控制流** | `if`/`elif`/`else` · `while` · `for-in` · `break`/`continue` · 循环 `else` |
| 🧩 **模式匹配** | `match` / `case` / `default` |
| 🛡️ **错误处理** | `try` / `catch` / `finally` / `throw` |
| 📦 **模块系统** | `import` / `as` · 循环引用检测 · 内置 `os`/`math`/`requests` |\n| 🔗 **FFI** | 调用 `.so`/`.dll`/`.dylib` · `ffi.load`/`lib.call` · 纯 Go 零 cgo |\n| 📱 **交互 UI** | `ui.button`/`input`/`slider`/`checkbox` · 生成 HTML · 手机 APK 内置渲染 |
| 📚 **包管理** | `.chm` 包 · `chpp install/search/info/list/freeze/pack` · 依赖与版本约束 |
| 🌏 **国际化** | Unicode 标识符：中文 / 日文 / 俄文均可命名 |
| 🖥️ **工具链** | `chpp serve` 静态服务器 · `chpp upgrade` 自更新 · **Chronospp Studio** 编辑器 |
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

### 交互式 UI

生成手机/桌面可用的界面，按钮点击、输入框、滑块都能调回 Chronospp 函数：

```chronos
import "ui"

ui.title("我的应用")
ui.text("欢迎使用 Chronospp！")

ui.button("点我打招呼", fn() {
    print("Hello!")
})
ui.input("你的名字", fn(name) {
    print("你好, " + name)
})
ui.slider("音量", 0, 100, 50, fn(v) {
    print("音量: " + v)
})
ui.checkbox("启用通知", true, fn(on) {
    print("通知: " + on)
})
ui.select("选择颜色", ["红","绿","蓝"], fn(c) {
    print("选择了: " + c)
})
ui.list(["任务1", "任务2"])
ui.progress(75)

let html = ui.show()    // 生成 HTML（含交互回调）
```

| 组件 | 交互 |
|------|------|
| `button("文本", fn)` | 点击 → 回调 |
| `input("提示", fn)` | 回车 → 传值 |
| `slider("标签", min, max, val, fn)` | 拖动 → 实时传值 |
| `checkbox("标签", checked, fn)` | 切换 → 传 1/0 |
| `select("标签", ["a","b"], fn)` | 选择 → 传选项 |
| `list([...])` | 显示列表 |
| `progress(75)` | 进度条 |
| `card(...)` / `row(...)` | 布局容器 |

> 手机 IDE 编译的 APK 内置 WebView，`ui.show()` 直接渲染。

### FFI 调用动态库

直接调用 `.so` / `.dll` / `.dylib` 中的 C 函数，**不需要 cgo、不需要编译器**。

```chronos
import "ffi"

// 加载共享库
let libm = ffi.load("libm.so.6")

// 整数函数（自动检测 ABI）
let libc = ffi.load("libc.so.6")
print(libc.call("strlen", "hello"))      // 5

// 浮点函数（声明签名后调用）
libm.declare("sqrt", "float", "float")
print(libm.call("sqrt", 16.0))           // 4.0

libm.declare("pow", "float", "float", "float")
print(libm.call("pow", 2.0, 10.0))       // 1024.0
```

**Windows 上调用 DLL**：

```chronos
import "ffi"
let user32 = ffi.load("user32.dll")
// user32.call("MessageBoxA", 0, ffi.ptr("Hello"), ffi.ptr("FFI"), 0)
```

API：
| 函数 | 说明 |
|------|------|
| `ffi.load(path)` | 加载动态库 |
| `lib.call(name, args...)` | 调用函数（自动检测 int/float ABI） |
| `lib.declare(name, ret, args...)` | 声明函数签名 |
| `ffi.ptr(str)` | 字符串 → C 指针 |
| `ffi.str(ptr)` | C 指针 → 字符串 |
| `ffi.alloc(n)` | 分配 n 字节内存 |

> 基于 purego（纯 Go FFI，运行时汇编 trampoline），零 cgo 依赖，单文件编译。

### 时间旅行

Chronospp 最独特的特性——**变量可以记住自己的历史，随时回退**。

```chronos
// traveller 变量：每次赋值都记录历史
let traveller int hp = 100
hp = 80
hp = 60
hp = 40

hp.history()     // [100, 80, 60]  所有历史
hp.at(0)         // 100            第 0 步
hp.at(1)         // 80             第 1 步
hp.step()        // 3              历史深度
hp.changes()     // [{from:100,to:80}, ...]  所有变化
hp.rollback()    // 退到 60
hp.reset()       // 清空历史
```

**全局时间点**：

```chronos
let hp = 100
let gold = 50

let t = mark("冒险前")     // 记录所有变量

hp = 20; gold = 0

diff(t)                   // ["hp", "gold"]  变了什么
restore(t)                // 全部恢复
```

**事务**：出错自动回滚

```chronos
try {
    transaction {
        hp = 0
        gold = 0
        throw "失败"
    }
} catch (e) {
    print(e)              // hp 和 gold 保持原样
}
```

用途：游戏存档读档、配置回滚、交互式程序的撤销栈、数据调试。

### 接口

```chronos
interface Speaker {
    fn speak()
    fn name()
}

class Dog implements Speaker {
    fn init(n) { this.n = n }
    fn speak() { return this.n + " 汪！" }
    fn name() { return this.n }
}

class Cat implements Speaker {
    fn speak() { return "喵" }
    // 缺 name() → 定义类时直接报错
}
```

多个接口用逗号分隔：`class Robot implements Speaker, Walker, Flyer`

### 指针

```chronos
let x = 42
let p = &x          // 取地址
print(*p)           // 解引用 → 42
*p = 100            // 通过指针修改
print(x)            // 100

int increment(int *ptr) {
    *ptr = *ptr + 1
    return *ptr
}

let n = 10
increment(&n)
print(n)            // 11（原变量被改了）
```

### 强类型函数

```chronos
int add(int a, int b) {
    return a + b
}

add(1, 2)           // ✅ 3
add(1)              // ❌ 需要 2 个参数，收到 1 个
add("x", "y")       // ❌ 参数 a 声明为 int，收到 STRING

int bad() { return "hi" }    // ❌ 声明返回 int，实际返回 STRING
```

类型为 `any` 或不写类型时不做检查。`fn` 风格与 C++ 风格可以混用。

### 数据流

```chronos
prints << "Hello" << endl        // 输出并换行
prints << "不换行"
prints << 42 << endl

let n = 0
inputs >> n                      // 读入到已声明类型的变量
```

`prints << x` 不换行（同 C++ `cout`），要换行写 `endl` 或 `"\n"`。

### 并行

```chronos
fn slow_a() { /* ... */ }
fn slow_b() { /* ... */ }

let results = start(slow_a, slow_b)   // 并行执行
print(results[0], results[1])

// 带参数
fn work(n) { return n * n }
start(work(3), work(4))               // → [9, 16]
```

### 包管理

```bash
chpp install json              # 从 GitHub 装
chpp install ./mypkg.chm       # 装本地文件
chpp search                    # 看看有哪些包
chpp info json                 # 查看详情
chpp list                      # 已安装
chpp freeze                    # 输出精确版本
chpp pack ./mypkg -o out.chm   # 打包
chpp uninstall json            # 卸载
```

依赖写在 `manifest.json`：

```json
{
  "name": "mypkg",
  "version": "1.0.0",
  "requires": { "json": ">=1.0.0", "string": "^1.2.0" }
}
```

版本约束支持 `>=` `<` `^` `~` `*`。

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
import "math_utils.chpp" as math
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
chpp run app.chpp          # 运行脚本
chpp build app.chpp        # 编译成独立可执行文件
chpp build app.chpp -p     # 明文模式（默认加密混淆）
chpp build app.chpp -o out # 指定输出名
chronos repl                # 交互式 REPL
chpp version             # 查看版本
chronos help                # 帮助
```

### 编译产物

```bash
chpp build examples/hello.chpp -o hello
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
        -dname "CN=Chronospp, OU=Dev, O=Chronospp, C=CN"
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

## 📚 模块与扩展

- **[模块开发指南](docs/MODULES.md)** —— 写可复用的 `.chpp` 模块，打包成 `.chm` 发布
- 内置模块：`import "os"`（文件/命令）、`import "math"`（数学）
- 安装第三方包：`chpp install 包名` / `chpp install 包.chm`
- 查看已安装：`chpp packages`

## 📄 许可证

[MIT](LICENSE) © 2026 Chronospp Contributors
