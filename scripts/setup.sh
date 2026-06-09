#!/bin/bash
# SubMill + Mihomo - Fully Offline Installer (Linux ARM64/AMD64)
# Usage: bash scripts/setup.sh
set -e

RED="\033[0;31m"; GREEN="\033[0;32m"; YELLOW="\033[1;33m"; CYAN="\033[0;36m"; NC="\033[0m"
log_info()  { echo -e "${GREEN}[INFO]${NC}  $1" | tee -a "$LOG_FILE"; }
log_warn()  { echo -e "${YELLOW}[WARN]${NC}  $1" | tee -a "$LOG_FILE"; }
log_error() { echo -e "${RED}[ERROR]${NC} $1" | tee -a "$LOG_FILE"; }
log_step()  { echo ""; echo -e "${CYAN}[STEP]${NC} $1" | tee -a "$LOG_FILE"; }

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJECT_DIR="$(dirname "$SCRIPT_DIR")"
LOG_FILE="$PROJECT_DIR/install.log"
SUDO_PASS="${SUDO_PASS:-}"
GO_VERSION="1.25.0"

# Start log
echo "============================================" | tee "$LOG_FILE"
echo "  SubMill + Mihomo Install Log" | tee -a "$LOG_FILE"
echo "  Time: $(date)" | tee -a "$LOG_FILE"
echo "============================================" | tee -a "$LOG_FILE"

# Prompt for sudo password if not set
if [ -z "$SUDO_PASS" ]; then
    echo -n "Enter sudo password: "
    read -s SUDO_PASS
    echo ""
fi

sudo_cmd() { echo "$SUDO_PASS" | sudo -S "$@"; }

# =========================================================================
# 1. Detect OS and Architecture
# =========================================================================
log_step "Detecting OS..."
. /etc/os-release 2>/dev/null || true
OS_ID="${ID:-unknown}"
ARCH=$(uname -m)
case "$ARCH" in
    x86_64|amd64)  ARCH="amd64" ;;
    aarch64|arm64) ARCH="arm64" ;;
    *) log_error "Unsupported arch: $ARCH"; exit 1 ;;
esac
log_info "System: $OS_ID  Arch: $ARCH"

# =========================================================================
# 2. System dependencies
# =========================================================================
log_step "Checking system dependencies..."
for cmd in tar curl; do
    if ! command -v $cmd &>/dev/null; then
        log_warn "Missing $cmd, installing..."
        case "$OS_ID" in
            ubuntu|debian|raspbian)
                sudo_cmd apt-get update -qq && sudo_cmd apt-get install -y -qq $cmd ;;
            centos|rhel|fedora|rocky|almalinux)
                sudo_cmd dnf install -y $cmd 2>/dev/null || sudo_cmd yum install -y $cmd ;;
            alpine) sudo_cmd apk add --no-cache $cmd ;;
            arch|manjaro) sudo_cmd pacman -S --noconfirm $cmd ;;
        esac
    fi
done
log_info "System dependencies OK"

# =========================================================================
# 3. Install Go from local tarball
# =========================================================================
GO_TAR="go${GO_VERSION}.linux-${ARCH}.tar.gz"
GO_PATH="$PROJECT_DIR/assets/go/$GO_TAR"

if command -v go &>/dev/null; then
    GO_VER=$(go version | awk '{print $3}' | sed 's/go//')
    log_info "Go $GO_VER already installed, skipping"
else
    log_step "Installing Go ${GO_VERSION}..."
    if [ ! -f "$GO_PATH" ]; then
        log_error "Go tarball not found: $GO_PATH"
        log_error "Please ensure assets/go/ contains the Linux ${ARCH} tarball"
        exit 1
    fi
    log_info "Extracting: assets/go/$GO_TAR"
    sudo_cmd rm -rf /usr/local/go
    sudo_cmd tar -C /usr/local -xzf "$GO_PATH"
    for rc in "$HOME/.bashrc" "$HOME/.zshrc" "$HOME/.profile"; do
        if [ -f "$rc" ] && ! grep -q "/usr/local/go/bin" "$rc" 2>/dev/null; then
            echo 'export PATH=$PATH:/usr/local/go/bin:$HOME/go/bin' >> "$rc"
        fi
    done
    export PATH=$PATH:/usr/local/go/bin:$HOME/go/bin
    log_info "Go installed: $(go version)"
fi

export PATH=$PATH:/usr/local/go/bin:$HOME/go/bin
export HOME="${HOME:-/root}"
export GOCACHE="${HOME}/.cache/go-build"

# =========================================================================
# 3.5 Extract vendor tarballs
# =========================================================================
log_step "Extracting vendor dependencies..."
cd "$PROJECT_DIR"
if [ ! -d "vendor" ]; then
    log_info "Extracting assets/vendor.tar.gz..."
    tar -xzf assets/vendor.tar.gz
fi
if [ ! -d "mihomo/vendor" ]; then
    log_info "Extracting assets/mihomo-vendor.tar.gz..."
    tar -xzf assets/mihomo-vendor.tar.gz
fi
log_info "Vendor dependencies ready"

# =========================================================================
# 4. Compile SubMill
# =========================================================================
log_step "Compiling SubMill..."
cd "$PROJECT_DIR"
if [ ! -d "vendor" ]; then
    log_error "vendor/ directory missing!"
    exit 1
fi
log_info "vendored deps: $(du -sh vendor | cut -f1)"

# Clean cache to avoid Go 1.25 telemetry SIGBUS on ARM64
rm -rf "$GOCACHE"

CGO_ENABLED=0 go build -mod=vendor -trimpath -ldflags="-s" -o submill . 2>&1 | tee -a "$LOG_FILE"
if [ -f submill ]; then
    log_info "SubMill compiled: $(ls -lh submill | awk '{print $5}')"
else
    log_error "SubMill compile failed!"
    exit 1
fi

# =========================================================================
# 5. Compile Mihomo
# =========================================================================
log_step "Compiling Mihomo..."
cd "$PROJECT_DIR/mihomo"
if [ ! -d "vendor" ]; then
    log_error "mihomo/vendor/ directory missing!"
    exit 1
fi
log_info "vendored deps: $(du -sh vendor | cut -f1)"

rm -rf "$GOCACHE"

CGO_ENABLED=0 go build -mod=vendor -trimpath -ldflags="-s" -o mihomo github.com/metacubex/mihomo 2>&1 | tee -a "$LOG_FILE"
if [ -f mihomo ]; then
    log_info "Mihomo compiled: $(ls -lh mihomo | awk '{print $5}')"
else
    log_error "Mihomo compile failed!"
    exit 1
fi

# =========================================================================
# 6. Configure
# =========================================================================
log_step "Setting up configuration..."
cd "$PROJECT_DIR"

# SubMill config: copy config.example.yaml to submill.yaml (only if not exists)
if [ ! -f "config/submill.yaml" ]; then
    cp config/config.example.yaml config/submill.yaml
    log_info "Created config/submill.yaml from template"
    log_warn "Edit config/submill.yaml to set your sub-urls"
else
    log_info "config/submill.yaml exists, skipping"
fi

# Mihomo config: copy mihomo.yaml to config.yaml (only if not exists)
if [ ! -f "config/config.yaml" ]; then
    cp config/mihomo.yaml config/config.yaml
    log_info "Mihomo config created: config/config.yaml"
else
    log_info "Mihomo config: config/config.yaml exists, skipping"
fi

mkdir -p config/output
ln -sf config/output output 2>/dev/null || true
log_info "output/ directory ready (inside config/ for mihomo safe-path)"

# =========================================================================
# 7. Register systemd services (Linux only)
# =========================================================================
if command -v systemctl &>/dev/null; then
    log_step "Registering systemd services..."

    cat > /tmp/submill.service << UNITEOF
[Unit]
Description=SubMill - Proxy Node Checker
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
ExecStart=${PROJECT_DIR}/submill
WorkingDirectory=${PROJECT_DIR}
Restart=on-failure
RestartSec=10

[Install]
WantedBy=multi-user.target
UNITEOF

    cat > /tmp/mihomo.service << UNITEOF
[Unit]
Description=Mihomo - Proxy Core
After=submill.service
Requires=submill.service

[Service]
Type=simple
ExecStart=${PROJECT_DIR}/mihomo/mihomo -d ${PROJECT_DIR}/config
WorkingDirectory=${PROJECT_DIR}
AmbientCapabilities=CAP_NET_BIND_SERVICE
Restart=on-failure
RestartSec=5

[Install]
WantedBy=multi-user.target
UNITEOF

    sudo_cmd mv /tmp/submill.service /etc/systemd/system/
    sudo_cmd mv /tmp/mihomo.service /etc/systemd/system/
    sudo_cmd systemctl daemon-reload
    sudo_cmd systemctl enable submill mihomo
    log_info "systemd services registered and enabled"
else
    log_info "No systemd detected, skipping service registration"
fi

# =========================================================================
# Done
# =========================================================================
echo "" | tee -a "$LOG_FILE"
echo "============================================" | tee -a "$LOG_FILE"
echo "  INSTALLATION COMPLETE" | tee -a "$LOG_FILE"
echo "" | tee -a "$LOG_FILE"
echo "  Start:   systemctl start submill mihomo" | tee -a "$LOG_FILE"
echo "  Enable:  systemctl enable submill mihomo" | tee -a "$LOG_FILE"
echo "  Status:  systemctl status submill mihomo" | tee -a "$LOG_FILE"
echo "  Logs:    journalctl -u submill -f" | tee -a "$LOG_FILE"
echo "" | tee -a "$LOG_FILE"
echo "  Mihomo proxy port: 7890 (HTTP/SOCKS5)" | tee -a "$LOG_FILE"
echo "  SubMill web panel: http://localhost:8199/admin" | tee -a "$LOG_FILE"
echo "" | tee -a "$LOG_FILE"
echo "  Install log: $LOG_FILE" | tee -a "$LOG_FILE"
echo "============================================" | tee -a "$LOG_FILE"