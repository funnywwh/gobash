#!/bin/bash
# 高级功能综合示例脚本
# 展示数组、关联数组、进程替换等新功能

echo "=== gobash 高级功能演示 ==="
echo ""

# ========== 数组功能 ==========
echo "--- 数组功能 ---"
arr=(1 2 3 4 5)
echo "数组赋值: arr=(1 2 3 4 5)"
echo "第一个元素: ${arr[0]}"
echo "第二个元素: ${arr[1]}"
echo "所有元素: $arr"

# 字符串数组
names=("Alice" "Bob" "Charlie")
echo "字符串数组: names=(\"Alice\" \"Bob\" \"Charlie\")"
echo "第一个名字: ${names[0]}"

# 在for循环中使用数组
echo "遍历数组:"
for i in $arr; do
    echo "  元素: $i"
done
echo ""

# ========== 关联数组功能 ==========
echo "--- 关联数组功能 ---"
declare -A assoc
assoc[hello]=world
assoc[foo]=bar
assoc[number]=123
echo "关联数组赋值:"
echo "  assoc[hello]=world"
echo "  assoc[foo]=bar"
echo "  assoc[number]=123"

echo "关联数组访问:"
echo "  assoc[hello] = ${assoc[hello]}"
echo "  assoc[foo] = ${assoc[foo]}"
echo "  assoc[number] = ${assoc[number]}"

# 使用变量作为键
key="foo"
echo "使用变量作为键: key=\"foo\""
echo "  assoc[\$key] = ${assoc[$key]}"
echo ""

# ========== 进程替换功能 ==========
echo "--- 进程替换功能 ---"
echo "进程替换（输入）: cat <(echo \"hello world\")"
cat <(echo "hello world")
echo ""

# 创建测试文件用于进程替换演示
echo "test1" > /tmp/gobash_test1.txt
echo "test2" > /tmp/gobash_test2.txt
echo "test3" > /tmp/gobash_test2.txt

echo "使用进程替换比较两个排序后的文件:"
echo "diff <(sort /tmp/gobash_test1.txt) <(sort /tmp/gobash_test2.txt)"
diff <(sort /tmp/gobash_test1.txt) <(sort /tmp/gobash_test2.txt) || echo "文件不同（这是正常的）"
echo ""

# ========== 组合使用示例 ==========
echo "--- 组合使用示例 ---"
echo "在命令替换中使用数组:"
echo "数组元素数量: $(echo $arr | wc -w)"
echo ""

echo "在算术展开中使用数组元素:"
echo "第一个元素加10: $(( ${arr[0]} + 10 ))"
echo ""

echo "=== 演示完成 ==="

