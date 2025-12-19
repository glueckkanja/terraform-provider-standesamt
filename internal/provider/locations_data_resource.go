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
	"github.com/hashicorp/terraform-plugin-log/tflog"

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

// LocationDataSource defines the data source implementation.
type LocationDataSource struct {
	config *ProviderConfig
}

func (d *LocationDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_locations"
}

func (d *LocationDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Data source to build a map of the locations. The source of locations depends on the provider configuration: either from the schema library or from the Azure Resource Manager API.",
		Attributes: map[string]schema.Attribute{
			"locations": schema.MapAttribute{
				Description:         "A map of location names to their short names. You can use this map to pass to the name function and use the location in the name.",
				MarkdownDescription: "A map of location names to their short names. You can use this map to pass to the name function and use the location in the name.",
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

	d.config = data
}

func (d *LocationDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var model locationDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &model)...)

	if resp.Diagnostics.HasError() {
		return
	}

	var locationsMap s.LocationsMapSchema
	var err error

	locationSource := d.config.ProviderData.LocationSource.ValueString()
	tflog.Debug(ctx, "Reading locations", map[string]interface{}{
		"location_source": locationSource,
	})

	switch locationSource {
	case "azure":
		// Fetch locations from Azure API
		if d.config.AzureConfig == nil {
			resp.Diagnostics.AddError(
				"Azure Configuration Missing",
				"location_source is 'azure' but Azure configuration is not available. Please configure the azure block in the provider.",
			)
			return
		}

		fetcher := s.NewAzureLocationFetcher(d.config.AzureConfig)
		locationsMap, err = fetcher.Fetch(ctx)
		if err != nil {
			resp.Diagnostics.AddError(
				"Failed to fetch Azure locations",
				fmt.Sprintf("Error fetching locations from Azure API: %s", err.Error()),
			)
			return
		}

		tflog.Debug(ctx, "Fetched locations from Azure API", map[string]interface{}{
			"count": len(locationsMap),
		})

	default:
		// Use schema library (existing behavior)
		result := s.Result{}
		process := s.NewProcessorClient(d.config.SourceRef)
		if err := process.Process(&result); err != nil {
			resp.Diagnostics.AddError("source_reference", err.Error())
			return
		}
		locationsMap = result.Locations

		tflog.Debug(ctx, "Loaded locations from schema library", map[string]interface{}{
			"count": len(locationsMap),
		})
	}

	// Apply location aliases if configured
	aliases, diags := d.config.ProviderData.getLocationAliases(ctx)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if len(aliases) > 0 {
		locationsMap = s.ApplyAliases(locationsMap, aliases)
		tflog.Debug(ctx, "Applied location aliases", map[string]interface{}{
			"alias_count": len(aliases),
		})
	}

	// Convert to Terraform types
	locations := make(map[string]attr.Value)
	for k, v := range locationsMap {
		locations[k] = types.StringValue(v)
	}

	model.Locations = types.MapValueMust(types.StringType, locations)

	resp.Diagnostics.Append(resp.State.Set(ctx, &model)...)
}
