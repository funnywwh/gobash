#!/bin/bash
# 测试算术展开在变量赋值中的问题

echo "=== 测试算术展开在变量赋值中 ==="

echo "1. 测试基本算术展开:"
i=$((1+1))
echo "i=$((1+1)) 应该输出 i=2，实际输出: i=$i"

echo ""
echo "2. 测试带变量的算术展开:"
j=5
k=$((j+3))
echo "j=5, k=\$((j+3)) 应该输出 k=8，实际输出: k=$k"

echo ""
echo "3. 测试在引号中的算术展开:"
test1="$((2+2))"
echo "test1=\"\$((2+2))\" 应该输出 test1=4，实际输出: test1=$test1"

echo ""
echo "4. 测试不带引号的算术展开:"
test2=$((2+2))
echo "test2=\$((2+2)) 应该输出 test2=4，实际输出: test2=$test2"

echo ""
echo "5. 测试复杂算术表达式:"
result=$((10 * 5 + 2))
echo "result=\$((10 * 5 + 2)) 应该输出 result=52，实际输出: result=$result"

echo ""
echo "=== 测试完成 ==="

