# Installing ionscale with Docker

This guide will walk you through installing and configuring ionscale using Docker containers. Docker provides a simple and consistent way to deploy ionscale across different environments.

## Prerequisites

Before you begin, make sure you have:

- A Linux server (virtual or physical)
- Docker installed ([official Docker installation guide](https://docs.docker.com/engine/install/))
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

## Container setup

### Creating a directory structure

Create a dedicated directory for ionscale files:

```bash
mkdir -p ionscale/data
cd ./ionscale
```

### Generating a configuration file

First, set up environment variables for the configuration:

```bash
export IONSCALE_ACME_EMAIL="your-email@example.com"  # Used for Let's Encrypt notifications
export IONSCALE_DOMAIN="ionscale.example.com"        # Your ionscale domain
export IONSCALE_SYSTEM_ADMIN_KEY=$(docker run --rm ghcr.io/jsiebens/ionscale:0.17.0 genkey -n)
```

!!! important "System admin key"
    The system admin key is a critical security component that provides full administrative access to your ionscale instance. Make sure to save this key securely.
    
    Alternatively, you can configure ionscale without a system admin key by using an OIDC provider and setting up system admin accounts through the OIDC configuration. See the [Authentication with OIDC](../configuration/auth-oidc.md) documentation for details.

Now create the configuration file:

```bash
cat > ./config.yaml <<EOF
listen_addr: ":443"
public_addr: "${IONSCALE_DOMAIN}:443"
stun_public_addr: "${IONSCALE_DOMAIN}:3478"

tls:
  acme: true
  acme_email: "${IONSCALE_ACME_EMAIL}"

keys:
  system_admin_key: "${IONSCALE_SYSTEM_ADMIN_KEY}"
  
database:
  url: "/data/ionscale.db?_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)"

logging:
  level: info
EOF
```

### Running the container

Start the ionscale container with the following command:

```bash
docker run -d \
  --name ionscale \
  --restart unless-stopped \
  -v $(pwd)/config.yaml:/etc/ionscale/config.yaml \
  -v $(pwd)/data:/data \
  -p 443:443 \
  -p 3478:3478/udp \
  ghcr.io/jsiebens/ionscale:0.17.0 server --config /etc/ionscale/config.yaml
```

This command:

- Creates a persistent container named "ionscale"
- Mounts your configuration file and data directory
- Maps the required ports to the host
- Automatically restarts the container if it stops unexpectedly
