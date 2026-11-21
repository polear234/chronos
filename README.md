# PicoX 语言

PicoX 是一个用 Go 语言编写的轻量级、动态类型的脚本语言。它的设计目标是简单易学、表达力强，适合嵌入式脚本、快速原型开发和教学用途。

## 特性

- **简洁语法**：类似 Python/JavaScript 的清晰语法
- **动态类型**：支持整数、浮点数、字符串、布尔值、数组、字典
- **函数式编程**：一等公民函数、闭包、高阶函数
- **面向对象**：基于原型/类的继承、多态、封装
- **模式匹配**：类似 Rust 的 match 表达式
- **错误处理**：try/catch/finally/throw
- **模块系统**：import/export 支持
- **内置标准库**：字符串、数组、字典、数学等常用操作

## 安装与运行

### 编译（生成完全独立的静态二进制）

```bash
# 普通编译（产出 picox 单文件，无任何运行时依赖）
make build

# 完全静态链接（连 libc 都不依赖，可在任何 Linux 上裸跑）
make static

# 交叉编译多平台（Linux/macOS/Windows，amd64/arm64）
make dist
# 或
./scripts/build-all.sh
```

### 命令行用法

```bash
picox                          # 无参数：启动交互式 REPL
picox run <file.px>            # 运行脚本文件
picox -run <file.px>           # 运行脚本（简写）
picox <file.px>                # 运行脚本（兼容旧用法）
picox run -                    # 从 stdin 读取并执行
picox version                  # 查看版本
picox help                     # 查看帮助

# 示例
picox -run examples/hello.px
echo 'print("Hello World!")' | picox run -
```

编译产物 `picox` 是**单文件静态二进制**，拷贝到任何同架构的 Linux 机器上即可直接运行，无需安装 Go 或任何依赖。

## 语法速览

### 变量声明

```picox
let x = 10          // 可变变量
const PI = 3.14159  // 常量
```

### 基本类型

```picox
let num = 42           // 整数
let pi = 3.14          // 浮点数
let str = "Hello"      // 字符串
let flag = true        // 布尔值
let nil = nil          // 空值
```

### 数组

```picox
let lst = [1, 2, 3, "hello"]

// 方法调用风格（推荐）
lst.push(4)          // [1, 2, 3, "hello", 4]
lst.pop()            // 移除并返回最后元素
lst.length()         // 长度
lst[0]               // 索引访问
lst.map(fn(x) { x * 2 })    // 映射
lst.filter(fn(x) { x > 1 }) // 过滤
lst.reduce(fn(acc, x) { acc + x }, 0) // 归约

// 全局函数风格（等价）
push(lst, 4)
pop(lst)
len(lst)
```

数组完整方法：`length` `push` `pop` `shift` `unshift` `insert` `removeAt` `indexOf` `contains` `slice` `concat` `clear` `reverse` `join`，以及高阶方法 `map` `filter` `forEach` `reduce` `find`。

### 字典

```picox
let dict = {"name": "Alice", "age": 25}
dict["name"]      // 访问
dict["age"] = 26  // 修改
keys(dict)        // 所有键
values(dict)      // 所有值
```

### 函数

```picox
fn add(a, b) {
    return a + b
}

// 高阶函数
fn applyTwice(f, x) {
    return f(f(x))
}
```

### 控制流

```picox
// if/elif/else
if x > 0 {
    print("positive")
} elif x < 0 {
    print("negative")
} else {
    print("zero")
}

// while 循环
while condition {
    // ...
}

// for 循环
for item in array {
    // ...
}

for i in range(10) {
    // 0 到 9
}
```

### 类与继承

```picox
class Animal {
    name = ""
    
    init(name) {
        this.name = name
    }
    
    speak() {
        print(this.name + " makes a sound")
    }
}

class Dog extends Animal {
    breed = ""
    
    init(name, breed) {
        super.init(name)
        this.breed = breed
    }
    
    speak() {
        print(this.name + " barks!")
    }
}

let dog = new Dog("Buddy", "Labrador")
dog.speak()
```

### 模式匹配

```picox
match (value) {
    case 1: { print("one") }
    case 2: { print("two") }
    case "hello": { print("greeting") }
    default: { print("other: " + str(value)) }
}
```

### 错误处理

```picox
try {
    riskyOperation()
} catch (e) {
    print("Error: " + e)
} finally {
    cleanup()
}

throw("Something went wrong")
```

## 内置函数

| 函数 | 说明 |
|------|------|
| `print(...)` | 打印到控制台 |
| `len(x)` | 返回长度（字符串/数组/字典） |
| `push(arr, x)` | 数组末尾添加元素 |
| `pop(arr)` | 移除并返回数组最后元素 |
| `shift(arr)` | 移除并返回数组第一个元素 |
| `unshift(arr, x)` | 数组开头添加元素 |
| `keys(dict)` | 返回字典所有键的数组 |
| `values(dict)` | 返回字典所有值的数组 |
| `type(x)` | 返回类型字符串 |
| `int(x)` | 转换为整数 |
| `float(x)` | 转换为浮点数 |
| `str(x)` | 转换为字符串 |
| `range(start, end, step?)` | 生成数字数组 |

## 示例程序

查看 `examples/` 目录：

- `hello.px` - 基础语法演示
- `fibonacci.px` - 斐波那契数列多种实现
- `oop.px` - 面向对象编程示例

## 架构

```
picox/
├── main.go           # 入口点
├── token/            # 词法标记定义
├── lexer/            # 词法分析器
├── parser/           # 语法分析器
├── ast/              # 抽象语法树节点
├── object/           # 运行时对象系统
├── interpreter/      # 解释器/求值器
├── repl/             # REPL 交互环境
├── stdlib/           # 标准库（待扩展）
└── examples/         # 示例程序
```

## 开发路线图

- [x] 模块系统（import 已完成）
- [ ] 标准库扩展（文件 IO、网络、正则等）
- [ ] 编译器优化（字节码 VM）
- [ ] 调试器支持
- [ ] 包管理器
- [ ] LSP 语言服务器

## 模块系统（import）

```picox
// math.px（被导入的模块）
let PI = 3.14159
fn square(x) { return x * x }

// main.px
import "math" as m        // as 别名
print(m.square(5))        // 25

import "math"             // 默认用文件名做别名
print(math.PI)            // 3.14159
```

- `import "xxx"` 读取同目录下的 `xxx.px`，在**独立环境**中执行
- 模块内所有变量和函数通过 `别名.成员` 访问
- 支持 `import "xxx" as 别名` 自定义命名
- 自动检测循环 import（A 导 B，B 导 A 会报错）
- 支持 `import "dir/xxx"` 子目录路径和 `import "xxx"` 自动补 `.px`

## 许可证

MIT License