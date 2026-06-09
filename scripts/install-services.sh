#!/bin/bash
# =============================================================================
# 注册系统服务
# Linux: systemd
# macOS: launchd
# =============================================================================
set -e

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJECT_DIR="$(dirname "$SCRIPT_DIR")"
SUBS_BIN="$PROJECT_DIR/submill"
MIHOMO_BIN="$PROJECT_DIR/mihomo/mihomo"
MIHOMO_CONF="$PROJECT_DIR/config"

install_linux() {
    echo ">>> 注册 systemd 服务..."

    cat > /tmp/submill.service << EOF
[Unit]
Description=submill - Proxy Node Checker
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
ExecStart=$SUBS_BIN
WorkingDirectory=$PROJECT_DIR
Restart=on-failure
RestartSec=10

[Install]
WantedBy=multi-user.target
EOF

    cat > /tmp/mihomo.service << EOF
[Unit]
Description=Mihomo - Proxy Core
After=submill.service
Requires=submill.service

[Service]
Type=simple
ExecStart=$MIHOMO_BIN -d $MIHOMO_CONF
WorkingDirectory=$PROJECT_DIR
AmbientCapabilities=CAP_NET_BIND_SERVICE
Restart=on-failure
RestartSec=5

[Install]
WantedBy=multi-user.target
EOF

    sudo mv /tmp/submill.service /etc/systemd/system/
    sudo mv /tmp/mihomo.service /etc/systemd/system/
    sudo systemctl daemon-reload

    echo ""
    echo "服务已注册，管理命令:"
    echo "  systemctl enable --now submill mihomo   # 开机自启"
    echo "  systemctl status submill mihomo          # 查看状态"
    echo "  journalctl -u submill -f                 # 日志"
}

install_macos() {
    echo ">>> 注册 launchd 服务..."

    mkdir -p ~/Library/LaunchAgents

    cat > ~/Library/LaunchAgents/com.submill.plist << EOF
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN"
  "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>com.submill</string>
    <key>ProgramArguments</key>
    <array>
        <string>$SUBS_BIN</string>
    </array>
    <key>WorkingDirectory</key>
    <string>$PROJECT_DIR</string>
    <key>RunAtLoad</key>
    <true/>
    <key>KeepAlive</key>
    <true/>
    <key>StandardOutPath</key>
    <string>$PROJECT_DIR/submill.log</string>
    <key>StandardErrorPath</key>
    <string>$PROJECT_DIR/submill.log</string>
</dict>
</plist>
EOF

    cat > ~/Library/LaunchAgents/com.mihomo.plist << EOF
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN"
  "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>com.mihomo</string>
    <key>ProgramArguments</key>
    <array>
        <string>$MIHOMO_BIN</string>
        <string>-d</string>
        <string>$MIHOMO_CONF</string>
    </array>
    <key>WorkingDirectory</key>
    <string>$PROJECT_DIR</string>
    <key>RunAtLoad</key>
    <true/>
    <key>KeepAlive</key>
    <true/>
    <key>StandardOutPath</key>
    <string>$PROJECT_DIR/mihomo.log</string>
    <key>StandardErrorPath</key>
    <string>$PROJECT_DIR/mihomo.log</string>
</dict>
</plist>
EOF

    launchctl unload ~/Library/LaunchAgents/com.submill.plist 2>/dev/null || true
    launchctl unload ~/Library/LaunchAgents/com.mihomo.plist 2>/dev/null || true
    launchctl load ~/Library/LaunchAgents/com.submill.plist
    launchctl load ~/Library/LaunchAgents/com.mihomo.plist

    echo ""
    echo "launchd 服务已注册并启动:"
    echo "  launchctl list | grep -E 'submill|mihomo'"
    echo "  停止: launchctl unload ~/Library/LaunchAgents/com.submill.plist"
}

# ---- main ----
if [ "$(uname)" = "Darwin" ]; then
    install_macos
elif command -v systemctl &>/dev/null; then
    install_linux
else
    echo "不支持的系统，请手动配置服务"
    exit 1
fi
