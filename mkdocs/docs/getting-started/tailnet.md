# Creating your first tailnet

A tailnet is a private network that connects your devices securely using Tailscale. This guide will walk you through creating your first tailnet with ionscale.

## Prerequisites

Before creating a tailnet, make sure you have:

- ionscale server installed and running
- The ionscale CLI installed and configured
- Authentication with system administrator privileges

## Creating a tailnet

The simplest way to create a tailnet is with the `tailnet create` command:

```bash
ionscale tailnet create --name "my-first-tailnet"
```

This creates a basic tailnet named "my-first-tailnet" with:

- A default ACL policy that allows all connections 
```
{"acls": [{"action": "accept", "src": ["*"], "dst": ["*:*"]}]}
```
- A default IAM policy that determines who can access the tailnet

!!! note
    The tailnet name must be unique within your ionscale instance and should only contain alphanumeric characters, hyphens, and underscores.

## Setting IAM policies for access control

!!! important "OIDC required"
    IAM policies are only relevant when an OIDC provider is configured. If your ionscale instance isn't using OIDC, access to tailnets is managed solely through auth keys, and the configuration in this section won't apply.

When using OIDC authentication, you'll need to configure who can access your tailnet through IAM policies.

### Configuring IAM during tailnet creation

You can set a basic IAM policy when creating a tailnet using flags:

```bash
# Allow all users with an @example.com email address
ionscale tailnet create --name "company-tailnet" --domain "example.com"

# Allow only a specific user
ionscale tailnet create --name "personal-tailnet" --email "user@example.com"
```

These flags provide quick ways to set up common IAM policies:

- `--domain example.com`: Creates a **shared tailnet** that allows all users with an email address from the specified domain. This is ideal for company or team networks where multiple users need access. The IAM policy will contain a filter rule like `domain == "example.com"`.

- `--email user@example.com`: Creates a **personal tailnet** that grants access only to the specific email address. This is suitable for individual use or when you want to tightly control access to a specific user. The IAM policy will contain an entry for the specified email.

!!! note
    You can't use both flags together. Choose either domain-based access for a shared network or email-based access for a personal network.

### Configuring IAM after tailnet creation

You can view and update the IAM policy for an existing tailnet:

```bash
# View current IAM policy
ionscale iam get-policy --tailnet "my-first-tailnet"

# Update IAM policy using a JSON file
ionscale iam update-policy --tailnet "my-first-tailnet" --file policy.json
```

Example policy.json file:
```json
{
  "filters": ["domain == example.com"],
  "emails": ["specific-user@otherdomain.com"],
  "roles": {
    "admin@example.com": "admin"
  }
}
```

## Connecting devices to your tailnet

There are two main methods to connect devices to your tailnet:

!!! tip
    For detailed instructions on configuring various Tailscale clients to use ionscale as a control server, refer to the [Tailscale Knowledge Base](https://tailscale.com/kb/1507/custom-control-server).

### Interactive login

When you have an OIDC provider configured, users can connect to their tailnet through an interactive web authentication flow:

```bash
tailscale up --login-server=https://ionscale.example.com
```

This opens a browser window where the user authenticates with the OIDC provider. After successful authentication, if the user has access based on the tailnet's IAM policy, the device will be connected to the tailnet.

!!! note
    Interactive login requires an OIDC provider to be configured on your ionscale instance.

### Using pre-authentication keys

Pre-authentication keys (auth keys) allow devices to join a tailnet without interactive authentication. This is useful for automated deployments, servers, or environments where browser-based authentication isn't practical.

To create an auth key:

```bash
# Create an auth key
ionscale auth-key create --tailnet "my-first-tailnet"

# Create an auth key with specific tags
ionscale auth-key create --tailnet "my-first-tailnet" --tags "tag:server"
```

The tags assigned to the key will determine what network access the device has once connected, based on your ACL rules.

!!! note
    In environments with OIDC, users with access to a tailnet can create auth keys for that tailnet. Without OIDC, only system administrators can create keys.

To connect a device using an auth key:

```bash
# Connect using the auth key
tailscale up --login-server=https://ionscale.example.com --auth-key=...
```

## Network access and security policies

By default, tailnets are created with an open policy that allows all connections between devices. For production environments, you'll want to configure:

- **[IAM Policies](iam-policies.md)**: Manage who can access your tailnet
- **[ACL Policies](acl-policies.md)**: Control which devices can communicate within your tailnet

!!! tip
    For detailed information on configuring security policies, see the dedicated documentation sections on [IAM Policies](iam-policies.md) and [ACL Policies](acl-policies.md).

## Managing multiple tailnets

You can create multiple tailnets to separate different network environments:

```bash
# List all tailnets
ionscale tailnet list

# Create tailnets for different teams or businesses
ionscale tailnet create --name "tailnet-a"
ionscale tailnet create --name "tailnet-b"
ionscale tailnet create --name "tailnet-c"
```

!!! note
    Each tailnet is a separate network with its own devices, ACLs, and IAM policies. Devices in different tailnets cannot communicate with each other by default.