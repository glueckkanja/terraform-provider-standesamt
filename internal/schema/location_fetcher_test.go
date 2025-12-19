// Copyright (c) glueckkanja AG
// SPDX-License-Identifier: MPL-2.0

package schema

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"terraform-provider-standesamt/internal/azure"
)

func TestApplyAliases(t *testing.T) {
	tests := []struct {
		name      string
		locations LocationsMapSchema
		aliases   map[string]string
		expected  LocationsMapSchema
	}{
		{
			name: "apply single alias",
			locations: LocationsMapSchema{
				"eastus":     "eastus",
				"westeurope": "westeurope",
			},
			aliases: map[string]string{
				"eastus": "eus",
			},
			expected: LocationsMapSchema{
				"eastus":     "eus",
				"westeurope": "westeurope",
			},
		},
		{
			name: "apply multiple aliases",
			locations: LocationsMapSchema{
				"eastus":             "eastus",
				"westeurope":         "westeurope",
				"germanywestcentral": "germanywestcentral",
			},
			aliases: map[string]string{
				"eastus":             "eus",
				"westeurope":         "weu",
				"germanywestcentral": "gwc",
			},
			expected: LocationsMapSchema{
				"eastus":             "eus",
				"westeurope":         "weu",
				"germanywestcentral": "gwc",
			},
		},
		{
			name: "no aliases",
			locations: LocationsMapSchema{
				"eastus":     "eastus",
				"westeurope": "westeurope",
			},
			aliases: nil,
			expected: LocationsMapSchema{
				"eastus":     "eastus",
				"westeurope": "westeurope",
			},
		},
		{
			name: "alias for non-existent location",
			locations: LocationsMapSchema{
				"eastus": "eastus",
			},
			aliases: map[string]string{
				"westeurope": "weu",
			},
			expected: LocationsMapSchema{
				"eastus": "eastus",
			},
		},
		{
			name:      "empty locations",
			locations: LocationsMapSchema{},
			aliases: map[string]string{
				"eastus": "eus",
			},
			expected: LocationsMapSchema{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ApplyAliases(tt.locations, tt.aliases)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestAzureLocationFetcher_CacheKey(t *testing.T) {
	config1 := &azure.Config{
		SubscriptionId: "sub-1",
		Environment:    azure.CloudEnvironmentPublic,
	}
	config2 := &azure.Config{
		SubscriptionId: "sub-2",
		Environment:    azure.CloudEnvironmentPublic,
	}
	config3 := &azure.Config{
		SubscriptionId: "sub-1",
		Environment:    azure.CloudEnvironmentUSGovernment,
	}

	fetcher1 := NewAzureLocationFetcher(config1)
	fetcher2 := NewAzureLocationFetcher(config2)
	fetcher3 := NewAzureLocationFetcher(config3)
	fetcher1Again := NewAzureLocationFetcher(config1)

	// Same config should produce same cache key
	assert.Equal(t, fetcher1.CacheKey(), fetcher1Again.CacheKey())

	// Different subscription should produce different cache key
	assert.NotEqual(t, fetcher1.CacheKey(), fetcher2.CacheKey())

	// Different environment should produce different cache key
	assert.NotEqual(t, fetcher1.CacheKey(), fetcher3.CacheKey())
}

func TestAzureLocationFetcher_Cache(t *testing.T) {
	// Create a temporary directory for cache
	tmpDir := t.TempDir()
	t.Setenv("SA_NAMING_DIR", tmpDir)

	config := &azure.Config{
		SubscriptionId: "test-subscription",
		Environment:    azure.CloudEnvironmentPublic,
	}

	fetcher := NewAzureLocationFetcher(config)

	// Test saveToCache and loadFromCache
	testLocations := LocationsMapSchema{
		"eastus":     "eastus",
		"westeurope": "westeurope",
	}

	err := fetcher.saveToCache(testLocations)
	require.NoError(t, err)

	// Verify cache file exists
	cachePath := fetcher.cacheFilePath()
	_, err = os.Stat(cachePath)
	require.NoError(t, err)

	// Load from cache
	loaded, err := fetcher.loadFromCache()
	require.NoError(t, err)
	assert.Equal(t, testLocations, loaded)
}

func TestAzureLocationFetcher_CacheExpiry(t *testing.T) {
	// Create a temporary directory for cache
	tmpDir := t.TempDir()
	t.Setenv("SA_NAMING_DIR", tmpDir)

	config := &azure.Config{
		SubscriptionId: "test-subscription",
		Environment:    azure.CloudEnvironmentPublic,
	}

	// Create fetcher with very short TTL
	fetcher := NewAzureLocationFetcher(config).WithCacheTTL(1 * time.Millisecond)

	testLocations := LocationsMapSchema{
		"eastus": "eastus",
	}

	err := fetcher.saveToCache(testLocations)
	require.NoError(t, err)

	// Wait for cache to expire
	time.Sleep(10 * time.Millisecond)

	// Should return error for expired cache
	_, err = fetcher.loadFromCache()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "expired")
}

func TestSchemaLocationFetcher_CacheKey(t *testing.T) {
	source1 := NewDefaultSource("azure/caf", "2025.04")
	source2 := NewDefaultSource("azure/caf", "2025.05")
	source3 := NewDefaultSource("other/path", "2025.04")

	fetcher1 := NewSchemaLocationFetcher(source1)
	fetcher2 := NewSchemaLocationFetcher(source2)
	fetcher3 := NewSchemaLocationFetcher(source3)
	fetcher1Again := NewSchemaLocationFetcher(source1)

	// Same source should produce same cache key
	assert.Equal(t, fetcher1.CacheKey(), fetcher1Again.CacheKey())

	// Different ref should produce different cache key
	assert.NotEqual(t, fetcher1.CacheKey(), fetcher2.CacheKey())

	// Different path should produce different cache key
	assert.NotEqual(t, fetcher1.CacheKey(), fetcher3.CacheKey())
}

func TestAzureLocationFetcher_CacheFilePath(t *testing.T) {
	config := &azure.Config{
		SubscriptionId: "test-sub",
		Environment:    azure.CloudEnvironmentPublic,
	}

	fetcher := NewAzureLocationFetcher(config)
	cachePath := fetcher.cacheFilePath()

	// Should contain the cache key
	assert.Contains(t, cachePath, "azure-locations-")
	assert.Contains(t, cachePath, ".json")

	// Should be in the cache directory
	dir := filepath.Dir(cachePath)
	assert.NotEmpty(t, dir)
}

func TestSchemaLocationFetcher_FetchWithoutDownload(t *testing.T) {
	source := NewDefaultSource("azure/caf", "2025.04")
	fetcher := NewSchemaLocationFetcher(source)

	// Should fail because source hasn't been downloaded
	_, err := fetcher.Fetch(context.Background())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not downloaded")
}
