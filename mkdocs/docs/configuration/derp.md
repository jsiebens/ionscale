# DERP Configuration

DERP (Designated Encrypted Relay for Packets) servers are relay servers that help Tailscale clients establish connections when direct connections aren't possible due to NAT, firewalls, or other network impediments. ionscale provides flexible DERP configuration options to support various deployment scenarios.

## Embedded DERP Server

ionscale includes a built-in DERP server that is enabled by default. This provides an out-of-the-box relay solution without requiring additional infrastructure.

### Configuration

The embedded DERP server can be configured in your ionscale configuration file:

```yaml
derp:
  server:
    disabled: false      # Set to true to disable the embedded DERP server
    region_id: 1000      # The region ID for the embedded DERP server
    region_code: "ionscale"  # The region code (short name)
    region_name: "ionscale Embedded DERP"  # Human-readable region name
```

Additional networking parameters that affect the DERP server:

```yaml
# The HTTP(S) listen address for the control plane (also used for DERP)
listen_addr: ":8080"

# The STUN listen address when using the embedded DERP
stun_listen_addr: ":3478"

# The DNS name of the server HTTP(S) endpoint as accessible by clients
public_addr: "ionscale.example.com:443"

# The DNS name of the STUN endpoint as accessible by clients
stun_public_addr: "ionscale.example.com:3478"
```

!!! important
    For the embedded DERP server to function properly, clients must be able to reach your ionscale server at the configured `public_addr` and `stun_public_addr`. Ensure these addresses are publicly accessible and have the appropriate ports open in your firewall.

## External DERP Sources

In addition to or instead of the embedded DERP server, ionscale can use external DERP servers. This is useful for:

- Using Tailscale's global DERP infrastructure
- Setting up your own geographically distributed DERP servers
- Optimizing connection paths for globally distributed teams

To configure external DERP sources:

```yaml
derp:
  sources:
    - https://controlplane.tailscale.com/derpmap/default  # Tailscale's default DERP map
    - https://example.com/my-custom-derpmap.json          # Custom DERP map
    - git::https://github.com/example/derpmap//config.json  # From a git repository
    - file:///etc/ionscale/derpmaps/custom.json           # From a local file
```

The `derp.sources` field accepts a list of URLs that point to JSON files containing DERP map configurations. These sources are loaded at server startup and merged with the embedded DERP configuration (if enabled).

!!! info "Source locations"
    ionscale uses HashiCorp's [go-getter](https://github.com/hashicorp/go-getter) library to fetch external sources, which supports multiple protocols and source types:
    
    - **HTTP/HTTPS**: `https://example.com/derpmap.json`
    - **Local files**: `file:///path/to/derpmap.json`
    - **Git repositories**: `git::https://github.com/user/repo//path/to/file.json`
    - **S3 buckets**: `s3::https://s3.amazonaws.com/bucket/derpmap.json`
    - **GCS buckets**: `gcs::https://www.googleapis.com/storage/v1/bucket/derpmap.json`
    
    This flexibility allows you to store and manage your DERP maps in various locations based on your organization's needs.
    
!!! note
    At the time of writing, ionscale only loads external DERP sources at startup and does not automatically poll them for changes. To apply changes to external DERP sources, you will need to restart the ionscale server.

## Instance and Tailnet DERP Configuration

ionscale provides a flexible DERP configuration model:

### Default Instance Configuration

By default, all tailnets use the DERP map defined at the instance level, which includes:

- The embedded DERP server (if enabled)
- Any external DERP sources configured in your ionscale configuration file

You can view the instance-level DERP map with:

```bash
ionscale system get-derp-map [--json]
```

### Tailnet-Specific Configuration

Each tailnet can be configured with its own custom DERP map. This gives you the flexibility to:

- Provide optimized DERP configurations for teams in different regions
- Test new DERP setups on specific tailnets before broader deployment
- Create specialized network paths for particular use cases

When a tailnet doesn't have a custom DERP map configured, it automatically uses the instance's default DERP map.

Managing tailnet-specific DERP maps:

```bash
# View a tailnet's current DERP map
ionscale tailnets get-derp-map --tailnet <tailnet-id> [--json]

# Set a custom DERP map for a specific tailnet
ionscale tailnets set-derp-map --tailnet <tailnet-id> --file <derpmap.json>

# Remove the custom configuration and revert to the instance default
ionscale tailnets reset-derp-map --tailnet <tailnet-id>
```

## Creating Custom DERP Maps

A DERP map is a JSON structure containing regions and nodes. Here's an example:

```json
{
  "Regions": {
    "1": {
      "RegionID": 1,
      "RegionCode": "nyc",
      "RegionName": "New York City",
      "Nodes": [
        {
          "Name": "1a",
          "RegionID": 1,
          "HostName": "derp1.example.com",
          "DERPPort": 443,
          "STUNPort": 3478
        }
      ]
    },
    "2": {
      "RegionID": 2,
      "RegionCode": "sfo",
      "RegionName": "San Francisco",
      "Nodes": [
        {
          "Name": "2a",
          "RegionID": 2,
          "HostName": "derp2.example.com",
          "DERPPort": 443,
          "STUNPort": 3478
        }
      ]
    }
  }
}
```

Important fields:

- `RegionID`: A unique identifier for the region (1-999 for custom regions, 1000+ reserved for ionscale)
- `RegionCode`: A short code for the region (e.g., "nyc", "sfo")
- `RegionName`: A human-readable name for the region
- `Nodes`: A list of DERP servers in the region
  - `Name`: A unique name for the node within the region
  - `HostName`: The DNS name of the DERP server
  - `DERPPort`: The port for DERP traffic (typically 443)
  - `STUNPort`: The port for STUN traffic (typically 3478)

To apply a custom DERP map to a tailnet:

```bash
# Create the DERP map file
cat > my-derpmap.json <<EOF
{
  "Regions": {
    "1": {
      "RegionID": 1,
      "RegionCode": "custom",
      "RegionName": "My Custom DERP",
      "Nodes": [
        {
          "Name": "derp1",
          "RegionID": 1,
          "HostName": "derp.example.com",
          "DERPPort": 443,
          "STUNPort": 3478
        }
      ]
    }
  }
}
EOF

# Apply it to a tailnet
ionscale tailnets set-derp-map --tailnet <tailnet-id> --file my-derpmap.json
```

!!! tip
    DERP servers primarily help with establishing connections when direct peer-to-peer connections aren't possible. Having DERP servers geographically close to your users can improve connection establishment times and provide better fallback performance.

!!! note
    Changes to DERP maps are automatically distributed to clients during their regular polling. There's no need to manually update clients when changing DERP configurations.