# gobash - Bash兼容跨平台Shell

一个使用Go语言开发的兼容Bash语法的跨平台Shell程序，优先支持Windows平台。

## 功能特性

- ✅ Bash脚本执行器（支持shebang和注释行）
- ✅ 交互式Shell（REPL）
- ✅ 多行输入支持（以`\`结尾的命令）
- ✅ 完整的命令历史（history命令，持久化存储）
- ✅ 命令别名（alias/unalias）
- ✅ 丰富的内置命令集（cd, pwd, echo, ls, cat, mkdir, rm等）
- ✅ 管道和重定向（|, >, <, >>），支持内置命令重定向
- ✅ 环境变量支持（单引号不展开，双引号展开变量）
- ✅ 命令替换（`$(command)` 和 `` `command` ``）
- ✅ 算术展开（`$((expr))`）
- ✅ 控制流语句（if/else, for, while）
- ✅ 函数定义和调用（支持参数传递）
- ✅ 增强的错误处理和提示
- ✅ Windows平台优化

## 编译

```bash
go build -o gobash.exe ./cmd/gobash
```

## 使用方法

### 交互式模式

直接运行可执行文件：

```bash
gobash.exe
```

### 执行脚本文件

```bash
gobash.exe script.sh
```

或使用 `-f` 参数：

```bash
gobash.exe -f script.sh
```

### 执行命令字符串

使用 `-c` 参数：

```bash
gobash.exe -c "echo hello world"
```

## 内置命令

### 目录操作
- `cd [目录]` - 改变当前目录（支持~展开）
- `pwd` - 显示当前工作目录

### 文件操作
- `ls [-l] [-a] [目录]` - 列出目录内容（-l长格式，-a显示隐藏文件）
- `cat [文件...]` - 显示文件内容
- `head [-n 行数] [文件...]` - 显示文件的前几行（默认10行）
- `tail [-n 行数] [文件...]` - 显示文件的后几行（默认10行）
- `wc [-l] [-w] [-c] [-m] [文件...]` - 统计行数、字数、字符数（-l行数，-w字数，-c字节数，-m字符数）
- `grep [-i] [-n] [-o] [模式] [文件...]` - 文本搜索（-i忽略大小写，-n显示行号，-o只显示匹配部分）
- `sort [-r] [-n] [-u] [文件...]` - 排序（-r逆序，-n数值排序，-u去重）
- `uniq [-c] [-d] [-i] [文件...]` - 去重（-c显示计数，-d只显示重复行，-i忽略大小写）
- `cut -d [分隔符] -f [字段列表] [文件...]` - 剪切字段（-d指定分隔符，-f指定字段，支持范围如1-3）
- `mkdir [-p] [目录...]` - 创建目录（-p创建父目录）
- `rmdir [目录...]` - 删除空目录
- `rm [-r] [-f] [文件/目录...]` - 删除文件或目录（-r递归，-f强制）
- `touch [文件...]` - 创建文件或更新时间戳

### 文本输出
- `echo [参数...]` - 打印参数
- `clear` - 清屏

### 环境变量
- `export [变量=值]` - 导出环境变量
- `unset [变量]` - 取消设置环境变量
- `env` - 显示所有环境变量
- `set` - 显示所有变量和shell选项
- `set -x` / `set +x` - 显示/隐藏执行的命令（xtrace）
- `set -e` / `set +e` - 遇到错误立即退出/继续执行（errexit）
- `set -u` / `set +u` - 使用未定义变量时报错/允许未定义变量（nounset）
- `set -xe` - 可以组合多个选项

### 控制
- `exit [退出码]` - 退出shell
- `alias [name=value]` - 设置或显示命令别名
- `unalias [name]` - 取消设置别名
- `history` - 显示命令历史
- `history -c` - 清除命令历史
- `which [命令...]` - 查找命令路径
- `type [命令...]` - 显示命令类型（内置/外部）
- `true` - 总是成功返回
- `false` - 总是失败返回

## 示例

### 基础命令

```bash
$ echo "Hello, World!"
Hello, World!

$ pwd
C:\Users\27027\gobash

$ cd ..
$ pwd
C:\Users\27027
```

### 管道

```bash
$ echo "hello" | echo
hello
```

### 重定向

```bash
# 输出重定向
$ echo "test" > output.txt
$ cat output.txt
test

# 追加重定向
$ echo "append" >> output.txt
$ cat output.txt
test
append

# 输入重定向
$ cat < output.txt
test
append

# 内置命令也支持重定向
$ echo "hello" > test.txt
$ cat test.txt
hello
```

### 环境变量

```bash
$ export MYVAR=hello
$ echo $MYVAR
hello

# 单引号字符串不展开变量
$ echo '$MYVAR'
$MYVAR

# 双引号字符串展开变量
$ echo "$MYVAR"
hello

# 支持转义的$符号
$ echo "\$MYVAR is $MYVAR"
$MYVAR is hello

# 支持${VAR}格式
$ echo "${MYVAR}world"
helloworld
```

### 命令替换

```bash
# 使用 $(command) 格式
$ echo "Result: $(echo hello)"
Result: hello

# 使用反引号格式（在PowerShell中需要转义）
$ echo `echo world`
world

# 在字符串中使用命令替换
$ echo "Current dir: $(pwd)"
Current dir: C:\Users\27027\wbash

# 嵌套使用
$ echo "Files: $(ls | head -1)"
Files: gobash.exe
```

### 算术展开

```bash
# 基本算术运算
$ echo $((1 + 2))
3

$ echo $((10 * 5))
50

$ echo $((100 / 4))
25

# 运算符优先级
$ echo $((2 + 3 * 4))
14

# 括号改变优先级
$ echo $(((2 + 3) * 4))
20

# 支持变量
$ export NUM=10
$ echo $((NUM + 5))
15
```

### 控制流

```bash
# if语句
if [ -f file.txt ]; then
    echo "file exists"
else
    echo "file not found"
fi

# for循环
for i in 1 2 3; do
    echo $i
done

# while循环
i=0
while [ $i -lt 3 ]; do
    echo $i
    i=$((i+1))
done
```

### 别名和函数

```bash
# 设置别名
alias ll='ls -l'
alias la='ls -a'

# 显示所有别名
alias

# 取消别名
unalias ll

# 定义函数
function greet() {
    echo "Hello, $1!"
}

# 或使用简写格式
greet() {
    echo "Hello, $1!"
}

# 调用函数
greet "World"
```

### 命令历史

```bash
# 显示所有历史命令
history

# 清除历史记录
history -c
```

历史记录会自动保存到 `~/.gobash_history` 文件，下次启动时会自动加载。

### Shell选项（set命令）

```bash
# 显示当前选项状态
$ set
--- Shell Options ---

# 启用命令跟踪（显示执行的命令）
$ set -x
$ echo "hello"
+ echo hello
hello

# 关闭命令跟踪
$ set +x

# 启用错误时立即退出
$ set -e
$ false
# 脚本会立即退出

# 启用未定义变量检查
$ set -u
$ echo $UNDEFINED_VAR
错误: 执行错误: 未定义的变量: UNDEFINED_VAR

# 组合多个选项
$ set -xe
```

### 多行输入

```bash
# 以反斜杠结尾的命令可以继续输入
$ echo "Hello" \
> "World"
Hello World

# 支持多行函数定义
$ function test() { \
> echo "line 1"; \
> echo "line 2"; \
> }
$ test
line 1
line 2
```

### 脚本执行

```bash
# 执行脚本文件（自动跳过shebang行和注释行）
$ gobash.exe script.sh

# 脚本示例（script.sh）
#!/bin/bash
# 这是注释行
echo "Hello from script"
export VAR=value
echo $VAR
```

## 项目结构

```
gobash/
├── cmd/
│   └── gobash/         # 主程序入口
├── internal/
│   ├── lexer/          # 词法分析器
│   ├── parser/         # 语法分析器
│   ├── executor/       # 执行器
│   ├── builtin/        # 内置命令
│   └── shell/          # Shell核心逻辑
├── pkg/
│   └── platform/       # 平台相关代码
├── go.mod
└── README.md
```

## 开发状态

### ✅ 已完成功能

**核心功能**
- [x] 词法分析器（支持字符串、变量、操作符等）
- [x] 语法分析器（构建AST）
- [x] 命令执行器（外部命令、内置命令、管道、重定向）
- [x] 交互式Shell（REPL循环）

**命令支持**
- [x] 内置命令（cd, pwd, echo, exit, export, unset, env, set）
- [x] 文件操作（ls, cat, mkdir, rmdir, rm, touch, clear）
- [x] 文本处理（head, tail, wc, grep, sort, uniq, cut）
- [x] 别名管理（alias, unalias）
- [x] 命令历史（history命令，持久化存储）

**语法特性**
- [x] 管道和重定向（|, >, <, >>），支持内置命令重定向
- [x] 环境变量（单引号不展开，双引号展开变量，支持${VAR}格式）
- [x] 控制流语句（if/else, for, while）
- [x] 函数定义和调用（支持参数传递，$1, $2, $#, $@）
- [x] 多行输入支持（以`\`结尾的命令）

**脚本执行**
- [x] 脚本文件执行（支持shebang行和注释行自动跳过）
- [x] 命令字符串执行（-c参数）

**平台支持**
- [x] Windows平台优化（路径处理、环境变量）
- [x] 跨平台兼容性

**用户体验**
- [x] 增强的错误处理和提示
- [x] 命令历史持久化（~/.gobash_history）

### 🔄 计划中的功能（可选增强）

- [ ] 箭头键浏览历史（需要readline库支持）
- [ ] 命令自动补全功能
- [ ] 更多Bash特性（数组、关联数组、进程替换等）
- [ ] 作业控制（后台任务、fg, bg, jobs）
- [x] 更多内置命令（head, tail, wc, grep, sort, uniq, cut）
- [ ] 更多测试用例和文档

## 技术实现

### 架构设计

项目采用模块化设计，主要组件包括：

- **词法分析器（Lexer）**：将输入字符串分解为token序列
- **语法分析器（Parser）**：构建抽象语法树（AST）
- **执行器（Executor）**：解释执行AST，处理命令、管道、重定向
- **内置命令（Builtin）**：实现常用shell命令
- **Shell核心（Shell）**：管理REPL循环、别名、历史记录

### 关键特性实现

1. **字符串变量展开**：区分单引号和双引号，双引号内支持变量展开和转义
2. **内置命令重定向**：通过临时替换os.Stdin/Stdout/Stderr实现
3. **文件名解析**：正确处理包含点号、连字符等特殊字符的文件名
4. **脚本执行**：自动识别并跳过shebang行和注释行

## 许可证

MIT License

