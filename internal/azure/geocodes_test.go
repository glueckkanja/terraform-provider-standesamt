// Copyright (c) glueckkanja AG
// SPDX-License-Identifier: MPL-2.0

package azure

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetGeoCode(t *testing.T) {
	tests := []struct {
		regionName   string
		expectedCode string
	}{
		// Common regions
		{"eastus", "eus"},
		{"westus", "wus"},
		{"westeurope", "we"},
		{"northeurope", "ne"},
		{"germanywestcentral", "gwc"},
		{"swedencentral", "sdc"},

		// Asia Pacific
		{"eastasia", "ea"},
		{"southeastasia", "sea"},
		{"australiaeast", "ae"},
		{"japaneast", "jpe"},

		// Government
		{"usgovvirginia", "ugv"},
		{"usgovarizona", "uga"},

		// China
		{"chinanorth", "bjb"},
		{"chinaeast", "sha"},

		// Unknown region should return the original name
		{"unknownregion", "unknownregion"},
		{"newregion2025", "newregion2025"},
	}

	for _, tt := range tests {
		t.Run(tt.regionName, func(t *testing.T) {
			result := GetGeoCode(tt.regionName)
			assert.Equal(t, tt.expectedCode, result)
		})
	}
}

func TestDefaultGeoCodeMappings_Coverage(t *testing.T) {
	// Verify that all expected regions have mappings
	expectedRegions := []string{
		"eastus", "eastus2", "westus", "westus2", "westus3",
		"centralus", "northcentralus", "southcentralus", "westcentralus",
		"northeurope", "westeurope", "uksouth", "ukwest",
		"francecentral", "francesouth", "germanywestcentral", "germanynorth",
		"swedencentral", "swedensouth", "norwayeast", "norwaywest",
		"switzerlandnorth", "switzerlandwest",
		"eastasia", "southeastasia",
		"australiaeast", "australiasoutheast", "australiacentral",
		"japaneast", "japanwest", "koreacentral", "koreasouth",
		"centralindia", "southindia", "westindia",
		"brazilsouth", "brazilsoutheast",
		"canadacentral", "canadaeast",
		"uaenorth", "uaecentral",
		"southafricanorth", "southafricawest",
		"qatarcentral", "israelcentral", "italynorth",
		"polandcentral", "spaincentral", "mexicocentral",
	}

	for _, region := range expectedRegions {
		t.Run(region, func(t *testing.T) {
			code := GetGeoCode(region)
			assert.NotEqual(t, region, code, "Region %s should have a geo-code mapping", region)
		})
	}
}

func TestDefaultGeoCodeMappings_Lowercase(t *testing.T) {
	// Verify all geo-codes are lowercase
	for region, code := range DefaultGeoCodeMappings {
		t.Run(region, func(t *testing.T) {
			assert.Equal(t, code, code, "Geo-code for %s should be lowercase", region)
		})
	}
}

func TestDefaultGeoCodeMappings_NotEmpty(t *testing.T) {
	assert.Greater(t, len(DefaultGeoCodeMappings), 50, "Should have at least 50 geo-code mappings")
}
