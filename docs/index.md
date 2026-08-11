# Espresso Provider

The Espresso provider manages Espresso accounts, Snowflake and Databricks credentials, and Warehouse Agent settings.

## Example Usage

```hcl
terraform {
  required_providers {
    espresso = {
      source = "espressocomputing/espresso"
    }
  }
}

provider "espresso" {}
```

In the Espresso dashboard, select an existing account, open **Tools → API Keys**, choose **Generate API key → Organization key**, and copy the complete `ok_` secret. It cannot be displayed again. Set it as `ESPRESSO_API_KEY`.

## Argument Reference

- `api_key` - (Optional, Sensitive) Espresso organization API key. May also be set with `ESPRESSO_API_KEY`.
- `endpoint` - (Optional) Espresso API origin.
