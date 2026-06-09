#!/bin/bash
# =============================================================================
# 跨平台编译 submill + mihomo
# 输出 x86_64 和 aarch64 二进制到 build/ 目录
# =============================================================================
set -e

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJECT_DIR="$(dirname "$SCRIPT_DIR")"
BUILD_DIR="$PROJECT_DIR/build"

RED='\033[0;31m'; GREEN='\033[0;32m'; NC='\033[0m'

# 检查 Go
if ! command -v go &>/dev/null; then
    echo -e "${RED}错误: 未找到 Go，请先运行: bash scripts/setup.sh${NC}"
    exit 1
fi

TARGETS=("linux/amd64" "linux/arm64")
mkdir -p "$BUILD_DIR"

echo "============================================"
echo "  跨平台编译"
echo "============================================"
echo ""

# ---- submill ----
echo ">>> 编译 submill..."
cd "$PROJECT_DIR"
[ -d vendor ] || { echo "先运行 bash scripts/setup.sh"; exit 1; }

for t in "${TARGETS[@]}"; do
    GOOS="${t%/*}"; GOARCH="${t#*/}"
    out="$BUILD_DIR/submill_${GOOS}_${GOARCH}"
    echo "    $t -> $out"
    CGO_ENABLED=0 GOOS="$GOOS" GOARCH="$GOARCH" \
        go build -mod=vendor -trimpath -ldflags="-s -w" -o "$out" .
done

# ---- mihomo ----
echo ">>> 编译 mihomo..."
cd "$PROJECT_DIR/mihomo"
[ -d vendor ] || { echo "先运行 bash scripts/setup.sh"; exit 1; }

for t in "${TARGETS[@]}"; do
    GOOS="${t%/*}"; GOARCH="${t#*/}"
    out="$BUILD_DIR/mihomo_${GOOS}_${GOARCH}"
    echo "    $t -> $out"
    CGO_ENABLED=0 GOOS="$GOOS" GOARCH="$GOARCH" \
        go build -mod=vendor -trimpath -ldflags="-s -w" -o "$out" github.com/metacubex/mihomo
done

echo ""
echo -e "${GREEN}============================================${NC}"
echo -e "${GREEN}  编译完成${NC}"
ls -lh "$BUILD_DIR/"
echo -e "${GREEN}============================================${NC}"
