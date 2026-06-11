#!/bin/bash
# SubMill + Mihomo + Worker - Fully Offline Installer (Linux AMD64/ARM64)
# Usage: SUDO_PASS='xxx' bash scripts/setup.sh
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

# ---- Fix HOME for headless users (no home dir) ----
if [ ! -d "$HOME" ] || [ ! -w "$HOME" ]; then
    export HOME="$PROJECT_DIR/.home"
    mkdir -p "$HOME"
fi
export GOCACHE="$PROJECT_DIR/.gocache"
mkdir -p "$GOCACHE"

# ---- Start log ----
echo "============================================" | tee "$LOG_FILE"
echo "  SubMill + Mihomo + Worker Install Log" | tee -a "$LOG_FILE"
echo "  Time: $(date)" | tee -a "$LOG_FILE"
echo "============================================" | tee -a "$LOG_FILE"

# ---- Sudo helper ----
if [ -z "$SUDO_PASS" ]; then
    echo -n "Enter sudo password: "
    read -s SUDO_PASS
    echo ""
fi
sudo_cmd() { echo "$SUDO_PASS" | sudo -S bash -c "$*"; }

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
# 2. System dependencies (inotify-tools for worker)
# =========================================================================
log_step "Checking system dependencies..."
for cmd in tar curl inotifywait python3; do
    if ! command -v $cmd &>/dev/null; then
        pkg=""
        case "$cmd" in
            inotifywait) pkg="inotify-tools" ;;
            python3)     pkg="python3 python3-yaml" ;;
            *)           pkg="$cmd" ;;
        esac
        log_warn "Missing $cmd, installing $pkg..."
        case "$OS_ID" in
            ubuntu|debian|raspbian)
                sudo_cmd "apt-get update -qq && apt-get install -y -qq $pkg" ;;
            centos|rhel|fedora|rocky|almalinux)
                sudo_cmd "dnf install -y $pkg 2>/dev/null || yum install -y $pkg" ;;
            alpine) sudo_cmd "apk add --no-cache $pkg" ;;
            arch|manjaro) sudo_cmd "pacman -S --noconfirm $pkg" ;;
        esac
    fi
done
# Verify PyYAML
python3 -c "import yaml" 2>/dev/null || {
    case "$OS_ID" in
        ubuntu|debian|raspbian) sudo_cmd "apt-get install -y -qq python3-yaml" ;;
        *) sudo_cmd "pip3 install pyyaml 2>/dev/null || pip install pyyaml" ;;
    esac
}
log_info "System dependencies OK"

# =========================================================================
# 3. Install Go
# =========================================================================
GO_TAR="go${GO_VERSION}.linux-${ARCH}.tar.gz"
GO_PATH="$PROJECT_DIR/assets/go/$GO_TAR"

if command -v go &>/dev/null; then
    GO_VER=$(go version | awk '{print $3}' | sed 's/go//')
    log_info "Go $GO_VER already installed, skipping"
else
    log_step "Installing Go ${GO_VERSION}..."
    if [ ! -f "$GO_PATH" ]; then
        log_warn "Go tarball not found locally, downloading..."
        GO_URL="https://go.dev/dl/${GO_TAR}"
        mkdir -p "$PROJECT_DIR/assets/go"
        curl -L -o "$GO_PATH" "$GO_URL" || {
            log_error "Download failed: $GO_URL"
            log_error "Place $GO_TAR in assets/go/ manually and retry"
            exit 1
        }
    fi
    log_info "Extracting: assets/go/$GO_TAR"
    sudo_cmd "rm -rf /usr/local/go && tar -C /usr/local -xzf $GO_PATH"
    for rc in /etc/profile /etc/bash.bashrc; do
        [ -f "$rc" ] && ! grep -q "/usr/local/go/bin" "$rc" 2>/dev/null && {
            echo 'export PATH=$PATH:/usr/local/go/bin' | sudo_cmd "tee -a $rc"
        }
    done
    export PATH=$PATH:/usr/local/go/bin
    log_info "Go installed: $(go version)"
fi

export PATH=$PATH:/usr/local/go/bin
export GOPATH="$PROJECT_DIR/.go"

# =========================================================================
# 4. Extract vendor tarballs
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
# 5. Compile SubMill
# =========================================================================
log_step "Compiling SubMill..."
cd "$PROJECT_DIR"
[ -d "vendor" ] || { log_error "vendor/ directory missing!"; exit 1; }
log_info "vendored deps: $(du -sh vendor | cut -f1)"

rm -rf "$GOCACHE"
CGO_ENABLED=0 go build -mod=vendor -trimpath \
    -ldflags="-s -w -X main.version=v1.0.0" \
    -o submill . 2>&1 | tail -3 | tee -a "$LOG_FILE"

[ -f "submill" ] || { log_error "SubMill compile failed!"; exit 1; }
log_info "SubMill compiled: $(du -sh submill | cut -f1)"

# =========================================================================
# 6. Compile Mihomo
# =========================================================================
log_step "Compiling Mihomo..."
cd "$PROJECT_DIR/mihomo"
[ -d "vendor" ] || { log_error "mihomo/vendor missing!"; exit 1; }
log_info "vendored deps: $(du -sh vendor | cut -f1)"

rm -rf "$GOCACHE"
CGO_ENABLED=0 go build -mod=vendor -trimpath \
    -ldflags="-s -w" \
    -o mihomo . 2>&1 | tail -3 | tee -a "$LOG_FILE"

[ -f "mihomo" ] || { log_error "Mihomo compile failed!"; exit 1; }
log_info "Mihomo compiled: $(du -sh mihomo | cut -f1)"

# =========================================================================
# 7. Configuration
# =========================================================================
log_step "Setting up configuration..."
cd "$PROJECT_DIR"

# SubMill config
if [ ! -f "config/submill.yaml" ]; then
    cp config/config.example.yaml config/submill.yaml 2>/dev/null || true
    [ -f "config/submill.yaml" ] && log_info "Created config/submill.yaml from template" \
        || log_warn "No config.example.yaml, skip. Create config/submill.yaml manually"
fi

# Mihomo config
if [ ! -f "config/config.yaml" ]; then
    cp config/mihomo.yaml config/config.yaml
    log_info "Mihomo config created: config/config.yaml"
fi

# Create directories
mkdir -p "$PROJECT_DIR/output" "$PROJECT_DIR/mihomo/nodes" "$PROJECT_DIR/config/mihomo"

# ---- Fix: symlink mihomo/nodes into config/ so SAFE_PATHS allows it ----
if [ ! -L "$PROJECT_DIR/config/mihomo/nodes" ]; then
    ln -sfn "$PROJECT_DIR/mihomo/nodes" "$PROJECT_DIR/config/mihomo/nodes" 2>/dev/null || {
        sudo_cmd "ln -sfn $PROJECT_DIR/mihomo/nodes $PROJECT_DIR/config/mihomo/nodes"
    }
    log_info "Symlinked config/mihomo/nodes -> mihomo/nodes (Mihomo SAFE_PATHS)"
fi

# Keep relative path in mihomo config (resolved via symlink)
sed -i 's|path: .*nodes/all.yaml|path: mihomo/nodes/all.yaml|' "$PROJECT_DIR/config/config.yaml" 2>/dev/null || true
log_info "Mihomo config: file provider reads mihomo/nodes/all.yaml"

# =========================================================================
# 8. Install Worker scripts
# =========================================================================
log_step "Installing Worker scripts..."
chmod +x "$PROJECT_DIR/scripts/watch-submill"
chmod +x "$PROJECT_DIR/scripts/sync-mihomo-nodes"

cat > /tmp/watch-submill.env << ENVEOF
SUBS_OUTPUT=$PROJECT_DIR/output/all.yaml
MIHOMO_NODES=$PROJECT_DIR/mihomo/nodes
SYNC_SCRIPT=$PROJECT_DIR/scripts/sync-mihomo-nodes
ENVEOF
log_info "Worker scripts installed"

# =========================================================================
# 9. Register systemd services
# =========================================================================
if command -v systemctl &>/dev/null; then
    log_step "Registering systemd services (3 services)..."

    cat > /tmp/submill.service << UNITEOF
[Unit]
Description=SubMill - Proxy Node Checker
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
ExecStart=${PROJECT_DIR}/submill
WorkingDirectory=${PROJECT_DIR}
Environment=HOME=${PROJECT_DIR}/.home
Environment=GOCACHE=${PROJECT_DIR}/.gocache
Restart=on-failure
RestartSec=10

[Install]
WantedBy=multi-user.target
UNITEOF

    cat > /tmp/watch-submill.service << UNITEOF
[Unit]
Description=SubMill Worker - Watch nodes and sync to Mihomo
After=submill.service
Requires=submill.service
Before=mihomo.service

[Service]
Type=simple
EnvironmentFile=-/tmp/watch-submill.env
Environment=HOME=${PROJECT_DIR}/.home
ExecStart=${PROJECT_DIR}/scripts/watch-submill
WorkingDirectory=${PROJECT_DIR}
Restart=always
RestartSec=10

[Install]
WantedBy=multi-user.target
UNITEOF

    cat > /tmp/mihomo.service << UNITEOF
[Unit]
Description=Mihomo - Proxy Core
After=watch-submill.service
Requires=watch-submill.service

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

    sudo_cmd "mv /tmp/submill.service /etc/systemd/system/"
    sudo_cmd "mv /tmp/watch-submill.service /etc/systemd/system/"
    sudo_cmd "mv /tmp/mihomo.service /etc/systemd/system/"
    sudo_cmd "systemctl daemon-reload"
    sudo_cmd "systemctl enable submill watch-submill mihomo"
    log_info "3 systemd services registered: submill -> watch-submill -> mihomo"
else
    log_info "No systemd detected, skipping service registration"
fi

# =========================================================================
# Done
# =========================================================================
echo "" | tee -a "$LOG_FILE"
echo "============================================" | tee -a "$LOG_FILE"
echo "  INSTALLATION COMPLETE" | tee -a "$LOG_FILE"
echo "  Mihomo proxy port: 7890 (HTTP/SOCKS5)" | tee -a "$LOG_FILE"
echo "  SubMill web panel: http://localhost:8199/admin" | tee -a "$LOG_FILE"
echo "  Start:  systemctl start submill watch-submill mihomo" | tee -a "$LOG_FILE"
echo "  Install log: $LOG_FILE" | tee -a "$LOG_FILE"
echo "============================================" | tee -a "$LOG_FILE"