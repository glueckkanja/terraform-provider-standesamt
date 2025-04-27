// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/stretchr/testify/assert"
	"os"
	s "terraform-provider-standesamt/internal/schema"
	"testing"
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

	diags := data.SchemaReference.As(context.Background(), &sourceRef, basetypes.ObjectAsOptions{})
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
