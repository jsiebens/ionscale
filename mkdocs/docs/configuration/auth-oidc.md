# Configuring authentication with OIDC

While ionscale can operate without an OIDC (OpenID Connect) provider using only static keys, configuring an OIDC provider is highly recommended for enhanced security, user management, and a smoother administrative experience.

## Why configure an OIDC provider?

Without an OIDC provider, ionscale operates in a key-only mode:

- System administrators can only use the system admin key for administrative tasks
- Tailscale devices can only connect using [tags](https://tailscale.com/kb/1068/tags) and [pre-authentication keys](https://tailscale.com/kb/1085/auth-keys)
- No user accounts or user-specific permissions are available

With an OIDC provider configured:

- Users can authenticate using their existing identity provider credentials
- Administrators can assign specific permissions to users
- System administration can be delegated to specific users
- Fine-grained access control based on user identity becomes possible
- More seamless user experience with browser-based authentication

## Supported OIDC providers

ionscale supports any standard OIDC provider, including:

- Google Workspace
- Auth0
- Okta
- Azure AD / Microsoft Entra ID
- Keycloak
- GitLab
- GitHub (OAuth 2.0 with OIDC extensions)
- And many others

## Basic OIDC configuration

To configure an OIDC provider, update your ionscale configuration file (`config.yaml`) with the following settings:

```yaml
auth:
  provider:
    # OIDC issuer URL where ionscale can find the OpenID Provider Configuration Document
    issuer: "https://your-oidc-provider.com"
    # OIDC client ID and secret
    client_id: "your-client-id"
    client_secret: "your-client-secret"
    # Optional: additional OIDC scopes used in the OIDC flow
    additional_scopes: "groups"
```

### Required configuration fields

- `issuer`: The URL to your OIDC provider's issuer. This URL is used to discover your provider's endpoints.
- `client_id`: The client ID from your OIDC provider application registration.
- `client_secret`: The client secret from your OIDC provider application registration.

### Optional configuration

- `additional_scopes`: A space-separated list of additional OAuth scopes to request during authentication.
  By default, ionscale requests the `openid`, `email`, and `profile` scopes.

## Configuring your OIDC provider

When registering ionscale with your OIDC provider, you'll need to configure the following:

**Redirect URI**: Set to `https://your-ionscale-domain.com/auth/callback`

## System administrator access

With OIDC configured, you'll want to specify which users have system administrator privileges.
This is done in the `auth.system_admins` section of your configuration:

```yaml
auth:
  # Configuration from previous section...

  system_admins:
    # By email address
    emails:
      - "admin@example.com"
      - "secadmin@example.com"
    
    # By subject identifier (sub claim from OIDC)
    subs:
      - "user|123456"
    
    # By attribute expression (using BEXPR syntax)
    filters:
      - "token.groups contains \"admin\""
      - "domain == \"admin.example.com\""
```

You can use one or more of these methods to designate system administrators:

1. **By Email**: List specific email addresses that should have admin privileges.
2. **By Subject ID**: List specific user IDs (the `sub` claim from your OIDC provider).
3. **By Expression**: Use BEXPR filters to determine admin status based on token claims.

## OIDC authentication flow

When a user attempts to authenticate with ionscale:

1. The user is redirected to the OIDC provider's login page.
2. After successful authentication, the user is redirected back to ionscale.
3. ionscale verifies the authentication and checks if:
   - The user is a system administrator (based on the `system_admins` configuration).
   - The user has access to any tailnets (based on IAM policies configured for individual tailnets).

## Provider-specific setup instructions

### Google

1. Go to the [Google Cloud Console](https://console.cloud.google.com/).
2. Create a new project or select an existing one.
3. Navigate to "APIs & Services" > "Credentials".
4. Click "Create Credentials" > "OAuth client ID".
5. Select "Web application" as the application type.
6. Add `https://your-ionscale-domain.com/auth/callback` as an authorized redirect URI.
7. Copy the client ID and client secret.

Configure ionscale:
```yaml
auth:
  provider:
    issuer: "https://accounts.google.com"
    client_id: "your-client-id.apps.googleusercontent.com"
    client_secret: "your-client-secret"
```

### Auth0

1. Go to the [Auth0 Dashboard](https://manage.auth0.com/).
2. Create a new application or select an existing one of type "Regular Web Application".
3. Under "Settings", configure:
   - Allowed Callback URLs: `https://your-ionscale-domain.com/auth/callback`
4. Copy the Domain, Client ID, and Client Secret.

Configure ionscale:
```yaml
auth:
  provider:
    issuer: "https://your-tenant.auth0.com/"
    client_id: "your-client-id"
    client_secret: "your-client-secret"
```

### Microsoft Azure AD / Entra ID

1. Go to the [Azure Portal](https://portal.azure.com/).
2. Navigate to "Azure Active Directory" > "App registrations".
3. Create a new registration.
4. Add `https://your-ionscale-domain.com/auth/callback` as a redirect URI of type "Web".
5. Under "Certificates & secrets", create a new client secret.
6. Copy the Application (client) ID and the new secret.

Configure ionscale:
```yaml
auth:
  provider:
    issuer: "https://login.microsoftonline.com/your-tenant-id/v2.0"
    client_id: "your-client-id"
    client_secret: "your-client-secret"
    additional_scopes: "offline_access"
```

## Complete configuration example

```yaml
auth:
  provider:
    issuer: "https://accounts.google.com"
    client_id: "your-client-id.apps.googleusercontent.com"
    client_secret: "your-client-secret"
    additional_scopes: "groups"
  
  system_admins:
    emails:
      - "admin@example.com"
    filters:
      - "domain == \"example.com\" && token.groups contains \"admin\""
```

## OIDC without system admin

If you've configured OIDC but no system administrators, you can still use the system admin key from your initial setup for administrative tasks:

```bash
export IONSCALE_ADDR="https://your-ionscale-domain.com"
export IONSCALE_SYSTEM_ADMIN_KEY="your-system-admin-key"
ionscale tailnet list
```

