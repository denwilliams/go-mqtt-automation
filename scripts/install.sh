#!/bin/bash

# MQTT Home Automation System Installation Script
# For Linux systems (tested on Ubuntu, Debian, Raspberry Pi OS)

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Configuration
USER="automation"
GROUP="automation"
INSTALL_DIR="/opt/automation"
DATA_DIR="/var/lib/automation"
CONFIG_DIR="/etc/automation"
LOG_DIR="/var/log/automation"

# Functions
log_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

check_root() {
    if [[ $EUID -ne 0 ]]; then
        log_error "This script must be run as root (use sudo)"
        exit 1
    fi
}

detect_architecture() {
    local arch=$(uname -m)
    case $arch in
        x86_64)
            echo "linux-amd64"
            ;;
        aarch64|arm64)
            echo "linux-arm64"
            ;;
        armv7l)
            echo "linux-arm"
            ;;
        *)
            log_error "Unsupported architecture: $arch"
            exit 1
            ;;
    esac
}

create_user() {
    if ! id "$USER" &>/dev/null; then
        log_info "Creating user: $USER"
        useradd -r -s /bin/false -d "$INSTALL_DIR" "$USER"
    else
        log_info "User $USER already exists"
    fi
}

create_directories() {
    log_info "Creating directories..."
    mkdir -p "$INSTALL_DIR" "$DATA_DIR" "$CONFIG_DIR" "$LOG_DIR"
    chown "$USER:$GROUP" "$INSTALL_DIR" "$DATA_DIR" "$LOG_DIR"
    chmod 755 "$INSTALL_DIR" "$DATA_DIR" "$LOG_DIR"
    chmod 750 "$CONFIG_DIR"
}

install_binary() {
    local version="${1:-latest}"
    local arch=$(detect_architecture)
    local binary_name="automation-server-$arch"

    log_info "Installing binary for architecture: $arch"

    if [[ -f "./$binary_name" ]]; then
        # Local installation
        log_info "Installing from local binary: $binary_name"
        cp "./$binary_name" "$INSTALL_DIR/automation-server"
    elif [[ -f "./automation-server" ]]; then
        # Development installation
        log_info "Installing from development build"
        cp "./automation-server" "$INSTALL_DIR/automation-server"
    else
        log_error "Binary not found. Please either:"
        log_error "  1. Place the binary in the current directory"
        log_error "  2. Run 'make build' first for development installation"
        exit 1
    fi

    chmod +x "$INSTALL_DIR/automation-server"
    chown "$USER:$GROUP" "$INSTALL_DIR/automation-server"
}

install_config() {
    if [[ ! -f "$CONFIG_DIR/config.yaml" ]]; then
        if [[ -f "./config/config.yaml" ]]; then
            log_info "Installing configuration file"
            cp "./config/config.yaml" "$CONFIG_DIR/config.yaml"
        elif [[ -f "./config/config.example.yaml" ]]; then
            log_info "Installing example configuration file"
            cp "./config/config.example.yaml" "$CONFIG_DIR/config.yaml"
        else
            log_warn "No configuration file found. You'll need to create $CONFIG_DIR/config.yaml manually"
        fi

        if [[ -f "$CONFIG_DIR/config.yaml" ]]; then
            chown root:$GROUP "$CONFIG_DIR/config.yaml"
            chmod 640 "$CONFIG_DIR/config.yaml"
        fi
    else
        log_info "Configuration file already exists, skipping"
    fi
}

install_systemd_service() {
    log_info "Installing systemd service"

    if [[ -f "./systemd/automation.service" ]]; then
        cp "./systemd/automation.service" "/etc/systemd/system/"
    else
        # Create service file inline
        cat > "/etc/systemd/system/automation.service" << EOF
[Unit]
Description=MQTT Home Automation Server
Documentation=https://github.com/denwilliams/go-mqtt-automation
After=network.target
Wants=network-online.target

[Service]
Type=simple
User=$USER
Group=$GROUP
WorkingDirectory=$INSTALL_DIR
ExecStart=$INSTALL_DIR/automation-server -config $CONFIG_DIR/config.yaml
ExecReload=/bin/kill -HUP \$MAINPID
Restart=on-failure
RestartSec=5s
TimeoutStopSec=30s

# Security hardening
NoNewPrivileges=yes
PrivateTmp=yes
ProtectSystem=strict
ProtectHome=yes
ReadWritePaths=$INSTALL_DIR $DATA_DIR $LOG_DIR

# Resource limits
LimitNOFILE=65536
MemoryMax=512M

# Environment
Environment=LOG_LEVEL=info

[Install]
WantedBy=multi-user.target
EOF
    fi

    systemctl daemon-reload
}

setup_database() {
    log_info "Setting up database directory"
    # Update config to use the data directory
    if [[ -f "$CONFIG_DIR/config.yaml" ]]; then
        sed -i "s|\\./automation\\.db|$DATA_DIR/automation.db|g" "$CONFIG_DIR/config.yaml"
    fi
}

show_status() {
    echo ""
    log_info "Installation completed successfully!"
    echo ""
    echo "Next steps:"
    echo "  1. Review and edit the configuration: $CONFIG_DIR/config.yaml"
    echo "  2. Start the service: systemctl start automation"
    echo "  3. Enable auto-start: systemctl enable automation"
    echo "  4. Check status: systemctl status automation"
    echo "  5. View logs: journalctl -u automation -f"
    echo ""
    echo "Web interface will be available at: http://localhost:8080"
    echo ""
}

# Main installation
main() {
    log_info "Starting MQTT Home Automation System installation..."

    check_root
    create_user
    create_directories
    install_binary "$@"
    install_config
    install_systemd_service
    setup_database
    show_status
}

# Run main function with all arguments
main "$@"