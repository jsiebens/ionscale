# Installation guide

ionscale can be installed in several ways, depending on your preferences and requirements. This section covers the different installation methods available.

## Choose your installation method

ionscale offers two primary installation methods:

### Docker installation

The [Docker installation](docker.md) method is recommended for:

- Quick deployments
- Testing and evaluation
- Users familiar with container environments
- Simplified upgrades and maintenance

Docker provides an isolated environment with all dependencies included, making it the easiest way to get started with ionscale.

### Linux installation

The [Linux installation](linux.md) method is suitable for:

- Production environments
- Integration with existing infrastructure
- More control over the installation
- Systems without Docker

This approach installs ionscale directly on your Linux server and configures it as a system service.

## Post-installation steps

After completing the installation, consider these next steps:

1. Configure an [OIDC provider](../configuration/auth-oidc.md) for user authentication
2. Set up a [DNS provider](../configuration/dns-providers.md) to enable HTTPS certificates for Tailscale nodes
3. Create and configure tailnets
4. Set up access controls and permissions