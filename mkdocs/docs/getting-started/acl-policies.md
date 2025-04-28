# ACL Policies

Access Control Lists (ACLs) define what network access is allowed within a tailnet. By default, tailnets are created with an open policy that allows all connections between devices.

## Understanding ACL policies

ACL policies in ionscale follow the same format and rules as Tailscale's ACL system. They allow you to control:

- Which devices can communicate with each other
- What ports and protocols are allowed
- Who can use exit nodes and other special features
- SSH access between machines
- Tag ownership and management

## Basic ACL structure

A basic ACL policy contains rules that specify which sources can access which destinations:

```json
{
  "acls": [
    {"action": "accept", "src": ["tag:web"], "dst": ["tag:db:5432"]},
    {"action": "accept", "src": ["group:admins"], "dst": ["*:*"]}
  ],
  "groups": {
    "admins": ["admin@example.com"]
  },
  "tagOwners": {
    "tag:web": ["admin@example.com"],
    "tag:db": ["admin@example.com"]
  }
}
```

In this example:
- Web servers (tagged `tag:web`) can only access database servers on port 5432
- Admins have full access to all resources
- Only admin@example.com can assign the web and database tags to machines

## Managing ACL policies

You can view and update the ACL policy for a tailnet using the ionscale CLI:

```bash
# View current ACL policy
ionscale acl get --tailnet "my-tailnet"

# Update ACL policy from a file
ionscale acl update --tailnet "my-tailnet" --file acl.json
```

!!! tip
    ACL changes take effect immediately for all devices in the tailnet.

## Common ACL patterns

### Allow specific tags to communicate

```json
{
  "acls": [
    {"action": "accept", "src": ["tag:web"], "dst": ["tag:api:8080"]},
    {"action": "accept", "src": ["tag:api"], "dst": ["tag:db:5432"]}
  ]
}
```

### Group-based access

```json
{
  "acls": [
    {"action": "accept", "src": ["group:developers"], "dst": ["tag:dev-env:*"]},
    {"action": "accept", "src": ["group:ops"], "dst": ["*:*"]}
  ],
  "groups": {
    "developers": ["alice@example.com", "bob@example.com"],
    "ops": ["charlie@example.com", "diana@example.com"]
  }
}
```

### SSH access control

```json
{
  "ssh": [
    {
      "action": "accept",
      "src": ["group:admins"],
      "dst": ["tag:server"],
      "users": ["root"]
    },
    {
      "action": "accept",
      "src": ["group:developers"],
      "dst": ["tag:dev"],
      "users": ["autogroup:nonroot"]
    }
  ]
}
```

### Auto-approving advertised routes

```json
{
  "autoApprovers": {
    "routes": {
      "10.0.0.0/24": ["group:network-admins"],
      "192.168.1.0/24": ["user@example.com"]
    },
    "exitNode": ["group:network-admins"]
  }
}
```

## Additional resources

For more detailed information on ACL syntax and capabilities, see the [Tailscale ACL documentation](https://tailscale.com/kb/1018/acls/).

!!! note "Feature support"
    Not all ACL features from the official Tailscale control plane are supported in ionscale. Some advanced features or newer functionality may not be available.