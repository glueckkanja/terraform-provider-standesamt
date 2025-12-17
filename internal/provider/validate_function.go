// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"encoding/json"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/function"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"regexp"
	"strings"
	"terraform-provider-standesamt/internal/random"
	s "terraform-provider-standesamt/internal/schema"
	"terraform-provider-standesamt/internal/tools"
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
	var (
		model             = configurationsModel{}
		name              types.String
		nameType          string
		result            buildNameResultModel
		configurations    types.Object
		settingsDynamic   types.Dynamic
		buildNameSettings s.BuildNameSettingsModel
		typeSchema        s.NamingSchema
		diagnose          diag.Diagnostics
	)

	if resp.Error = req.Arguments.Get(ctx, &configurations, &nameType, &settingsDynamic, &name); resp.Error != nil {
		return
	}

	diags := configurations.As(ctx, &model, basetypes.ObjectAsOptions{})
	resp.Error = function.ConcatFuncErrors(resp.Error, function.FuncErrorFromDiags(ctx, diags))

	for k, o := range model.Schema {
		if k == nameType {
			diagnose = o.As(ctx, &typeSchema, basetypes.ObjectAsOptions{})

			resp.Error = function.ConcatFuncErrors(resp.Error, function.FuncErrorFromDiags(ctx, diagnose))
			break
		}
	}

	if !settingsDynamic.IsNull() && !settingsDynamic.IsUnderlyingValueNull() {
		switch settingsDynamic.UnderlyingValue().(type) {
		case types.Object:
			// This may be the sickest workaround ever to get optional attributes to work
			// The String() function will return a json representation of the object
			// And we can unmarshal it into our struct leveraging the json omitempty
			err := json.Unmarshal([]byte(settingsDynamic.UnderlyingValue().String()), &buildNameSettings)
			if err != nil {
				resp.Error = function.ConcatFuncErrors(resp.Error, function.NewArgumentFuncError(2, err.Error()))
				break
			}
		default:
			resp.Error = function.ConcatFuncErrors(resp.Error, function.NewArgumentFuncError(2, "settingsDynamic is not an object"))
			return
		}
	}

	result.SetConvention(&buildNameSettings, &model)

	if result.Convention.ValueString() == "default" {
		var location string
		if buildNameSettings.Location != "" {
			location = buildNameSettings.Location
		} else if !model.Configuration.Location.IsNull() {
			location = model.Configuration.Location.ValueString()
		} else {
			location = ""
		}

		if location != "" {
			if v, ok := model.Locations[location]; ok {
				result.Location = v
			} else {
				resp.Error = function.ConcatFuncErrors(resp.Error, function.NewArgumentFuncError(0, "location not found in provided locations map"))
			}
		}

		if buildNameSettings.Environment != "" {
			result.Environment = types.StringValue(buildNameSettings.Environment)
		} else if typeSchema.Configuration.UseEnvironment.ValueBool() {
			result.Environment = model.Configuration.Environment
		} else {
			result.Environment = types.StringValue("")
		}

		if buildNameSettings.Separator != "" {
			result.Separator = types.StringValue(buildNameSettings.Separator)
		} else if typeSchema.Configuration.UseSeparator.ValueBool() {
			result.Separator = model.Configuration.Separator
		} else {
			result.Separator = types.StringValue("")
		}

		result.NamePrecedence, diagnose = types.ListValueFrom(ctx, types.StringType, s.DefaultNamePrecedence[:])
		resp.Error = function.ConcatFuncErrors(resp.Error, function.FuncErrorFromDiags(ctx, diagnose))

		var itemsNamePrecedence = typeSchema.Configuration.NamePrecedence.Elements()
		if len(itemsNamePrecedence) > 0 {
			tflog.Debug(ctx, "build_resource_name: setting NamePrecedence from schema")
			result.NamePrecedence = typeSchema.Configuration.NamePrecedence
		}

		if len(buildNameSettings.NamePrecedence) > 0 {
			result.NamePrecedence, diagnose = types.ListValueFrom(ctx, types.StringType, buildNameSettings.NamePrecedence)
			resp.Error = function.ConcatFuncErrors(resp.Error, function.FuncErrorFromDiags(ctx, diagnose))
		}

		if len(buildNameSettings.Prefixes) == 0 || buildNameSettings.Prefixes == nil {
			result.Prefixes = model.Configuration.Prefixes
		} else {
			result.Prefixes, diagnose = types.ListValueFrom(ctx, types.StringType, buildNameSettings.Prefixes)
			resp.Error = function.ConcatFuncErrors(resp.Error, function.FuncErrorFromDiags(ctx, diagnose))
		}

		if len(buildNameSettings.Suffixes) == 0 || buildNameSettings.Suffixes == nil {
			result.Suffixes = model.Configuration.Suffixes
		} else {
			result.Suffixes, diagnose = types.ListValueFrom(ctx, types.StringType, buildNameSettings.Suffixes)
			resp.Error = function.ConcatFuncErrors(resp.Error, function.FuncErrorFromDiags(ctx, diagnose))
		}

		if buildNameSettings.HashLength > 0 {
			result.HashLength = types.Int32Value(buildNameSettings.HashLength)
		} else if model.Configuration.HashLength.ValueInt32() > 0 {
			result.HashLength = model.Configuration.HashLength
		} else {
			result.HashLength = typeSchema.Configuration.HashLength
		}

		if buildNameSettings.RandomSeed > 0 {
			result.RandomSeed = types.Int64Value(buildNameSettings.RandomSeed)
		} else {
			result.RandomSeed = model.Configuration.RandomSeed
		}

		var calculatedContent []string

		for i := 0; i < len(result.NamePrecedence.Elements()); i++ {

			switch c := (result.NamePrecedence.Elements())[i].String(); strings.Trim(c, "\"") {
			case "abbreviation":
				if len(typeSchema.Abbreviation.String()) > 0 {
					calculatedContent = append(calculatedContent, tools.GetBaseString(typeSchema.Abbreviation))
				}
			case "prefixes":
				for j := 0; j < len(result.Prefixes.Elements()); j++ {
					calculatedContent = append(calculatedContent,
						strings.Trim(result.Prefixes.Elements()[j].String(), "\""))
				}
			case "suffixes":
				for j := 0; j < len(result.Suffixes.Elements()); j++ {
					calculatedContent = append(calculatedContent,
						strings.Trim(result.Suffixes.Elements()[j].String(), "\""))
				}
			case "name":
				if len(name.String()) > 0 {
					calculatedContent = append(calculatedContent, tools.GetBaseString(name))
				}
			case "environment":
				if len(result.Environment.ValueString()) > 0 {
					calculatedContent = append(calculatedContent, tools.GetBaseString(result.Environment))
				}
			case "location":
				if len(result.Location.ValueString()) > 0 {
					calculatedContent = append(calculatedContent, tools.GetBaseString(result.Location))
				}
			case "hash":
				if !result.HashLength.IsNull() {
					var hashLength = result.HashLength.ValueInt32()
					if hashLength > 0 {
						randomHash := random.Hash(int(hashLength), result.RandomSeed.ValueInt64())
						calculatedContent = append(calculatedContent, randomHash)
					}
				}
			}
		}
		result.Name = types.StringValue(strings.Join(calculatedContent, result.Separator.ValueString()))
	} else { // end if result.Configuration.Convention.String() == "default"
		tflog.Debug(ctx, "configuring with passthrough convention")
		result.Name = name
	}

	// Check if any of the use_lower_case settings are set to true
	// and convert the full name to lower case before validation
	if typeSchema.Configuration.UseLowerCase.ValueBool() || model.Configuration.Lowercase.ValueBool() || buildNameSettings.Lowercase {
		result.Name = toLower(result.Name)
	}

	resultNameStr := tools.GetBaseString(result.Name)

	// Perform validation and collect results
	regexValid := true
	lengthValid := true
	doubleHyphensFound := strings.Contains(resultNameStr, "--")

	re := regexp.MustCompile(tools.GetBaseString(typeSchema.ValidationRegex))
	if !re.MatchString(resultNameStr) {
		regexValid = false
	}

	nameLength := int64(len(resultNameStr))
	if nameLength > typeSchema.MaxLength.ValueInt64() || nameLength < typeSchema.MinLength.ValueInt64() {
		lengthValid = false
	}

	// Build the validation result map
	regexObj, diags := types.ObjectValue(
		map[string]attr.Type{
			"valid": types.BoolType,
			"match": types.StringType,
		},
		map[string]attr.Value{
			"valid": types.BoolValue(regexValid),
			"match": types.StringValue(tools.GetBaseString(typeSchema.ValidationRegex)),
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
			"valid": types.BoolValue(lengthValid),
			"is":    types.Int64Value(nameLength),
			"max":   types.Int64Value(typeSchema.MaxLength.ValueInt64()),
			"min":   types.Int64Value(typeSchema.MinLength.ValueInt64()),
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
			"name":                  types.StringValue(resultNameStr),
			"double_hyphens_denied": types.BoolValue(typeSchema.Configuration.DenyDoubleHyphens.ValueBool()),
			"double_hyphens_found":  types.BoolValue(doubleHyphensFound),
		},
	)
	if diags.HasError() {
		resp.Error = function.ConcatFuncErrors(resp.Error, function.FuncErrorFromDiags(ctx, diags))
		return
	}

	resp.Error = function.ConcatFuncErrors(resp.Error, resp.Result.Set(ctx, validationResult))
}
