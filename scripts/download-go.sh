#!/bin/bash
# =============================================================================
# 下载 Go 安装包到 assets/go/（离线安装用）
# 用法: bash scripts/download-go.sh
# =============================================================================
set -e

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJECT_DIR="$(dirname "$SCRIPT_DIR")"
GO_DIR="$PROJECT_DIR/assets/go"
GO_VERSION="1.25.0"

# 镜像源（按优先级尝试）
MIRRORS=(
    "https://mirrors.aliyun.com/golang"
    "https://go.dev/dl"
)

mkdir -p "$GO_DIR"

TARGETS=(
    "linux-amd64"
    "linux-arm64"
    "darwin-amd64"
    "darwin-arm64"
)

echo ">>> 下载 Go ${GO_VERSION} 安装包..."
echo ""

for target in "${TARGETS[@]}"; do
    file="go${GO_VERSION}.${target}.tar.gz"
    dest="$GO_DIR/$file"

    if [ -f "$dest" ]; then
        echo "  $file  已存在，跳过"
        continue
    fi

    echo "  $file  下载中..."
    downloaded=false
    for mirror in "${MIRRORS[@]}"; do
        url="$mirror/$file"
        if curl -fsSL --connect-timeout 10 --max-time 300 "$url" -o "$dest" 2>/dev/null; then
            downloaded=true
            break
        fi
    done

    if [ "$downloaded" = false ]; then
        echo "    错误: 所有镜像均下载失败"
        rm -f "$dest"
        exit 1
    fi
done

echo ""
echo "============================================"
echo "  下载完成"
ls -lh "$GO_DIR/"
echo "============================================"
