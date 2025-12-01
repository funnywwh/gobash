#!/bin/bash
# 模拟 build.sh 脚本的关键部分，用于测试

set -e

echo "开始构建测试..."

# 解析命令行参数（模拟 build.sh 的参数解析）
PLATFORM=""
BUILD_WINDOWS=false
BUILD_MAC=false

# 模拟 while 循环解析参数
while [[ $# -gt 0 ]]; do
    case $1 in
        --platform)
            PLATFORM="$2"
            shift 2
            ;;
        --win)
            BUILD_WINDOWS=true
            shift
            ;;
        --mac)
            BUILD_MAC=true
            shift
            ;;
        *)
            echo "未知选项: $1"
            exit 1
            ;;
    esac
done

echo "参数解析完成: PLATFORM=$PLATFORM, BUILD_WINDOWS=$BUILD_WINDOWS, BUILD_MAC=$BUILD_MAC"

# 测试条件判断
if [[ -z "$PLATFORM" ]] || [[ "$PLATFORM" == "linux"* ]]; then
    echo "Linux 平台构建"
fi

# 测试变量赋值和算术展开
counter=0
max_count=3

while [[ $counter -lt $max_count ]]; do
    echo "构建步骤 $counter"
    counter=$((counter+1))
    
    # 防止无限循环
    if [[ $counter -gt 10 ]]; then
        echo "错误: 循环超过10次，可能卡死！"
        break
    fi
done

echo "构建完成！"

