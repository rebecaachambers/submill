#!/bin/sh
# SubMill Docker Entrypoint - starts submill + worker + mihomo

set -e

echo "============================================"
echo "  SubMill Docker Container Starting..."
echo "============================================"

# Ensure necessary directories
mkdir -p /app/config /app/output /app/mihomo/nodes

# Generate default configs if missing
if [ ! -f /app/config/submill.yaml ]; then
    echo "[init] Creating default submill.yaml..."
    cp /app/config/config.example.yaml /app/config/submill.yaml 2>/dev/null || true
fi

if [ ! -f /app/config/config.yaml ]; then
    echo "[init] Creating default mihomo config..."
    cat > /app/config/config.yaml << MIHOMOEOF
mixed-port: 7890
bind-address: "*"
allow-lan: true
mode: rule
log-level: info
ipv6: false
geo-auto-update: false
geo-update-interval: 99999

profile:
  store-selected: true
  store-fake-ip: true

dns:
  enable: true
  ipv6: false
  enhanced-mode: fake-ip
  fake-ip-range: 198.18.0.1/16
  nameserver:
    - 223.5.5.5
    - 119.29.29.29

proxy-providers:
  submill:
    type: file
    path: /app/mihomo/nodes/all.yaml
    health-check:
      enable: true
      url: "http://www.gstatic.com/generate_204"
      interval: 300

proxy-groups:
  - name: PROXY
    type: select
    proxies:
      - auto
      - balance
      - DIRECT

  - name: auto
    type: url-test
    use:
      - submill
    url: "http://www.gstatic.com/generate_204"
    interval: 300
    tolerance: 20

  - name: balance
    type: load-balance
    use:
      - submill
    url: "http://www.gstatic.com/generate_204"
    interval: 300
    strategy: consistent-hashing

  - name: FALLBACK
    type: fallback
    proxies:
      - auto
      - balance
      - DIRECT

rules:
  - IP-CIDR,192.168.0.0/16,DIRECT
  - IP-CIDR,10.0.0.0/8,DIRECT
  - IP-CIDR,172.16.0.0/12,DIRECT
  - IP-CIDR,127.0.0.0/8,DIRECT
  - MATCH,PROXY
MIHOMOEOF
fi

# Set proxy for submill itself (so it can reach subscription URLs)
if [ -n "$SUBMILL_PROXY" ]; then
    export HTTP_PROXY="$SUBMILL_PROXY"
    export HTTPS_PROXY="$SUBMILL_PROXY"
fi

# Trap for graceful shutdown
cleanup() {
    echo ""
    echo ">>> Shutting down..."
    [ -n "$WORKER_PID" ] && kill "$WORKER_PID" 2>/dev/null
    [ -n "$MIHOMO_PID" ] && kill "$MIHOMO_PID" 2>/dev/null
    [ -n "$SUBS_PID" ] && kill "$SUBS_PID" 2>/dev/null
    wait
    echo ">>> All services stopped."
}
trap cleanup INT TERM

# Start SubMill
echo ">>> Starting SubMill..."
/app/submill "$@" &
SUBS_PID=$!

# Wait for SubMill to be ready
echo ">>> Waiting for SubMill to be ready..."
for i in $(seq 1 30); do
    if curl -s http://127.0.0.1:8199/sub/ > /dev/null 2>&1; then
        echo ">>> SubMill is ready (PID=$SUBS_PID)"
        break
    fi
    if ! kill -0 "$SUBS_PID" 2>/dev/null; then
        echo "[ERROR] SubMill failed to start"
        exit 1
    fi
    sleep 2
done

# Start Worker
echo ">>> Starting Worker (watch + sync)..."
SUBS_OUTPUT="/app/output/all.yaml" \
    MIHOMO_NODES="/app/mihomo/nodes" \
    SYNC_SCRIPT="/app/scripts/sync-mihomo-nodes" \
    /app/scripts/watch-submill &
WORKER_PID=$!
echo ">>> Worker started (PID=$WORKER_PID)"

# Give worker a moment to do initial sync
sleep 2

# Start Mihomo
echo ">>> Starting Mihomo..."
/app/mihomo -d /app/config &
MIHOMO_PID=$!

echo ""
echo "============================================"
echo "  SubMill Docker - All services running"
echo "  SubMill : PID=$SUBS_PID    http://127.0.0.1:8199"
echo "  Worker  : PID=$WORKER_PID  output/ -> mihomo/nodes/"
echo "  Mihomo  : PID=$MIHOMO_PID  mixed-port=7890"
echo "============================================"

# Keep container alive
wait