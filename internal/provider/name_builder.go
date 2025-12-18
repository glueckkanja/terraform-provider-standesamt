// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"terraform-provider-standesamt/internal/random"
	s "terraform-provider-standesamt/internal/schema"
	"terraform-provider-standesamt/internal/tools"

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

// validateJSONString checks if a JSON string contains invalid patterns like <null>
func validateJSONString(jsonStr string) error {
	// Check for <null> pattern which is invalid JSON
	if strings.Contains(jsonStr, "<null>") {
		return fmt.Errorf("invalid JSON: contains <null> pattern")
	}
	// Check if it starts with < which could indicate malformed null values
	trimmed := strings.TrimSpace(jsonStr)
	if strings.HasPrefix(trimmed, "<") {
		return fmt.Errorf("invalid JSON: starts with < character")
	}
	// Try to validate it's proper JSON
	var temp interface{}
	return json.Unmarshal([]byte(jsonStr), &temp)
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
		resp.Error = function.NewArgumentFuncError(1, "name_type not found in schema")
		return nil, "", nil, types.String{}, nil, resp.Error
	}

	// Parse optional settings from dynamic parameter
	if !settingsDynamic.IsNull() && !settingsDynamic.IsUnderlyingValueNull() {
		switch settingsDynamic.UnderlyingValue().(type) {
		case types.Object:
			// Parse optional settings from dynamic parameter
			// The String() function returns a JSON representation of the object,
			// which we can unmarshal into our struct leveraging json.omitempty tags
			// to handle optional attributes that may not be present
			jsonStr := settingsDynamic.UnderlyingValue().String()

			// Validate JSON before unmarshalling to catch invalid patterns like <null>
			if err := validateJSONString(jsonStr); err != nil {
				resp.Error = function.ConcatFuncErrors(resp.Error, function.NewArgumentFuncError(2, "invalid JSON in settings parameter: "+err.Error()))
				return nil, "", nil, types.String{}, nil, resp.Error
			}

			err := json.Unmarshal([]byte(jsonStr), &buildNameSettings)
			if err != nil {
				resp.Error = function.ConcatFuncErrors(resp.Error, function.NewArgumentFuncError(2, err.Error()))
				return nil, "", nil, types.String{}, nil, resp.Error
			}
		default:
			resp.Error = function.ConcatFuncErrors(resp.Error, function.NewArgumentFuncError(2, "settingsDynamic is not an object"))
			return nil, "", nil, types.String{}, nil, resp.Error
		}
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
