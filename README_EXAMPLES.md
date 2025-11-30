# gobash 使用示例

本文档提供了 gobash 的各种使用示例。

## 目录

- [基础功能](#基础功能)
- [文本处理](#文本处理)
- [高级功能](#高级功能)
- [脚本编写](#脚本编写)

## 基础功能

### 环境变量

```bash
# 设置环境变量
export MY_VAR="hello"
echo $MY_VAR

# 单引号不展开变量
echo '$MY_VAR'  # 输出: $MY_VAR

# 双引号展开变量
echo "$MY_VAR"  # 输出: hello
```

### 命令替换

```bash
# 使用 $(command) 格式
echo "当前目录: $(pwd)"
echo "当前时间: $(date)"

# 使用 `command` 格式（反引号）
echo "用户: `whoami`"
```

### 算术展开

```bash
# 基本运算
echo $((1 + 1))        # 输出: 2
echo $((10 * 5))       # 输出: 50
echo $((100 / 4))      # 输出: 25
echo $((10 - 3))       # 输出: 7

# 变量参与运算
a=10
b=20
echo $((a + b))        # 输出: 30
```

### 条件判断

```bash
# 文件测试
if [ -f file.txt ]; then
    echo "文件存在"
fi

if [ -d /path/to/dir ]; then
    echo "目录存在"
fi

# 字符串比较
if [ "hello" = "hello" ]; then
    echo "字符串相等"
fi

# 数值比较
if [ 10 -gt 5 ]; then
    echo "10 大于 5"
fi
```

### 循环

```bash
# for 循环
for i in 1 2 3; do
    echo "数字: $i"
done

# for 循环（位置参数）
for arg; do
    echo "参数: $arg"
done

# while 循环
count=1
while [ $count -le 5 ]; do
    echo "计数: $count"
    count=$((count + 1))
done
```

### 函数

```bash
# 定义函数
function greet() {
    echo "Hello, $1!"
}

# 调用函数
greet "World"

# 带多个参数
function add() {
    echo $(( $1 + $2 ))
}
add 10 20  # 输出: 30
```

## 文本处理

### head 和 tail

```bash
# 显示文件前10行（默认）
head file.txt

# 显示文件前5行
head -n 5 file.txt

# 显示文件后10行（默认）
tail file.txt

# 显示文件后5行
tail -n 5 file.txt
```

### wc（统计）

```bash
# 统计行数
wc -l file.txt

# 统计字数
wc -w file.txt

# 统计字符数
wc -c file.txt

# 组合使用
wc -l -w -c file.txt
```

### grep（搜索）

```bash
# 基本搜索
grep "pattern" file.txt

# 忽略大小写
grep -i "pattern" file.txt

# 显示行号
grep -n "pattern" file.txt

# 只显示匹配部分
grep -o "pattern" file.txt
```

### sort（排序）

```bash
# 字母排序
sort file.txt

# 逆序
sort -r file.txt

# 数值排序
sort -n numbers.txt

# 去重
sort -u file.txt
```

### uniq（去重）

```bash
# 基本去重
uniq file.txt

# 显示计数
uniq -c file.txt

# 只显示重复行
uniq -d file.txt

# 忽略大小写
uniq -i file.txt
```

### cut（剪切）

```bash
# 按逗号分隔，提取第1和第3列
cut -d, -f1,3 data.csv

# 提取第1到第3列
cut -d, -f1-3 data.csv

# 组合使用
cut -d, -f1-2,4 data.csv
```

## 高级功能

### set 命令选项

```bash
# 显示执行的命令（xtrace）
set -x
echo "这条命令会被显示"
set +x

# 遇到错误立即退出（errexit）
set -e
false  # 脚本会在这里退出
echo "这行不会执行"

# 使用未定义变量时报错（nounset）
set -u
echo $UNDEFINED_VAR  # 会报错
```

### 管道和重定向

```bash
# 输出重定向
echo "hello" > file.txt

# 追加重定向
echo "world" >> file.txt

# 输入重定向
cat < file.txt

# 管道
echo "hello world" | grep "hello"
ls | head -n 5
```

### 别名

```bash
# 设置别名
alias ll='ls -l'
alias la='ls -a'
alias grep='grep --color'

# 显示所有别名
alias

# 取消别名
unalias ll
```

### 命令历史

```bash
# 显示历史
history

# 清除历史
history -c

# 使用箭头键浏览历史（交互式模式）
# ↑ 键：上一条命令
# ↓ 键：下一条命令
```

## 脚本编写

### 基本脚本结构

```bash
#!/usr/bin/env gobash
# 脚本注释

# 设置变量
VAR="value"

# 执行命令
echo "Hello, World!"

# 使用函数
function my_function() {
    echo "Function called"
}

my_function
```

### 错误处理

```bash
#!/usr/bin/env gobash

# 启用错误时退出
set -e

# 如果命令失败，脚本会退出
mkdir /tmp/test_dir
cd /tmp/test_dir
echo "成功进入目录"
```

### 参数处理

```bash
#!/usr/bin/env gobash

# 处理脚本参数
if [ $# -eq 0 ]; then
    echo "用法: $0 <参数>"
    exit 1
fi

echo "第一个参数: $1"
echo "参数总数: $#"
echo "所有参数: $@"
```

## 运行示例

运行示例脚本：

```bash
# 运行基础示例
gobash.exe examples/basic.sh

# 运行文本处理示例
gobash.exe examples/text_processing.sh

# 运行高级功能示例
gobash.exe examples/advanced.sh
```

## 数组和关联数组

### 数组

```bash
# 数组赋值
arr=(1 2 3 4 5)
names=("Alice" "Bob" "Charlie")

# 数组访问
echo ${arr[0]}  # 输出: 1
echo ${arr[1]}  # 输出: 2

# 数组变量展开（所有元素）
echo $arr  # 输出: 1 2 3 4 5

# 在for循环中使用数组
for i in $arr; do
    echo "数字: $i"
done
```

### 关联数组

```bash
# 声明关联数组
declare -A arr

# 关联数组赋值
arr[hello]=world
arr[foo]=bar
arr[number]=123

# 关联数组访问
echo ${arr[hello]}  # 输出: world
echo ${arr[foo]}     # 输出: bar

# 使用变量作为键
key="foo"
echo ${arr[$key]}    # 输出: bar
```

## 进程替换

```bash
# 进程替换（输入）：将命令输出作为文件读取
cat <(echo "hello world")
# 输出: hello world

# 比较两个排序后的文件
diff <(sort file1.txt) <(sort file2.txt)

# 进程替换（输出）：将命令输入作为文件写入
echo "test" >(cat)
# 注意：>(command)通常用于将输出重定向到命令的输入
```

## 更多信息

查看 [README.md](README.md) 了解完整的功能列表和安装说明。

