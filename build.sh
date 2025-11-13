#!/bin/bash

# 本地构建脚本 - 用于构建 Linux 二进制文件

set -e

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# 打印带颜色的信息
print_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

# 检查 Go 是否安装
if ! command -v go &> /dev/null; then
    print_error "Go 未安装，请先安装 Go"
    exit 1
fi

print_info "Go 版本: $(go version)"

# 创建输出目录
OUTPUT_DIR="dist"
rm -rf "$OUTPUT_DIR"
mkdir -p "$OUTPUT_DIR"

# 版本信息（可以从 git tag 获取）
VERSION=${VERSION:-"dev"}
if git describe --tags --exact-match 2>/dev/null; then
    VERSION=$(git describe --tags --exact-match)
fi
print_info "构建版本: $VERSION"

# 构建函数
build_binary() {
    local os=$1
    local arch=$2
    local arm=$3
    local output_name=$4

    print_info "构建 $output_name..."

    env GOOS="$os" GOARCH="$arch" GOARM="$arm" \
        go build -ldflags="-s -w -X main.Version=$VERSION" \
        -o "$OUTPUT_DIR/$output_name" .

    if [ $? -eq 0 ]; then
        chmod +x "$OUTPUT_DIR/$output_name"
        print_info "✓ $output_name 构建成功"
    else
        print_error "✗ $output_name 构建失败"
        return 1
    fi
}

# 开始构建
print_info "开始构建 Linux 二进制文件..."

# Linux amd64 (最常用)
build_binary "linux" "amd64" "" "fl_limit-linux-amd64"

# Linux arm64
build_binary "linux" "arm64" "" "fl_limit-linux-arm64"

# Linux 386
build_binary "linux" "386" "" "fl_limit-linux-386"

# Linux arm v7 (树莓派等)
build_binary "linux" "arm" "7" "fl_limit-linux-armv7"

# 压缩文件
print_info "压缩二进制文件..."
cd "$OUTPUT_DIR"
for file in fl_limit-*; do
    if [ -f "$file" ]; then
        tar czf "${file}.tar.gz" "$file"
        print_info "✓ 已压缩: ${file}.tar.gz"
    fi
done

# 生成 SHA256 校验和
print_info "生成 SHA256 校验和..."
sha256sum *.tar.gz > SHA256SUMS
print_info "✓ SHA256SUMS 已生成"

# 显示构建结果
echo ""
print_info "构建完成！文件列表："
ls -lah *.tar.gz
echo ""
print_info "SHA256 校验和："
cat SHA256SUMS

# 返回原目录
cd ..

print_info "所有文件已保存到 $OUTPUT_DIR 目录"
print_info "要测试本地构建，运行: ./$OUTPUT_DIR/fl_limit-linux-amd64 -config config.yaml"