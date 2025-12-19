// Copyright (c) glueckkanja AG
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// TestAccLocationsDataSource_Schema tests the locations data source with schema source (default)
func TestAccLocationsDataSource_Schema(t *testing.T) {
	if os.Getenv("TF_ACC") == "" {
		t.Skip("Acceptance tests skipped unless env 'TF_ACC' set")
	}

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactoriesUnique(),
		Steps: []resource.TestStep{
			{
				Config: testAccLocationsDataSourceConfig_Schema(),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.standesamt_locations.test", "locations.%"),
				),
			},
		},
	})
}

// TestAccLocationsDataSource_SchemaWithAliases tests locations with aliases
func TestAccLocationsDataSource_SchemaWithAliases(t *testing.T) {
	if os.Getenv("TF_ACC") == "" {
		t.Skip("Acceptance tests skipped unless env 'TF_ACC' set")
	}

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactoriesUnique(),
		Steps: []resource.TestStep{
			{
				Config: testAccLocationsDataSourceConfig_SchemaWithAliases(),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.standesamt_locations.test", "locations.%"),
					// Verify alias is applied - westeurope should have value "weu"
					resource.TestCheckResourceAttr("data.standesamt_locations.test", "locations.westeurope", "weu"),
					// Verify alias is applied - eastus should have value "eus"
					resource.TestCheckResourceAttr("data.standesamt_locations.test", "locations.eastus", "eus"),
				),
			},
		},
	})
}

// TestAccLocationsDataSource_Azure tests the locations data source with Azure source
// This test requires valid Azure credentials and will be skipped if not available
func TestAccLocationsDataSource_Azure(t *testing.T) {
	if os.Getenv("TF_ACC") == "" {
		t.Skip("Acceptance tests skipped unless env 'TF_ACC' set")
	}

	subscriptionId := os.Getenv("ARM_SUBSCRIPTION_ID")
	if subscriptionId == "" {
		t.Skip("ARM_SUBSCRIPTION_ID must be set for Azure location source tests")
	}

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactoriesUnique(),
		Steps: []resource.TestStep{
			{
				Config: testAccLocationsDataSourceConfig_Azure(subscriptionId),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.standesamt_locations.test", "locations.%"),
					// Azure should return common regions with geo-codes applied
					resource.TestCheckResourceAttr("data.standesamt_locations.test", "locations.eastus", "eus"),
					resource.TestCheckResourceAttr("data.standesamt_locations.test", "locations.westeurope", "we"),
					resource.TestCheckResourceAttr("data.standesamt_locations.test", "locations.germanywestcentral", "gwc"),
				),
			},
		},
	})
}

// TestAccLocationsDataSource_AzureWithAliases tests Azure locations with aliases
func TestAccLocationsDataSource_AzureWithAliases(t *testing.T) {
	if os.Getenv("TF_ACC") == "" {
		t.Skip("Acceptance tests skipped unless env 'TF_ACC' set")
	}

	subscriptionId := os.Getenv("ARM_SUBSCRIPTION_ID")
	if subscriptionId == "" {
		t.Skip("ARM_SUBSCRIPTION_ID must be set for Azure location source tests")
	}

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactoriesUnique(),
		Steps: []resource.TestStep{
			{
				Config: testAccLocationsDataSourceConfig_AzureWithAliases(subscriptionId),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.standesamt_locations.test", "locations.%"),
					// Verify aliases are applied
					resource.TestCheckResourceAttr("data.standesamt_locations.test", "locations.eastus", "eus"),
					resource.TestCheckResourceAttr("data.standesamt_locations.test", "locations.westeurope", "weu"),
				),
			},
		},
	})
}

// TestAccLocationsDataSource_AzureWithEnvAuth tests Azure locations using environment auth
func TestAccLocationsDataSource_AzureWithEnvAuth(t *testing.T) {
	if os.Getenv("TF_ACC") == "" {
		t.Skip("Acceptance tests skipped unless env 'TF_ACC' set")
	}

	subscriptionId := os.Getenv("ARM_SUBSCRIPTION_ID")
	if subscriptionId == "" {
		t.Skip("ARM_SUBSCRIPTION_ID must be set for Azure location source tests")
	}

	// This test relies on ARM_* environment variables being set
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactoriesUnique(),
		Steps: []resource.TestStep{
			{
				Config: testAccLocationsDataSourceConfig_AzureEnvAuth(),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.standesamt_locations.test", "locations.%"),
				),
			},
		},
	})
}

func testAccLocationsDataSourceConfig_Schema() string {
	return `
provider "standesamt" {
  schema_reference = {
    path = "azure/caf"
    ref  = "2025.04"
  }
}

data "standesamt_locations" "test" {}
`
}

func testAccLocationsDataSourceConfig_SchemaWithAliases() string {
	return `
provider "standesamt" {
  schema_reference = {
    path = "azure/caf"
    ref  = "2025.04"
  }
  location_aliases = {
    westeurope = "weu"
    eastus     = "eus"
  }
}

data "standesamt_locations" "test" {}
`
}

func testAccLocationsDataSourceConfig_Azure(subscriptionId string) string {
	return fmt.Sprintf(`
provider "standesamt" {
  location_source = "azure"
  
  azure = {
    subscription_id = %q
    use_cli         = true
  }

  schema_reference = {
    path = "azure/caf"
    ref  = "2025.04"
  }
}

data "standesamt_locations" "test" {}
`, subscriptionId)
}

func testAccLocationsDataSourceConfig_AzureWithAliases(subscriptionId string) string {
	return fmt.Sprintf(`
provider "standesamt" {
  location_source = "azure"
  
  azure = {
    subscription_id = %q
    use_cli         = true
  }

  location_aliases = {
    eastus     = "eus"
    westeurope = "weu"
    germanywestcentral = "gwc"
  }

  schema_reference = {
    path = "azure/caf"
    ref  = "2025.04"
  }
}

data "standesamt_locations" "test" {}
`, subscriptionId)
}

func testAccLocationsDataSourceConfig_AzureEnvAuth() string {
	return `
provider "standesamt" {
  location_source = "azure"

  schema_reference = {
    path = "azure/caf"
    ref  = "2025.04"
  }
}

data "standesamt_locations" "test" {}
`
}
