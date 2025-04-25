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
		"location":    types.StringType,
	}
}

func (d *SchemaDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		// This description is used by the documentation generator and the language server.
		MarkdownDescription: "Schema data source",

		Attributes: map[string]schema.Attribute{
			"convention": schema.StringAttribute{
				Optional:            true,
				Sensitive:           false,
				Description:         "Default convention for all naming results. Possible values 'default', 'passthrough'. Default 'default'",
				MarkdownDescription: "Default convention for all naming results. Possible values 'default', 'passthrough'. Default 'default'",
				Validators: []validator.String{
					stringvalidator.OneOf("default", "passthrough"),
				},
			},
			"environment": schema.StringAttribute{
				Optional:            true,
				Description:         "Environment parameter.",
				MarkdownDescription: "Environment parameter.",
			},
			"separator": schema.StringAttribute{
				Optional:            true,
				Description:         "Naming schema separator. Default '-'",
				MarkdownDescription: "Naming schema separator. Default '-'",
			},
			"random_seed": schema.Int64Attribute{
				Optional:            true,
				Description:         "Random seed for naming schema.",
				MarkdownDescription: "Random seed for naming schema.",
			},
			"hash_length": schema.Int32Attribute{
				Optional:            true,
				Description:         "Default hash length for resource schema. Overrrides all naminng schema configurations defined in json files.",
				MarkdownDescription: "Default hash length for resource schema. Overrrides all naminng schema configurations defined in json files.",
			},
			"lowercase": schema.BoolAttribute{
				Optional:            true,
				Description:         "Namig result formating. Default 'false'",
				MarkdownDescription: "Namig result formating. Default 'false'",
			},
			"prefixes": schema.ListAttribute{
				Optional:            true,
				MarkdownDescription: "Prefixes for naming schema. Default '[]'",
				ElementType:         types.StringType,
			},
			"suffixes": schema.ListAttribute{
				Optional:            true,
				MarkdownDescription: "Suffixes for naming schema. Default '[]'",
				ElementType:         types.StringType,
			},
			"location": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "Location parameter.",
			},
			"schema": schema.MapAttribute{
				MarkdownDescription: "Schema",
				Computed:            true,
				ElementType: types.ObjectType{
					AttrTypes: s.SchemaTypeAttributes(),
				},
			},
			"configuration": schema.ObjectAttribute{
				MarkdownDescription: "Configuration",
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
