// Copyright (c) glueckkanja AG
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	s "terraform-provider-standesamt/internal/schema"
	"terraform-provider-standesamt/internal/tools"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/function"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ function.Function = &ValidateFunction{}

type ValidateFunction struct{}

func NewValidateFunction() function.Function {
	return &ValidateFunction{}
}

func (f *ValidateFunction) Metadata(_ context.Context, _ function.MetadataRequest, resp *function.MetadataResponse) {
	resp.Name = "validate"
}

func (f *ValidateFunction) Definition(_ context.Context, _ function.DefinitionRequest, resp *function.DefinitionResponse) {
	resp.Definition = function.Definition{
		Summary:             "Validate a resource name and return detailed validation results",
		Description:         "Build a resource name based on the provided configuration and name type, then return detailed validation results as a map.",
		MarkdownDescription: "Build a resource name based on the provided configuration and name type, then return detailed validation results as a map containing regex validation, length validation, and resource type information.",
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
		Return: function.ObjectReturn{
			AttributeTypes: map[string]attr.Type{
				"regex": types.ObjectType{
					AttrTypes: map[string]attr.Type{
						"valid": types.BoolType,
						"match": types.StringType,
					},
				},
				"length": types.ObjectType{
					AttrTypes: map[string]attr.Type{
						"valid": types.BoolType,
						"is":    types.Int64Type,
						"max":   types.Int64Type,
						"min":   types.Int64Type,
					},
				},
				"type":                  types.StringType,
				"name":                  types.StringType,
				"double_hyphens_denied": types.BoolType,
				"double_hyphens_found":  types.BoolType,
			},
		},
	}
}

func (f *ValidateFunction) Run(ctx context.Context, req function.RunRequest, resp *function.RunResponse) {
	// Parse and validate input arguments
	model, nameType, buildNameSettings, name, typeSchema, err := parseArguments(ctx, req, resp)
	if err != nil || resp.Error != nil {
		return
	}

	// Build the resource name using the nameBuilder
	builder := newNameBuilder(ctx, model, typeSchema, buildNameSettings)
	resultName := builder.buildName(name, resp)
	if resp.Error != nil {
		return
	}

	resultNameStr := tools.GetBaseString(resultName)

	// Perform validation and collect results
	validation := validateName(resultNameStr, typeSchema)

	// Build the validation result map
	regexObj, diags := types.ObjectValue(
		map[string]attr.Type{
			"valid": types.BoolType,
			"match": types.StringType,
		},
		map[string]attr.Value{
			"valid": types.BoolValue(validation.RegexValid),
			"match": types.StringValue(validation.ValidationRegex),
		},
	)
	if diags.HasError() {
		resp.Error = function.ConcatFuncErrors(resp.Error, function.FuncErrorFromDiags(ctx, diags))
		return
	}

	lengthObj, diags := types.ObjectValue(
		map[string]attr.Type{
			"valid": types.BoolType,
			"is":    types.Int64Type,
			"max":   types.Int64Type,
			"min":   types.Int64Type,
		},
		map[string]attr.Value{
			"valid": types.BoolValue(validation.LengthValid),
			"is":    types.Int64Value(validation.NameLength),
			"max":   types.Int64Value(validation.MaxLength),
			"min":   types.Int64Value(validation.MinLength),
		},
	)
	if diags.HasError() {
		resp.Error = function.ConcatFuncErrors(resp.Error, function.FuncErrorFromDiags(ctx, diags))
		return
	}

	validationResult, diags := types.ObjectValue(
		map[string]attr.Type{
			"regex": types.ObjectType{
				AttrTypes: map[string]attr.Type{
					"valid": types.BoolType,
					"match": types.StringType,
				},
			},
			"length": types.ObjectType{
				AttrTypes: map[string]attr.Type{
					"valid": types.BoolType,
					"is":    types.Int64Type,
					"max":   types.Int64Type,
					"min":   types.Int64Type,
				},
			},
			"type":                  types.StringType,
			"name":                  types.StringType,
			"double_hyphens_denied": types.BoolType,
			"double_hyphens_found":  types.BoolType,
		},
		map[string]attr.Value{
			"regex":                 regexObj,
			"length":                lengthObj,
			"type":                  types.StringValue(nameType),
			"name":                  types.StringValue(validation.Name),
			"double_hyphens_denied": types.BoolValue(validation.DenyDoubleHyphens),
			"double_hyphens_found":  types.BoolValue(validation.DoubleHyphensFound),
		},
	)
	if diags.HasError() {
		resp.Error = function.ConcatFuncErrors(resp.Error, function.FuncErrorFromDiags(ctx, diags))
		return
	}

	resp.Error = function.ConcatFuncErrors(resp.Error, resp.Result.Set(ctx, validationResult))
}
