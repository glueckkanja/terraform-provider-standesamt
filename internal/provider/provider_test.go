// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"os"
	s "terraform-provider-standesamt/internal/schema"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/stretchr/testify/assert"
)

// testAccProtoV6ProviderFactories is used to instantiate a provider during acceptance testing.
// The factory function is called for each Terraform CLI command to create a provider
// server that the CLI can connect to and interact with.
//var testAccProtoV6ProviderFactories = map[string]func() (tfprotov6.ProviderServer, error){
//	"standesamt": providerserver.NewProtocol6WithError(New("test")()),
//}

func testAccProtoV6ProviderFactoriesUnique() map[string]func() (tfprotov6.ProviderServer, error) {
	return map[string]func() (tfprotov6.ProviderServer, error){
		"standesamt": providerserver.NewProtocol6WithError(New("test")()),
	}
}

// testAccProtoV6ProviderFactoriesWithEcho includes the echo provider alongside the scaffolding provider.
// It allows for testing assertions on data returned by an ephemeral resource during Open.
// The echoprovider is used to arrange tests by echoing ephemeral data into the Terraform state.
// This lets the data be referenced in test assertions with state checks.
//var testAccProtoV6ProviderFactoriesWithEcho = map[string]func() (tfprotov6.ProviderServer, error){
//	"naming": providerserver.NewProtocol6WithError(New("test")()),
//	"echo":   echoprovider.NewProviderServer(),
//}

func testAccPreCheck(t *testing.T) {
	// You can add code here to run prior to any test case execution, for example assertions
	// about the appropriate environment variables being set are common to see in a pre-check
	// function.

}

func TestProviderDefaults(t *testing.T) {
	// This test checks that the default values are set correctly in the provider.
	// It does not check that the values are actually used in the provider logic.
	// The actual logic is tested in the other tests.
	_ = os.Unsetenv("SA_ENVIRONMENT")
	_ = os.Unsetenv("SA_CONVENTION")
	_ = os.Unsetenv("SA_SEPARATOR")
	_ = os.Unsetenv("SA_RANDOM_SEED")
	_ = os.Unsetenv("SA_HASH_LENGTH")
	_ = os.Unsetenv("SA_LOWERCASE")

	data := &providerData{}
	data.configProviderDefaults()

	var sourceRef s.SourceValue

	diags := data.SchemaReference.As(t.Context(), &sourceRef, basetypes.ObjectAsOptions{})
	assert.False(t, diags.HasError())

	assert.Equal(t, "default", data.Convention.ValueString())
	assert.Equal(t, "", data.Environment.ValueString())
	assert.Equal(t, "-", data.Separator.ValueString())
	assert.Equal(t, int64(1337), data.RandomSeed.ValueInt64())
	assert.Equal(t, int32(0), data.HashLength.ValueInt32())
	assert.Equal(t, false, data.Lowercase.ValueBool())
	assert.Equal(t, "2025.04", sourceRef.Ref.ValueString())
	assert.Equal(t, "azure/caf", sourceRef.Path.ValueString())
	assert.Equal(t, "", sourceRef.CustomUrl.ValueString())

}

func TestConfigureFromEnvironment(t *testing.T) {
	var diags diag.Diagnostics
	// Unset all environment variables
	_ = os.Unsetenv("SA_ENVIRONMENT")
	_ = os.Unsetenv("SA_CONVENTION")
	_ = os.Unsetenv("SA_SEPARATOR")
	_ = os.Unsetenv("SA_RANDOM_SEED")
	_ = os.Unsetenv("SA_HASH_LENGTH")
	_ = os.Unsetenv("SA_LOWERCASE")
	_ = os.Unsetenv("SA_LOCATION_SOURCE")

	data := &providerData{}
	data.configProviderFromEnvironment()

	assert.True(t, data.Environment.IsNull())

	t.Setenv("SA_ENVIRONMENT", "tst")
	t.Setenv("SA_CONVENTION", "default")
	t.Setenv("SA_SEPARATOR", "-")
	t.Setenv("SA_RANDOM_SEED", "1234")
	t.Setenv("SA_HASH_LENGTH", "8")
	t.Setenv("SA_LOWERCASE", "true")

	data = &providerData{}
	diags = data.configProviderFromEnvironment()

	assert.Equal(t, "tst", data.Environment.ValueString())
	assert.Equal(t, "default", data.Convention.ValueString())
	assert.Equal(t, "-", data.Separator.ValueString())
	assert.Equal(t, int64(1234), data.RandomSeed.ValueInt64())
	assert.Equal(t, int32(8), data.HashLength.ValueInt32())
	assert.Equal(t, true, data.Lowercase.ValueBool())
	assert.Empty(t, diags)

	// Unset all environment variables
	_ = os.Unsetenv("SA_ENVIRONMENT")
	_ = os.Unsetenv("SA_CONVENTION")
	_ = os.Unsetenv("SA_SEPARATOR")
	_ = os.Unsetenv("SA_RANDOM_SEED")
	_ = os.Unsetenv("SA_HASH_LENGTH")
	_ = os.Unsetenv("SA_LOWERCASE")

	t.Setenv("SA_CONVENTION", "invalid")

	data = &providerData{}

	diags = data.configProviderFromEnvironment()
	assert.True(t, data.Convention.IsNull())
	assert.True(t, diags.HasError())
}

func TestConfigureLocationSourceFromEnvironment(t *testing.T) {
	// Unset all environment variables
	_ = os.Unsetenv("SA_LOCATION_SOURCE")

	data := &providerData{}
	diags := data.configProviderFromEnvironment()
	assert.False(t, diags.HasError())
	assert.True(t, data.LocationSource.IsNull())

	// Test valid value: schema
	t.Setenv("SA_LOCATION_SOURCE", "schema")
	data = &providerData{}
	diags = data.configProviderFromEnvironment()
	assert.False(t, diags.HasError())
	assert.Equal(t, "schema", data.LocationSource.ValueString())

	// Test valid value: azure
	t.Setenv("SA_LOCATION_SOURCE", "azure")
	data = &providerData{}
	diags = data.configProviderFromEnvironment()
	assert.False(t, diags.HasError())
	assert.Equal(t, "azure", data.LocationSource.ValueString())

	// Test invalid value
	t.Setenv("SA_LOCATION_SOURCE", "invalid")
	data = &providerData{}
	diags = data.configProviderFromEnvironment()
	assert.True(t, diags.HasError())
	assert.True(t, data.LocationSource.IsNull())
}

func TestConfigureAzureFromEnvironment(t *testing.T) {
	// Clean up all ARM_* environment variables
	armVars := []string{
		"ARM_CLIENT_ID",
		"ARM_CLIENT_SECRET",
		"ARM_CLIENT_CERTIFICATE_PATH",
		"ARM_CLIENT_CERTIFICATE_PASSWORD",
		"ARM_TENANT_ID",
		"ARM_SUBSCRIPTION_ID",
		"ARM_ENVIRONMENT",
		"ARM_USE_CLI",
		"ARM_USE_MSI",
		"ARM_USE_OIDC",
	}
	for _, v := range armVars {
		_ = os.Unsetenv(v)
	}

	// Test: no ARM variables set, AzureConfig should remain null
	data := &providerData{}
	err := data.configAzureFromEnvironment()
	assert.NoError(t, err)
	assert.True(t, data.AzureConfig.IsNull())

	// Test: set some ARM variables
	t.Setenv("ARM_SUBSCRIPTION_ID", "test-sub-id")
	t.Setenv("ARM_TENANT_ID", "test-tenant-id")
	t.Setenv("ARM_CLIENT_ID", "test-client-id")
	t.Setenv("ARM_USE_CLI", "true")

	data = &providerData{}
	err = data.configAzureFromEnvironment()
	assert.NoError(t, err)
	assert.False(t, data.AzureConfig.IsNull())

	// Extract and verify values
	azureConfig, diags := data.getAzureConfig(t.Context())
	assert.False(t, diags.HasError())
	assert.NotNil(t, azureConfig)
	assert.Equal(t, "test-sub-id", azureConfig.SubscriptionId)
	assert.Equal(t, "test-tenant-id", azureConfig.TenantId)
	assert.Equal(t, "test-client-id", azureConfig.ClientId)
	assert.True(t, azureConfig.UseCli)
}

func TestConfigureAzureEnvironmentValidation(t *testing.T) {
	// Clean up
	_ = os.Unsetenv("ARM_ENVIRONMENT")
	_ = os.Unsetenv("ARM_SUBSCRIPTION_ID")

	// Test valid environments
	validEnvs := []string{"public", "usgovernment", "china"}
	for _, env := range validEnvs {
		t.Run("valid_"+env, func(t *testing.T) {
			t.Setenv("ARM_ENVIRONMENT", env)
			t.Setenv("ARM_SUBSCRIPTION_ID", "test-sub")

			data := &providerData{}
			err := data.configAzureFromEnvironment()
			assert.NoError(t, err)
		})
	}

	// Test invalid environment
	t.Run("invalid_environment", func(t *testing.T) {
		t.Setenv("ARM_ENVIRONMENT", "invalid")
		t.Setenv("ARM_SUBSCRIPTION_ID", "test-sub")

		data := &providerData{}
		err := data.configAzureFromEnvironment()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid value for ARM_ENVIRONMENT")
	})
}

func TestProviderDefaultsLocationSource(t *testing.T) {
	_ = os.Unsetenv("SA_LOCATION_SOURCE")

	data := &providerData{}
	data.configProviderDefaults()

	assert.Equal(t, "schema", data.LocationSource.ValueString())
}

func TestAzureConfigValueToAzureConfig(t *testing.T) {
	tests := []struct {
		name     string
		input    AzureConfigValue
		expected struct {
			useCli         bool
			useMsi         bool
			useOidc        bool
			clientId       string
			subscriptionId string
			environment    string
		}
	}{
		{
			name: "basic CLI config",
			input: AzureConfigValue{
				UseCli:         basetypes.NewBoolValue(true),
				UseMsi:         basetypes.NewBoolValue(false),
				UseOidc:        basetypes.NewBoolValue(false),
				SubscriptionId: basetypes.NewStringValue("sub-123"),
				ClientId:       basetypes.NewStringNull(),
				ClientSecret:   basetypes.NewStringNull(),
				TenantId:       basetypes.NewStringNull(),
				Environment:    basetypes.NewStringNull(),
			},
			expected: struct {
				useCli         bool
				useMsi         bool
				useOidc        bool
				clientId       string
				subscriptionId string
				environment    string
			}{
				useCli:         true,
				useMsi:         false,
				useOidc:        false,
				subscriptionId: "sub-123",
				environment:    "public",
			},
		},
		{
			name: "service principal config",
			input: AzureConfigValue{
				UseCli:         basetypes.NewBoolValue(false),
				UseMsi:         basetypes.NewBoolValue(false),
				UseOidc:        basetypes.NewBoolValue(false),
				SubscriptionId: basetypes.NewStringValue("sub-456"),
				ClientId:       basetypes.NewStringValue("client-id"),
				ClientSecret:   basetypes.NewStringValue("client-secret"),
				TenantId:       basetypes.NewStringValue("tenant-id"),
				Environment:    basetypes.NewStringValue("usgovernment"),
			},
			expected: struct {
				useCli         bool
				useMsi         bool
				useOidc        bool
				clientId       string
				subscriptionId string
				environment    string
			}{
				useCli:         false,
				subscriptionId: "sub-456",
				clientId:       "client-id",
				environment:    "usgovernment",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.input.ToAzureConfig()

			assert.Equal(t, tt.expected.useCli, result.UseCli)
			assert.Equal(t, tt.expected.useMsi, result.UseMsi)
			assert.Equal(t, tt.expected.useOidc, result.UseOidc)
			assert.Equal(t, tt.expected.subscriptionId, result.SubscriptionId)
			assert.Equal(t, tt.expected.clientId, result.ClientId)
		})
	}
}
