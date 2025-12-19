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
# Provider configuration using environment variables
# The following environment variables are supported:
# - SA_ENVIRONMENT: Sets the environment (e.g., 'prod', 'dev', 'test')
# - SA_CONVENTION: Sets the naming convention ('default' or 'passthrough')
# - SA_SEPARATOR: Sets the separator character
# - SA_RANDOM_SEED: Sets the random seed for unique name generation
# - SA_HASH_LENGTH: Sets the default hash length
# - SA_LOWERCASE: Controls lowercase output ('true' or 'false')
provider "standesamt" {
  alias = "from_env"
  schema_reference = {
    path = "azure/caf"
    ref  = "2025.04"
  }
}
