# espresso_snowflake_warehouse_agent Resource

Manages global Snowflake Warehouse Agent settings for an Espresso account. Removing this resource stops Terraform management without changing the current Espresso settings.

## Example Usage

```hcl
resource "espresso_snowflake_warehouse_agent" "production" {
  account     = espresso_account.production.slug
  enabled     = true
  auto_opt_in = true
  notes       = "Managed by Terraform"
}
```

## Argument Reference

- `account` - (Required) Espresso account slug. Changing it replaces the resource.
- `enabled` - (Required) Whether the Warehouse Agent is enabled.
- `auto_opt_in` - (Optional) Whether discovered warehouses are automatically opted in. Defaults to `false`.
- `notes` - (Optional) Audit note sent with updates.

## Attribute Reference

- `id` - Espresso account and Warehouse Agent identifier.
