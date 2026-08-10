# espresso_snowflake_public_key Data Source

Reads the public half of an existing Espresso Snowflake key pair.

## Example Usage

```hcl
data "espresso_snowflake_public_key" "production" {
  account = espresso_account.production.slug
}
```

## Argument Reference

- `account` - (Required) Espresso account slug.

## Attribute Reference

- `public_key` - Public key to assign to the Snowflake service user.
- `id` - Espresso account and public-key identifier.
