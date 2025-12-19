// Copyright (c) glueckkanja AG
// SPDX-License-Identifier: MPL-2.0

package azure

// DefaultGeoCodeMappings contains the official Microsoft Azure Backup geo-code mappings.
// These mappings are used to convert Azure region display names to their short geo-codes.
// Source: Microsoft Azure Backup GeoCodeList XML
//
// The map key is the normalized region name (lowercase, no spaces) as returned by the Azure API.
// The value is the official geo-code abbreviation.
var DefaultGeoCodeMappings = map[string]string{
	// Asia Pacific
	"eastasia":           "ea",
	"southeastasia":      "sea",
	"australiaeast":      "ae",
	"australiasoutheast": "ase",
	"australiacentral":   "acl",
	"australiacentral2":  "acl2",
	"japaneast":          "jpe",
	"japanwest":          "jpw",
	"koreacentral":       "krc",
	"koreasouth":         "krs",
	"centralindia":       "inc",
	"southindia":         "ins",
	"westindia":          "inw",
	"jioindiacentral":    "jic",
	"jioindiawest":       "jiw",
	"malaysiasouth":      "mys",
	"malaysiawest":       "myw",
	"taiwannorth":        "twn",
	"taiwannorthwest":    "twnr",
	"indonesiacentral":   "idc",
	"newzealandnorth":    "nzn",

	// Americas
	"eastus":          "eus",
	"eastus2":         "eus2",
	"westus":          "wus",
	"westus2":         "wus2",
	"westus3":         "wus3",
	"centralus":       "cus",
	"northcentralus":  "ncus",
	"southcentralus":  "scus",
	"westcentralus":   "wcus",
	"canadacentral":   "cnc",
	"canadaeast":      "cne",
	"brazilsouth":     "brs",
	"brazilsoutheast": "bse",
	"mexicocentral":   "mxc",
	"chilecentral":    "clc",
	"southeastus":     "use",

	// Europe
	"northeurope":        "ne",
	"westeurope":         "we",
	"uksouth":            "uks",
	"ukwest":             "ukw",
	"francecentral":      "frc",
	"francesouth":        "frs",
	"switzerlandnorth":   "szn",
	"switzerlandwest":    "szw",
	"germanynorth":       "gn",
	"germanywestcentral": "gwc",
	"norwayeast":         "nwe",
	"norwaywest":         "nww",
	"swedencentral":      "sdc",
	"swedensouth":        "sds",
	"polandcentral":      "plc",
	"italynorth":         "itn",
	"spaincentral":       "spc",

	// Middle East & Africa
	"uaecentral":       "uac",
	"uaenorth":         "uan",
	"southafricanorth": "san",
	"southafricawest":  "saw",
	"qatarcentral":     "qac",
	"israelcentral":    "ilc",

	// Azure Government
	"usgovvirginia": "ugv",
	"usgovarizona":  "uga",
	"usgovtexas":    "ugt",
	"usdodcentral":  "udc",
	"usdodeast":     "ude",

	// Azure China
	"chinanorth":  "bjb",
	"chinaeast":   "sha",
	"chinanorth2": "bjb2",
	"chinaeast2":  "sha2",
	"chinanorth3": "bjb3",
	"chinaeast3":  "sha3",

	// Preview/EUAP regions
	"centraluseuap": "ccy",
	"eastus2euap":   "ecy",
}

// GetGeoCode returns the geo-code for a given Azure region name.
// If no mapping exists, it returns the original region name.
func GetGeoCode(regionName string) string {
	if code, ok := DefaultGeoCodeMappings[regionName]; ok {
		return code
	}
	return regionName
}
