#!/bin/bash
echo "第1行: 开始执行"

echo "第2行: 测试变量"
VAR="test"
echo "VAR=$VAR"

echo "第3行: 测试函数定义"
test_func() {
    echo "函数内部"
}
echo "函数定义完成"

echo "第4行: 测试 while 循环"
i=0
while [[ $i -lt 3 ]]; do
    echo "  循环: i=$i"
    i=$((i+1))
    if [[ $i -gt 10 ]]; then
        echo "  错误: 卡死！"
        break
    fi
done
echo "while 循环结束"

echo "第5行: 测试 case 语句"
case "test" in
    test)
        echo "  case 匹配"
        ;;
    *)
        echo "  case 不匹配"
        ;;
esac
echo "case 语句结束"

echo "第6行: 脚本执行完成"




