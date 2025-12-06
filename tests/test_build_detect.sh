#!/bin/bash
echo "开始测试 detect_platform"

detect_platform() {
    case "$(uname -s)" in
        Linux*)
            echo "linux"
            ;;
        Darwin*)
            echo "darwin"
            ;;
        MINGW*|MSYS*|CYGWIN*)
            echo "windows"
            ;;
        *)
            echo "unknown"
            ;;
    esac
}

echo "调用 detect_platform"
result=$(detect_platform)
echo "结果: $result"
echo "测试完成"





