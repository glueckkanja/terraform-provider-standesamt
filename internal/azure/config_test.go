// Copyright (c) glueckkanja AG
// SPDX-License-Identifier: MPL-2.0

package azure

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestConfig_Validate(t *testing.T) {
	tests := []struct {
		name        string
		config      Config
		wantErr     bool
		errContains string
	}{
		{
			name: "valid config with subscription_id and use_cli",
			config: Config{
				SubscriptionId: "12345678-1234-1234-1234-123456789abc",
				UseCli:         true,
			},
			wantErr: false,
		},
		{
			name: "valid config with service principal",
			config: Config{
				SubscriptionId: "12345678-1234-1234-1234-123456789abc",
				ClientId:       "client-id",
				ClientSecret:   "client-secret",
				TenantId:       "tenant-id",
			},
			wantErr: false,
		},
		{
			name: "missing subscription_id",
			config: Config{
				UseCli: true,
			},
			wantErr:     true,
			errContains: "subscription_id is required",
		},
		{
			name: "no auth method defaults to CLI",
			config: Config{
				SubscriptionId: "12345678-1234-1234-1234-123456789abc",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if tt.wantErr {
				assert.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestConfig_GetCloudConfig(t *testing.T) {
	tests := []struct {
		name        string
		environment CloudEnvironment
		wantName    string
	}{
		{
			name:        "public cloud",
			environment: CloudEnvironmentPublic,
			wantName:    "AzurePublic",
		},
		{
			name:        "us government cloud",
			environment: CloudEnvironmentUSGovernment,
			wantName:    "AzureGovernment",
		},
		{
			name:        "china cloud",
			environment: CloudEnvironmentChina,
			wantName:    "AzureChina",
		},
		{
			name:        "empty defaults to public",
			environment: "",
			wantName:    "AzurePublic",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := Config{Environment: tt.environment}
			cloudConfig := config.GetCloudConfig()

			// Verify by checking the ActiveDirectoryAuthorityHost
			switch tt.wantName {
			case "AzurePublic":
				assert.Contains(t, cloudConfig.ActiveDirectoryAuthorityHost, "login.microsoftonline.com")
			case "AzureGovernment":
				assert.Contains(t, cloudConfig.ActiveDirectoryAuthorityHost, "login.microsoftonline.us")
			case "AzureChina":
				assert.Contains(t, cloudConfig.ActiveDirectoryAuthorityHost, "login.chinacloudapi.cn")
			}
		})
	}
}

func TestConfig_ValidateDefaultsToCliAuth(t *testing.T) {
	config := Config{
		SubscriptionId: "12345678-1234-1234-1234-123456789abc",
	}

	err := config.Validate()
	assert.NoError(t, err)
	assert.True(t, config.UseCli, "Should default to CLI auth when no auth method specified")
}
