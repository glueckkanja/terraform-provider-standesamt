// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"terraform-provider-standesamt/internal/random"
	s "terraform-provider-standesamt/internal/schema"
	"terraform-provider-standesamt/internal/tools"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/function"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// nameBuilder encapsulates the logic for building resource names
type nameBuilder struct {
	ctx               context.Context
	model             *configurationsModel
	typeSchema        *s.NamingSchema
	buildNameSettings *s.BuildNameSettingsModel
	result            *buildNameResultModel
}

// extractStringSlice extracts a string slice from a types.List or types.Tuple
func extractStringSlice(value attr.Value) []string {
	var result []string

	switch v := value.(type) {
	case types.List:
		if v.IsNull() || v.IsUnknown() {
			return nil
		}
		for _, elem := range v.Elements() {
			if str, ok := elem.(types.String); ok && !str.IsNull() && !str.IsUnknown() {
				result = append(result, str.ValueString())
			}
		}
	case types.Tuple:
		if v.IsNull() || v.IsUnknown() {
			return nil
		}
		for _, elem := range v.Elements() {
			if str, ok := elem.(types.String); ok && !str.IsNull() && !str.IsUnknown() {
				result = append(result, str.ValueString())
			}
		}
	}

	return result
}

// parseSettingsFromDynamic extracts settings from a dynamic parameter without JSON
func parseSettingsFromDynamic(settingsDynamic types.Dynamic) (*s.BuildNameSettingsModel, error) {
	settings := &s.BuildNameSettingsModel{}

	if settingsDynamic.IsNull() || settingsDynamic.IsUnderlyingValueNull() {
		return settings, nil
	}

	obj, ok := settingsDynamic.UnderlyingValue().(types.Object)
	if !ok {
		return nil, fmt.Errorf("settings must be an object")
	}

	attrs := obj.Attributes()

	// Extract each attribute with null/unknown checks
	if v, ok := attrs["convention"].(types.String); ok && !v.IsNull() && !v.IsUnknown() {
		settings.Convention = v.ValueString()
	}

	if v, ok := attrs["location"].(types.String); ok && !v.IsNull() && !v.IsUnknown() {
		settings.Location = v.ValueString()
	}

	if v, ok := attrs["environment"].(types.String); ok && !v.IsNull() && !v.IsUnknown() {
		settings.Environment = v.ValueString()
	}

	if v, ok := attrs["separator"].(types.String); ok && !v.IsNull() && !v.IsUnknown() {
		settings.Separator = v.ValueString()
	}

	// Handle hash_length - can be types.Int32, types.Int64, or types.Number
	if v, ok := attrs["hash_length"].(types.Int32); ok && !v.IsNull() && !v.IsUnknown() {
		settings.HashLength = v.ValueInt32()
	} else if v, ok := attrs["hash_length"].(types.Int64); ok && !v.IsNull() && !v.IsUnknown() {
		settings.HashLength = int32(v.ValueInt64())
	} else if v, ok := attrs["hash_length"].(types.Number); ok && !v.IsNull() && !v.IsUnknown() {
		val, _ := v.ValueBigFloat().Int64()
		settings.HashLength = int32(val)
	}

	// Handle random_seed - can be types.Int64 or types.Number
	if v, ok := attrs["random_seed"].(types.Int64); ok && !v.IsNull() && !v.IsUnknown() {
		settings.RandomSeed = v.ValueInt64()
	} else if v, ok := attrs["random_seed"].(types.Number); ok && !v.IsNull() && !v.IsUnknown() {
		val, _ := v.ValueBigFloat().Int64()
		settings.RandomSeed = val
	}

	if v, ok := attrs["lowercase"].(types.Bool); ok && !v.IsNull() && !v.IsUnknown() {
		settings.Lowercase = v.ValueBool()
	}

	// Handle list/tuple attributes - HCL uses tuples for literal lists
	if v, ok := attrs["prefixes"]; ok {
		settings.Prefixes = extractStringSlice(v)
	}

	if v, ok := attrs["suffixes"]; ok {
		settings.Suffixes = extractStringSlice(v)
	}

	if v, ok := attrs["name_precedence"]; ok {
		settings.NamePrecedence = extractStringSlice(v)
	}

	return settings, nil
}

// parseArguments extracts and validates the function arguments
func parseArguments(
	ctx context.Context,
	req function.RunRequest,
	resp *function.RunResponse,
) (*configurationsModel, string, *s.BuildNameSettingsModel, types.String, *s.NamingSchema, error) {
	var (
		model             = configurationsModel{}
		name              types.String
		nameType          string
		configurations    types.Object
		settingsDynamic   types.Dynamic
		buildNameSettings s.BuildNameSettingsModel
		typeSchema        s.NamingSchema
	)

	if resp.Error = req.Arguments.Get(ctx, &configurations, &nameType, &settingsDynamic, &name); resp.Error != nil {
		return nil, "", nil, types.String{}, nil, resp.Error
	}

	diags := configurations.As(ctx, &model, basetypes.ObjectAsOptions{})
	resp.Error = function.ConcatFuncErrors(resp.Error, function.FuncErrorFromDiags(ctx, diags))
	if resp.Error != nil {
		return nil, "", nil, types.String{}, nil, resp.Error
	}

	// Find the schema for the requested name type
	schemaFound := false
	for k, o := range model.Schema {
		if k == nameType {
			diagnose := o.As(ctx, &typeSchema, basetypes.ObjectAsOptions{})
			resp.Error = function.ConcatFuncErrors(resp.Error, function.FuncErrorFromDiags(ctx, diagnose))
			if resp.Error != nil {
				return nil, "", nil, types.String{}, nil, resp.Error
			}
			schemaFound = true
			break
		}
	}

	if !schemaFound {
		// Collect available resource types for helpful error message
		availableTypes := make([]string, 0, len(model.Schema))
		for k := range model.Schema {
			availableTypes = append(availableTypes, k)
		}
		
		errorMsg := fmt.Sprintf("resource type '%s' not found in schema. Available resource types: %s", nameType, strings.Join(availableTypes, ", "))
		resp.Error = function.NewArgumentFuncError(1, errorMsg)
		return nil, "", nil, types.String{}, nil, resp.Error
	}

	// Parse optional settings from dynamic parameter
	if !settingsDynamic.IsNull() && !settingsDynamic.IsUnderlyingValueNull() {
		parsedSettings, err := parseSettingsFromDynamic(settingsDynamic)
		if err != nil {
			resp.Error = function.ConcatFuncErrors(resp.Error, function.NewArgumentFuncError(2, err.Error()))
			return nil, "", nil, types.String{}, nil, resp.Error
		}
		buildNameSettings = *parsedSettings
	}

	return &model, nameType, &buildNameSettings, name, &typeSchema, nil
}

// newNameBuilder creates a new nameBuilder instance
func newNameBuilder(
	ctx context.Context,
	model *configurationsModel,
	typeSchema *s.NamingSchema,
	buildNameSettings *s.BuildNameSettingsModel,
) *nameBuilder {
	return &nameBuilder{
		ctx:               ctx,
		model:             model,
		typeSchema:        typeSchema,
		buildNameSettings: buildNameSettings,
		result:            &buildNameResultModel{},
	}
}

// setConvention configures the naming convention
func (nb *nameBuilder) setConvention() {
	nb.result.SetConvention(nb.buildNameSettings, nb.model)
}

// resolveLocation determines the location to use
func (nb *nameBuilder) resolveLocation(resp *function.RunResponse) {
	var location string
	if nb.buildNameSettings.Location != "" {
		location = nb.buildNameSettings.Location
	} else if !nb.model.Configuration.Location.IsNull() {
		location = nb.model.Configuration.Location.ValueString()
	} else {
		location = ""
	}

	if location != "" {
		if v, ok := nb.model.Locations[location]; ok {
			nb.result.Location = v
		} else {
			resp.Error = function.ConcatFuncErrors(resp.Error, function.NewArgumentFuncError(0, "location not found in provided locations map"))
		}
	}
}

// resolveEnvironment determines the environment to use
func (nb *nameBuilder) resolveEnvironment() {
	if nb.buildNameSettings.Environment != "" {
		nb.result.Environment = types.StringValue(nb.buildNameSettings.Environment)
	} else if nb.typeSchema.Configuration.UseEnvironment.ValueBool() {
		nb.result.Environment = nb.model.Configuration.Environment
	} else {
		nb.result.Environment = types.StringValue("")
	}
}

// resolveSeparator determines the separator to use
func (nb *nameBuilder) resolveSeparator() {
	if nb.buildNameSettings.Separator != "" {
		nb.result.Separator = types.StringValue(nb.buildNameSettings.Separator)
	} else if nb.typeSchema.Configuration.UseSeparator.ValueBool() {
		nb.result.Separator = nb.model.Configuration.Separator
	} else {
		nb.result.Separator = types.StringValue("")
	}
}

// resolveNamePrecedence determines the name precedence order
func (nb *nameBuilder) resolveNamePrecedence(resp *function.RunResponse) {
	var diagnose diag.Diagnostics
	nb.result.NamePrecedence, diagnose = types.ListValueFrom(nb.ctx, types.StringType, s.DefaultNamePrecedence[:])
	resp.Error = function.ConcatFuncErrors(resp.Error, function.FuncErrorFromDiags(nb.ctx, diagnose))

	var itemsNamePrecedence = nb.typeSchema.Configuration.NamePrecedence.Elements()
	if len(itemsNamePrecedence) > 0 {
		tflog.Debug(nb.ctx, "build_resource_name: setting NamePrecedence from schema")
		nb.result.NamePrecedence = nb.typeSchema.Configuration.NamePrecedence
	}

	if len(nb.buildNameSettings.NamePrecedence) > 0 {
		nb.result.NamePrecedence, diagnose = types.ListValueFrom(nb.ctx, types.StringType, nb.buildNameSettings.NamePrecedence)
		resp.Error = function.ConcatFuncErrors(resp.Error, function.FuncErrorFromDiags(nb.ctx, diagnose))
	}
}

// resolvePrefixes determines the prefixes to use
func (nb *nameBuilder) resolvePrefixes(resp *function.RunResponse) {
	if len(nb.buildNameSettings.Prefixes) == 0 || nb.buildNameSettings.Prefixes == nil {
		nb.result.Prefixes = nb.model.Configuration.Prefixes
	} else {
		var diagnose diag.Diagnostics
		nb.result.Prefixes, diagnose = types.ListValueFrom(nb.ctx, types.StringType, nb.buildNameSettings.Prefixes)
		resp.Error = function.ConcatFuncErrors(resp.Error, function.FuncErrorFromDiags(nb.ctx, diagnose))
	}
}

// resolveSuffixes determines the suffixes to use
func (nb *nameBuilder) resolveSuffixes(resp *function.RunResponse) {
	if len(nb.buildNameSettings.Suffixes) == 0 || nb.buildNameSettings.Suffixes == nil {
		nb.result.Suffixes = nb.model.Configuration.Suffixes
	} else {
		var diagnose diag.Diagnostics
		nb.result.Suffixes, diagnose = types.ListValueFrom(nb.ctx, types.StringType, nb.buildNameSettings.Suffixes)
		resp.Error = function.ConcatFuncErrors(resp.Error, function.FuncErrorFromDiags(nb.ctx, diagnose))
	}
}

// resolveHashLength determines the hash length to use
func (nb *nameBuilder) resolveHashLength() {
	if nb.buildNameSettings.HashLength > 0 {
		nb.result.HashLength = types.Int32Value(nb.buildNameSettings.HashLength)
	} else if nb.model.Configuration.HashLength.ValueInt32() > 0 {
		nb.result.HashLength = nb.model.Configuration.HashLength
	} else {
		nb.result.HashLength = nb.typeSchema.Configuration.HashLength
	}
}

// resolveRandomSeed determines the random seed to use
func (nb *nameBuilder) resolveRandomSeed() {
	if nb.buildNameSettings.RandomSeed > 0 {
		nb.result.RandomSeed = types.Int64Value(nb.buildNameSettings.RandomSeed)
	} else {
		nb.result.RandomSeed = nb.model.Configuration.RandomSeed
	}
}

// buildNameComponents constructs the name from individual components
func (nb *nameBuilder) buildNameComponents(name types.String) {
	var calculatedContent []string

	for i := 0; i < len(nb.result.NamePrecedence.Elements()); i++ {
		switch c := (nb.result.NamePrecedence.Elements())[i].String(); strings.Trim(c, "\"") {
		case "abbreviation":
			if len(nb.typeSchema.Abbreviation.String()) > 0 {
				calculatedContent = append(calculatedContent, tools.GetBaseString(nb.typeSchema.Abbreviation))
			}
		case "prefixes":
			for j := 0; j < len(nb.result.Prefixes.Elements()); j++ {
				calculatedContent = append(calculatedContent,
					strings.Trim(nb.result.Prefixes.Elements()[j].String(), "\""))
			}
		case "suffixes":
			for j := 0; j < len(nb.result.Suffixes.Elements()); j++ {
				calculatedContent = append(calculatedContent,
					strings.Trim(nb.result.Suffixes.Elements()[j].String(), "\""))
			}
		case "name":
			if len(name.String()) > 0 {
				calculatedContent = append(calculatedContent, tools.GetBaseString(name))
			}
		case "environment":
			if len(nb.result.Environment.ValueString()) > 0 {
				calculatedContent = append(calculatedContent, tools.GetBaseString(nb.result.Environment))
			}
		case "location":
			if len(nb.result.Location.ValueString()) > 0 {
				calculatedContent = append(calculatedContent, tools.GetBaseString(nb.result.Location))
			}
		case "hash":
			if !nb.result.HashLength.IsNull() {
				var hashLength = nb.result.HashLength.ValueInt32()
				if hashLength > 0 {
					randomHash := random.Hash(int(hashLength), nb.result.RandomSeed.ValueInt64())
					calculatedContent = append(calculatedContent, randomHash)
				}
			}
		}
	}
	nb.result.Name = types.StringValue(strings.Join(calculatedContent, nb.result.Separator.ValueString()))
}

// applyLowercase converts the name to lowercase if needed
func (nb *nameBuilder) applyLowercase() {
	if nb.typeSchema.Configuration.UseLowerCase.ValueBool() || nb.model.Configuration.Lowercase.ValueBool() || nb.buildNameSettings.Lowercase {
		nb.result.Name = toLower(nb.result.Name)
	}
}

// buildName orchestrates the name building process
func (nb *nameBuilder) buildName(name types.String, resp *function.RunResponse) types.String {
	nb.setConvention()

	if nb.result.Convention.ValueString() == "default" {
		nb.resolveLocation(resp)
		nb.resolveEnvironment()
		nb.resolveSeparator()
		nb.resolveNamePrecedence(resp)
		nb.resolvePrefixes(resp)
		nb.resolveSuffixes(resp)
		nb.resolveHashLength()
		nb.resolveRandomSeed()
		nb.buildNameComponents(name)
	} else {
		tflog.Debug(nb.ctx, "configuring with passthrough convention")
		nb.result.Name = name
	}

	nb.applyLowercase()
	return nb.result.Name
}

// validationResult encapsulates the validation results for a name
type validationResult struct {
	RegexValid         bool
	LengthValid        bool
	DoubleHyphensFound bool
	Name               string
	NameLength         int64
	ValidationRegex    string
	MaxLength          int64
	MinLength          int64
	DenyDoubleHyphens  bool
}

// validateName performs validation checks on a name and returns structured results
func validateName(name string, schema *s.NamingSchema) *validationResult {
	result := &validationResult{
		Name:              name,
		NameLength:        int64(len(name)),
		ValidationRegex:   tools.GetBaseString(schema.ValidationRegex),
		MaxLength:         schema.MaxLength.ValueInt64(),
		MinLength:         schema.MinLength.ValueInt64(),
		DenyDoubleHyphens: schema.Configuration.DenyDoubleHyphens.ValueBool(),
		RegexValid:        true,
		LengthValid:       true,
	}

	// Check regex validation
	re := regexp.MustCompile(result.ValidationRegex)
	if !re.MatchString(name) {
		result.RegexValid = false
	}

	// Check length validation
	if result.NameLength > result.MaxLength || result.NameLength < result.MinLength {
		result.LengthValid = false
	}

	// Check for double hyphens
	result.DoubleHyphensFound = strings.Contains(name, "--")

	return result
}
