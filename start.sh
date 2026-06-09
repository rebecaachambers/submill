#!/bin/bash
# =============================================================================
# 启动 submill + mihomo
# 首次使用前请运行: bash scripts/setup.sh
# =============================================================================
set -e

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
cd "$SCRIPT_DIR"

SUBS_BIN="$SCRIPT_DIR/submill"
MIHOMO_BIN="$SCRIPT_DIR/mihomo/mihomo"
MIHOMO_CONF="$SCRIPT_DIR/config/config.yaml"
MIHOMO_CONF_DIR="$SCRIPT_DIR/config"

check() {
    if [ ! -f "$SUBS_BIN" ]; then
        echo "[ERROR] 未找到 submill，请先运行: bash scripts/setup.sh"
        exit 1
    fi
    if [ ! -f "$MIHOMO_BIN" ]; then
        echo "[ERROR] 未找到 mihomo，请先运行: bash scripts/setup.sh"
        exit 1
    fi
    if [ ! -f "$MIHOMO_CONF" ]; then
        echo "[ERROR] 未找到 mihomo 配置: $MIHOMO_CONF"
        exit 1
    fi
}

cleanup() {
    echo ""
    echo ">>> 正在关闭所有服务..."
    [ -n "$SUBS_PID" ] && kill "$SUBS_PID" 2>/dev/null && wait "$SUBS_PID" 2>/dev/null
    [ -n "$MIHOMO_PID" ] && kill "$MIHOMO_PID" 2>/dev/null && wait "$MIHOMO_PID" 2>/dev/null
    echo ">>> 全部服务已停止"
}
trap cleanup EXIT INT TERM

check

echo "============================================"
echo "  启动 submill ..."
echo "============================================"
"$SUBS_BIN" "$@" &
SUBS_PID=$!

echo ">>> 等待 submill 就绪..."
for i in $(seq 1 60); do
    if curl -s http://127.0.0.1:8199/sub/ > /dev/null 2>&1; then
        echo ">>> submill 已就绪 (PID=$SUBS_PID)"
        break
    fi
    if ! kill -0 "$SUBS_PID" 2>/dev/null; then
        echo "[ERROR] submill 启动失败"
        exit 1
    fi
    sleep 2
done

echo ">>> 启动 mihomo..."
"$MIHOMO_BIN" -d "$MIHOMO_CONF_DIR" &
MIHOMO_PID=$!

echo ""
echo "============================================"
echo "  全部服务已启动:"
echo "  submill : PID=$SUBS_PID   http://127.0.0.1:8199"
echo "  mihomo     : PID=$MIHOMO_PID  mixed-port=7890"
echo "============================================"
echo "  按 Ctrl+C 停止"
echo ""

# macOS bash 3.x 不支持 wait -n，用循环替代
while kill -0 "$SUBS_PID" 2>/dev/null && kill -0 "$MIHOMO_PID" 2>/dev/null; do
    sleep 2
done
