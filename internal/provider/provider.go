// Copyright (c) glueckkanja AG
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"crypto/sha256"
	"fmt"
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
	"io/fs"
	"os"
	"strconv"
	s "terraform-provider-standesamt/internal/schema"
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
}

// Metadata returns the provider type name.
func (p *StandesamtProvider) Metadata(_ context.Context, _ provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "standesamt"
	resp.Version = p.version
}

func (m providerData) getSourceRef(ctx context.Context) (s.Source, diag.Diagnostics) {

	var sourceValue s.SourceValue

	diags := m.SchemaReference.As(ctx, &sourceValue, basetypes.ObjectAsOptions{
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
						Description:         "This is the version of the schema reference to use, e.g. `v1`. Also requires `path`. Conflicts with `custom_url`.",
						MarkdownDescription: "This is the version of the schema reference to use, e.g. `v1`. Also requires `path`. Conflicts with `custom_url`.",
						Validators: []validator.String{
							stringvalidator.ConflictsWith(path.MatchRelative().AtParent().AtName("custom_url")),
							stringvalidator.AlsoRequires(path.MatchRelative().AtParent().AtName("path")),
						},
					},
				},
				Optional:            true,
				Description:         "A reference to a naming schema library to use. The reference should either contain a `path` (e.g. `azure/caf`) and the `ref` (e.g. `v1`), or a `custom_url` to be supplied to go-getter.\nIf this value is not specified, the default value will be used, which is:\n\n```terraform\nschema_reference= {\n path = \"azure/caf\", ref = \"v1\"\n}\n```\n\n",
				MarkdownDescription: "A reference to a Naming schema library to use. The reference should either contain a `path` (e.g. `azure/caf`) and the `ref` (e.g. `v1`), or a `custom_url` to be supplied to go-getter.\nIf this value is not specified, the default value will be used, which is:\n\n```terraform\nschema_reference= {\n path = \"azure/caf\", ref = \"v1\"\n}\n```\n\n",
			},
		},
	}
}

// Configure prepares an API client for data sources and resources.
func (p *StandesamtProvider) Configure(ctx context.Context, req provider.ConfigureRequest, response *provider.ConfigureResponse) {
	var model providerData
	tflog.Debug(ctx, "Provider configuration started")

	if p.config != nil {
		tflog.Debug(ctx, "provider naming already present, skipping configuration")
		return
	}

	if response.Diagnostics.Append(req.Config.Get(ctx, &model)...); response.Diagnostics.HasError() {
		return
	}

	if model.Environment.IsNull() {
		if v := os.Getenv("SA_ENVIRONMENT"); v != "" {
			model.Environment = types.StringValue(v)
		}
	}

	if model.Convention.IsNull() {
		if v := os.Getenv("SA_CONVENTION"); v != "" {
			model.Convention = types.StringValue(v)
		} else {
			model.Convention = types.StringValue("default")
		}
	}

	if model.Separator.IsNull() {
		if v := os.Getenv("SA_SEPARATOR"); v != "" {
			model.Separator = types.StringValue(v)
		} else {
			model.Separator = types.StringValue("-")
		}
	}

	if model.RandomSeed.IsNull() {
		if v := os.Getenv("SA_RANDOM_SEED"); v != "" {
			i, err := strconv.ParseInt(v, 10, 64)
			if err != nil {
				response.Diagnostics.AddError("random_seed", "Invalid random seed value")
				return
			}
			model.RandomSeed = types.Int64Value(i)
		} else {
			model.RandomSeed = types.Int64Value(1337)
		}
	}

	if model.HashLength.IsNull() {
		if v := os.Getenv("SA_HASH_LENGTH"); v != "" {
			i, err := strconv.Atoi(v)
			if err != nil {
				response.Diagnostics.AddError("hash_length", "Invalid hash length value")
				return
			}
			model.HashLength = types.Int32Value(int32(i))
		}
	}

	if model.Lowercase.IsNull() {
		if v := os.Getenv("SA_LOWERCASE"); v != "" {
			model.Lowercase = types.BoolValue(v == "true")
		} else {
			model.Lowercase = types.BoolValue(false)
		}
	}

	if model.SchemaReference.IsNull() {
		model.SchemaReference, _ = types.ObjectValue(
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

	sourceRef, diags := model.getSourceRef(ctx)
	response.Diagnostics = append(response.Diagnostics, diags...)
	if response.Diagnostics.HasError() {
		return
	}

	// Download the schema reference
	f, err := sourceRef.Download(ctx, hash(sourceRef))
	if err != nil {
		response.Diagnostics.AddError("source_reference", err.Error())
		return
	}

	p.config = &ProviderConfig{
		SourceRef:    f,
		ProviderData: model,
	}

	response.DataSourceData = p.config
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
	}
}
