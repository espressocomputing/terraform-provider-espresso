# espresso_snowflake_credentials Resource

Configures key-pair Snowflake credentials for an Espresso account and tests the connection. Removing this resource stops Terraform management without deleting the stored credentials.

## Example Usage

```hcl
resource "espresso_snowflake_credentials" "production" {
  account           = espresso_account.production.slug
  snowflake_account = "acme-org-acme-production"
  username          = "ESPRESSO_AI_USER"
  role              = "ESPRESSO_AI_ROLE"
  warehouse         = "ESPRESSO_AI_WH"
}
```

## Argument Reference

- `account` - (Required) Espresso account slug. Changing it replaces the resource.
- `snowflake_account` - (Required) Snowflake organization and account identifier.
- `host` - (Optional) Snowflake host. Omit it to derive the host from `snowflake_account`.
- `username` - (Required) Snowflake service-user name.
- `role` - (Required) Snowflake role used by Espresso.
- `warehouse` - (Required) Default Snowflake warehouse.

## Attribute Reference

- `host` - Effective Snowflake host returned by Espresso.
- `id` - Espresso account and credential identifier.
