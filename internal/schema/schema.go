// Copyright (c) glueckkanja AG
// SPDX-License-Identifier: MPL-2.0

package schema

import (
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// The following type definitions are re-used from aztools
// to have downward compatibility with the existing codebase.
type LocationsMapSchema map[string]string

var DefaultNamePrecedence = [...]string{"abbreviation", "prefixes", "name", "location", "environment", "hash", "suffixes"}

type JsonNamingSchema struct {
	ResourceType    string                  `json:"resourceType"`
	Abbreviation    string                  `json:"abbreviation"`
	MinLength       int                     `json:"minLength"`
	MaxLength       int                     `json:"maxLength"`
	ValidationRegex string                  `json:"validationRegex"`
	Configuration   JsonConfigurationSchema `json:"configuration"`
}

type JsonConfigurationSchema struct {
	UseEnvironment    bool     `json:"useEnvironment"`
	UseLowerCase      bool     `json:"useLowerCase"`
	UseSeparator      bool     `json:"useSeparator"`
	DenyDoubleHyphens bool     `json:"denyDoubleHyphens"`
	NamePrecedence    []string `json:"namePrecedence"`
	HashLength        int      `json:"hashLength"`
}

type JsonNamingSchemaMap map[string]JsonNamingSchema

// BuildNameSettingsModel contains optional settings that can override
// the default naming configuration. All fields use Go zero values
// to indicate "not set", which allows the calling code to only apply
// settings that were explicitly provided.
type BuildNameSettingsModel struct {
	Convention     string
	Environment    string
	Prefixes       []string
	Suffixes       []string
	NamePrecedence []string
	HashLength     int32
	RandomSeed     int64
	Separator      string
	Location       string
	Lowercase      bool
}

type NamingSchemaMap map[string]NamingSchema

type NamingSchema struct {
	ResourceType    types.String  `tfsdk:"resource_type"`
	Abbreviation    types.String  `tfsdk:"abbreviation"`
	MinLength       types.Int64   `tfsdk:"min_length"`
	MaxLength       types.Int64   `tfsdk:"max_length"`
	ValidationRegex types.String  `tfsdk:"validation_regex"`
	Configuration   Configuration `tfsdk:"configuration"`
}

type Configuration struct {
	UseEnvironment    types.Bool  `tfsdk:"use_environment"`
	UseLowerCase      types.Bool  `tfsdk:"use_lower_case"`
	UseSeparator      types.Bool  `tfsdk:"use_separator"`
	DenyDoubleHyphens types.Bool  `tfsdk:"deny_double_hyphens"`
	NamePrecedence    types.List  `tfsdk:"name_precedence"`
	HashLength        types.Int32 `tfsdk:"hash_length"`
}

func NewNamingSchemaMap(schemas []JsonNamingSchema) NamingSchemaMap {
	m := make(NamingSchemaMap, len(schemas))
	for _, s := range schemas {
		precedenceElements := make([]attr.Value, 0)

		if len(s.Configuration.NamePrecedence) == 0 {
			s.Configuration.NamePrecedence = DefaultNamePrecedence[:]
		}

		for _, v := range s.Configuration.NamePrecedence {
			precedenceElements = append(precedenceElements, types.StringValue(v))
		}

		m[s.ResourceType] = NamingSchema{
			ResourceType:    types.StringValue(s.ResourceType),
			Abbreviation:    types.StringValue(s.Abbreviation),
			MinLength:       types.Int64Value(int64(s.MinLength)),
			MaxLength:       types.Int64Value(int64(s.MaxLength)),
			ValidationRegex: types.StringValue(s.ValidationRegex),
			Configuration: Configuration{
				UseEnvironment:    types.BoolValue(s.Configuration.UseEnvironment),
				UseLowerCase:      types.BoolValue(s.Configuration.UseLowerCase),
				UseSeparator:      types.BoolValue(s.Configuration.UseSeparator),
				DenyDoubleHyphens: types.BoolValue(s.Configuration.DenyDoubleHyphens),
				NamePrecedence:    types.ListValueMust(types.StringType, precedenceElements),
				HashLength:        types.Int32Value(int32(s.Configuration.HashLength)),
			},
		}
	}

	return m
}

func (m JsonNamingSchemaMap) GetByResourceType(resourceType string) (JsonNamingSchema, bool) {
	s, ok := m[resourceType]
	return s, ok
}

func SchemaTypeAttributes() map[string]attr.Type {
	return map[string]attr.Type{
		"resource_type":    types.StringType,
		"abbreviation":     types.StringType,
		"min_length":       types.Int64Type,
		"max_length":       types.Int64Type,
		"validation_regex": types.StringType,
		"configuration": types.ObjectType{
			AttrTypes: map[string]attr.Type{
				"use_environment":     types.BoolType,
				"use_lower_case":      types.BoolType,
				"use_separator":       types.BoolType,
				"deny_double_hyphens": types.BoolType,
				"name_precedence":     types.ListType{ElemType: types.StringType},
				"hash_length":         types.Int32Type,
			},
		},
	}
}
