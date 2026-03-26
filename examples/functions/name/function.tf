terraform {
  required_providers {
    standesamt = {
      source  = "glueckkanja/standesamt"
      version = "0.1.0"
    }
  }
}

provider "standesamt" {
  schema_reference = {
    path = "azure/caf"
    ref  = "2025.04"
  }
}

data "standesamt_config" "default" {}

data "standesamt_locations" "default" {}

locals {
  config = {
    configuration = data.standesamt_config.default.configuration
    locations     = data.standesamt_locations.default.locations
    schema        = data.standesamt_config.default.schema
  }
}

# Basic usage — no overrides, uses provider defaults
output "name_basic" {
  value = provider::standesamt::name(local.config, "azurerm_resource_group", {}, "example")
}

# Override the separator for a specific resource type on a per-call basis
output "name_custom_separator" {
  value = provider::standesamt::name(local.config, "azurerm_resource_group", { separator = "." }, "example")
}

# Full settings override example
output "name_full_settings" {
  value = provider::standesamt::name(
    local.config,
    "azurerm_resource_group",
    {
      environment = "prd"
      separator   = "-"
      prefixes    = ["team"]
      suffixes    = ["001"]
      lowercase   = true
    },
    "example"
  )
}
