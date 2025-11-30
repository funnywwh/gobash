#!/usr/bin/env gobash
# gobash 文本处理命令示例

echo "=== 文本处理命令演示 ==="

# 创建测试文件
cat > /tmp/test_data.txt <<EOF
apple
banana
cherry
date
elderberry
EOF

echo "1. head 命令（显示前3行）："
head -n 3 /tmp/test_data.txt

echo ""
echo "2. tail 命令（显示后2行）："
tail -n 2 /tmp/test_data.txt

echo ""
echo "3. wc 命令（统计）："
wc -l /tmp/test_data.txt
wc -w /tmp/test_data.txt

echo ""
echo "4. grep 命令（搜索）："
grep "a" /tmp/test_data.txt
echo "带行号："
grep -n "a" /tmp/test_data.txt

echo ""
echo "5. sort 命令（排序）："
sort /tmp/test_data.txt
echo "逆序："
sort -r /tmp/test_data.txt

echo ""
echo "6. uniq 命令（去重）："
echo -e "apple\napple\nbanana\nbanana\ncherry" | uniq
echo "带计数："
echo -e "apple\napple\nbanana\nbanana\ncherry" | uniq -c

echo ""
echo "7. cut 命令（剪切字段）："
echo "name,age,city" > /tmp/csv.txt
echo "Alice,25,Beijing" >> /tmp/csv.txt
echo "Bob,30,Shanghai" >> /tmp/csv.txt
cut -d, -f1,3 /tmp/csv.txt

# 清理
rm -f /tmp/test_data.txt /tmp/csv.txt

echo ""
echo "=== 演示完成 ==="

