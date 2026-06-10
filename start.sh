#!/bin/bash
# =============================================================================
# Start SubMill + Worker + Mihomo (foreground)
# =============================================================================
set -e

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
cd "$SCRIPT_DIR"

SUBS_BIN="$SCRIPT_DIR/submill"
WORKER_BIN="$SCRIPT_DIR/scripts/watch-submill"
MIHOMO_BIN="$SCRIPT_DIR/mihomo/mihomo"
MIHOMO_CONF="$SCRIPT_DIR/config"

check() {
    if [ ! -f "$SUBS_BIN" ]; then
        echo "[ERROR] submill not found, run: bash scripts/setup.sh"
        exit 1
    fi
    if [ ! -f "$MIHOMO_BIN" ]; then
        echo "[ERROR] mihomo not found, run: bash scripts/setup.sh"
        exit 1
    fi
    if [ ! -f "$MIHOMO_CONF/config.yaml" ]; then
        echo "[ERROR] mihomo config not found: $MIHOMO_CONF/config.yaml"
        exit 1
    fi
}

cleanup() {
    echo ""
    echo ">>> Stopping all services..."
    [ -n "$WORKER_PID" ] && kill "$WORKER_PID" 2>/dev/null && wait "$WORKER_PID" 2>/dev/null
    [ -n "$MIHOMO_PID" ] && kill "$MIHOMO_PID" 2>/dev/null && wait "$MIHOMO_PID" 2>/dev/null
    [ -n "$SUBS_PID" ] && kill "$SUBS_PID" 2>/dev/null && wait "$SUBS_PID" 2>/dev/null
    echo ">>> All services stopped."
}
trap cleanup EXIT INT TERM

check

# Create directories
mkdir -p "$SCRIPT_DIR/output" "$SCRIPT_DIR/mihomo/nodes"

echo "============================================"
echo "  Starting SubMill ..."
echo "============================================"
"$SUBS_BIN" "$@" &
SUBS_PID=$!

echo ">>> Waiting for SubMill..."
for i in $(seq 1 60); do
    if curl -s http://127.0.0.1:8199/sub/ > /dev/null 2>&1; then
        echo ">>> SubMill ready (PID=$SUBS_PID)"
        break
    fi
    if ! kill -0 "$SUBS_PID" 2>/dev/null; then
        echo "[ERROR] SubMill failed to start"
        exit 1
    fi
    sleep 2
done

echo ">>> Starting Worker..."
SUBS_OUTPUT="$SCRIPT_DIR/output/all.yaml" \
    MIHOMO_NODES="$SCRIPT_DIR/mihomo/nodes" \
    SYNC_SCRIPT="$SCRIPT_DIR/scripts/sync-mihomo-nodes" \
    "$WORKER_BIN" &
WORKER_PID=$!
echo ">>> Worker ready (PID=$WORKER_PID)"

sleep 2

echo ">>> Starting Mihomo..."
"$MIHOMO_BIN" -d "$MIHOMO_CONF" &
MIHOMO_PID=$!

echo ""
echo "============================================"
echo "  All services running"
echo "  SubMill : PID=$SUBS_PID    http://127.0.0.1:8199"
echo "  Worker  : PID=$WORKER_PID  output/ -> mihomo/nodes/"
echo "  Mihomo  : PID=$MIHOMO_PID  mixed-port=7890"
echo "============================================"
echo "  Press Ctrl+C to stop"
echo ""

while kill -0 "$SUBS_PID" 2>/dev/null && kill -0 "$MIHOMO_PID" 2>/dev/null; do
    sleep 2
done