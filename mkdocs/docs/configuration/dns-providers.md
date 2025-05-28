# Configuring DNS providers

ionscale supports integration with various DNS providers to enable Tailscale's HTTPS certificate functionality. When a DNS provider is properly configured, ionscale can automatically manage TXT records required for the DNS-01 challenge when requesting certificates.

## Why configure a DNS provider

While not strictly required for basic ionscale operation, configuring a DNS provider enables important Tailscale features:

1. **Tailscale HTTPS Certificates**: Allows nodes to receive valid HTTPS certificates for their Tailscale hostnames, enabling secure web services within your tailnet.
2. **Tailscale Serve**: Supports the `tailscale serve` feature, which allows users to easily share web services with proper HTTPS.

Without a configured DNS provider, these features will not be available to your users.

## Supported DNS providers

ionscale supports DNS providers through two methods:

### Built-in providers (deprecated)

ionscale includes built-in support for the following DNS providers using [libdns](https://github.com/libdns) libraries:

- [Azure DNS](https://github.com/libdns/azure)
- [Cloudflare](https://github.com/libdns/cloudflare)
- [DigitalOcean](https://github.com/libdns/digitalocean)
- [Google Cloud DNS](https://github.com/libdns/googleclouddns)
- [Amazon Route 53](https://github.com/libdns/route53)

!!! warning "Built-in providers are deprecated"
    The built-in DNS providers are deprecated and will be removed in a future release. Please migrate to external DNS plugins for continued support.

### External DNS plugins (recommended)

ionscale now supports external DNS plugins through a plugin system. This allows for:

- **Extensibility**: Add support for any DNS provider without modifying ionscale
- **Maintainability**: Plugins are maintained independently
- **Flexibility**: Plugin configuration specific to each provider's needs

!!! info "Plugin availability"
    External DNS plugins implement the [libdns-plugin](https://github.com/libdns/libdns-plugin) interface. Official plugin implementations can be found in the [ionscale GitHub organization](https://github.com/ionscale) with repositories named `ionscale-<provider>-dns`. You can also create your own following the plugin specification.

## DNS provider configuration

To configure a DNS provider, add the appropriate settings to your ionscale configuration file (`config.yaml`):

### Built-in provider configuration

```yaml
dns:
  # The base domain for MagicDNS FQDN hostnames
  magic_dns_suffix: "ionscale.net"
  
  # DNS provider configuration for HTTPS certificates
  provider:
    # Name of your DNS provider
    name: "cloudflare"
    # The DNS zone to use (typically your domain name)
    zone: "example.com"
    # Provider-specific configuration (varies by provider)
    config:
      # Provider-specific credentials and settings go here
      # See provider-specific examples below
```

### External plugin configuration

```yaml
dns:
  # The base domain for MagicDNS FQDN hostnames
  magic_dns_suffix: "ionscale.net"
  
  # DNS provider configuration for HTTPS certificates
  provider:
    # Path to your DNS plugin executable
    plugin_path: "/path/to/your/dns-plugin"
    # The DNS zone to use (typically your domain name)
    zone: "example.com"
    # Plugin-specific configuration (varies by plugin)
    config:
      # Plugin-specific credentials and settings go here
      # See plugin documentation for configuration options
```

### Important requirements

1. The `magic_dns_suffix` must be a subdomain of the provider's zone (or the zone itself).
2. MagicDNS must be enabled for the tailnet to use HTTPS certificates.
3. You must have administrative access to the DNS zone to configure the provider.
4. For external plugins, the plugin executable must be accessible and executable by the ionscale process.

## Provider-specific examples

### Cloudflare

```yaml
dns:
  magic_dns_suffix: "ts.example.com"
  provider:
    name: "cloudflare"
    zone: "example.com"
    config:
      api_token: "your-cloudflare-api-token"
```

For Cloudflare, create an API token with the "Edit" permission for "Zone:DNS".

### Azure DNS

```yaml
dns:
  magic_dns_suffix: "ts.example.com"
  provider:
    name: "azure"
    zone: "example.com"
    config:
      tenant_id: "your-tenant-id"
      client_id: "your-client-id"
      client_secret: "your-client-secret"
      subscription_id: "your-subscription-id"
      resource_group_name: "your-resource-group"
```

For Azure, create a service principal with "DNS Zone Contributor" role for your DNS zone's resource group.

### Amazon Route 53

```yaml
dns:
  magic_dns_suffix: "ts.example.com"
  provider:
    name: "route53"
    zone: "example.com"
    config:
      access_key_id: "your-access-key-id"
      secret_access_key: "your-secret-access-key"
      # Optional region, defaults to us-east-1
      region: "us-east-1"
      ...
```

For AWS Route 53, create an IAM user with permissions to modify records in your hosted zone.

### Google Cloud DNS

```yaml
dns:
  magic_dns_suffix: "ts.example.com"
  provider:
    name: "googleclouddns"
    zone: "example-com" # Note: GCP uses zone names without dots
    config:
      gcp_project: "your-project-id"
      # Optional path to a service account key file
      # gcp_application_default: "/path/to/service-account-key.json"
```

For Google Cloud DNS, create a service account with the "DNS Administrator" role.

### DigitalOcean

```yaml
dns:
  magic_dns_suffix: "ts.example.com"
  provider:
    name: "digitalocean"
    zone: "example.com"
    config:
      auth_token: "your-digitalocean-api-token"
```

For DigitalOcean, create an API token with read and write access.

## External DNS plugin examples

### Using a hypothetical Cloudflare plugin

```yaml
dns:
  magic_dns_suffix: "ts.example.com"
  provider:
    plugin_path: "/usr/local/bin/libdns-cloudflare-plugin"
    zone: "example.com"
    config:
      api_token: "your-cloudflare-api-token"
```

### Using a custom DNS plugin

```yaml
dns:
  magic_dns_suffix: "ts.example.com"
  provider:
    plugin_path: "/opt/dns-plugins/my-custom-provider"
    zone: "example.com"
    config:
      # Configuration specific to your custom plugin
      endpoint: "https://api.mydnsprovider.com"
      api_key: "your-api-key"
      custom_setting: "value"
```

!!! tip "Plugin development"
    To create your own DNS plugin, implement the [libdns-plugin](https://github.com/libdns/libdns-plugin) interface. The plugin system uses HashiCorp's go-plugin framework for communication between ionscale and your plugin.

## Enabling HTTPS certificates for a tailnet

After configuring a DNS provider in your ionscale server, you can enable HTTPS certificates for a tailnet:

```bash
# Enable MagicDNS and HTTPS certificates for a tailnet
ionscale dns update --tailnet "my-tailnet" --magic-dns --https-certs
```

## Verifying DNS provider configuration

To verify that your DNS provider is correctly configured:

1. Configure a tailnet with MagicDNS and HTTPS certificates enabled.
2. Connect a device to the tailnet.
3. On the device, run:
   ```bash
   tailscale cert <hostname>
   ```
4. If successful, the command will obtain a certificate for the hostname.
5. You should see TXT records created in your DNS zone for the ACME challenge.

## Troubleshooting

If you encounter issues with DNS provider integration:

### General troubleshooting

1. **Check DNS Provider Credentials**: Ensure the credentials in your configuration have sufficient permissions.
2. **Verify Zone Configuration**: Make sure the zone in your configuration matches your actual DNS zone.
3. **Check MagicDNS Settings**: Confirm that `magic_dns_suffix` is properly configured as a subdomain of your zone.
4. **Review Server Logs**: The ionscale server logs may contain error messages related to DNS provider integration.
5. **Test DNS Record Creation**: Some providers offer tools to test API access for creating and updating DNS records.

### External plugin troubleshooting

1. **Plugin Executable**: Ensure the plugin path is correct and the executable has proper permissions.
2. **Plugin Logs**: Check both ionscale logs and any plugin-specific logs for error messages.
3. **Plugin Health**: ionscale automatically restarts failed plugins, but persistent failures may indicate configuration issues.

