# IAM Policies

Identity and Access Management (IAM) policies in ionscale control who can access a tailnet and what administrative permissions they have.

!!! important "OIDC required"
    IAM policies are only relevant when an OIDC provider is configured. If your ionscale instance isn't using OIDC, access to tailnets is managed solely through auth keys.

## Understanding IAM policies

IAM policies determine:

- Which users can join a tailnet
- What roles and permissions users have within the tailnet
- How access decisions are made based on user attributes

An IAM policy consists of:

```json
{
  "subs": ["auth0|123456789"],
  "filters": ["domain == example.com"],
  "emails": ["specific-user@otherdomain.com"],
  "roles": {
    "admin@example.com": "admin"
  }
}
```

## IAM policy components

### Subs

The `subs` list provides direct access based on user IDs (subjects):

```json
"subs": ["auth0|123456789", "google-oauth2|12345"]
```

Any user whose ID matches an entry in this list will be granted access to the tailnet. User IDs are typically provided by the OIDC provider and are unique identifiers for each user.

### Emails

The `emails` list provides direct access to specific email addresses:

```json
"emails": ["alice@example.com", "bob@otherdomain.com"]
```

Any user with an email in this list will be granted access to the tailnet, regardless of filters.

### Filters

Filters are expressions that evaluate user attributes:

```json
"filters": ["domain == example.com", "email.endsWith('@engineering.example.com')"]
```

These expressions determine if a user can access the tailnet based on their identity attributes. Users matching any filter expression will be granted access.

### Roles

The `roles` map assigns specific roles to users:

```json
"roles": {
  "admin@example.com": "admin",
  "devops@example.com": "admin",
  "developer@example.com": "member"
}
```

Available roles:
- `admin`: Can manage tailnet settings, ACLs, and auth keys
- `member`: Standard access to use the tailnet (default)

## Managing IAM policies

View and update IAM policies using the ionscale CLI:

```bash
# View current IAM policy
ionscale iam get-policy --tailnet "my-tailnet"

# Update IAM policy using a JSON file
ionscale iam update-policy --tailnet "my-tailnet" --file policy.json
```

## Common IAM patterns

### Domain-based tailnet

Grant access to everyone with the same email domain:

```json
{
  "filters": ["domain == example.com"],
  "roles": {
    "admin1@example.com": "admin",
    "admin2@example.com": "admin"
  }
}
```

### Personal tailnet

Create a tailnet for individual use:

```json
{
  "emails": ["personal@example.com"],
  "roles": {
    "personal@example.com": "admin"
  }
}
```

## Setting IAM during tailnet creation

You can set basic IAM policies during tailnet creation with CLI flags:

```bash
# Allow all users with an @example.com email address
ionscale tailnet create --name "shared-tailnet" --domain "example.com"

# Allow only a specific user
ionscale tailnet create --name "personal-tailnet" --email "user@example.com"
```

These shortcuts create appropriate filter rules or email entries in the IAM policy.

## IAM policy evaluation

When a user attempts to access a tailnet, the following checks occur:

1. Is the user's ID in the `subs` list? If yes, grant access.
2. Is the user's email in the `emails` list? If yes, grant access.
3. Does the user match any expression in the `filters` list? If yes, grant access.
4. If none of these conditions are met, access is denied.

For role determination:

1. Check if the user has an entry in the `roles` map
2. If yes, assign that role
3. If no, assign the default `member` role

## Security considerations

- **Principle of least privilege**: Start with minimal access and add users or filters as needed
- **Regular audits**: Periodically review IAM policies to ensure only appropriate users have access
- **Admin roles**: Limit admin roles to trusted users who need to manage tailnet settings

## Troubleshooting access issues

If a user is having trouble accessing a tailnet:

1. Verify the user's email is correct and matches their OIDC identity
2. Check filter expressions to ensure they match the user's attributes
3. Verify the user is authenticating against the correct ionscale instance
4. Check OIDC provider configuration and token issuance