@echo off
setlocal EnableDelayedExpansion

:: 版本号
set VERSION=1.0.0
set BINARY_NAME=StellarServer

:: 创建输出目录
if not exist output mkdir output
if not exist output\templates\email mkdir output\templates\email

:: 检查是否安装了 UPX
where upx >nul 2>nul
if %ERRORLEVEL% NEQ 0 (
    echo 错误：请先安装 UPX！
    echo 下载地址：https://github.com/upx/upx/releases
    exit /b 1
)

set GOOS=linux
set GOARCH=amd64
go build -trimpath -ldflags "-s -w" -o "output/%BINARY_NAME%" ./cmd/server/main.go
upx --best --lzma "output/%BINARY_NAME%"

:: 复制邮件模板
copy templates\email\*.html output\templates\email\

echo.
echo 编译完成！
echo 输出文件在 output 目录中

endlocal