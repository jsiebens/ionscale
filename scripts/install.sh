#!/bin/bash
set -e

# ionscale linux installation script
# This script automates the installation of ionscale on a Linux server with systemd

# Display functions
info() {
  echo "===> [INFO]" "$@"
}

warn() {
  echo "===> [WARN]" "$@"
}

fatal() {
  echo "===> [ERROR]" "$@"
  exit 1
}

# Welcome message
echo "===================================================="
echo "ionscale Installation Script"
echo "===================================================="

# Check for systemd
if ! [ -d /run/systemd ]; then
  fatal "Cannot find systemd to use as a process supervisor for ionscale"
fi

# Check for root or sudo privileges
SUDO=sudo
if [ "$(id -u)" -eq 0 ]; then
  SUDO=
  info "Running as root"
else
  info "Running with sudo"
fi

# Get required input
if [ -z "${IONSCALE_DOMAIN}" ]; then
  read -p "Enter your ionscale domain (e.g. ionscale.example.com): " IONSCALE_DOMAIN
  if [ -z "${IONSCALE_DOMAIN}" ]; then
    fatal "Domain is required"
  fi
fi

if [ -z "${IONSCALE_ACME_EMAIL}" ]; then
  read -p "Enter your email address (for Let's Encrypt notifications): " IONSCALE_ACME_EMAIL
  if [ -z "${IONSCALE_ACME_EMAIL}" ]; then
    fatal "Email address is required"
  fi
fi

# Set up directories and paths
IONSCALE_VERSION=v0.17.0
IONSCALE_DATA_DIR=/var/lib/ionscale
IONSCALE_CONFIG_DIR=/etc/ionscale
IONSCALE_SERVICE_FILE=/etc/systemd/system/ionscale.service
IONSCALE_ENV_FILE=/etc/default/ionscale
BIN_DIR=/usr/local/bin

# --- Architecture detection ---
setup_arch() {
  info "Detecting architecture"
  if [ -z "$ARCH" ]; then
    ARCH=$(uname -m)
  fi
  case $ARCH in
  amd64|x86_64)
    SUFFIX=amd64
    ;;
  arm64|aarch64)
    SUFFIX=arm64
    ;;
  *)
    fatal "Unsupported architecture $ARCH"
    ;;
  esac
  info "Architecture: $ARCH (using $SUFFIX)"
}

# --- Dependencies check ---
install_dependencies() {
  info "Checking for dependencies"
  if ! [ -x "$(command -v curl)" ]; then
    info "Installing curl"
    if [ -n "$(command -v apt-get)" ]; then
      $SUDO apt-get update
      $SUDO apt-get install -y curl
    elif [ -n "$(command -v yum)" ]; then
      $SUDO yum install -y curl
    else
      fatal "Could not find apt-get or yum. Cannot install dependencies on this OS"
    fi
  fi
}

# --- Create service user ---
create_service_user() {
  info "Creating service user"

  # Only create user if it doesn't exist
  if ! id ionscale &>/dev/null; then
    $SUDO useradd --system --no-create-home --shell /bin/false ionscale
    info "Created user 'ionscale'"
  else
    fatal "User 'ionscale' already exists"
  fi
}

# --- Binary installation ---
download_and_install() {
  info "Downloading ionscale binary $IONSCALE_VERSION"
  $SUDO curl -o "$BIN_DIR/ionscale" -sfL "https://github.com/jsiebens/ionscale/releases/download/${IONSCALE_VERSION}/ionscale_linux_${SUFFIX}"
  $SUDO chmod +x "$BIN_DIR/ionscale"

  # Verify installation
  if [ -x "$BIN_DIR/ionscale" ]; then
    info "ionscale binary installed successfully"
    $SUDO $BIN_DIR/ionscale version
  else
    fatal "Failed to install ionscale binary"
  fi
}

# --- Create configuration files ---
create_folders_and_config() {
  info "Creating directories"
  $SUDO mkdir -p ${IONSCALE_DATA_DIR}
  $SUDO mkdir -p ${IONSCALE_CONFIG_DIR}

  # Set appropriate ownership
  $SUDO chown ionscale:ionscale ${IONSCALE_DATA_DIR}
  $SUDO chown ionscale:ionscale ${IONSCALE_CONFIG_DIR}

  info "Generating system admin key"
  ADMIN_KEY=$($BIN_DIR/ionscale genkey -n)
  $SUDO tee $IONSCALE_ENV_FILE >/dev/null <<EOF
IONSCALE_KEYS_SYSTEM_ADMIN_KEY=$ADMIN_KEY
EOF
  info "Generated system admin key:"
  echo "$ADMIN_KEY"
  echo "IMPORTANT: Save this key securely. You'll need it to administer ionscale."

  info "Creating configuration file"
  $SUDO tee ${IONSCALE_CONFIG_DIR}/config.yaml >/dev/null <<EOF
listen_addr: ":443"
public_addr: "${IONSCALE_DOMAIN}:443"
stun_listen_addr: ":3478"
stun_public_addr: "${IONSCALE_DOMAIN}:3478"

tls:
  acme: true
  acme_email: "${IONSCALE_ACME_EMAIL}"

database:
  type: sqlite
  url: "${IONSCALE_DATA_DIR}/ionscale.db?_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)&_pragma=foreign_keys(ON)"

keys:
  system_admin_key: "\${IONSCALE_KEYS_SYSTEM_ADMIN_KEY}"

logging:
  level: info
EOF

  info "Configuration created at ${IONSCALE_CONFIG_DIR}/config.yaml"
}

# --- Create systemd service file ---
create_systemd_service_file() {
  info "Creating systemd service file ${IONSCALE_SERVICE_FILE}"
  $SUDO tee ${IONSCALE_SERVICE_FILE} >/dev/null <<EOF
[Unit]
Description=ionscale - a Tailscale control server
Requires=network-online.target
After=network.target

[Service]
EnvironmentFile=/etc/default/ionscale
User=ionscale
Group=ionscale
ExecStart=${BIN_DIR}/ionscale server --config ${IONSCALE_CONFIG_DIR}/config.yaml
Restart=on-failure
RestartSec=10s
AmbientCapabilities=CAP_NET_BIND_SERVICE

[Install]
WantedBy=multi-user.target
EOF
}

# --- Enable and start service ---
systemd_enable_and_start() {
  if [ "${SKIP_ENABLE}" = true ]; then
    info "Skipping service enablement (SKIP_ENABLE=true)"
    return
  fi

  info "Enabling systemd service"
  $SUDO systemctl daemon-reload >/dev/null
  $SUDO systemctl enable ${IONSCALE_SERVICE_FILE} >/dev/null

  if [ "${SKIP_START}" = true ]; then
    info "Skipping service start (SKIP_START=true)"
    return
  fi

  info "Starting ionscale service"
  $SUDO systemctl restart ionscale

  # Check service status
  ATTEMPTS=0
  MAX_ATTEMPTS=5
  DELAY=2

  while [ $ATTEMPTS -lt $MAX_ATTEMPTS ]; do
    if $SUDO systemctl is-active --quiet ionscale; then
      info "ionscale service is running"
      break
    else
      ATTEMPTS=$((ATTEMPTS + 1))
      if [ $ATTEMPTS -eq $MAX_ATTEMPTS ]; then
        warn "ionscale service failed to start. Check status with: sudo systemctl status ionscale"
      else
        info "Waiting for service to start ($ATTEMPTS/$MAX_ATTEMPTS)..."
        sleep $DELAY
      fi
    fi
  done
}

# Main execution sequence
setup_arch
install_dependencies
create_service_user
download_and_install
create_folders_and_config
create_systemd_service_file
systemd_enable_and_start

# Completion message
echo
echo "===================================================="
echo "ionscale installation complete!"
echo "===================================================="
echo
echo "Your ionscale instance is now available at: https://${IONSCALE_DOMAIN}"
echo
echo "Next steps:"
echo "1. Configure OIDC authentication if needed"
echo "2. Set up DNS provider if needed"
echo "3. Create your first tailnet"
echo
echo "To view logs: sudo journalctl -u ionscale -f"
echo "To restart the service: sudo systemctl restart ionscale"
echo
echo "Configure ionscale CLI:"
echo "export IONSCALE_ADDR=https://${IONSCALE_DOMAIN}"
echo "export IONSCALE_SYSTEM_ADMIN_KEY=${ADMIN_KEY}"
echo "Then you can run: ionscale tailnets list"