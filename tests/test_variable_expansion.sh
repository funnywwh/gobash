#!/bin/bash
# 测试变量展开的各种情况

echo "=== 测试变量展开 ==="

echo "1. 测试基本变量赋值:"
VAR1="hello"
echo "VAR1=$VAR1"

echo ""
echo "2. 测试算术展开赋值:"
VAR2=$((1+1))
echo "VAR2=\$((1+1)) 应该输出 VAR2=2，实际输出: VAR2=$VAR2"

echo ""
echo "3. 测试变量在算术表达式中:"
A=5
B=3
C=$((A+B))
echo "A=5, B=3, C=\$((A+B)) 应该输出 C=8，实际输出: C=$C"

echo ""
echo "4. 测试变量在条件判断中:"
X=10
if [[ $X -gt 5 ]]; then
    echo "X=$X 大于 5 (正确)"
else
    echo "X=$X 不大于 5 (错误)"
fi

echo ""
echo "5. 测试变量在 while 循环中:"
Y=0
while [[ $Y -lt 3 ]]; do
    echo "  Y=$Y"
    Y=$((Y+1))
    if [[ $Y -gt 10 ]]; then
        echo "  错误: Y 超过10，可能卡死！"
        break
    fi
done
echo "循环结束，Y=$Y"

echo ""
echo "6. 测试位置参数 \$#:"
echo "当前参数个数: \$#=$#"

echo ""
echo "=== 测试完成 ==="

