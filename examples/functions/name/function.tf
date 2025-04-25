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
    ref  = "main"
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

output "name" {
  value = provider::standesamt::name(local.config, "azurerm_resource_group", {}, "example")
}

