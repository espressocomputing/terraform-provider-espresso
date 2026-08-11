# espresso_account Resource

Registers and manages the display name of an Espresso account. Removing this resource from Terraform state does not delete the account from Espresso.

If an account with the canonical slug already exists, the provider adopts it and reconciles its display name instead of creating a duplicate.

## Example Usage

```hcl
resource "espresso_account" "production" {
  slug         = "acme_production"
  display_name = "Acme Production"
  product      = "databricks"
}
```

## Argument Reference

- `slug` - (Required) Permanent account slug. Espresso prefixes Databricks slugs with `databricks_` when needed. Changing it replaces the resource.
- `display_name` - (Required) Display name. This value can be updated in place.
- `product` - (Required) Account product, such as `databricks` or `snowflake`. Changing it replaces the resource.

## Attribute Reference

- `id` - Canonical Espresso account slug.
