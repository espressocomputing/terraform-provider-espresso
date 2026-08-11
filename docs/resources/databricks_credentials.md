# espresso_databricks_credentials Resource

Tests and saves the Databricks service-principal connection used by Espresso. The client secret is a write-only attribute and is not retained in this resource's state.

```hcl
resource "espresso_databricks_credentials" "production" {
  account                = espresso_account.production.slug
  workspace_url          = "https://1234567890123456.cloud.databricks.com"
  workspace_id           = "1234567890123456"
  workspace_name         = "Production"
  client_id              = databricks_service_principal.espresso.application_id
  client_secret          = databricks_service_principal_secret.espresso.secret
  service_principal_id   = databricks_service_principal.espresso.id
  service_principal_name = databricks_service_principal.espresso.display_name
  warehouse_id           = databricks_sql_endpoint.espresso.id
  warehouse_name         = databricks_sql_endpoint.espresso.name
}
```

## Arguments

- `account` - (Required) Espresso account slug.
- `workspace_url` - (Required) Databricks workspace URL.
- `workspace_id` - (Required) Databricks workspace ID.
- `workspace_name` - (Optional) Databricks workspace name.
- `client_id` - (Required) Service-principal application ID.
- `client_secret` - (Required, Sensitive, Write-only) Service-principal OAuth secret.
- `service_principal_id` - (Required) Databricks service-principal ID.
- `service_principal_name` - (Required) Databricks service-principal display name.
- `warehouse_id` - (Required) Espresso SQL warehouse ID.
- `warehouse_name` - (Optional) Espresso SQL warehouse name.

Saving tests authentication and warehouse access before replacing the stored credentials. Removing the resource stops Terraform management without deleting the stored credentials.
