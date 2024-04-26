# Getting started with Docker

You can install and run __ionscale__ using the Docker images published on [GitHub Container Registry](https://github.com/jsiebens/ionscale/pkgs/container/ionscale).

## Requirements 

- A Linux machine with port 443 and 3478 open to ingress traffic.
- Docker installed. See the [official installation documentation](https://docs.docker.com/install/)
- A registered domain name.

## Step 1. Configure DNS

Set up a `A` DNS records: `ionscale.example.com` (We are assuming that your domain name is example.com.)

!!! tip

    You can use `dig` to make sure that DNS records are propagated:

    ``` bash
    $ dig ionscale.example.com
    ```

## Step 2. Run ionscale with Docker

### Configure ionscale

``` bash
mkdir -p ionscale/data
cd ./ionscale
```

Generate a configuration file for __ionscale__ with the following commands:

``` bash
export IONSCALE_ACME_EMAIL=<your email>
export IONSCALE_DOMAIN=ionscale.example.com
export IONSCALE_SYSTEM_ADMIN_KEY=$(docker run ghcr.io/jsiebens/ionscale:0.15.0 genkey -n)
```

``` bash
tee ./config.yaml >/dev/null <<EOF
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

### Start ionscale

Run an __ionscale__ instance with the following command:

``` bash
docker run \
  -v $(pwd)/config.yaml:/etc/ionscale/config.yaml \
  -v $(pwd)/data:/data \
  -p 443:443 \
  -p 3478:3478/udp \
  ghcr.io/jsiebens/ionscale:0.15.0 server --config /etc/ionscale/config.yaml
```