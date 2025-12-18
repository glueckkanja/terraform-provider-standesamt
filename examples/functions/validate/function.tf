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

# Example: Validate a resource group name
output "validation_result" {
  value = provider::standesamt::validate(local.config, "azurerm_resource_group", {}, "example")
}

# Example: Validate a name that exceeds max length
output "validation_result_too_long" {
  value = provider::standesamt::validate(local.config, "azurerm_resource_group", {}, "this-is-a-very-long-name-that-exceeds-the-maximum-length")
}

# Example: Validate a name with invalid characters
output "validation_result_invalid_chars" {
  value = provider::standesamt::validate(local.config, "azurerm_resource_group", {}, "test#invalid")
}

# Example: Using validation result in conditional logic
locals {
  proposed_name = "example"
  validation    = provider::standesamt::validate(local.config, "azurerm_resource_group", {}, local.proposed_name)

  # Check if name is valid
  is_valid = (
    local.validation.regex.valid &&
    local.validation.length.valid &&
    (!local.validation.double_hyphens_denied || !local.validation.double_hyphens_found)
  )
}

output "is_name_valid" {
  value = local.is_valid
}

output "validation_details" {
  value = {
    name                  = local.validation.name
    type                  = local.validation.type
    regex_valid           = local.validation.regex.valid
    regex_pattern         = local.validation.regex.match
    length_valid          = local.validation.length.valid
    length_current        = local.validation.length.is
    length_min            = local.validation.length.min
    length_max            = local.validation.length.max
    double_hyphens_denied = local.validation.double_hyphens_denied
    double_hyphens_found  = local.validation.double_hyphens_found
  }
}
