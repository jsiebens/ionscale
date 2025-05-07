# Getting started with ionscale

After installing ionscale, you'll need to configure the CLI to interact with your server. This guide will walk you through the initial setup and explain the authentication options available.

## Installing the ionscale CLI

The ionscale CLI is the primary tool for managing your ionscale instance. It allows you to create and manage tailnets, users, and access controls.

```bash
# Download the CLI (adjust the URL for your system architecture)
curl -L -o ionscale https://github.com/jsiebens/ionscale/releases/download/v0.17.0/ionscale_linux_amd64

# Make it executable
chmod +x ionscale

# Move to system path
sudo mv ionscale /usr/local/bin/
```

## Authentication requirements

To use the ionscale CLI, you must authenticate with the server using one of two methods:

!!! important "Administrator access required"
    All management operations require either:
    
    1. **System admin key** authentication, or
    2. **OIDC user** authentication with system administrator privileges

### Option 1: Using the system admin key

If you configured ionscale with a system admin key during installation, you can authenticate using that key:

```bash
# Configure environment variables
export IONSCALE_ADDR="https://ionscale.example.com"
export IONSCALE_SYSTEM_ADMIN_KEY="your-system-admin-key"

# Verify connection
ionscale version
```

The system admin key provides full administrative access to your ionscale instance. This is the default authentication method when OIDC is not configured.

### Option 2: Using OIDC authentication

If you configured ionscale with an OIDC provider, users designated as system administrators in the OIDC configuration can authenticate:

```bash
# Configure URL only
export IONSCALE_ADDR="https://ionscale.example.com"

# Authenticate through OIDC
ionscale auth login
```

This will open a browser window where you can authenticate with your OIDC provider. After successful authentication, if your account has system administrator privileges, you'll be able to use the CLI.

!!! tip "OIDC system administrators"
    System administrators are defined in the ionscale configuration under the `auth.system_admins` section. See the [Authentication with OIDC](../configuration/auth-oidc.md) documentation for details.

## Basic CLI commands

Once authenticated, you can use the ionscale CLI to manage your instance:

```bash
# View general information
ionscale version                # Show version information
ionscale help                   # Display help information

# Tailnet management
ionscale tailnet list           # List all tailnets
ionscale tailnet create -n NAME # Create a new tailnet

# Auth key management
ionscale auth-key list --tailnet NAME    # List auth keys for a tailnet
ionscale auth-key create --tailnet NAME  # Create a new auth key
```