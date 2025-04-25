// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfversion"
	"testing"
)

func TestNameParse_Passthrough(t *testing.T) {
	resource.UnitTest(t, resource.TestCase{
		TerraformVersionChecks: []tfversion.TerraformVersionCheck{
			tfversion.SkipBelow(tfversion.Version1_8_0),
		},
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactoriesUnique(),

		Steps: []resource.TestStep{
			{
				Config: testBasicPassthrough(),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownOutputValue(
						"test",
						knownvalue.StringExact("test"),
					),
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckOutput("test", "test")),
			},
		},
	})
}

//func TestNameParse_RemoteSchemaBasic(t *testing.T) {
//	resource.UnitTest(t, resource.TestCase{
//		TerraformVersionChecks: []tfversion.TerraformVersionCheck{
//			tfversion.SkipBelow(tfversion.Version1_8_0),
//		},
//		ProtoV6ProviderFactories: testAccProtoV6ProviderFactoriesUnique(),
//
//		Steps: []resource.TestStep{
//			{
//				Config: testConfigBasicRemote(),
//				ConfigStateChecks: []statecheck.StateCheck{
//					statecheck.ExpectKnownOutputValue(
//						"test",
//						knownvalue.StringExact("rg-example"),
//					),
//				},
//				Check: resource.ComposeAggregateTestCheckFunc(
//					resource.TestCheckOutput("test", "rg-example")),
//			},
//		},
//	})
//}

func testBasicPassthrough() string {
	return `
provider "standesamt" {
  convention = "passthrough"
}

data "standesamt_config" "default" {}

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
