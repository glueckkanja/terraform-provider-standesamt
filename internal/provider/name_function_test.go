// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"fmt"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"regexp"
	"testing"
)

func TestNameFunction_Null(t *testing.T) {
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactoriesUnique(),
		Steps: []resource.TestStep{
			{
				Config: `output "test" {
							value = provider::standesamt::name(null, null, null, null)
						}`,
				ExpectError: regexp.MustCompile(`Invalid value for "configurations" parameter: argument must not be null\.`),
			},
		},
	})
}

func TestNameFunction_ResourceGroup(t *testing.T) {
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactoriesUnique(),
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf("%s %s", default_config_with_no_settings_default_precedence, `output "test" {
					value = provider::standesamt::name(local.config, "azurerm_resource_group", local.settings, "test")
				}`),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownOutputValue("test", knownvalue.StringExact("rg-test-we")),
				},
			},
		},
	})
}

func TestNameFunction_AzureCaf(t *testing.T) {
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactoriesUnique(),
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf("%s %s", schema_config_with_no_settings, `output "test" {
					value = provider::standesamt::name(local.config, "azurerm_resource_group", local.settings, "test")
				}`),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownOutputValue("test", knownvalue.StringExact("rg-test")),
				},
			},
		},
	})
}

func testBasicPassthrough() string {
	return `
data "standesamt_config" "default" {
 convention = "passthrough"
}

data "standesamt_locations" "default" {}

locals {
  config = {
    configuration = data.standesamt_config.default.configuration
    schema        = data.standesamt_config.default.schema
    locations     = data.standesamt_locations.default.locations
  }
}

output "test" {
  value = provider::standesamt::name(local.config, "azurerm_resource_group", {}, "test")
}
`
}

const schema_config = `
data "standesamt_config" "example" {}

locals {
	settings = {
		%s
	}
	config = {
		configuration 	= data.standesamt_config.example.configuration
		schema 			= data.standesamt_config.example.schema
		locations = {
			"westeurope" = "we"
		}
	}
}
`

var schema_config_with_no_settings = fmt.Sprintf(schema_config, ``)

const default_config = `
locals {
	settings = {
		%s
	}
	config = {
		configuration = {
			convention 		= "default"
			environment 		= ""
			prefixes 			= []
			suffixes			= []
			name_precedence 	= [%s]
			hash_length 		= 0
			random_seed 		= 1337
			separator 			= "-"
			location 			= "westeurope"
			lowercase 			=  true
		}
		schema = {
			azurerm_resource_group = {
				resource_type 		= "azurerm_resource_group"
				abbreviation 		= "rg"
				min_length 			=  1
				max_length			=  90
				validation_regex 	= "^[a-zA-Z0-9-._()]{0,89}[a-zA-Z0-9-_()]$"
				configuration = {
				  use_environment		= true
				  use_lower_case 		= false
				  use_separator 		= true
				  deny_double_hyphens = false
				  name_precedence		= []
				  hash_length			= 0
				}				
			}
		}
		locations = {
			"westeurope" = "we"
		}
	}
}
`

var default_config_with_no_settings_default_precedence = fmt.Sprintf(default_config, ``, `"abbreviation", "prefixes", "name", "location", "environment", "hash", "suffixes"`)
