// Copyright (c) glueckkanja AG
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
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

func TestNameFunction_MaxLength(t *testing.T) {
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactoriesUnique(),
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf("%s %s", default_config_with_no_settings_default_precedence, `output "test" {
					value = provider::standesamt::name(local.config, "azurerm_resource_group", local.settings, "12345678901234567890")
				}`),
				ExpectError: regexp.MustCompile(`Name has 26 characters,\s+but maximum is set to 20\.`),
			},
		},
	})
}

func TestNameFunction_DoubleHyphenError(t *testing.T) {
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactoriesUnique(),
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf("%s %s", default_config_with_no_settings_default_precedence, `output "test" {
					value = provider::standesamt::name(local.config, "azurerm_resource_group", local.settings, "12345--67890")
				}`),
				ExpectError: regexp.MustCompile(`Invalid name:\s+'rg-12345--67890-we' contains double hyphens`),
			},
		},
	})
}

func TestNameFunction_DoubleHyphenNoError(t *testing.T) {
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactoriesUnique(),
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf("%s %s", remote_schema_config_with_no_settings, `output "test" {
					value = provider::standesamt::name(local.config, "azurerm_resource_group", local.settings, "te--st")
				}`),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownOutputValue("test", knownvalue.StringExact("rg-te--st")),
				},
			},
		},
	})
}

func TestNameFunction_MinLength(t *testing.T) {
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactoriesUnique(),
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf("%s %s", default_config_with_no_settings_default_precedence, `output "test" {
					value = provider::standesamt::name(local.config, "azurerm_resource_group", local.settings, "t")
				}`),
				ExpectError: regexp.MustCompile(`Name has 7 characters,\s+but minimum is set to 8\.`),
			},
		},
	})
}

func TestNameFunction_RegEx(t *testing.T) {
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactoriesUnique(),
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf("%s %s", default_config_with_no_settings_default_precedence, `output "test" {
					value = provider::standesamt::name(local.config, "azurerm_resource_group", local.settings, "test#")
				}`),
				ExpectError: regexp.MustCompile(`Name does not match\s+regex\.`),
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

func TestNameFunction_LowerCase(t *testing.T) {
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactoriesUnique(),
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf("%s %s", default_config_with_no_settings_default_precedence, `output "test" {
					value = provider::standesamt::name(local.config, "azurerm_resource_group", local.settings, "UPPERCASE")
				}`),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownOutputValue("test", knownvalue.StringExact("rg-uppercase-we")),
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
				Config: fmt.Sprintf("%s %s", remote_schema_config_with_no_settings, `output "test" {
					value = provider::standesamt::name(local.config, "azurerm_resource_group", local.settings, "test")
				}`),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownOutputValue("test", knownvalue.StringExact("rg-test")),
				},
			},
		},
	})
}

func TestNameFunction_AzureCaf_Full(t *testing.T) {
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactoriesUnique(),
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf("%s %s", remote_schema_config_with_full_settings, `output "test" {
					value = provider::standesamt::name(local.config, "azurerm_resource_group", local.settings, "TEST")
				}`),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownOutputValue("test", knownvalue.StringExact("rg_pre1_pre2_test_we_tst_qffc_suf1_suf2")),
				},
			},
		},
	})
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

var remote_schema_config_with_no_settings = fmt.Sprintf(schema_config, ``)

var remote_schema_config_with_full_settings = fmt.Sprintf(schema_config, `
convention = "default"
environment = "tst"
prefixes = ["pre1", "pre2"]
suffixes = ["suf1", "suf2"]
name_precedence = ["abbreviation", "prefixes", "name", "location", "environment", "hash", "suffixes"]
hash_length = 4
random_seed = 1234
separator = "_"
location = "westeurope"
lowercase = true
`)

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
			lowercase 			= true
		}
		schema = {
			azurerm_resource_group = {
				resource_type 		= "azurerm_resource_group"
				abbreviation 		= "rg"
				min_length 			=  8
				max_length			=  20
				validation_regex 	= "^[a-zA-Z0-9-._()]{0,89}[a-zA-Z0-9-_()]$"
				configuration = {
				  use_environment		= true
				  use_lower_case 		= false
				  use_separator 		= true
				  deny_double_hyphens = true
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
