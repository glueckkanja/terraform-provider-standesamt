package provider

import (
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	//"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	//"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"testing"
)

func TestAccStandesamtRemoteSchema(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactoriesUnique(),
		ExternalProviders:        map[string]resource.ExternalProvider{},
		Steps: []resource.TestStep{
			{
				Config: testAccConfigurationDataSourceConfigNoAttributes(),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.standesamt_config.test", "schema.azurerm_resource_group.abbreviation", "rg"),
					resource.TestCheckResourceAttr("data.standesamt_config.test", "schema.azurerm_resource_group.resource_type", "azurerm_resource_group"),
				),
			},
		},
	})
}

func TestAccStandesamtNoAttributes(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactoriesUnique(),
		ExternalProviders:        map[string]resource.ExternalProvider{},
		Steps: []resource.TestStep{
			{
				Config: testAccConfigurationDataSourceConfigNoAttributes(),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckNoResourceAttr("data.standesamt_config.test", "convention"),
					resource.TestCheckNoResourceAttr("data.standesamt_config.test", "environment"),
					resource.TestCheckNoResourceAttr("data.standesamt_config.test", "separator"),
					resource.TestCheckNoResourceAttr("data.standesamt_config.test", "random_seed"),
					resource.TestCheckNoResourceAttr("data.standesamt_config.test", "hash_length"),
					resource.TestCheckNoResourceAttr("data.standesamt_config.test", "lowercase"),
					resource.TestCheckResourceAttr("data.standesamt_config.test", "prefixes.#", "0"),
					resource.TestCheckResourceAttr("data.standesamt_config.test", "suffixes.#", "0"),
					resource.TestCheckResourceAttr("data.standesamt_config.test", "configuration.convention", "default"),
					resource.TestCheckResourceAttr("data.standesamt_config.test", "configuration.environment", ""),
					resource.TestCheckResourceAttr("data.standesamt_config.test", "configuration.separator", "-"),
					resource.TestCheckResourceAttr("data.standesamt_config.test", "configuration.random_seed", "1337"),
					resource.TestCheckResourceAttr("data.standesamt_config.test", "configuration.hash_length", "0"),
					resource.TestCheckResourceAttr("data.standesamt_config.test", "configuration.lowercase", "false"),
					resource.TestCheckResourceAttr("data.standesamt_config.test", "configuration.prefixes.#", "0"),
					resource.TestCheckResourceAttr("data.standesamt_config.test", "configuration.suffixes.#", "0"),
				),
			},
		},
	})
}

func TestAccStandesamtFullAttributes(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactoriesUnique(),
		ExternalProviders:        map[string]resource.ExternalProvider{},
		Steps: []resource.TestStep{
			{
				Config: testAccConfigurationDataSourceConfigFullAttributes(),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.standesamt_config.test", "convention", "passthrough"),
					resource.TestCheckResourceAttr("data.standesamt_config.test", "environment", "tst"),
					resource.TestCheckResourceAttr("data.standesamt_config.test", "separator", "_"),
					resource.TestCheckResourceAttr("data.standesamt_config.test", "random_seed", "1234"),
					resource.TestCheckResourceAttr("data.standesamt_config.test", "hash_length", "4"),
					resource.TestCheckResourceAttr("data.standesamt_config.test", "lowercase", "true"),
					resource.TestCheckResourceAttr("data.standesamt_config.test", "prefixes.#", "2"),
					resource.TestCheckResourceAttr("data.standesamt_config.test", "suffixes.#", "2"),
					resource.TestCheckResourceAttr("data.standesamt_config.test", "location", "westeurope"),
					resource.TestCheckResourceAttr("data.standesamt_config.test", "configuration.convention", "passthrough"),
					resource.TestCheckResourceAttr("data.standesamt_config.test", "configuration.environment", "tst"),
					resource.TestCheckResourceAttr("data.standesamt_config.test", "configuration.separator", "_"),
					resource.TestCheckResourceAttr("data.standesamt_config.test", "configuration.random_seed", "1234"),
					resource.TestCheckResourceAttr("data.standesamt_config.test", "configuration.hash_length", "4"),
					resource.TestCheckResourceAttr("data.standesamt_config.test", "configuration.lowercase", "true"),
					resource.TestCheckResourceAttr("data.standesamt_config.test", "configuration.prefixes.#", "2"),
					resource.TestCheckResourceAttr("data.standesamt_config.test", "configuration.suffixes.#", "2"),
					resource.TestCheckResourceAttr("data.standesamt_config.test", "configuration.location", "westeurope"),
				),
			},
		},
	})
}

func testAccConfigurationDataSourceConfigNoAttributes() string {
	return `
data "standesamt_config" "test" {}
`
}

func testAccConfigurationDataSourceConfigFullAttributes() string {
	return `
data "standesamt_config" "test" {
	convention = "passthrough"
	environment = "tst"
	prefixes = ["pre1", "pre2"]
	suffixes = ["suf1", "suf2"]
	separator = "_"
	random_seed = 1234
	hash_length = 4
	lowercase = true
	location = "westeurope"
}
`
}
