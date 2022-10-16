# Getting started on a Linux Server

This tutorial will guide you through the steps needed to install and run __ionscale__ on a Linux machine.

## Prerequisites 

- A Linux machine with port 80 and 443 open to ingress traffic.
- A registered domain name.

## Step 1. Configure DNS

Set up a `A` DNS records: `ionscale.example.com` (We are assuming that your domain name is example.com.)

!!! tip

    You can use `dig` to make sure that DNS records are propagated:

    ``` bash
    $ dig ionscale.example.com
    ```

## Step 2. Set up ionscale on your Linux host

### Prepare installation

Run the following commands to prepare the installation:

``` bash
sudo mkdir -p /etc/ionscale
sudo mkdir -p /var/lib/ionscale

sudo useradd --system --no-create-home --shell /bin/false ionscale
sudo chown ionscale:ionscale /etc/ionscale
sudo chown ionscale:ionscale /var/lib/ionscale
```

### Install ionscale

Run the following commands to install the __ionscale__ binary on your Linux host:

``` bash
sudo curl \
    -o "/usr/local/bin/ionscale" \
    -sfL "https://github.com/jsiebens/ionscale/releases/download/v0.4.0/ionscale_linux_amd64"

sudo chmod +x "/usr/local/bin/ionscale"
```

### Configure ionscale

Generate a system admin key for __ionscale__ using the `ionscale genkey` command and write it the an environment file:

``` bash
sudo tee /etc/default/ionscale >/dev/null <<EOF
IONSCALE_KEYS_SYSTEM_ADMIN_KEY=$(ionscale genkey -n)
EOF
```

Generate a configuration file for __ionscale__ with the following commands:

``` bash
export IONSCALE_DOMAIN=example.com
export IONSCALE_ACME_EMAIL=<your email>
```

``` bash
sudo tee /etc/ionscale/config.yaml >/dev/null <<EOF
http_listen_addr: ":80"
https_listen_addr: ":443"
server_url: "https://${IONSCALE_DOMAIN}"

tls:
  acme: true
  acme_email: "${IONSCALE_ACME_EMAIL}"
  acme_path: "/var/lib/ionscale/acme"

database:
  url: "/var/lib/ionscale/ionscale.db?_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)"

logging:
  level: info
EOF
```

Create a systemd service file for __ionscale__ with the following commands:

``` bash
sudo tee /etc/systemd/system/ionscale.service >/dev/null <<EOF
[Unit]
Description=ionscale - a Tailscale Controller server
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

### Start ionscale

On your Linux machine, run the following commands to enable and start the __ionscale__ daemon:

``` bash
sudo systemctl daemon-reload
sudo systemctl enable ionscale
sudo systemctl start ionscale
```
