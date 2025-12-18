// Copyright (c) glueckkanja AG
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"testing"

	s "terraform-provider-standesamt/internal/schema"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stretchr/testify/assert"
)

func TestExtractStringSlice(t *testing.T) {
	tests := []struct {
		name     string
		list     types.List
		expected []string
	}{
		{
			name:     "empty list",
			list:     types.ListValueMust(types.StringType, []attr.Value{}),
			expected: nil,
		},
		{
			name: "list with values",
			list: types.ListValueMust(types.StringType, []attr.Value{
				types.StringValue("prefix1"),
				types.StringValue("prefix2"),
			}),
			expected: []string{"prefix1", "prefix2"},
		},
		{
			name:     "null list",
			list:     types.ListNull(types.StringType),
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractStringSlice(tt.list)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestParseSettingsFromDynamic(t *testing.T) {
	tests := []struct {
		name        string
		dynamic     types.Dynamic
		wantErr     bool
		checkResult func(*testing.T, *parseSettingsResult)
	}{
		{
			name:    "null dynamic",
			dynamic: types.DynamicNull(),
			wantErr: false,
			checkResult: func(t *testing.T, result *parseSettingsResult) {
				assert.Empty(t, result.settings.Convention)
				assert.Empty(t, result.settings.Environment)
			},
		},
		{
			name: "valid object with convention",
			dynamic: types.DynamicValue(types.ObjectValueMust(
				map[string]attr.Type{
					"convention": types.StringType,
				},
				map[string]attr.Value{
					"convention": types.StringValue("passthrough"),
				},
			)),
			wantErr: false,
			checkResult: func(t *testing.T, result *parseSettingsResult) {
				assert.Equal(t, "passthrough", result.settings.Convention)
			},
		},
		{
			name: "valid object with multiple fields",
			dynamic: types.DynamicValue(types.ObjectValueMust(
				map[string]attr.Type{
					"convention":  types.StringType,
					"environment": types.StringType,
					"lowercase":   types.BoolType,
				},
				map[string]attr.Value{
					"convention":  types.StringValue("default"),
					"environment": types.StringValue("prod"),
					"lowercase":   types.BoolValue(true),
				},
			)),
			wantErr: false,
			checkResult: func(t *testing.T, result *parseSettingsResult) {
				assert.Equal(t, "default", result.settings.Convention)
				assert.Equal(t, "prod", result.settings.Environment)
				assert.True(t, result.settings.Lowercase)
			},
		},
		{
			name:    "non-object value",
			dynamic: types.DynamicValue(types.StringValue("not an object")),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			settings, err := parseSettingsFromDynamic(tt.dynamic)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				if tt.checkResult != nil {
					tt.checkResult(t, &parseSettingsResult{settings: settings})
				}
			}
		})
	}
}

type parseSettingsResult struct {
	settings *s.BuildNameSettingsModel
}
