#!/bin/bash
# 带超时的测试脚本 - 用于验证 while 循环卡死问题
echo "开始测试 - 如果脚本在5秒内没有完成，说明卡死了"
timeout 5 gobash.exe test_while_basic.sh 2>&1
if [ $? -eq 124 ]; then
    echo "脚本超时 - 确认卡死！"
    exit 1
elif [ $? -eq 0 ]; then
    echo "脚本正常完成"
    exit 0
else
    echo "脚本执行出错"
    exit 1
fi


