#!/bin/bash
# 简化版 build.sh 测试脚本

echo "开始执行"

# 测试 while 循环
echo "测试 while 循环解析参数"
i=0
while [[ $# -gt 0 ]]; do
    echo "  参数: $1"
    shift
    i=$((i+1))
    if [[ $i -gt 10 ]]; then
        echo "  错误: 循环超过10次，可能卡死！"
        break
    fi
done
echo "while 循环结束，参数个数: $#"

# 测试 uname 命令
echo "测试 uname -s"
uname_result=$(uname -s)
echo "uname 结果: $uname_result"

# 测试 command 命令
echo "测试 command -v go"
if command -v go &> /dev/null; then
    echo "go 命令找到"
else
    echo "go 命令未找到（这是正常的，如果 go 未安装）"
fi

echo "脚本执行完成"





