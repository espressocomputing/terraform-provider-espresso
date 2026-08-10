# espresso_databricks_warehouse_agent Resource

Manages global Databricks Warehouse Agent settings for an Espresso account. Removing this resource stops Terraform management without changing the current Espresso settings.

## Example Usage

```hcl
resource "espresso_databricks_warehouse_agent" "workspace" {
  account     = espresso_account.workspace.slug
  enabled     = true
  auto_opt_in = false
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
