# gobash - Bash兼容跨平台Shell

一个使用Go语言开发的兼容Bash语法的跨平台Shell程序，优先支持Windows平台。

## 功能特性

- ✅ Bash脚本执行器
- ✅ 交互式Shell（REPL）
- ✅ 基础命令支持（cd, pwd, echo, exit等）
- ✅ 管道和重定向（|, >, <, >>）
- ✅ 环境变量支持
- ✅ 控制流语句（if/else, for, while）
- ✅ 函数定义（基础支持）

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
- `set` - 显示所有变量

### 控制
- `exit [退出码]` - 退出shell
- `alias [name=value]` - 设置或显示命令别名
- `unalias [name]` - 取消设置别名
- `history` - 显示命令历史
- `history -c` - 清除命令历史

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
$ echo "test" > output.txt
$ cat output.txt
test

$ echo "append" >> output.txt
$ cat output.txt
test
append
```

### 环境变量

```bash
$ export MYVAR=hello
$ echo $MYVAR
hello
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

当前版本实现了基础的Bash兼容功能，包括：

- [x] 词法分析器
- [x] 语法分析器
- [x] 命令执行
- [x] 管道和重定向
- [x] 内置命令（cd, pwd, echo, ls, cat, mkdir, rm等）
- [x] 控制流语句（if/else, for, while）
- [x] 交互式Shell
- [x] 多行输入支持（以\结尾）
- [x] 错误处理和提示
- [x] Windows平台优化
- [x] 命令别名（alias/unalias）
- [x] 函数定义和调用
- [x] 命令历史（history命令，历史记录持久化）
- [ ] 箭头键浏览历史（可选增强）
- [ ] 自动补全

## 许可证

MIT License

