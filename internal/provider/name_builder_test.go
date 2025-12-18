// Copyright (c) glueckkanja AG
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestValidateJSONString(t *testing.T) {
	tests := []struct {
		name    string
		jsonStr string
		wantErr bool
	}{
		{
			name:    "valid empty object",
			jsonStr: "{}",
			wantErr: false,
		},
		{
			name:    "valid object with fields",
			jsonStr: `{"convention":"default","environment":"test"}`,
			wantErr: false,
		},
		{
			name:    "valid object with null values",
			jsonStr: `{"convention":"default","environment":null}`,
			wantErr: false,
		},
		{
			name:    "invalid JSON with <null> pattern",
			jsonStr: `{"convention":"default","environment":<null>}`,
			wantErr: true,
		},
		{
			name:    "invalid JSON starting with <",
			jsonStr: `<null>`,
			wantErr: true,
		},
		{
			name:    "invalid JSON with < in middle",
			jsonStr: `{"test":<null>}`,
			wantErr: true,
		},
		{
			name:    "valid array",
			jsonStr: `["item1","item2"]`,
			wantErr: false,
		},
		{
			name:    "invalid JSON syntax",
			jsonStr: `{invalid json}`,
			wantErr: true,
		},
		{
			name:    "empty string",
			jsonStr: "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateJSONString(tt.jsonStr)
			if tt.wantErr {
				assert.Error(t, err, "validateJSONString() should return error for input: %s", tt.jsonStr)
			} else {
				assert.NoError(t, err, "validateJSONString() should not return error for input: %s", tt.jsonStr)
			}
		})
	}
}
