#!/bin/bash
# 测试 case 语句（build.sh 中使用了 case）

echo "=== 测试 case 语句 ==="

test_case() {
    case $1 in
        --platform)
            echo "匹配到 --platform"
            ;;
        --win)
            echo "匹配到 --win"
            ;;
        --mac)
            echo "匹配到 --mac"
            ;;
        *)
            echo "未知选项: $1"
            ;;
    esac
}

echo "1. 测试 case 语句:"
test_case "--platform"
test_case "--win"
test_case "--mac"
test_case "--unknown"

echo ""
echo "2. 测试 case 在 while 循环中:"
args=("--platform" "linux/amd64" "--win")
i=0
while [[ $i -lt ${#args[@]} ]]; do
    case ${args[$i]} in
        --platform)
            echo "  找到 --platform，下一个参数: ${args[$((i+1))]}"
            i=$((i+2))
            ;;
        --win)
            echo "  找到 --win"
            i=$((i+1))
            ;;
        *)
            echo "  未知: ${args[$i]}"
            i=$((i+1))
            ;;
    esac
    
    # 防止无限循环
    if [[ $i -gt 10 ]]; then
        echo "  错误: 循环超过10次，可能卡死！"
        break
    fi
done

echo ""
echo "=== 测试完成 ==="

