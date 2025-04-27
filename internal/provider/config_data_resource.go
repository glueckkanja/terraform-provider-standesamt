// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"io/fs"
	s "terraform-provider-standesamt/internal/schema"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ datasource.DataSource = &SchemaDataSource{}

func NewSchemaDataSource() datasource.DataSource {
	return &SchemaDataSource{}
}

// SchemaDataSource defines the data source implementation.
type SchemaDataSource struct {
	sourceRef        fs.FS
	providerSettings providerData
}

type configurationModel struct {
	Convention  types.String `tfsdk:"convention"`
	Environment types.String `tfsdk:"environment"`
	Separator   types.String `tfsdk:"separator"`
	RandomSeed  types.Int64  `tfsdk:"random_seed"`
	HashLength  types.Int32  `tfsdk:"hash_length"`
	Lowercase   types.Bool   `tfsdk:"lowercase"`
	Prefixes    types.List   `tfsdk:"prefixes"`
	Suffixes    types.List   `tfsdk:"suffixes"`
	Location    types.String `tfsdk:"location"`
}

// SchemaDataSourceModel describes the data source data model.
type schemaDataSourceModel struct {
	Convention    types.String `tfsdk:"convention"`
	Environment   types.String `tfsdk:"environment"`
	Separator     types.String `tfsdk:"separator"`
	RandomSeed    types.Int64  `tfsdk:"random_seed"`
	HashLength    types.Int32  `tfsdk:"hash_length"`
	Lowercase     types.Bool   `tfsdk:"lowercase"`
	Prefixes      types.List   `tfsdk:"prefixes"`
	Suffixes      types.List   `tfsdk:"suffixes"`
	Schema        types.Map    `tfsdk:"schema"`
	Configuration types.Object `tfsdk:"configuration"`
	Location      types.String `tfsdk:"location"`
}

func (d *SchemaDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_config"
}

func configurationTypeAttributes() map[string]attr.Type {
	return map[string]attr.Type{
		"convention":  types.StringType,
		"environment": types.StringType,
		"separator":   types.StringType,
		"random_seed": types.Int64Type,
		"hash_length": types.Int32Type,
		"lowercase":   types.BoolType,
		"prefixes":    types.ListType{ElemType: types.StringType},
		"suffixes":    types.ListType{ElemType: types.StringType},
		"location":    types.StringType, //TODO
	}
}

func (d *SchemaDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		// This description is used by the documentation generator and the language server.
		Description:         "Data source to generate the naming schema and configuration for the Standesamt provider.",
		MarkdownDescription: "Data source to generate the naming schema and configuration for the Standesamt provider.",
		Attributes: map[string]schema.Attribute{
			"convention": schema.StringAttribute{
				Optional:            true,
				Sensitive:           false,
				Description:         "Define the convention for naming results. Possible values are 'default' and 'passthrough'. Will override the convention defined in the provider settings.",
				MarkdownDescription: "Define the convention for naming results. Possible values are 'default' and 'passthrough'. Will override the convention defined in the provider settings.",
				Validators: []validator.String{
					stringvalidator.OneOf("default", "passthrough"),
				},
			},
			"environment": schema.StringAttribute{
				Optional:            true,
				Description:         "Define the environment for the naming schema. Normally this is the name of the environment, e.g. 'prod', 'dev', 'test'. Will override the environment defined in the provider settings.",
				MarkdownDescription: "Define the environment for the naming schema. Normally this is the name of the environment, e.g. 'prod', 'dev', 'test'. Will override the environment defined in the provider settings.",
			},
			"separator": schema.StringAttribute{
				Optional:            true,
				Description:         "The separator to use for generating the resulting name. Will override the separator defined in the provider settings.",
				MarkdownDescription: "The separator to use for generating the resulting name. Will override the separator defined in the provider settings.",
			},
			"random_seed": schema.Int64Attribute{
				Optional:            true,
				Description:         "A random seed used by the random number generator. This is used to generate a random name for the naming schema. The default value is 1337. Make sure to update this value to avoid collisions for globally unique names. Will override the random seed defined in the provider settings.",
				MarkdownDescription: "A random seed used by the random number generator. This is used to generate a random name for the naming schema. The default value is 1337. Make sure to update this value to avoid collisions for globally unique names. Will override the random seed defined in the provider settings.",
			},
			"hash_length": schema.Int32Attribute{
				Optional:            true,
				Description:         "Default hash length. Overrides all schema configurations. Overrides the default hash length defined in the provider settings.",
				MarkdownDescription: "Default hash length. Overrides all schema configurations. Overrides the default hash length defined in the provider settings.",
			},
			"lowercase": schema.BoolAttribute{
				Optional:            true,
				Description:         "Control if the resulting name should be lower case. Overrides all schema configurations. Overrides the default lowercase setting defined in the provider settings.",
				MarkdownDescription: "Control if the resulting name should be lower case. Overrides all schema configurations. Overrides the default lowercase setting defined in the provider settings.",
			},
			"prefixes": schema.ListAttribute{
				Optional:            true,
				Description:         "A list of strings used as prefixes for the resulting name. Each prefix will be used in order and separated by the separator. Default '[]'",
				MarkdownDescription: "A list of strings used as prefixes for the resulting name. Each prefix will be used in order and separated by the separator. Default '[]'",
				ElementType:         types.StringType,
			},
			"suffixes": schema.ListAttribute{
				Optional:            true,
				Description:         "A list of strings used as suffixes for the resulting name. Each suffix will be used in order and separated by the separator. Default '[]'",
				MarkdownDescription: "A list of strings used as suffixes for the resulting name. Each suffix will be used in order and separated by the separator. Default '[]'",
				ElementType:         types.StringType,
			},
			"location": schema.StringAttribute{
				Optional:            true,
				Description:         "A location string used to lookup in the locations schema. In the default schema library this is a list of locations for Azure. If you set the location to 'westeurope' the resulting name will be 'we'.",
				MarkdownDescription: "A location string used to lookup in the locations schema. In the default schema library this is a list of locations for Azure. If you set the location to 'westeurope' the resulting name will be 'we'.",
			},
			"schema": schema.MapAttribute{
				Description:         "A map of naming schema objects that is generated from the schema library file schema.naming.json. This attribute is used to get passed to the naming function.",
				MarkdownDescription: "A map of naming schema objects that is generated from the schema library file schema.naming.json. This attribute is used to get passed to the naming function.",
				Computed:            true,
				ElementType: types.ObjectType{
					AttrTypes: s.SchemaTypeAttributes(),
				},
			},
			"configuration": schema.ObjectAttribute{
				Description:         "Configuration object that contains the resulting configuration for the naming schema. This is used to pass the configuration to the naming function.",
				MarkdownDescription: "Configuration object that contains the resulting configuration for the naming schema. This is used to pass the configuration to the naming function.",
				Computed:            true,
				AttributeTypes:      configurationTypeAttributes(),
			},
		},
	}
}

func (d *SchemaDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil {
		return
	}

	data, ok := req.ProviderData.(*ProviderConfig)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected *ProviderConfig, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	d.sourceRef = data.SourceRef
	d.providerSettings = data.ProviderData
}

func (d *SchemaDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data schemaDataSourceModel

	var configuration configurationModel

	// Read configuration data into the model
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	result := s.Result{}
	process := s.NewProcessorClient(d.sourceRef)
	if err := process.Process(&result); err != nil {
		resp.Diagnostics.AddError("source_reference", err.Error())
		return
	}

	configuration.Convention = data.Convention
	if configuration.Convention.IsNull() {
		if d.providerSettings.Convention.IsNull() {
			configuration.Convention = types.StringValue("default")
		} else {
			configuration.Convention = d.providerSettings.Convention
		}
	}

	configuration.Separator = data.Separator
	if configuration.Separator.IsNull() {
		configuration.Separator = d.providerSettings.Separator
	}

	configuration.Prefixes = data.Prefixes
	if configuration.Prefixes.IsNull() || len(configuration.Prefixes.Elements()) == 0 {
		configuration.Prefixes = types.ListValueMust(types.StringType, []attr.Value{})
	}

	configuration.Suffixes = data.Suffixes
	if configuration.Suffixes.IsNull() || len(configuration.Suffixes.Elements()) == 0 {
		configuration.Suffixes = types.ListValueMust(types.StringType, []attr.Value{})
	}

	configuration.RandomSeed = data.RandomSeed
	if configuration.RandomSeed.IsNull() {
		configuration.RandomSeed = d.providerSettings.RandomSeed
	}

	configuration.HashLength = data.HashLength
	if configuration.HashLength.IsNull() {
		configuration.HashLength = d.providerSettings.HashLength
	}

	configuration.Lowercase = data.Lowercase
	if configuration.Lowercase.IsNull() {
		configuration.Lowercase = d.providerSettings.Lowercase
	}

	configuration.Environment = data.Environment
	if configuration.Environment.IsNull() {
		configuration.Environment = d.providerSettings.Environment
	}

	configuration.Location = data.Location

	resultingNamingSchemaMap, _ := types.MapValueFrom(ctx, types.ObjectType{AttrTypes: s.SchemaTypeAttributes()}, s.NewNamingSchemaMap(result.NamingSchemas))

	data.Schema = resultingNamingSchemaMap
	var configObj, diagnostic = types.ObjectValueFrom(ctx, configurationTypeAttributes(), configuration)
	if diagnostic.HasError() {
		resp.Diagnostics.Append(diagnostic.Errors()...)
		return
	}
	data.Configuration = configObj

	// Save data into state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
