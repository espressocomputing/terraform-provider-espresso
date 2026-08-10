# espresso_snowflake_warehouse_agent_warehouse Resource

Manages Warehouse Agent settings for one Snowflake warehouse. Removing this resource stops Terraform management without changing the current settings or deleting the warehouse.

## Example Usage

```hcl
resource "espresso_snowflake_warehouse_agent_warehouse" "transforming" {
  account        = espresso_account.production.slug
  name           = "TRANSFORMING"
  enabled        = true
  min_clusters   = 1
  max_clusters   = 4
  scaling_policy = "STANDARD"
}
```

## Argument Reference

- `account` - (Required) Espresso account slug. Changing it replaces the resource.
- `name` - (Required) Snowflake warehouse name. Changing it replaces the resource.
- `enabled` - (Optional) Whether Espresso manages the warehouse.
- `min_clusters` - (Optional) Minimum cluster count.
- `max_clusters` - (Optional) Maximum cluster count.
- `scaling_policy` - (Optional) Snowflake scaling policy.
- `notes` - (Optional) Audit note sent with updates.

Omitting an optional setting adopts its current value without changing it.

## Attribute Reference

- `id` - Espresso account, agent, and warehouse identifier.
