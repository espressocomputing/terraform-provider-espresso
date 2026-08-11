# Espresso Terraform provider

Manage Espresso accounts, Snowflake and Databricks credentials, and Warehouse Agent settings with Terraform.

## Using the provider

```hcl
terraform {
  required_providers {
    espresso = {
      source  = "espressocomputing/espresso"
      version = "~> 0.1"
    }
  }
}

provider "espresso" {}
```

See the [provider documentation](docs/index.md) and individual resource and data-source pages under `docs/`.

## Authentication

In the Espresso dashboard, select an existing account, open **Tools → API Keys**, choose **Generate API key → Organization key**, and copy the complete `ok_` secret. It cannot be displayed again. Set it as `ESPRESSO_API_KEY`.

## Development

This provider requires Go 1.25. Build and test it with:

```shell
go build ./...
go test ./...
```

## One account per Databricks workspace

An `espresso_account` is the Espresso account boundary. Key the Terraform resources by Databricks workspace ID and give each workspace a permanent Espresso slug:

```hcl
variable "databricks_workspaces" {
  type = map(object({
    espresso_slug = string
    display_name  = string
  }))
}

resource "espresso_account" "databricks_workspace" {
  for_each = var.databricks_workspaces

  slug         = each.value.espresso_slug
  display_name = each.value.display_name
  product      = "databricks"
}

resource "espresso_databricks_warehouse_agent" "workspace" {
  for_each = var.databricks_workspaces

  account     = espresso_account.databricks_workspace[each.key].slug
  enabled     = false
  auto_opt_in = false
}

output "espresso_account_by_workspace_id" {
  value = {
    for workspace_id, account in espresso_account.databricks_workspace :
    workspace_id => account.slug
  }
}
```

For example:

```hcl
databricks_workspaces = {
  "1234567890123456" = {
    espresso_slug = "acme_production"
    display_name  = "Acme Production"
  }
  "9876543210987654" = {
    espresso_slug = "acme_staging"
    display_name  = "Acme Staging"
  }
}
```

Espresso prepends `databricks_` when a Databricks slug omits it, so these accounts are stored as `databricks_acme_production` and `databricks_acme_staging`. Every global and warehouse setting for workspace `1234567890123456` must use `espresso_account.databricks_workspace["1234567890123456"].slug` as its `account`.

An account's `display_name` can be updated in place. Its `slug` and `product` are immutable. Removing an account resource from Terraform stops managing it but leaves the account in Espresso.

Onboarding still needs to be run.

## Snowflake credentials

```hcl
resource "espresso_account" "snowflake_production" {
  slug         = "acme_snowflake_production"
  display_name = "Acme Snowflake Production"
  product      = "snowflake"
}

data "espresso_snowflake_public_key" "production" {
  account = espresso_account.snowflake_production.slug
}

resource "snowflake_service_user" "espresso" {
  name              = "ESPRESSO_AI_USER"
  default_role      = "ESPRESSO_AI_ROLE"
  default_warehouse = "ESPRESSO_AI_WH"
  rsa_public_key    = data.espresso_snowflake_public_key.production.public_key
}

resource "espresso_snowflake_credentials" "production" {
  account           = espresso_account.snowflake_production.slug
  snowflake_account = "acme-org-acme-production"
  host              = "acme-org-acme-production.snowflakecomputing.com"
  username          = snowflake_service_user.espresso.name
  role              = "ESPRESSO_AI_ROLE"
  warehouse         = "ESPRESSO_AI_WH"
}
```

`espresso_snowflake_public_key` only reads the public half of an existing Espresso keypair. Configure the keypair through Espresso onboarding before reading it, then assign the value to the Snowflake service user before creating `espresso_snowflake_credentials`. The credentials resource always uses the stored keypair and tests the Snowflake connection before saving the remaining connection settings. Omit `host` to derive it from `snowflake_account`. Removing the credentials resource stops Terraform management without deleting the stored credentials.

## Warehouse Agent settings

```hcl
resource "espresso_snowflake_warehouse_agent" "production" {
  account     = espresso_account.snowflake_production.slug
  enabled     = true
  auto_opt_in = true
  notes       = "Managed by Terraform"
}

locals {
  transforming = {
    min_clusters   = 1
    max_clusters   = 4
    scaling_policy = "STANDARD"
  }
}

resource "snowflake_warehouse" "transforming" {
  name              = "TRANSFORMING"
  min_cluster_count = local.transforming.min_clusters
  max_cluster_count = local.transforming.max_clusters
  scaling_policy    = local.transforming.scaling_policy

  lifecycle {
    ignore_changes = [min_cluster_count, max_cluster_count, scaling_policy]
  }
}

resource "espresso_snowflake_warehouse_agent_warehouse" "transforming" {
  account        = espresso_account.snowflake_production.slug
  name           = snowflake_warehouse.transforming.name
  enabled        = true
  min_clusters   = local.transforming.min_clusters
  max_clusters   = local.transforming.max_clusters
  scaling_policy = local.transforming.scaling_policy
}

locals {
  shared_workspace_id = "1234567890123456"
  shared = {
    min_clusters = 1
    max_clusters = 8
  }
}

resource "databricks_sql_endpoint" "shared" {
  name             = "Shared SQL"
  min_num_clusters = local.shared.min_clusters
  max_num_clusters = local.shared.max_clusters
  cluster_size     = "Large"
  warehouse_type   = "PRO"

  lifecycle {
    ignore_changes = [min_num_clusters, max_num_clusters]
  }
}

resource "espresso_databricks_warehouse_agent_warehouse" "shared" {
  account        = espresso_account.databricks_workspace[local.shared_workspace_id].slug
  name           = databricks_sql_endpoint.shared.name
  enabled        = true
  min_clusters   = local.shared.min_clusters
  max_clusters   = local.shared.max_clusters
}
```

The lifecycle lists prevent the native providers from fighting the Warehouse Agent. Terraform lifecycle values cannot be conditional. To return control safely, first set the Espresso warehouse's `enabled` to `false` and apply, then remove its `ignore_changes` entries and apply again. The native provider then reconciles the warehouse to the values in the shared local.

Each Warehouse Agent warehouse configuration is managed as a discrete resource. Its settings fields are optional, so an `account` and `name` can adopt the current values without changing them. Removing a Warehouse Agent resource stops Terraform management without changing the current Espresso settings or the underlying warehouse.
