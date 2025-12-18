// Copyright (c) HashiCorp, Inc.
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

type NameFunction struct{}

func NewNameFunction() function.Function {
	return &NameFunction{}
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
				Name:                "settings",
				MarkdownDescription: "A map of settings to apply to the name string.",
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
	model, _, buildNameSettings, name, typeSchema, err := parseArguments(ctx, req, resp)
	if err != nil || resp.Error != nil {
		// Error is already set in resp.Error by parseArguments, just return
		return
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
