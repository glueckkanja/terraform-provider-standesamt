# Basic usage - use defaults from the provider
data "standesamt_config" "default" {
}
# Advanced usage - override specific settings
data "standesamt_config" "production" {
  environment = "prod"
  separator   = "-"
  lowercase   = true
  prefixes    = ["mycompany"]
  suffixes    = ["001"]
}
# Usage with location
data "standesamt_config" "westeurope" {
  environment = "dev"
  location    = "westeurope"
}
# Combine with name function
data "standesamt_locations" "default" {
}
locals {
  config = {
    configuration = data.standesamt_config.default.configuration
    locations     = data.standesamt_locations.default.locations
    schema        = data.standesamt_config.default.schema
  }
}
output "resource_group_name" {
  value = provider::standesamt::name(local.config, "azurerm_resource_group", {}, "example")
}
