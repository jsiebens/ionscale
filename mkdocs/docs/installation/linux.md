# Linux installation

This guide walks you through installing ionscale directly on a Linux server. This approach gives you more control over the installation and is suitable for production environments.

## Prerequisites

Before you begin, make sure you have:

- A Linux server (virtual or physical)
- Root or sudo access
- A registered domain name pointing to your server
- Ports 443 (HTTPS) and 3478/UDP (STUN) open in your firewall
- Basic familiarity with the Linux command line

### Domain and DNS configuration

ionscale requires a domain name to function properly. This enables secure HTTPS connections and proper Tailscale device discovery.

1. Configure an A record in your domain's DNS settings:
   ```
   ionscale.example.com  →  YOUR_SERVER_IP
   ```
   (Replace "example.com" with your actual domain and "YOUR_SERVER_IP" with your server's public IP address)

2. Verify the DNS record has propagated:
   ```bash
   dig ionscale.example.com
   ```
   The command should return your server's public IP address.

## Quick deployment

If you prefer an automated deployment, you can use our installation script:

```bash
# Download the script
curl -fsSL https://raw.githubusercontent.com/jsiebens/ionscale/main/scripts/install.sh -o install.sh
chmod +x install.sh

# Run the script (interactive mode)
./install.sh
```

The script will prompt you for:
1. Your domain name for ionscale
2. Your email address (for Let's Encrypt notifications)

For non-interactive installation, set the required environment variables:

```bash
export IONSCALE_DOMAIN="ionscale.example.com"
export IONSCALE_ACME_EMAIL="your-email@example.com"
./install.sh
```

The script automatically:

1. Determines your system architecture
2. Creates a dedicated service user
3. Downloads and installs the latest ionscale binary
4. Generates a secure system admin key
5. Creates necessary configuration files
6. Sets up and starts the systemd service

For a detailed explanation of each step, continue reading the manual installation instructions below.

## System preparation

### Create a dedicated service user

For security reasons, ionscale should run under a dedicated, unprivileged system user:

```bash
# Create service user
sudo useradd --system --no-create-home --shell /bin/false ionscale

# Create directories
sudo mkdir -p /etc/ionscale
sudo mkdir -p /var/lib/ionscale

# Set appropriate permissions
sudo chown ionscale:ionscale /etc/ionscale
sudo chown ionscale:ionscale /var/lib/ionscale
```

## Installation

### Download and install the binary

Install the ionscale binary on your system:

```bash
# Download the latest version (adjust for your CPU architecture)
sudo curl -o "/usr/local/bin/ionscale" \
  -sfL "https://github.com/jsiebens/ionscale/releases/download/v0.17.0/ionscale_linux_amd64"

# Make it executable
sudo chmod +x "/usr/local/bin/ionscale"

# Verify installation
ionscale version
```

### Configuration

1. Generate a system admin key and store it in an environment file:

```bash
sudo tee /etc/default/ionscale >/dev/null <<EOF
IONSCALE_KEYS_SYSTEM_ADMIN_KEY=$(ionscale genkey -n)
EOF
```

!!! important "System admin key"
    The system admin key is a critical security component that provides full administrative access to your ionscale instance. Make sure to save this key securely.
    
    Alternatively, you can configure ionscale without a system admin key by using an OIDC provider and setting up system admin accounts through the OIDC configuration. See the [Authentication with OIDC](../configuration/auth-oidc.md) documentation for details.

2. Set up environment variables for the configuration:

```bash
export IONSCALE_ACME_EMAIL="your-email@example.com"  # For Let's Encrypt notifications
export IONSCALE_DOMAIN="ionscale.example.com"        # Your ionscale domain
```

3. Create the configuration file:

```bash
sudo tee /etc/ionscale/config.yaml >/dev/null <<EOF
listen_addr: ":443"
public_addr: "${IONSCALE_DOMAIN}:443"
stun_public_addr: "${IONSCALE_DOMAIN}:3478"

tls:
  acme: true
  acme_email: "${IONSCALE_ACME_EMAIL}"

database:
  url: "/var/lib/ionscale/ionscale.db?_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)"

keys:
  system_admin_key: "\${IONSCALE_KEYS_SYSTEM_ADMIN_KEY}"
  
logging:
  level: info
EOF
```

## Setting up systemd service

Create a systemd service file to manage the ionscale process:

```bash
sudo tee /etc/systemd/system/ionscale.service >/dev/null <<EOF
[Unit]
Description=ionscale - a Tailscale control server
Requires=network-online.target
After=network.target

[Service]
EnvironmentFile=/etc/default/ionscale
User=ionscale
Group=ionscale
ExecStart=/usr/local/bin/ionscale server --config /etc/ionscale/config.yaml
Restart=on-failure
RestartSec=10s
AmbientCapabilities=CAP_NET_BIND_SERVICE

[Install]
WantedBy=multi-user.target
EOF
```

The `AmbientCapabilities=CAP_NET_BIND_SERVICE` line allows ionscale to bind to privileged ports (443) without running as root.

## Starting ionscale

Enable and start the ionscale service:

```bash
# Reload systemd to recognize the new service
sudo systemctl daemon-reload

# Enable service to start on boot
sudo systemctl enable ionscale

# Start the service
sudo systemctl start ionscale

# Check status
sudo systemctl status ionscale
```

If everything started successfully, you should see an "active (running)" status.
