#!/bin/bash
# 数组功能测试脚本

echo "=== 数组功能测试 ==="

# 测试数组赋值
arr=(1 2 3 4 5)
echo "数组赋值: arr=(1 2 3 4 5)"

# 测试数组访问
echo "第一个元素: ${arr[0]}"
echo "第二个元素: ${arr[1]}"
echo "第三个元素: ${arr[2]}"

# 测试数组变量展开（所有元素）
echo "所有元素: $arr"

# 测试在for循环中使用数组
echo "遍历数组:"
for i in "${arr[@]}"; do
    echo "  $i"
done

# 测试字符串数组
names=("Alice" "Bob" "Charlie")
echo "字符串数组: names=(\"Alice\" \"Bob\" \"Charlie\")"
echo "第一个名字: ${names[0]}"
echo "第二个名字: ${names[1]}"

echo "=== 测试完成 ==="

