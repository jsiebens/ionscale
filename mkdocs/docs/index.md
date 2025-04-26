# Welcome to ionscale

ionscale is an open-source alternative to Tailscale's control server, designed to provide a self-hosted coordination service for your Tailscale networks.

!!! info "Beta status"
    ionscale is currently in beta. While it's stable for production use for small tailnets, we're actively developing new features and improvements.

!!! warning "Documentation status"
    This documentation is a work in progress. Some sections may be incomplete or missing. We're continuously improving the documentation to provide comprehensive coverage of all features.

## What is ionscale?

Tailscale allows your devices to communicate securely across networks using WireGuard®. While Tailscale's client software is open source, their centralized coordination server (which manages public keys and network configurations) is proprietary.

**ionscale aims to implement a lightweight, open-source control server that:**

- Acts as a drop-in replacement for Tailscale's coordination server
- Can be self-hosted on your infrastructure
- Gives you full control over your network configuration
- Works with the standard Tailscale clients
- Supports a [wide range of Tailscale features](./overview/features.md)

## Getting started

- [**Installation guide**](./installation/index.md) - Install ionscale using Docker or directly on Linux
- [**CLI configuration**](./getting-started/index.md) - Set up the ionscale CLI and authenticate
- [**Creating a tailnet**](./getting-started/tailnet.md) - Create and manage your first tailnet
- [**OIDC authentication**](./configuration/auth-oidc.md) - Configure user authentication via OIDC
- [**DNS providers**](./configuration/dns-providers.md) - Set up DNS integration for HTTPS certificates

<small>
Disclaimer: This is not an official Tailscale or Tailscale Inc. project. Tailscale and WireGuard are trademarks of their respective owners.
</small>