// Copyright (c) glueckkanja AG
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"crypto/sha256"
	"fmt"
	"io/fs"
	"math"
	"os"
	"strconv"

	"terraform-provider-standesamt/internal/azure"
	s "terraform-provider-standesamt/internal/schema"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/function"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

const (
	standesamtLibRef  = "2025.04"
	standesamtLibPath = "azure/caf"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ provider.Provider              = &StandesamtProvider{}
	_ provider.ProviderWithFunctions = &StandesamtProvider{}
)

// New is a helper function to simplify provider server and testing implementation.
func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &StandesamtProvider{
			version: version,
		}
	}
}

type ProviderConfig struct {
	SourceRef    fs.FS
	ProviderData providerData
	AzureConfig  *azure.Config // Azure configuration for location fetching (nil if not using Azure)
}

// StandesamtProvider is the provider implementation.
type StandesamtProvider struct {
	// version is set to the provider version on release, "dev" when the
	// provider is built and ran locally, and "test" when running acceptance
	// testing.
	version string
	config  *ProviderConfig
}

type providerData struct {
	Convention      types.String `tfsdk:"convention"`
	Environment     types.String `tfsdk:"environment"`
	Separator       types.String `tfsdk:"separator"`
	HashLength      types.Int32  `tfsdk:"hash_length"`
	Lowercase       types.Bool   `tfsdk:"lowercase"`
	RandomSeed      types.Int64  `tfsdk:"random_seed"`
	SchemaReference types.Object `tfsdk:"schema_reference"`
	LocationSource  types.String `tfsdk:"location_source"`
	LocationAliases types.Map    `tfsdk:"location_aliases"`
	AzureConfig     types.Object `tfsdk:"azure"`
}

// AzureConfigValue represents the Azure authentication configuration
type AzureConfigValue struct {
	UseCli                    types.Bool   `tfsdk:"use_cli"`
	UseMsi                    types.Bool   `tfsdk:"use_msi"`
	UseOidc                   types.Bool   `tfsdk:"use_oidc"`
	ClientId                  types.String `tfsdk:"client_id"`
	ClientSecret              types.String `tfsdk:"client_secret"`
	ClientCertificatePath     types.String `tfsdk:"client_certificate_path"`
	ClientCertificatePassword types.String `tfsdk:"client_certificate_password"`
	TenantId                  types.String `tfsdk:"tenant_id"`
	SubscriptionId            types.String `tfsdk:"subscription_id"`
	Environment               types.String `tfsdk:"environment"`
}

// ToAzureConfig converts AzureConfigValue to azure.Config
func (a *AzureConfigValue) ToAzureConfig() *azure.Config {
	config := &azure.Config{
		UseCli:                    a.UseCli.ValueBool(),
		UseMsi:                    a.UseMsi.ValueBool(),
		UseOidc:                   a.UseOidc.ValueBool(),
		ClientId:                  a.ClientId.ValueString(),
		ClientSecret:              a.ClientSecret.ValueString(),
		ClientCertificatePath:     a.ClientCertificatePath.ValueString(),
		ClientCertificatePassword: a.ClientCertificatePassword.ValueString(),
		TenantId:                  a.TenantId.ValueString(),
		SubscriptionId:            a.SubscriptionId.ValueString(),
	}

	// Set environment
	env := a.Environment.ValueString()
	switch env {
	case "usgovernment":
		config.Environment = azure.CloudEnvironmentUSGovernment
	case "china":
		config.Environment = azure.CloudEnvironmentChina
	default:
		config.Environment = azure.CloudEnvironmentPublic
	}

	return config
}

// getAzureConfig extracts and converts the Azure configuration from providerData
func (d *providerData) getAzureConfig(ctx context.Context) (*azure.Config, diag.Diagnostics) {
	if d.AzureConfig.IsNull() {
		return nil, nil
	}

	var azureConfigValue AzureConfigValue
	diags := d.AzureConfig.As(ctx, &azureConfigValue, basetypes.ObjectAsOptions{
		UnhandledNullAsEmpty:    true,
		UnhandledUnknownAsEmpty: true,
	})
	if diags.HasError() {
		return nil, diags
	}

	return azureConfigValue.ToAzureConfig(), nil
}

// getLocationAliases extracts the location aliases map from providerData
func (d *providerData) getLocationAliases(ctx context.Context) (map[string]string, diag.Diagnostics) {
	if d.LocationAliases.IsNull() {
		return nil, nil
	}

	aliases := make(map[string]string)
	diags := d.LocationAliases.ElementsAs(ctx, &aliases, false)
	if diags.HasError() {
		return nil, diags
	}

	return aliases, nil
}

// Metadata returns the provider type name.
func (p *StandesamtProvider) Metadata(_ context.Context, _ provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "standesamt"
	resp.Version = p.version
}

func (d providerData) getSourceRef(ctx context.Context) (s.Source, diag.Diagnostics) {

	var sourceValue s.SourceValue

	diags := d.SchemaReference.As(ctx, &sourceValue, basetypes.ObjectAsOptions{
		UnhandledNullAsEmpty:    false,
		UnhandledUnknownAsEmpty: false,
	})
	if diags.HasError() {
		return nil, diags
	}

	if sourceValue.CustomUrl.IsNull() {
		return s.NewDefaultSource(sourceValue.Path.ValueString(), sourceValue.Ref.ValueString()), nil
	}

	return s.NewCustomSource(sourceValue.CustomUrl.ValueString()), nil

}

// Schema defines the provider-level schema for configuration data.
func (p *StandesamtProvider) Schema(_ context.Context, _ provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"convention": schema.StringAttribute{
				Optional:            true,
				Description:         "Define the convention for naming results. Possible values are 'default' and 'passthrough'. Default 'default'",
				MarkdownDescription: "Define the convention for naming results. Possible values are 'default' and 'passthrough'. Default 'default'",
				Validators: []validator.String{
					stringvalidator.OneOf("default", "passthrough"),
				},
			},
			"environment": schema.StringAttribute{
				Optional:            true,
				Description:         "Define the environment for the naming schema. Normally this is the name of the environment, e.g. 'prod', 'dev', 'test'.",
				MarkdownDescription: "Define the environment for the naming schema. Normally this is the name of the environment, e.g. 'prod', 'dev', 'test'.",
			},
			"separator": schema.StringAttribute{
				Optional:            true,
				Description:         "The separator to use for generating the resulting name. Default '-'",
				MarkdownDescription: "The separator to use for generating the resulting name. Default '-'",
			},
			"random_seed": schema.Int64Attribute{
				Optional:            true,
				Description:         "A random seed used by the random number generator. This is used to generate a random name for the naming schema. The default value is 1337. Make sure to update this value to avoid collisions for globally unique names.",
				MarkdownDescription: "A random seed used by the random number generator. This is used to generate a random name for the naming schema. The default value is 1337. Make sure to update this value to avoid collisions for globally unique names.",
			},
			"hash_length": schema.Int32Attribute{
				Optional:            true,
				Description:         "Default hash length. Overrides all schema configurations.",
				MarkdownDescription: "Default hash length. Overrides all schema configurations.",
			},
			"lowercase": schema.BoolAttribute{
				Optional:            true,
				Description:         "Control if the resulting name should be lower case. Default 'false'",
				MarkdownDescription: "Control if the resulting name should be lower case. Default 'false'",
			},
			"schema_reference": schema.SingleNestedAttribute{
				Attributes: map[string]schema.Attribute{
					"custom_url": schema.StringAttribute{
						Optional:            true,
						Sensitive:           true,
						Description:         "A custom path/URL to the schema reference to use. Conflicts with `path` and `ref`. For supported protocols, see [go-getter](https://pkg.go.dev/github.com/hashicorp/go-getter/v2). Value is marked sensitive as may contain secrets.",
						MarkdownDescription: "A custom path/URL to the schema reference to use. Conflicts with `path` and `ref`. For supported protocols, see [go-getter](https://pkg.go.dev/github.com/hashicorp/go-getter/v2). Value is marked sensitive as may contain secrets.",
						Validators: []validator.String{
							stringvalidator.ConflictsWith(path.MatchRelative().AtParent().AtName("path")),
							stringvalidator.ConflictsWith(path.MatchRelative().AtParent().AtName("ref")),
						},
					},
					"path": schema.StringAttribute{
						Optional:            true,
						Description:         "The path in the default schema library, e.g. `azure/caf`. Also requires `ref`. Conflicts with `custom_url`.",
						MarkdownDescription: "The path in the default schema library, e.g. `azure/caf`. Also requires `ref`. Conflicts with `custom_url`.",
						Validators: []validator.String{
							stringvalidator.ConflictsWith(path.MatchRelative().AtParent().AtName("custom_url")),
							stringvalidator.AlsoRequires(path.MatchRelative().AtParent().AtName("ref")),
						},
					},
					"ref": schema.StringAttribute{
						Optional:            true,
						Description:         "This is the version of the schema reference to use, e.g. `2025.04`. Also requires `path`. Conflicts with `custom_url`.",
						MarkdownDescription: "This is the version of the schema reference to use, e.g. `2025.04`. Also requires `path`. Conflicts with `custom_url`.",
						Validators: []validator.String{
							stringvalidator.ConflictsWith(path.MatchRelative().AtParent().AtName("custom_url")),
							stringvalidator.AlsoRequires(path.MatchRelative().AtParent().AtName("path")),
						},
					},
				},
				Optional:            true,
				Description:         "A reference to a naming schema library to use. The reference should either contain a `path` (e.g. `azure/caf`) and the `ref` (e.g. `2025.04`), or a `custom_url` to be supplied to go-getter.\n    If this value is not specified, the default value will be used, which is:\n\n    ```terraform\n\n    schema_reference = {\n      path = \"azure/caf\",\n      ref = \"2025.04\"\n    }\n\n    ```\n\n    The reference is using the [default standesamt library](https://github.com/glueckkanja/standesamt-schema-library).",
				MarkdownDescription: "A reference to a Naming schema library to use. The reference should either contain a `path` (e.g. `azure/caf`) and the `ref` (e.g. `2025.04`), or a `custom_url` to be supplied to go-getter.\n    If this value is not specified, the default value will be used, which is:\n\n    ```terraform\n\n    schema_reference = {\n      path = \"azure/caf\",\n      ref = \"2025.04\"\n    }\n\n    ```\n\n    The reference is using the [default standesamt library](https://github.com/glueckkanja/standesamt-schema-library).",
			},
			"location_source": schema.StringAttribute{
				Optional:            true,
				Description:         "The source for location data. Possible values are 'schema' (default) to use the schema library, or 'azure' to fetch locations from the Azure Resource Manager API.",
				MarkdownDescription: "The source for location data. Possible values are `schema` (default) to use the schema library, or `azure` to fetch locations from the Azure Resource Manager API.",
				Validators: []validator.String{
					stringvalidator.OneOf("schema", "azure"),
				},
			},
			"location_aliases": schema.MapAttribute{
				Optional:            true,
				ElementType:         types.StringType,
				Description:         "A map of location name aliases. Use this to remap location short names, e.g. { eastus = \"eus\", westeurope = \"weu\" }. The key is the original name (from schema or Azure API), the value is the replacement.",
				MarkdownDescription: "A map of location name aliases. Use this to remap location short names, e.g. `{ eastus = \"eus\", westeurope = \"weu\" }`. The key is the original name (from schema or Azure API), the value is the replacement.",
			},
			"azure": schema.SingleNestedAttribute{
				Attributes: map[string]schema.Attribute{
					"use_cli": schema.BoolAttribute{
						Optional:            true,
						Description:         "Use Azure CLI for authentication. Default 'true'.",
						MarkdownDescription: "Use Azure CLI for authentication. Default `true`.",
					},
					"use_msi": schema.BoolAttribute{
						Optional:            true,
						Description:         "Use Managed Service Identity for authentication. Default 'false'.",
						MarkdownDescription: "Use Managed Service Identity for authentication. Default `false`.",
					},
					"use_oidc": schema.BoolAttribute{
						Optional:            true,
						Description:         "Use OpenID Connect for authentication. Default 'false'.",
						MarkdownDescription: "Use OpenID Connect for authentication. Default `false`.",
					},
					"client_id": schema.StringAttribute{
						Optional:            true,
						Description:         "The Client ID for Service Principal authentication.",
						MarkdownDescription: "The Client ID for Service Principal authentication.",
					},
					"client_secret": schema.StringAttribute{
						Optional:            true,
						Sensitive:           true,
						Description:         "The Client Secret for Service Principal authentication.",
						MarkdownDescription: "The Client Secret for Service Principal authentication.",
					},
					"client_certificate_path": schema.StringAttribute{
						Optional:            true,
						Description:         "The path to a client certificate for Service Principal authentication.",
						MarkdownDescription: "The path to a client certificate for Service Principal authentication.",
					},
					"client_certificate_password": schema.StringAttribute{
						Optional:            true,
						Sensitive:           true,
						Description:         "The password for the client certificate.",
						MarkdownDescription: "The password for the client certificate.",
					},
					"tenant_id": schema.StringAttribute{
						Optional:            true,
						Description:         "The Tenant ID for authentication.",
						MarkdownDescription: "The Tenant ID for authentication.",
					},
					"subscription_id": schema.StringAttribute{
						Optional:            true,
						Description:         "The Subscription ID to use for fetching Azure locations. Required when location_source is 'azure'.",
						MarkdownDescription: "The Subscription ID to use for fetching Azure locations. Required when `location_source` is `azure`.",
					},
					"environment": schema.StringAttribute{
						Optional:            true,
						Description:         "The Azure environment to use. Possible values are 'public', 'usgovernment', 'china'. Default 'public'.",
						MarkdownDescription: "The Azure environment to use. Possible values are `public`, `usgovernment`, `china`. Default `public`.",
						Validators: []validator.String{
							stringvalidator.OneOf("public", "usgovernment", "china"),
						},
					},
				},
				Optional:            true,
				Description:         "Azure authentication configuration. Required when location_source is 'azure'. Supports multiple authentication methods similar to the azurerm provider.",
				MarkdownDescription: "Azure authentication configuration. Required when `location_source` is `azure`. Supports multiple authentication methods similar to the azurerm provider.",
			},
		},
	}
}

func (d *providerData) configProviderFromEnvironment() diag.Diagnostics {
	var diags diag.Diagnostics

	if val := os.Getenv("SA_ENVIRONMENT"); val != "" && d.Environment.IsNull() {
		d.Environment = types.StringValue(val)
	}

	if val := os.Getenv("SA_CONVENTION"); val != "" && d.Convention.IsNull() {
		if val != "default" && val != "passthrough" {
			diags.AddError("Invalid Environment Variable", fmt.Sprintf("Invalid value for SA_CONVENTION: %s", val))
			return diags
		}
		d.Convention = types.StringValue(val)
	}

	if val := os.Getenv("SA_SEPARATOR"); val != "" && d.Separator.IsNull() {
		d.Separator = types.StringValue(val)
	}

	if val := os.Getenv("SA_RANDOM_SEED"); val != "" && d.RandomSeed.IsNull() {
		i, err := strconv.ParseInt(val, 10, 64)
		if err != nil {
			diags.AddError("Invalid Environment Variable", fmt.Sprintf("Invalid value for SA_RANDOM_SEED: %s", err))
			return diags
		}
		d.RandomSeed = types.Int64Value(i)
	}

	if val := os.Getenv("SA_HASH_LENGTH"); val != "" && d.HashLength.IsNull() {
		i, err := strconv.Atoi(val)
		if err != nil {
			diags.AddError("Invalid Environment Variable", fmt.Sprintf("Invalid value for SA_HASH_LENGTH: %s", err))
			return diags
		}
		if i > 0 && i <= math.MaxInt32 {
			d.HashLength = types.Int32Value(int32(i))
		} else {
			diags.AddError("Invalid Environment Variable", fmt.Sprintf("Invalid value for SA_HASH_LENGTH: %s (parsed as %d), must be between 1 and %d", val, i, math.MaxInt32))
			return diags
		}
	}

	if val := os.Getenv("SA_LOWERCASE"); val != "" && d.Lowercase.IsNull() {
		d.Lowercase = types.BoolValue(val == "true")
	}

	if val := os.Getenv("SA_LOCATION_SOURCE"); val != "" && d.LocationSource.IsNull() {
		if val != "schema" && val != "azure" {
			diags.AddError("Invalid Environment Variable", fmt.Sprintf("Invalid value for SA_LOCATION_SOURCE: %s. Must be 'schema' or 'azure'.", val))
			return diags
		}
		d.LocationSource = types.StringValue(val)
	}

	// Configure Azure settings from environment variables (ARM_* for compatibility with azurerm)
	if err := d.configAzureFromEnvironment(); err != nil {
		diags.AddError("Invalid Environment Variable", err.Error())
		return diags
	}

	return nil
}

func (d *providerData) configProviderDefaults() {
	if d.Convention.IsNull() {
		d.Convention = types.StringValue("default")
	}

	if d.Environment.IsNull() {
		d.Environment = types.StringValue("")
	}

	if d.Separator.IsNull() {
		d.Separator = types.StringValue("-")
	}

	if d.RandomSeed.IsNull() {
		d.RandomSeed = types.Int64Value(1337)
	}

	if d.HashLength.IsNull() {
		d.HashLength = types.Int32Value(0)
	}

	if d.Lowercase.IsNull() {
		d.Lowercase = types.BoolValue(false)
	}

	if d.SchemaReference.IsNull() {
		d.SchemaReference, _ = types.ObjectValue(
			map[string]attr.Type{
				"ref":        types.StringType,
				"path":       types.StringType,
				"custom_url": types.StringType,
			},
			map[string]attr.Value{
				"ref":        types.StringValue(standesamtLibRef),
				"path":       types.StringValue(standesamtLibPath),
				"custom_url": types.StringNull(),
			})
	}

	if d.LocationSource.IsNull() {
		d.LocationSource = types.StringValue("schema")
	}
}

// configAzureFromEnvironment configures Azure settings from environment variables.
// Uses ARM_* variables for compatibility with the azurerm provider.
func (d *providerData) configAzureFromEnvironment() error {
	// If AzureConfig is already set, extract current values
	var azureConfig AzureConfigValue
	hasExistingConfig := !d.AzureConfig.IsNull()

	if hasExistingConfig {
		// We need to manually check each field since we can't easily convert
		// For now, we'll only set values if the entire azure block is null
		return nil
	}

	// Check if any Azure environment variables are set
	envVars := map[string]string{
		"ARM_CLIENT_ID":                   os.Getenv("ARM_CLIENT_ID"),
		"ARM_CLIENT_SECRET":               os.Getenv("ARM_CLIENT_SECRET"),
		"ARM_CLIENT_CERTIFICATE_PATH":     os.Getenv("ARM_CLIENT_CERTIFICATE_PATH"),
		"ARM_CLIENT_CERTIFICATE_PASSWORD": os.Getenv("ARM_CLIENT_CERTIFICATE_PASSWORD"),
		"ARM_TENANT_ID":                   os.Getenv("ARM_TENANT_ID"),
		"ARM_SUBSCRIPTION_ID":             os.Getenv("ARM_SUBSCRIPTION_ID"),
		"ARM_ENVIRONMENT":                 os.Getenv("ARM_ENVIRONMENT"),
		"ARM_USE_CLI":                     os.Getenv("ARM_USE_CLI"),
		"ARM_USE_MSI":                     os.Getenv("ARM_USE_MSI"),
		"ARM_USE_OIDC":                    os.Getenv("ARM_USE_OIDC"),
	}

	// Check if any ARM_* variables are set
	hasAnyEnvVar := false
	for _, v := range envVars {
		if v != "" {
			hasAnyEnvVar = true
			break
		}
	}

	if !hasAnyEnvVar {
		return nil
	}

	// Build AzureConfigValue from environment variables
	azureConfig = AzureConfigValue{
		ClientId:                  types.StringNull(),
		ClientSecret:              types.StringNull(),
		ClientCertificatePath:     types.StringNull(),
		ClientCertificatePassword: types.StringNull(),
		TenantId:                  types.StringNull(),
		SubscriptionId:            types.StringNull(),
		Environment:               types.StringNull(),
		UseCli:                    types.BoolNull(),
		UseMsi:                    types.BoolNull(),
		UseOidc:                   types.BoolNull(),
	}

	if v := envVars["ARM_CLIENT_ID"]; v != "" {
		azureConfig.ClientId = types.StringValue(v)
	}
	if v := envVars["ARM_CLIENT_SECRET"]; v != "" {
		azureConfig.ClientSecret = types.StringValue(v)
	}
	if v := envVars["ARM_CLIENT_CERTIFICATE_PATH"]; v != "" {
		azureConfig.ClientCertificatePath = types.StringValue(v)
	}
	if v := envVars["ARM_CLIENT_CERTIFICATE_PASSWORD"]; v != "" {
		azureConfig.ClientCertificatePassword = types.StringValue(v)
	}
	if v := envVars["ARM_TENANT_ID"]; v != "" {
		azureConfig.TenantId = types.StringValue(v)
	}
	if v := envVars["ARM_SUBSCRIPTION_ID"]; v != "" {
		azureConfig.SubscriptionId = types.StringValue(v)
	}
	if v := envVars["ARM_ENVIRONMENT"]; v != "" {
		if v != "public" && v != "usgovernment" && v != "china" {
			return fmt.Errorf("invalid value for ARM_ENVIRONMENT: %s. Must be 'public', 'usgovernment', or 'china'", v)
		}
		azureConfig.Environment = types.StringValue(v)
	}
	if v := envVars["ARM_USE_CLI"]; v != "" {
		azureConfig.UseCli = types.BoolValue(v == "true")
	}
	if v := envVars["ARM_USE_MSI"]; v != "" {
		azureConfig.UseMsi = types.BoolValue(v == "true")
	}
	if v := envVars["ARM_USE_OIDC"]; v != "" {
		azureConfig.UseOidc = types.BoolValue(v == "true")
	}

	// Convert to types.Object
	azureConfigObj, diags := types.ObjectValueFrom(context.Background(), map[string]attr.Type{
		"use_cli":                     types.BoolType,
		"use_msi":                     types.BoolType,
		"use_oidc":                    types.BoolType,
		"client_id":                   types.StringType,
		"client_secret":               types.StringType,
		"client_certificate_path":     types.StringType,
		"client_certificate_password": types.StringType,
		"tenant_id":                   types.StringType,
		"subscription_id":             types.StringType,
		"environment":                 types.StringType,
	}, azureConfig)

	if diags.HasError() {
		return fmt.Errorf("failed to create Azure config from environment variables: %v", diags)
	}

	d.AzureConfig = azureConfigObj
	return nil
}

// Configure prepares an API client for data sources and resources.
func (p *StandesamtProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var data providerData
	tflog.Debug(ctx, "Provider configuration started.")

	if p.config != nil {
		tflog.Debug(ctx, "Provider configuration is already present, skipping configuration part.")
		resp.DataSourceData = p.config
		return
	}

	if resp.Diagnostics.Append(req.Config.Get(ctx, &data)...); resp.Diagnostics.HasError() {
		return
	}

	if resp.Diagnostics.Append(data.configProviderFromEnvironment()...); resp.Diagnostics.HasError() {
		return
	}

	data.configProviderDefaults()

	sourceRef, diags := data.getSourceRef(ctx)
	resp.Diagnostics = append(resp.Diagnostics, diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Download the schema reference
	f, err := sourceRef.Download(ctx, hash(sourceRef))
	if err != nil {
		resp.Diagnostics.AddError("source_reference", err.Error())
		return
	}

	// Extract Azure configuration if location_source is 'azure'
	var azureConfig *azure.Config
	if data.LocationSource.ValueString() == "azure" {
		azureConfig, diags = data.getAzureConfig(ctx)
		resp.Diagnostics = append(resp.Diagnostics, diags...)
		if resp.Diagnostics.HasError() {
			return
		}

		if azureConfig == nil {
			resp.Diagnostics.AddError(
				"Missing Azure Configuration",
				"When location_source is 'azure', the azure block must be configured with at least a subscription_id.",
			)
			return
		}

		if err := azureConfig.Validate(); err != nil {
			resp.Diagnostics.AddError("Azure Configuration Error", err.Error())
			return
		}

		tflog.Debug(ctx, "Azure location source configured", map[string]interface{}{
			"subscription_id": azureConfig.SubscriptionId,
			"environment":     azureConfig.Environment,
		})
	}

	p.config = &ProviderConfig{
		SourceRef:    f,
		ProviderData: data,
		AzureConfig:  azureConfig,
	}

	resp.DataSourceData = p.config
}

func hash(s fmt.Stringer) string {
	return hashStr(s.String())
}

// hash returns the SHA224 hash of a string, as a string.
func hashStr(s string) string {
	return fmt.Sprintf("%x", sha256.Sum224([]byte(s)))
}

// DataSources defines the data sources implemented in the provider.
func (p *StandesamtProvider) DataSources(_ context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		NewSchemaDataSource,
		NewLocationDataSource,
	}
}

// Resources defines the resources implemented in the provider.
func (p *StandesamtProvider) Resources(_ context.Context) []func() resource.Resource {
	return nil
}

// Functions defines the functions implemented in the provider.
func (p *StandesamtProvider) Functions(_ context.Context) []func() function.Function {
	return []func() function.Function{
		NewNameFunction,
		NewValidateFunction,
	}
}
