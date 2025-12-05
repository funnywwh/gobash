#!/bin/bash
# gobash 构建脚本
# 支持 Linux、macOS 和 Windows 平台构建

set -e

# 颜色输出
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# 默认构建选项
PLATFORM=""
OUTPUT_DIR="."
CLEAN=false
CROSS_COMPILE=false
VERSION=""

# 显示帮助信息
show_help() {
    cat << EOF
gobash 构建脚本

用法: $0 [选项]

选项:
    -p, --platform PLATFORM    目标平台 (linux, darwin, windows)
    -o, --output DIR            输出目录 (默认: 当前目录)
    -c, --clean                 构建前清理旧的构建文件
    -x, --cross                 交叉编译所有平台
    -v, --version VERSION       设置版本号
    -h, --help                  显示此帮助信息

示例:
    $0                          # 构建当前平台
    $0 -p windows               # 构建 Windows 版本
    $0 -p linux -o ./dist      # 构建 Linux 版本到 dist 目录
    $0 -x                       # 交叉编译所有平台
    $0 -c                       # 清理后构建
EOF
}

# 解析命令行参数
while [[ $# -gt 0 ]]; do
    case $1 in
        -p|--platform)
            PLATFORM="$2"
            shift 2
            ;;
        -o|--output)
            OUTPUT_DIR="$2"
            shift 2
            ;;
        -c|--clean)
            CLEAN=true
            shift
            ;;
        -x|--cross)
            CROSS_COMPILE=true
            shift
            ;;
        -v|--version)
            VERSION="$2"
            shift 2
            ;;
        -h|--help)
            show_help
            exit 0
            ;;
        *)
            echo -e "${RED}错误: 未知选项 $1${NC}"
            show_help
            exit 1
            ;;
    esac
done

# 检测当前平台
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

# 构建函数
build() {
    local platform=$1
    local output_name="gobash"
    local ldflags=""
    
    # Windows 平台使用 .exe 扩展名
    if [[ "$platform" == "windows" ]]; then
        output_name="gobash.exe"
    fi
    
    # 设置版本信息
    if [[ -n "$VERSION" ]]; then
        ldflags="-X main.version=$VERSION"
    fi
    
    # 设置输出路径
    local output_path="$OUTPUT_DIR/$output_name"
    if [[ "$OUTPUT_DIR" != "." ]]; then
        mkdir -p "$OUTPUT_DIR"
    fi
    
    echo -e "${GREEN}正在构建 $platform 平台...${NC}"
    
    if [[ "$platform" == "$(detect_platform)" ]]; then
        # 本地构建
        go build -ldflags "$ldflags" -o "$output_path" ./cmd/gobash
    else
        # 交叉编译
        GOOS=$(echo $platform | sed 's/darwin/macos/')
        if [[ "$platform" == "windows" ]]; then
            GOOS=windows
        elif [[ "$platform" == "darwin" ]]; then
            GOOS=darwin
        else
            GOOS=linux
        fi
        
        GOARCH=amd64
        if [[ "$platform" == "darwin" ]]; then
            # macOS 可能需要 arm64
            GOARCH=amd64
        fi
        
        CGO_ENABLED=0 GOOS=$GOOS GOARCH=$GOARCH go build -ldflags "$ldflags" -o "$output_path" ./cmd/gobash
    fi
    
    if [[ -f "$output_path" ]]; then
        echo -e "${GREEN}✓ 构建成功: $output_path${NC}"
        ls -lh "$output_path" 2>/dev/null || echo "文件大小: $(stat -f%z "$output_path" 2>/dev/null || stat -c%s "$output_path" 2>/dev/null) 字节"
    else
        echo -e "${RED}✗ 构建失败: $output_path${NC}"
        return 1
    fi
}

# 清理函数
clean() {
    echo -e "${YELLOW}清理旧的构建文件...${NC}"
    rm -f gobash gobash.exe
    if [[ -d "$OUTPUT_DIR" ]] && [[ "$OUTPUT_DIR" != "." ]]; then
        rm -f "$OUTPUT_DIR/gobash" "$OUTPUT_DIR/gobash.exe"
    fi
    echo -e "${GREEN}清理完成${NC}"
}

# 主函数
main() {
    echo -e "${GREEN}=== gobash 构建脚本 ===${NC}"
    
    # 检查 Go 环境
    if ! command -v go &> /dev/null; then
        echo -e "${RED}错误: 未找到 Go 编译器，请先安装 Go${NC}"
        exit 1
    fi
    
    echo -e "Go 版本: $(go version)"
    
    # 清理
    if [[ "$CLEAN" == true ]]; then
        clean
    fi
    
    # 交叉编译所有平台
    if [[ "$CROSS_COMPILE" == true ]]; then
        echo -e "${YELLOW}交叉编译所有平台...${NC}"
        build "linux"
        build "darwin"
        build "windows"
        echo -e "${GREEN}=== 所有平台构建完成 ===${NC}"
        exit 0
    fi
    
    # 单平台构建
    if [[ -z "$PLATFORM" ]]; then
        PLATFORM=$(detect_platform)
        echo -e "检测到平台: $PLATFORM"
    fi
    
    case "$PLATFORM" in
        linux|darwin|windows)
            build "$PLATFORM"
            ;;
        *)
            echo -e "${RED}错误: 不支持的平台 '$PLATFORM'${NC}"
            echo -e "支持的平台: linux, darwin, windows"
            exit 1
            ;;
    esac
    
    echo -e "${GREEN}=== 构建完成 ===${NC}"
}

# 运行主函数
main



