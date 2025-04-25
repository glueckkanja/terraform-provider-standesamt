// Copyright (c) glueckkanja AG
// SPDX-License-Identifier: MPL-2.0

package tools

import "testing"

func TestNamingSchemaCacheDir(t *testing.T) {
	tests := []struct {
		name string
		want string
	}{
		{
			name: "default",
			want: ".standesamt",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NamingSchemaCacheDir(); got != tt.want {
				t.Errorf("NamingSchemaCacheDir() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNamingSchemaGitUrl(t *testing.T) {
	tests := []struct {
		name string
		want string
	}{
		{
			name: "default",
			want: "github.com/glueckkanja/standesamt-schema-library",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NamingSchemaGitUrl(); got != tt.want {
				t.Errorf("NamingSchemaGitUrl() = %v, want %v", got, tt.want)
			}
		})
	}
}
