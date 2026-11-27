# 模块开发指南

Chronospp 的模块就是一个 `.chpp` 文件（或一个目录），用 `import` 引入。本文档讲怎么写出可复用的模块，以及怎么打包成 `.chm` 发布给别人用。

---

## 目录

- [最简单的模块](#最简单的模块)
- [导出多个东西](#导出多个东西)
- [模块目录与入口](#模块目录与入口)
- [内置模块](#内置模块)
- [打包成 .chm](#打包成-chm)
- [发布到 GitHub](#发布到-github)
- [常见问题](#常见问题)

---

## 最简单的模块

新建 `greet.chpp`：

```chronos
fn hello(name) {
    return "你好，" + name
}
```

别处用：

```chronos
import "greet"

print(greet.hello("Chronospp"))
```

`import` 后面写**相对路径**（不带扩展名也可以）：

```chronos
import "./greet"          // 同目录
import "../lib/greet"     // 上一级
import "lib/greet"        // 子目录
```

---

## 导出多个东西

模块里的 `let`、`fn` 默认都能被访问。想显式声明导出，用 `export`：

```chronos
// mathx.chpp
let PI = 3.14159

fn square(x) {
    return x * x
}

fn cube(x) {
    return x * x * x
}

export square
export cube
```

调用：

```chronos
import "mathx"

print(mathx.square(5))    // 25
print(mathx.PI)           // 3.14159
```

### 别名

```chronos
import "mathx" as mx

print(mx.cube(3))         // 27
```

---

## 模块目录与入口

模块也可以是个**目录**，入口文件叫 `index.chpp`：

```
mylib/
├── index.chpp
├── helpers.chpp
└── config.chpp
```

`index.chpp`：

```chronos
import "./helpers"
import "./config"

let name = "mylib"

fn info() {
    return name + " v" + config.VERSION
}
```

用：

```chronos
import "mylib"

print(mylib.info())
```

---

## 内置模块

不用安装，直接 `import` 就有：

### os —— 文件与系统

```chronos
import "os"

os.readFile(path)          // 读文件
os.writeFile(path, text)   // 写文件（自动建父目录）
os.appendFile(path, text)  // 追加
os.exists(path)            // 是否存在 → true/false
os.remove(path)            // 删除
os.mkdir(path)             // 建目录
os.listDir(path)           // 列出目录内容 → 数组

os.join("a", "b", "c")     // 拼路径
os.abs(path)               // 绝对路径
os.basename(path)          // 文件名
os.dirname(path)           // 目录名

os.getenv(name)            // 读环境变量
os.setenv(name, value)     // 设环境变量
os.cwd()                   // 当前目录
os.chdir(path)             // 切换目录

os.run("ls", "-l")         // 执行命令 → { code, out, err }
os.exec("ls -l | head")    // 走 shell → 输出字符串
```

> **注意**：`os` 在浏览器 / WASM 环境不可用，调用会报错。手机 IDE 里能用文件相关的函数，但子进程不行。

### math —— 数学

```chronos
import "math"

math.sqrt(16)         // 4
math.pow(2, 10)       // 1024
math.abs(-42)         // 42（保持整数）
math.floor(3.7)       // 3
math.ceil(3.2)        // 4
math.round(3.7)       // 4

math.sin/cos/tan(x)
math.asin/acos/atan(x)
math.log/log2/log10(x)
math.exp(x)

math.min(1, 5, 3)     // 1
math.max(1, 5, 3)     // 5
math.random()         // 0 到 1 的随机数
math.randInt(1, 100)  // 1 到 99 的随机整数

math.PI               // 3.141592653589793
math.E                // 2.718281828459045
math.inf              // +Inf
```

### 优先级

模块解析顺序：

1. **当前目录**的文件（本地优先）
2. **已安装的第三方包**（`~/.chronos/packages/`）
3. **内置模块**（`os`、`math`）

所以你自己写一个 `math.chpp` 放在同目录，会覆盖内置的 `math`。

---

## 打包成 .chm

`.chm` 是 Chronospp 的第三方包格式，本质是 **tar.gz**，里面必须有 `manifest.json`。

### 1. 准备目录

```
mypkg/
├── manifest.json
├── index.chpp
└── extra.chpp
```

### 2. 写 manifest.json

```json
{
  "name": "mypkg",
  "version": "1.0.0",
  "description": "一句话说明这个包做什么",
  "author": "你的名字",
  "main": "index.chpp"
}
```

| 字段 | 必填 | 说明 |
|------|------|------|
| `name` | ✅ | 包名，`import "包名"` 就用它 |
| `version` | 建议 | 版本号，`chpp packages` 会显示 |
| `description` | 建议 | 一句话描述 |
| `author` | 可选 | 作者 |
| `main` | 可选 | 入口文件，默认 `index.chpp` |

### 3. 打包

```bash
tar czf mypkg.chm -C mypkg .
```

> 注意 `-C mypkg .`：让包内是 `./manifest.json` 而不是 `mypkg/manifest.json`。

检查一下内容：

```bash
tar tzf mypkg.chm
# ./
# ./manifest.json
# ./index.chpp
# ./extra.chpp
```

### 4. 安装测试

```bash
chpp install mypkg.chm
chpp packages
```

然后写个脚本试：

```chronos
import "mypkg"
print(mypkg.something())
```

### 5. 卸载

```bash
chpp uninstall mypkg
```

---

## 发布到 GitHub

Chronospp 会自动从 `polear234/chronos-packages` 仓库拉包。目录结构：

```
chronos-packages/
└── packages/
    ├── mypkg/
    │   └── mypkg.chm
    └── otherpkg/
        └── otherpkg.chm
```

发布后别人就能直接：

```bash
chpp install mypkg
```

---

## 常见问题

### import 找不到模块

```
ERROR: cannot resolve import path: mymod
```

按顺序检查：

1. 文件真的存在吗？`ls mymod.chpp`
2. 路径写对了吗？`import "./mymod"` 和 `import "mymod"` 都行，但 `import "lib/mymod"` 要从当前文件所在目录算起
3. 装成包了吗？`chpp packages` 看看

### 循环 import

```
ERROR: circular import detected: a.chpp
```

`a.chpp` 里 `import "b"`，`b.chpp` 里又 `import "a"`。把共用的部分拆到第三个文件里。

### 模块里的变量改了，外面看不到

模块有独立作用域。想让外面改，导出函数：

```chronos
// counter.chpp
let count = 0

fn inc() {
    count = count + 1
    return count
}

fn get() {
    return count
}

export inc
export get
```

### os.run 拿不到输出

`os.run` 返回一个哈希：

```chronos
let r = os.run("echo", "hello")
print(r["out"])     // "hello\n"
print(r["code"])    // 0
print(r["err"])     // 错误输出
```

命令不存在会报错；命令失败（非 0 退出）不报错，看 `code`。

### 想在浏览器里也能用

`os` 的多数函数在 WASM 环境不可用。写模块时把文件操作抽出来，纯逻辑部分不依赖 `os`：

```chronos
// 好：纯逻辑
fn parseLine(line) {
    return line.split(",")
}

// 不好：写死在模块顶层
let data = os.readFile("config.txt")
```

---

## 完整示例

一个读配置文件的小模块：

```chronos
// config.chpp
import "os"

let CACHE = {}

fn load(path) {
    if !os.exists(path) {
        return {}
    }
    let text = os.readFile(path)
    let result = {}
    for line in text.split("\n") {
        let t = line.trim()
        if t == "" || t[0] == "#" {
            continue
        }
        let parts = t.split("=")
        if len(parts) == 2 {
            result[parts[0].trim()] = parts[1].trim()
        }
    }
    return result
}

fn get(cfg, key, fallback) {
    if cfg[key] == nil {
        return fallback
    }
    return cfg[key]
}

export load
export get
```

用：

```chronos
import "config"

let cfg = config.load("app.conf")
print(config.get(cfg, "host", "localhost"))
```
