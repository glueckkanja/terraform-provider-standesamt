# Fetch all available locations from the schema library (default)
data "standesamt_locations" "default" {
}

# Use locations map with config data source
data "standesamt_config" "default" {
}

locals {
  config = {
    configuration = data.standesamt_config.default.configuration
    locations     = data.standesamt_locations.default.locations
    schema        = data.standesamt_config.default.schema
  }
}

# Example: Generate name with location abbreviation
output "storage_account_name" {
  value = provider::standesamt::name(local.config, "azurerm_storage_account", {
    location = "westeurope"
  }, "example")
}

# Output all available locations
output "all_locations" {
  value = data.standesamt_locations.default.locations
}

# ============================================================================
# Example: Using Azure as location source with automatic geo-code mappings
# ============================================================================
# When using location_source = "azure", the provider fetches locations directly
# from the Azure Resource Manager API and automatically applies official
# Microsoft Azure Backup geo-code mappings.
#
# For example:
#   - eastus -> eus
#   - westeurope -> we
#   - germanywestcentral -> gwc
#
# You can override any mapping using location_aliases in the provider config.
# ============================================================================

