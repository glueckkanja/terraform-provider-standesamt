# Basic provider configuration using the default schema library
provider "standesamt" {
  schema_reference = {
    path = "azure/caf"
    ref  = "2025.04"
  }
}
# Provider configuration with environment and naming options
provider "standesamt" {
  alias       = "production"
  environment = "prod"
  separator   = "-"
  lowercase   = true
  random_seed = 42
  schema_reference = {
    path = "azure/caf"
    ref  = "2025.04"
  }
}
# Provider configuration with a custom schema URL
provider "standesamt" {
  alias = "custom"
  schema_reference = {
    custom_url = "https://example.com/path/to/schema.zip"
  }
}

# Provider configuration using Azure as location source
# This fetches locations directly from the Azure Resource Manager API
provider "standesamt" {
  alias           = "azure_locations"
  location_source = "azure"

  # Azure authentication configuration
  azure = {
    subscription_id = "00000000-0000-0000-0000-000000000000"
    use_cli         = true # Uses Azure CLI for authentication
  }

  # Optional: Remap location short names
  location_aliases = {
    eastus             = "eus"
    westeurope         = "weu"
    germanywestcentral = "gwc"
  }

  schema_reference = {
    path = "azure/caf"
    ref  = "2025.04"
  }
}

# Provider configuration with Azure locations using Service Principal
provider "standesamt" {
  alias           = "azure_spn"
  location_source = "azure"

  azure = {
    subscription_id = "00000000-0000-0000-0000-000000000000"
    tenant_id       = "00000000-0000-0000-0000-000000000000"
    client_id       = "00000000-0000-0000-0000-000000000000"
    client_secret   = "your-client-secret"
  }

  schema_reference = {
    path = "azure/caf"
    ref  = "2025.04"
  }
}

# Provider configuration using environment variables
# The following environment variables are supported:
# - SA_ENVIRONMENT: Sets the environment (e.g., 'prod', 'dev', 'test')
# - SA_CONVENTION: Sets the naming convention ('default' or 'passthrough')
# - SA_SEPARATOR: Sets the separator character
# - SA_RANDOM_SEED: Sets the random seed for unique name generation
# - SA_HASH_LENGTH: Sets the default hash length
# - SA_LOWERCASE: Controls lowercase output ('true' or 'false')
# - SA_LOCATION_SOURCE: Sets the location source ('schema' or 'azure')
#
# Azure authentication environment variables (compatible with azurerm):
# - ARM_SUBSCRIPTION_ID: Azure subscription ID
# - ARM_TENANT_ID: Azure tenant ID
# - ARM_CLIENT_ID: Service principal client ID
# - ARM_CLIENT_SECRET: Service principal client secret
# - ARM_USE_CLI: Use Azure CLI authentication ('true' or 'false')
# - ARM_USE_MSI: Use Managed Service Identity ('true' or 'false')
# - ARM_USE_OIDC: Use OpenID Connect authentication ('true' or 'false')
# - ARM_ENVIRONMENT: Azure environment ('public', 'usgovernment', 'china')
provider "standesamt" {
  alias = "from_env"
  schema_reference = {
    path = "azure/caf"
    ref  = "2025.04"
  }
}
