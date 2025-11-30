#!/usr/bin/env gobash
# gobash 高级功能示例

echo "=== gobash 高级功能演示 ==="

# 1. set 命令选项
echo "1. set 命令选项："
echo "启用 xtrace (-x)："
set -x
echo "这条命令会被显示"
set +x

echo ""
echo "2. 函数参数传递："
function show_args() {
    echo "参数数量: $#"
    echo "参数列表: $@"
    echo "第一个参数: $1"
    echo "第二个参数: $2"
}
show_args "arg1" "arg2" "arg3"

echo ""
echo "3. for 循环位置参数："
function process_args() {
    for arg; do
        echo "  处理: $arg"
    done
}
process_args "file1" "file2" "file3"

echo ""
echo "4. 命令替换和算术展开组合："
files=$(ls examples/*.sh 2>/dev/null | wc -l)
echo "示例脚本数量: $files"

echo ""
echo "5. 条件判断组合："
if [ -f examples/advanced.sh ] && [ -r examples/advanced.sh ]; then
    echo "文件存在且可读"
fi

if [ ! -d /nonexistent ]; then
    echo "目录不存在"
fi

echo ""
echo "6. 别名："
alias ll='ls -l'
alias la='ls -a'
# 注意：别名在脚本中可能不会自动展开

echo ""
echo "=== 演示完成 ==="

