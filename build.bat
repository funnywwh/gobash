@echo off
REM gobash Windows 构建脚本

setlocal enabledelayedexpansion

set PLATFORM=windows
set OUTPUT_DIR=.
set CLEAN=false
set VERSION=

REM 解析命令行参数
:parse_args
if "%~1"=="" goto :build
if /i "%~1"=="-p" set PLATFORM=%~2 & shift & shift & goto :parse_args
if /i "%~1"=="--platform" set PLATFORM=%~2 & shift & shift & goto :parse_args
if /i "%~1"=="-o" set OUTPUT_DIR=%~2 & shift & shift & goto :parse_args
if /i "%~1"=="--output" set OUTPUT_DIR=%~2 & shift & shift & goto :parse_args
if /i "%~1"=="-c" set CLEAN=true & shift & goto :parse_args
if /i "%~1"=="--clean" set CLEAN=true & shift & goto :parse_args
if /i "%~1"=="-v" set VERSION=%~2 & shift & shift & goto :parse_args
if /i "%~1"=="--version" set VERSION=%~2 & shift & shift & goto :parse_args
if /i "%~1"=="-h" goto :show_help
if /i "%~1"=="--help" goto :show_help
echo 错误: 未知选项 %~1
goto :show_help

:show_help
echo gobash Windows 构建脚本
echo.
echo 用法: %~nx0 [选项]
echo.
echo 选项:
echo     -p, --platform PLATFORM    目标平台 (windows, linux, darwin)
echo     -o, --output DIR            输出目录 (默认: 当前目录)
echo     -c, --clean                 构建前清理旧的构建文件
echo     -v, --version VERSION       设置版本号
echo     -h, --help                  显示此帮助信息
echo.
echo 示例:
echo     %~nx0                       构建 Windows 版本
echo     %~nx0 -p linux              交叉编译 Linux 版本
echo     %~nx0 -o .\dist             构建到 dist 目录
exit /b 0

:build
echo === gobash 构建脚本 ===

REM 检查 Go 环境
where go >nul 2>&1
if errorlevel 1 (
    echo 错误: 未找到 Go 编译器，请先安装 Go
    exit /b 1
)

go version

REM 清理
if "%CLEAN%"=="true" (
    echo 清理旧的构建文件...
    if exist gobash.exe del /f /q gobash.exe
    if exist gobash del /f /q gobash
    if exist "%OUTPUT_DIR%\gobash.exe" del /f /q "%OUTPUT_DIR%\gobash.exe"
    if exist "%OUTPUT_DIR%\gobash" del /f /q "%OUTPUT_DIR%\gobash"
    echo 清理完成
)

REM 设置输出路径
set OUTPUT_NAME=gobash.exe
if not "%PLATFORM%"=="windows" set OUTPUT_NAME=gobash

if not "%OUTPUT_DIR%"=="." (
    if not exist "%OUTPUT_DIR%" mkdir "%OUTPUT_DIR%"
    set OUTPUT_PATH=%OUTPUT_DIR%\!OUTPUT_NAME!
) else (
    set OUTPUT_PATH=!OUTPUT_NAME!
)

REM 设置构建参数
set LDFLAGS=
if not "%VERSION%"=="" set LDFLAGS=-ldflags "-X main.version=%VERSION%"

REM 构建
echo 正在构建 %PLATFORM% 平台...

if "%PLATFORM%"=="windows" (
    go build %LDFLAGS% -o "%OUTPUT_PATH%" ./cmd/gobash
) else (
    REM 交叉编译
    if "%PLATFORM%"=="linux" (
        set GOOS=linux
        set GOARCH=amd64
    ) else if "%PLATFORM%"=="darwin" (
        set GOOS=darwin
        set GOARCH=amd64
    ) else (
        echo 错误: 不支持的平台 '%PLATFORM%'
        echo 支持的平台: windows, linux, darwin
        exit /b 1
    )
    set CGO_ENABLED=0
    go build %LDFLAGS% -o "%OUTPUT_PATH%" ./cmd/gobash
)

if exist "%OUTPUT_PATH%" (
    echo ✓ 构建成功: %OUTPUT_PATH%
    dir "%OUTPUT_PATH%" | findstr /C:"%OUTPUT_NAME%"
) else (
    echo ✗ 构建失败: %OUTPUT_PATH%
    exit /b 1
)

echo === 构建完成 ===
endlocal

