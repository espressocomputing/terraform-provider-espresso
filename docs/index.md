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

Set `ESPRESSO_API_KEY` to an Espresso organization API key and `ESPRESSO_ENDPOINT` to the Espresso API origin.

## Argument Reference

- `api_key` - (Optional, Sensitive) Espresso organization API key. May also be set with `ESPRESSO_API_KEY`.
- `endpoint` - (Optional) Espresso API origin. May also be set with `ESPRESSO_ENDPOINT`.
