# Configuring DNS providers

ionscale supports integration with various DNS providers to enable Tailscale's HTTPS certificate functionality. When a DNS provider is properly configured, ionscale can automatically manage TXT records required for the DNS-01 challenge when requesting certificates.

## Why configure a DNS provider

While not strictly required for basic ionscale operation, configuring a DNS provider enables important Tailscale features:

1. **Tailscale HTTPS Certificates**: Allows nodes to receive valid HTTPS certificates for their Tailscale hostnames, enabling secure web services within your tailnet.
2. **Tailscale Serve**: Supports the `tailscale serve` feature, which allows users to easily share web services with proper HTTPS.

Without a configured DNS provider, these features will not be available to your users.

## Supported DNS providers

ionscale uses the [libdns](https://github.com/libdns) libraries to support the following DNS providers:

- [Azure DNS](https://github.com/libdns/azure)
- [Cloudflare](https://github.com/libdns/cloudflare)
- [DigitalOcean](https://github.com/libdns/digitalocean)
- [Google Cloud DNS](https://github.com/libdns/googleclouddns)
- [Amazon Route 53](https://github.com/libdns/route53)

!!! info "Provider availability"
    These are the DNS providers supported at the time of this writing. Since ionscale uses the libdns ecosystem, additional providers may be added in future releases as the ecosystem expands.
    
    If you need support for a different DNS provider, check the [libdns GitHub organization](https://github.com/libdns) for newly available providers or consider contributing an implementation for your provider.

!!! note "Future plugin system"
    A plugin system for DNS providers is currently in the ideation phase. This would make it significantly easier to add and configure additional DNS providers without modifying the core ionscale code. Stay tuned for updates on this feature in future releases.

## DNS provider configuration

To configure a DNS provider, add the appropriate settings to your ionscale configuration file (`config.yaml`):

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

### Important requirements

1. The `magic_dns_suffix` must be a subdomain of the provider's zone (or the zone itself).
2. MagicDNS must be enabled for the tailnet to use HTTPS certificates.
3. You must have administrative access to the DNS zone to configure the provider.

## Provider-specific examples

### Cloudflare

```yaml
dns:
  magic_dns_suffix: "ts.example.com"
  provider:
    name: "cloudflare"
    zone: "example.com"
    config:
      auth_token: "your-cloudflare-api-token"
      # OR use email/api_key authentication 
      # email: "your-email@example.com"
      # api_key: "your-global-api-key"
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
      project: "your-project-id"
      # Either provide service account JSON directly
      service_account_json: '{"type":"service_account","project_id":"your-project",...}'
      # OR specify a path to a service account key file
      # service_account_key_file: "/path/to/service-account-key.json"
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
      token: "your-digitalocean-api-token"
```

For DigitalOcean, create an API token with read and write access.

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

1. **Check DNS Provider Credentials**: Ensure the credentials in your configuration have sufficient permissions.
2. **Verify Zone Configuration**: Make sure the zone in your configuration matches your actual DNS zone.
3. **Check MagicDNS Settings**: Confirm that `magic_dns_suffix` is properly configured as a subdomain of your zone.
4. **Review Server Logs**: The ionscale server logs may contain error messages related to DNS provider integration.
5. **Test DNS Record Creation**: Some providers offer tools to test API access for creating and updating DNS records.

