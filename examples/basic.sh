#!/usr/bin/env gobash
# gobash 基础功能示例脚本

echo "=== gobash 基础功能演示 ==="

# 1. 基本命令
echo "1. 基本命令："
pwd
echo "当前目录：$(pwd)"

# 2. 环境变量
echo ""
echo "2. 环境变量："
export DEMO_VAR="Hello from gobash"
echo "DEMO_VAR=$DEMO_VAR"
echo "单引号不展开: '$DEMO_VAR'"
echo "双引号展开: \"$DEMO_VAR\""

# 3. 命令替换
echo ""
echo "3. 命令替换："
echo "当前时间: $(date)"
echo "当前用户: $(whoami)"

# 4. 算术展开
echo ""
echo "4. 算术展开："
echo "1 + 1 = $((1 + 1))"
echo "10 * 5 = $((10 * 5))"
echo "100 / 4 = $((100 / 4))"

# 5. 条件判断
echo ""
echo "5. 条件判断："
if [ -f examples/basic.sh ]; then
    echo "basic.sh 文件存在"
fi

if [ -d examples ]; then
    echo "examples 目录存在"
fi

# 6. 循环
echo ""
echo "6. 循环："
echo "for 循环："
for i in 1 2 3; do
    echo "  循环 $i"
done

echo "while 循环："
count=1
while [ $count -le 3 ]; do
    echo "  计数: $count"
    count=$((count + 1))
done

# 7. 函数
echo ""
echo "7. 函数："
function greet() {
    echo "  你好, $1!"
}
greet "gobash用户"

# 8. 管道和重定向
echo ""
echo "8. 管道和重定向："
echo "test line 1" > /tmp/gobash_test.txt
echo "test line 2" >> /tmp/gobash_test.txt
cat /tmp/gobash_test.txt
rm /tmp/gobash_test.txt

echo ""
echo "=== 演示完成 ==="

