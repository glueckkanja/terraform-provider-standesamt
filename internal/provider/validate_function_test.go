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

func TestValidateFunction_Null(t *testing.T) {
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactoriesUnique(),
		Steps: []resource.TestStep{
			{
				Config: `output "test" {
							value = provider::standesamt::validate(null, null, null, null)
						}`,
				ExpectError: regexp.MustCompile(`Invalid value for "configurations" parameter: argument must not be null\.`),
			},
		},
	})
}

func TestValidateFunction_MissingResourceType(t *testing.T) {
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactoriesUnique(),
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf("%s %s", default_config_with_no_settings_default_precedence, `output "test" {
					value = provider::standesamt::validate(local.config, "invalid_resource_type", local.settings, "test")
				}`),
				ExpectError: regexp.MustCompile(`(?s)resource type\s+'invalid_resource_type' not found in schema.*Available resource types:\s+\[azurerm_resource_group\]`),
			},
		},
	})
}

func TestValidateFunction_ValidName(t *testing.T) {
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactoriesUnique(),
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf("%s %s", default_config_with_no_settings_default_precedence, `output "test" {
					value = provider::standesamt::validate(local.config, "azurerm_resource_group", local.settings, "test")
				}`),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownOutputValue("test", knownvalue.ObjectExact(map[string]knownvalue.Check{
						"name": knownvalue.StringExact("rg-test-we"),
						"type": knownvalue.StringExact("azurerm_resource_group"),
						"regex": knownvalue.ObjectExact(map[string]knownvalue.Check{
							"valid": knownvalue.Bool(true),
							"match": knownvalue.StringExact("^[a-zA-Z0-9-._()]{0,89}[a-zA-Z0-9-_()]$"),
						}),
						"length": knownvalue.ObjectExact(map[string]knownvalue.Check{
							"valid": knownvalue.Bool(true),
							"is":    knownvalue.Int64Exact(10),
							"max":   knownvalue.Int64Exact(20),
							"min":   knownvalue.Int64Exact(8),
						}),
						"double_hyphens_denied": knownvalue.Bool(true),
						"double_hyphens_found":  knownvalue.Bool(false),
					})),
				},
			},
		},
	})
}

func TestValidateFunction_MaxLength(t *testing.T) {
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactoriesUnique(),
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf("%s %s", default_config_with_no_settings_default_precedence, `output "test" {
					value = provider::standesamt::validate(local.config, "azurerm_resource_group", local.settings, "12345678901234567890")
				}`),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownOutputValue("test", knownvalue.ObjectExact(map[string]knownvalue.Check{
						"name": knownvalue.StringExact("rg-12345678901234567890-we"),
						"type": knownvalue.StringExact("azurerm_resource_group"),
						"regex": knownvalue.ObjectExact(map[string]knownvalue.Check{
							"valid": knownvalue.Bool(true),
							"match": knownvalue.StringExact("^[a-zA-Z0-9-._()]{0,89}[a-zA-Z0-9-_()]$"),
						}),
						"length": knownvalue.ObjectExact(map[string]knownvalue.Check{
							"valid": knownvalue.Bool(false),
							"is":    knownvalue.Int64Exact(26),
							"max":   knownvalue.Int64Exact(20),
							"min":   knownvalue.Int64Exact(8),
						}),
						"double_hyphens_denied": knownvalue.Bool(true),
						"double_hyphens_found":  knownvalue.Bool(false),
					})),
				},
			},
		},
	})
}

func TestValidateFunction_MinLength(t *testing.T) {
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactoriesUnique(),
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf("%s %s", default_config_with_no_settings_default_precedence, `output "test" {
					value = provider::standesamt::validate(local.config, "azurerm_resource_group", local.settings, "t")
				}`),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownOutputValue("test", knownvalue.ObjectExact(map[string]knownvalue.Check{
						"name": knownvalue.StringExact("rg-t-we"),
						"type": knownvalue.StringExact("azurerm_resource_group"),
						"regex": knownvalue.ObjectExact(map[string]knownvalue.Check{
							"valid": knownvalue.Bool(true),
							"match": knownvalue.StringExact("^[a-zA-Z0-9-._()]{0,89}[a-zA-Z0-9-_()]$"),
						}),
						"length": knownvalue.ObjectExact(map[string]knownvalue.Check{
							"valid": knownvalue.Bool(false),
							"is":    knownvalue.Int64Exact(7),
							"max":   knownvalue.Int64Exact(20),
							"min":   knownvalue.Int64Exact(8),
						}),
						"double_hyphens_denied": knownvalue.Bool(true),
						"double_hyphens_found":  knownvalue.Bool(false),
					})),
				},
			},
		},
	})
}

func TestValidateFunction_RegEx(t *testing.T) {
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactoriesUnique(),
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf("%s %s", default_config_with_no_settings_default_precedence, `output "test" {
					value = provider::standesamt::validate(local.config, "azurerm_resource_group", local.settings, "test#")
				}`),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownOutputValue("test", knownvalue.ObjectExact(map[string]knownvalue.Check{
						"name": knownvalue.StringExact("rg-test#-we"),
						"type": knownvalue.StringExact("azurerm_resource_group"),
						"regex": knownvalue.ObjectExact(map[string]knownvalue.Check{
							"valid": knownvalue.Bool(false),
							"match": knownvalue.StringExact("^[a-zA-Z0-9-._()]{0,89}[a-zA-Z0-9-_()]$"),
						}),
						"length": knownvalue.ObjectExact(map[string]knownvalue.Check{
							"valid": knownvalue.Bool(true),
							"is":    knownvalue.Int64Exact(11),
							"max":   knownvalue.Int64Exact(20),
							"min":   knownvalue.Int64Exact(8),
						}),
						"double_hyphens_denied": knownvalue.Bool(true),
						"double_hyphens_found":  knownvalue.Bool(false),
					})),
				},
			},
		},
	})
}

func TestValidateFunction_DoubleHyphensFound(t *testing.T) {
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactoriesUnique(),
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf("%s %s", default_config_with_no_settings_default_precedence, `output "test" {
					value = provider::standesamt::validate(local.config, "azurerm_resource_group", local.settings, "12345--67890")
				}`),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownOutputValue("test", knownvalue.ObjectExact(map[string]knownvalue.Check{
						"name": knownvalue.StringExact("rg-12345--67890-we"),
						"type": knownvalue.StringExact("azurerm_resource_group"),
						"regex": knownvalue.ObjectExact(map[string]knownvalue.Check{
							"valid": knownvalue.Bool(true),
							"match": knownvalue.StringExact("^[a-zA-Z0-9-._()]{0,89}[a-zA-Z0-9-_()]$"),
						}),
						"length": knownvalue.ObjectExact(map[string]knownvalue.Check{
							"valid": knownvalue.Bool(true),
							"is":    knownvalue.Int64Exact(18),
							"max":   knownvalue.Int64Exact(20),
							"min":   knownvalue.Int64Exact(8),
						}),
						"double_hyphens_denied": knownvalue.Bool(true),
						"double_hyphens_found":  knownvalue.Bool(true),
					})),
				},
			},
		},
	})
}

func TestValidateFunction_DoubleHyphensNotDenied(t *testing.T) {
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactoriesUnique(),
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf("%s %s", remote_schema_config_with_no_settings, `output "test" {
					value = provider::standesamt::validate(local.config, "azurerm_resource_group", local.settings, "te--st")
				}`),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownOutputValue("test", knownvalue.ObjectExact(map[string]knownvalue.Check{
						"name": knownvalue.StringExact("rg-te--st"),
						"type": knownvalue.StringExact("azurerm_resource_group"),
						"regex": knownvalue.ObjectExact(map[string]knownvalue.Check{
							"valid": knownvalue.Bool(true),
							"match": knownvalue.StringExact("^[a-zA-Z0-9-._()]{0,89}[a-zA-Z0-9-_()]$"),
						}),
						"length": knownvalue.ObjectExact(map[string]knownvalue.Check{
							"valid": knownvalue.Bool(true),
							"is":    knownvalue.Int64Exact(9),
							"max":   knownvalue.Int64Exact(90),
							"min":   knownvalue.Int64Exact(1),
						}),
						"double_hyphens_denied": knownvalue.Bool(false),
						"double_hyphens_found":  knownvalue.Bool(true),
					})),
				},
			},
		},
	})
}

func TestValidateFunction_MultipleValidationFailures(t *testing.T) {
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactoriesUnique(),
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf("%s %s", default_config_with_no_settings_default_precedence, `output "test" {
					value = provider::standesamt::validate(local.config, "azurerm_resource_group", local.settings, "test#12345678901234567890")
				}`),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownOutputValue("test", knownvalue.ObjectExact(map[string]knownvalue.Check{
						"name": knownvalue.StringExact("rg-test#12345678901234567890-we"),
						"type": knownvalue.StringExact("azurerm_resource_group"),
						"regex": knownvalue.ObjectExact(map[string]knownvalue.Check{
							"valid": knownvalue.Bool(false),
							"match": knownvalue.StringExact("^[a-zA-Z0-9-._()]{0,89}[a-zA-Z0-9-_()]$"),
						}),
						"length": knownvalue.ObjectExact(map[string]knownvalue.Check{
							"valid": knownvalue.Bool(false),
							"is":    knownvalue.Int64Exact(31),
							"max":   knownvalue.Int64Exact(20),
							"min":   knownvalue.Int64Exact(8),
						}),
						"double_hyphens_denied": knownvalue.Bool(true),
						"double_hyphens_found":  knownvalue.Bool(false),
					})),
				},
			},
		},
	})
}

func TestValidateFunction_LowerCase(t *testing.T) {
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactoriesUnique(),
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf("%s %s", default_config_with_no_settings_default_precedence, `output "test" {
					value = provider::standesamt::validate(local.config, "azurerm_resource_group", local.settings, "UPPERCASE")
				}`),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownOutputValue("test", knownvalue.ObjectExact(map[string]knownvalue.Check{
						"name": knownvalue.StringExact("rg-uppercase-we"),
						"type": knownvalue.StringExact("azurerm_resource_group"),
						"regex": knownvalue.ObjectExact(map[string]knownvalue.Check{
							"valid": knownvalue.Bool(true),
							"match": knownvalue.StringExact("^[a-zA-Z0-9-._()]{0,89}[a-zA-Z0-9-_()]$"),
						}),
						"length": knownvalue.ObjectExact(map[string]knownvalue.Check{
							"valid": knownvalue.Bool(true),
							"is":    knownvalue.Int64Exact(15),
							"max":   knownvalue.Int64Exact(20),
							"min":   knownvalue.Int64Exact(8),
						}),
						"double_hyphens_denied": knownvalue.Bool(true),
						"double_hyphens_found":  knownvalue.Bool(false),
					})),
				},
			},
		},
	})
}

func TestValidateFunction_AzureCaf(t *testing.T) {
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactoriesUnique(),
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf("%s %s", remote_schema_config_with_no_settings, `output "test" {
					value = provider::standesamt::validate(local.config, "azurerm_resource_group", local.settings, "test")
				}`),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownOutputValue("test", knownvalue.ObjectExact(map[string]knownvalue.Check{
						"name": knownvalue.StringExact("rg-test"),
						"type": knownvalue.StringExact("azurerm_resource_group"),
						"regex": knownvalue.ObjectExact(map[string]knownvalue.Check{
							"valid": knownvalue.Bool(true),
							"match": knownvalue.StringExact("^[a-zA-Z0-9-._()]{0,89}[a-zA-Z0-9-_()]$"),
						}),
						"length": knownvalue.ObjectExact(map[string]knownvalue.Check{
							"valid": knownvalue.Bool(true),
							"is":    knownvalue.Int64Exact(7),
							"max":   knownvalue.Int64Exact(90),
							"min":   knownvalue.Int64Exact(1),
						}),
						"double_hyphens_denied": knownvalue.Bool(false),
						"double_hyphens_found":  knownvalue.Bool(false),
					})),
				},
			},
		},
	})
}

func TestValidateFunction_AzureCaf_Full(t *testing.T) {
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactoriesUnique(),
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf("%s %s", remote_schema_config_with_full_settings, `output "test" {
					value = provider::standesamt::validate(local.config, "azurerm_resource_group", local.settings, "TEST")
				}`),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownOutputValue("test", knownvalue.ObjectExact(map[string]knownvalue.Check{
						"name": knownvalue.StringExact("rg_pre1_pre2_test_we_tst_qffc_suf1_suf2"),
						"type": knownvalue.StringExact("azurerm_resource_group"),
						"regex": knownvalue.ObjectExact(map[string]knownvalue.Check{
							"valid": knownvalue.Bool(true),
							"match": knownvalue.StringExact("^[a-zA-Z0-9-._()]{0,89}[a-zA-Z0-9-_()]$"),
						}),
						"length": knownvalue.ObjectExact(map[string]knownvalue.Check{
							"valid": knownvalue.Bool(true),
							"is":    knownvalue.Int64Exact(39),
							"max":   knownvalue.Int64Exact(90),
							"min":   knownvalue.Int64Exact(1),
						}),
						"double_hyphens_denied": knownvalue.Bool(false),
						"double_hyphens_found":  knownvalue.Bool(false),
					})),
				},
			},
		},
	})
}

func TestValidateFunction_AzureCaf_AllNullValues(t *testing.T) {
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactoriesUnique(),
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf("%s %s", remote_schema_config_with_null_values, `output "test" {
					value = provider::standesamt::validate(local.config, "azurerm_resource_group", local.settings, "test")
				}`),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownOutputValue("test", knownvalue.ObjectExact(map[string]knownvalue.Check{
						"name": knownvalue.StringExact("rg-test"),
						"type": knownvalue.StringExact("azurerm_resource_group"),
						"regex": knownvalue.ObjectExact(map[string]knownvalue.Check{
							"valid": knownvalue.Bool(true),
							"match": knownvalue.StringExact("^[a-zA-Z0-9-._()]{0,89}[a-zA-Z0-9-_()]$"),
						}),
						"length": knownvalue.ObjectExact(map[string]knownvalue.Check{
							"valid": knownvalue.Bool(true),
							"is":    knownvalue.Int64Exact(7),
							"max":   knownvalue.Int64Exact(90),
							"min":   knownvalue.Int64Exact(1),
						}),
						"double_hyphens_denied": knownvalue.Bool(false),
						"double_hyphens_found":  knownvalue.Bool(false),
					})),
				},
			},
		},
	})
}

func TestValidateFunction_AzureCaf_PartialNullValues(t *testing.T) {
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactoriesUnique(),
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf("%s %s", remote_schema_config_with_partial_null_values, `output "test" {
					value = provider::standesamt::validate(local.config, "azurerm_resource_group", local.settings, "TEST")
				}`),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownOutputValue("test", knownvalue.ObjectExact(map[string]knownvalue.Check{
						"name": knownvalue.StringExact("rg_test_we_tst_qffc"),
						"type": knownvalue.StringExact("azurerm_resource_group"),
						"regex": knownvalue.ObjectExact(map[string]knownvalue.Check{
							"valid": knownvalue.Bool(true),
							"match": knownvalue.StringExact("^[a-zA-Z0-9-._()]{0,89}[a-zA-Z0-9-_()]$"),
						}),
						"length": knownvalue.ObjectExact(map[string]knownvalue.Check{
							"valid": knownvalue.Bool(true),
							"is":    knownvalue.Int64Exact(19),
							"max":   knownvalue.Int64Exact(90),
							"min":   knownvalue.Int64Exact(1),
						}),
						"double_hyphens_denied": knownvalue.Bool(false),
						"double_hyphens_found":  knownvalue.Bool(false),
					})),
				},
			},
		},
	})
}
