// Copyright (c) glueckkanja AG
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"io/fs"
	s "terraform-provider-standesamt/internal/schema"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ datasource.DataSource = &LocationDataSource{}

type locationDataSourceModel struct {
	Locations types.Map `tfsdk:"locations"`
}

func NewLocationDataSource() datasource.DataSource {
	return &LocationDataSource{}
}

// SchemaDataSource defines the data source implementation.
type LocationDataSource struct {
	sourceRef fs.FS
}

func (d *LocationDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_locations"
}

func (d *LocationDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "location data source",
		Attributes: map[string]schema.Attribute{
			"locations": schema.MapAttribute{
				MarkdownDescription: "The location map.",
				Computed:            true,
				ElementType:         types.StringType,
			},
		},
	}
}

func (d *LocationDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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
}

func (d *LocationDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var model locationDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &model)...)

	if resp.Diagnostics.HasError() {
		return
	}

	result := s.Result{}
	process := s.NewProcessorClient(d.sourceRef)
	if err := process.Process(&result); err != nil {
		resp.Diagnostics.AddError("source_reference", err.Error())
		return
	}

	locations := make(map[string]attr.Value)

	for k, v := range result.Locations {
		locations[k] = types.StringValue(v)
	}

	model.Locations = types.MapValueMust(types.StringType, locations)

	resp.Diagnostics.Append(resp.State.Set(ctx, &model)...)
}
