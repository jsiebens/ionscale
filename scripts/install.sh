#!/bin/bash
set -e

info() {
  echo '[INFO] ->' "$@"
}

fatal() {
  echo '[ERROR] ->' "$@"
  exit 1
}

verify_system() {
  if ! [ -d /run/systemd ]; then
    fatal 'Can not find systemd to use as a process supervisor for ionscale'
  fi
}

setup_env() {
  SUDO=sudo
  if [ "$(id -u)" -eq 0 ]; then
    SUDO=
  fi

  if [ -z "${IONSCALE_DOMAIN}" ]; then
    fatal "env variable IONSCALE_DOMAIN is undefined"
  fi

  if [ -z "${IONSCALE_ACME_EMAIL}" ]; then
    fatal "env variable IONSCALE_ACME_EMAIL is undefined"
  fi

  IONSCALE_VERSION=v0.6.0
  IONSCALE_DATA_DIR=/var/lib/ionscale
  IONSCALE_CONFIG_DIR=/etc/ionscale
  IONSCALE_SERVICE_FILE=/etc/systemd/system/ionscale.service

  BIN_DIR=/usr/local/bin
}

# --- set arch and suffix, fatal if architecture not supported ---
setup_verify_arch() {
  if [ -z "$ARCH" ]; then
    ARCH=$(uname -m)
  fi
  case $ARCH in
  amd64)
    SUFFIX=amd64
    ;;
  x86_64)
    SUFFIX=amd64
    ;;
  arm64)
    SUFFIX=arm64
    ;;
  aarch64)
    SUFFIX=arm64
    ;;
  *)
    fatal "Unsupported architecture $ARCH"
    ;;
  esac
}

has_yum() {
  [ -n "$(command -v yum)" ]
}

has_apt_get() {
  [ -n "$(command -v apt-get)" ]
}

install_dependencies() {
  if ! [ -x "$(command -v curl)" ]; then
    if $(has_apt_get); then
      $SUDO apt-get install -y curl
    elif $(has_yum); then
      $SUDO yum install -y curl
    else
      fatal "Could not find apt-get or yum. Cannot install dependencies on this OS"
      exit 1
    fi
  fi
}

download_and_install() {
  info "Downloading ionscale binary"
  $SUDO curl -o "$BIN_DIR/ionscale" -sfL "https://github.com/jsiebens/ionscale/releases/download/${IONSCALE_VERSION}/ionscale_linux_${SUFFIX}"
  $SUDO chmod +x "$BIN_DIR/ionscale"
}

create_folders_and_config() {
  $SUDO mkdir --parents ${IONSCALE_DATA_DIR}
  $SUDO mkdir --parents ${IONSCALE_CONFIG_DIR}

  if [ ! -f "/etc/default/ionscale" ]; then
    $SUDO tee /etc/default/ionscale >/dev/null <<EOF
IONSCALE_KEYS_SYSTEM_ADMIN_KEY=$($BIN_DIR/ionscale genkey -n)
EOF
  fi

    $SUDO tee ${IONSCALE_CONFIG_DIR}/config.yaml >/dev/null <<EOF
http_listen_addr: ":80"
https_listen_addr: ":443"
metrics_listen_addr: 127.0.0.1:9090
server_url: "https://${IONSCALE_DOMAIN}"

tls:
  acme: true
  acme_email: "${IONSCALE_ACME_EMAIL}"
  acme_path: "${IONSCALE_DATA_DIR}/acme"

database:
  type: sqlite
  url: "${IONSCALE_DATA_DIR}/ionscale.db?_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)&_pragma=foreign_keys(ON)"

logging:
  level: info
EOF

}

# --- write systemd service file ---
create_systemd_service_file() {
  info "Adding systemd service file ${IONSCALE_SERVICE_FILE}"
  $SUDO tee ${IONSCALE_SERVICE_FILE} >/dev/null <<EOF
[Unit]
Description=ionscale - a Tailscale Controller server
After=syslog.target
After=network.target

[Service]
EnvironmentFile=/etc/default/ionscale
ExecStart=${BIN_DIR}/ionscale server --config ${IONSCALE_CONFIG_DIR}/config.yaml
Restart=on-failure
RestartSec=10s

[Install]
WantedBy=multi-user.target
EOF
}

# --- startup systemd service ---
systemd_enable_and_start() {
  [ "${SKIP_ENABLE}" = true ] && return

  info "Enabling systemd service"
  $SUDO systemctl enable ${IONSCALE_SERVICE_FILE} >/dev/null
  $SUDO systemctl daemon-reload >/dev/null

  [ "${SKIP_START}" = true ] && return

  info "Starting systemd service"
  $SUDO systemctl restart ionscale

  return 0
}

setup_env
setup_verify_arch
verify_system
install_dependencies
download_and_install
create_folders_and_config
create_systemd_service_file
systemd_enable_and_start
