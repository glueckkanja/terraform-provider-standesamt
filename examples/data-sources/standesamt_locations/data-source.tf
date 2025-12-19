# Fetch all available locations
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
