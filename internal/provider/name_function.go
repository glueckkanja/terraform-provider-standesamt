// Copyright glueckkanja AG 2025, 2026
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
	"strings"
	s "terraform-provider-standesamt/internal/schema"
	"terraform-provider-standesamt/internal/tools"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/function"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

//var namingReturnAttrTypes = map[string]attr.Type{
//	"Result": types.StringType,
//}

type configurationsModel struct {
	Configuration configurationModel      `tfsdk:"configuration"`
	Locations     map[string]types.String `tfsdk:"locations"`
	Schema        map[string]types.Object `tfsdk:"schema"`
}

type buildNameResultModel struct {
	Name           types.String
	Convention     types.String
	Environment    types.String
	Separator      types.String
	HashLength     types.Int32
	RandomSeed     types.Int64
	Prefixes       types.List
	Suffixes       types.List
	NamePrecedence types.List
	Location       types.String
	Lowercase      types.Bool
}

func (r *buildNameResultModel) GetName() types.String {
	return r.Name
}

func (r *buildNameResultModel) SetConvention(override *s.BuildNameSettingsModel, model *configurationsModel) {
	if override.Convention != "" {
		r.Convention = types.StringValue(override.Convention)
	} else {
		r.Convention = model.Configuration.Convention
	}
}

var _ function.Function = &NameFunction{}

type NameFunction struct {
	provider *StandesamtProvider
}

func (f *NameFunction) Metadata(_ context.Context, _ function.MetadataRequest, resp *function.MetadataResponse) {
	resp.Name = "name"
}

func (f *NameFunction) Definition(_ context.Context, _ function.DefinitionRequest, resp *function.DefinitionResponse) {
	resp.Definition = function.Definition{
		Summary:             "Provide a valid resource name",
		Description:         "Build a resource name based on the provided configuration and name type.",
		MarkdownDescription: "Build a resource name based on the provided configuration and name type.",
		Parameters: []function.Parameter{
			function.ObjectParameter{
				Name:                "configurations",
				MarkdownDescription: "A configuration object that contains the variables and formats to use for the name.",
				AttributeTypes: map[string]attr.Type{
					"configuration": types.ObjectType{
						AttrTypes: configurationTypeAttributes(),
					},
					"locations": types.MapType{
						ElemType: types.StringType,
					},
					"schema": types.MapType{
						ElemType: types.ObjectType{
							AttrTypes: s.SchemaTypeAttributes(),
						},
					},
				},
				Description: "Configuration for the naming object",
			},
			function.StringParameter{
				Name:        "name_type",
				Description: "The resource type to use for the name.",
			},
			function.DynamicParameter{
				Name: "settings",
				MarkdownDescription: "An optional map of per-call overrides. All keys are optional and take " +
					"precedence over the provider-level configuration.\n\n" +
					"Supported keys:\n\n" +
					"| Key | Type | Description |\n" +
					"|---|---|---|\n" +
					"| `convention` | `string` | Naming convention (`default` or `passthrough`). |\n" +
					"| `environment` | `string` | Environment abbreviation (e.g. `prd`, `tst`). |\n" +
					"| `location` | `string` | Azure location key resolved via the `locations` map. |\n" +
					"| `separator` | `string` | Separator between name parts — overrides the schema default on a per-call basis. |\n" +
					"| `prefixes` | `list(string)` | Prefix segments to prepend. |\n" +
					"| `suffixes` | `list(string)` | Suffix segments to append. |\n" +
					"| `name_precedence` | `list(string)` | Order of name segments. |\n" +
					"| `hash_length` | `number` | Length of the random hash segment (0 = disabled). |\n" +
					"| `random_seed` | `number` | Seed for the hash generator (for reproducible names). |\n" +
					"| `lowercase` | `bool` | Convert the final name to lowercase. |\n\n" +
					"Pass `{}` or `null` to use provider defaults for all settings.",
			},
			function.StringParameter{
				Name:        "name",
				Description: "Name to parse",
			},
		},
		Return: function.StringReturn{},
	}
}

func (f *NameFunction) Run(ctx context.Context, req function.RunRequest, resp *function.RunResponse) {
	// Parse and validate input arguments
	model, nameType, buildNameSettings, name, typeSchema, err := parseArguments(ctx, req, resp)
	if err != nil || resp.Error != nil {
		// Error is already set in resp.Error by parseArguments, just return
		return
	}

	// Inject the schema-level separator from the JSON library into typeSchema.
	// The field has no tfsdk tag so it is not populated during HCL unmarshal.
	if f.provider != nil && f.provider.config != nil {
		if jsonSchema, ok := f.provider.config.NamingSchemas[nameType]; ok {
			typeSchema.Configuration.Separator = jsonSchema.Configuration.Separator
		}
	}

	// Build the resource name using the nameBuilder
	builder := newNameBuilder(ctx, model, typeSchema, buildNameSettings)
	resultName := builder.buildName(name, resp)
	if resp.Error != nil {
		return
	}

	resultNameStr := tools.GetBaseString(resultName)

	// Validate the final name against the naming schema constraints
	validation := validateName(resultNameStr, typeSchema)

	if validation.DenyDoubleHyphens && validation.DoubleHyphensFound {
		resp.Error = function.ConcatFuncErrors(resp.Error, function.NewFuncError(fmt.Sprintf("Invalid name: '%s' contains double hyphens", resultNameStr)))
	}

	if !validation.RegexValid {
		resp.Error = function.ConcatFuncErrors(resp.Error, function.NewFuncError("Name does not match regex"))
	} else if !validation.LengthValid {
		if validation.NameLength > validation.MaxLength {
			resp.Error = function.ConcatFuncErrors(resp.Error, function.NewFuncError(fmt.Sprintf("Name has %d characters, but maximum is set to %d", validation.NameLength, validation.MaxLength)))
		} else if validation.NameLength < validation.MinLength {
			resp.Error = function.ConcatFuncErrors(resp.Error, function.NewFuncError(fmt.Sprintf("Name has %d characters, but minimum is set to %d", validation.NameLength, validation.MinLength)))
		}
	}

	// Set the result
	resp.Error = function.ConcatFuncErrors(resp.Error, resp.Result.Set(ctx, &resultName))
}

func toLower(s types.String) types.String {
	if s.IsNull() || s.IsUnknown() {
		return s
	}

	return types.StringValue(strings.ToLower(s.ValueString()))
}
