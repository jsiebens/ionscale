# Supported features

ionscale implements key Tailscale features to provide a complete control server experience. This page outlines the major features that ionscale supports.

## Multi-tailnet support

ionscale allows you to create and manage multiple tailnets (Tailscale private networks) from a single server:

- Create isolated networks for different teams or environments
- Manage separate tailnets for personal and organizational use
- Configure each tailnet with its own ACLs and settings

## User management

- **Multi-user support**: Multiple users can access and use the same tailnet based on permissions
- **OIDC integration**: Optional but recommended for user authentication and management

## Authentication

- **[Auth keys](https://tailscale.com/kb/1085/auth-keys/)**: Generate and manage pre-authentication keys for devices
- **[Device tagging](https://tailscale.com/kb/1068/tags/)**: Apply tags to devices for better management and ACL control

## Network control

- **[Access control lists (ACLs)](https://tailscale.com/kb/1018/acls/)**: Define fine-grained rules for who can access what
- **[Subnet routers](https://tailscale.com/kb/1019/subnets/)**: Connect existing networks to your tailnet
- **[Exit nodes](https://tailscale.com/kb/1103/exit-nodes/)**: Configure nodes to act as VPN exit points

## DNS management

- **[MagicDNS](https://tailscale.com/kb/1081/magicdns/)**: Automatic DNS for tailnet devices
- **[Split DNS](https://tailscale.com/kb/1054/dns/)**: Route specific DNS queries to specific resolvers
- **Custom nameservers**: Configure any DNS servers for your tailnet

## HTTPS and certificates

- **[HTTPS certificates](https://tailscale.com/kb/1153/enabling-https/)**: Automatic SSL/TLS certificates for devices
- **DNS provider integration**: Support for various DNS providers to facilitate ACME challenges
- **[Tailscale Serve](https://tailscale.com/kb/1242/tailscale-serve/)**: Share web services easily with HTTPS

## SSH

- **[Tailscale SSH](https://tailscale.com/kb/1193/tailscale-ssh/)**: Built-in SSH server support
- **SSH policy management**: Control who can SSH into which devices

## DERP

- **Embedded DERP server**
- **Custom DERP maps**: Configure your own DERP servers

## File sharing

- **[Taildrop](https://tailscale.com/kb/1106/taildrop/)**: Send files directly between tailnet devices

## Feature status

Most features are fully implemented and compatible with the official Tailscale clients. As ionscale is continuously developed, new features from Tailscale are regularly added.

If you find any issues with specific features or have requests for additional functionality, please check the [project repository](https://github.com/jsiebens/ionscale) for the latest updates or to submit feedback.