# Espresso Provider

The Espresso provider manages Espresso customer accounts, Snowflake credentials, and Snowflake and Databricks Warehouse Agent settings.

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

Set `ESPRESSO_API_KEY` to an Espresso organization API key and `ESPRESSO_ENDPOINT` to the Espresso dashboard API origin.

## Argument Reference

- `api_key` - (Optional, Sensitive) Espresso organization API key. May also be set with `ESPRESSO_API_KEY`.
- `endpoint` - (Optional) Espresso dashboard API origin. May also be set with `ESPRESSO_ENDPOINT`.
