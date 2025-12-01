#!/bin/bash
# 测试 while 循环卡死问题

echo "=== 测试 while 循环 ==="

echo "1. 测试基本 while 循环（应该循环3次）:"
i=0
count=0
while [[ $i -lt 3 ]]; do
    echo "  循环 $count: i=$i"
    i=$((i+1))
    count=$((count+1))
    if [[ $count -gt 10 ]]; then
        echo "  错误: 循环超过10次，可能卡死！"
        break
    fi
done
echo "循环结束，i=$i, count=$count"

echo ""
echo "2. 测试 while 循环中的变量更新:"
j=0
while [[ $j -lt 5 ]]; do
    echo "  j=$j"
    j=$((j+1))
    if [[ $j -gt 10 ]]; then
        echo "  错误: j 超过10，可能卡死！"
        break
    fi
done
echo "循环结束，j=$j"

echo ""
echo "3. 测试 while 循环条件（应该立即退出）:"
k=10
while [[ $k -lt 5 ]]; do
    echo "  这行不应该执行: k=$k"
    k=$((k+1))
done
echo "循环结束，k=$k (应该还是10)"

echo ""
echo "=== 测试完成 ==="

