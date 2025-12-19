// Copyright (c) glueckkanja AG
// SPDX-License-Identifier: MPL-2.0

package schema

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/hashicorp/terraform-plugin-log/tflog"
	"terraform-provider-standesamt/internal/azure"
	"terraform-provider-standesamt/internal/tools"
)

// LocationFetcher defines the interface for fetching location data
type LocationFetcher interface {
	// Fetch retrieves the location map (key: location name, value: short name)
	Fetch(ctx context.Context) (LocationsMapSchema, error)
	// CacheKey returns a unique identifier for caching purposes
	CacheKey() string
}

// SchemaLocationFetcher fetches locations from the schema library (existing behavior)
type SchemaLocationFetcher struct {
	source Source
}

// NewSchemaLocationFetcher creates a new fetcher that uses the schema library
func NewSchemaLocationFetcher(source Source) *SchemaLocationFetcher {
	return &SchemaLocationFetcher{source: source}
}

func (f *SchemaLocationFetcher) Fetch(ctx context.Context) (LocationsMapSchema, error) {
	// The schema source should already be downloaded
	if f.source.Dst() == nil {
		return nil, fmt.Errorf("schema source not downloaded")
	}

	processor := NewProcessorClient(f.source.Dst())
	result := &Result{}
	if err := processor.Process(result); err != nil {
		return nil, fmt.Errorf("failed to process schema: %w", err)
	}

	return result.Locations, nil
}

func (f *SchemaLocationFetcher) CacheKey() string {
	return f.source.String()
}

// AzureLocationFetcher fetches locations from the Azure Resource Manager API
type AzureLocationFetcher struct {
	config   *azure.Config
	cacheTTL time.Duration
}

// NewAzureLocationFetcher creates a new fetcher that uses the Azure API
func NewAzureLocationFetcher(config *azure.Config) *AzureLocationFetcher {
	return &AzureLocationFetcher{
		config:   config,
		cacheTTL: 24 * time.Hour, // Cache for 24 hours by default
	}
}

// WithCacheTTL sets the cache TTL for Azure locations
func (f *AzureLocationFetcher) WithCacheTTL(ttl time.Duration) *AzureLocationFetcher {
	f.cacheTTL = ttl
	return f
}

func (f *AzureLocationFetcher) Fetch(ctx context.Context) (LocationsMapSchema, error) {
	// Check cache first
	cached, err := f.loadFromCache()
	if err == nil && cached != nil {
		return cached, nil
	}

	// Fetch from Azure API
	client, err := azure.NewLocationClient(f.config)
	if err != nil {
		return nil, fmt.Errorf("failed to create Azure location client: %w", err)
	}

	locationsMap, err := client.GetLocationsMap(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch Azure locations: %w", err)
	}

	// Save to cache
	if err := f.saveToCache(locationsMap); err != nil {
		// Log warning but don't fail
		tflog.Warn(ctx, "Failed to cache Azure locations", map[string]interface{}{"error": err.Error()})
	}

	return locationsMap, nil
}

func (f *AzureLocationFetcher) CacheKey() string {
	// Create a unique cache key based on subscription ID and environment
	key := fmt.Sprintf("azure-%s-%s", f.config.SubscriptionId, f.config.Environment)
	hash := sha256.Sum256([]byte(key))
	return hex.EncodeToString(hash[:8])
}

func (f *AzureLocationFetcher) cacheFilePath() string {
	cacheDir := tools.NamingSchemaCacheDir()
	return filepath.Join(cacheDir, fmt.Sprintf("azure-locations-%s.json", f.CacheKey()))
}

type azureLocationCache struct {
	Locations LocationsMapSchema `json:"locations"`
	Timestamp time.Time          `json:"timestamp"`
}

func (f *AzureLocationFetcher) loadFromCache() (LocationsMapSchema, error) {
	cachePath := f.cacheFilePath()

	data, err := os.ReadFile(cachePath)
	if err != nil {
		return nil, err
	}

	var cache azureLocationCache
	if err := json.Unmarshal(data, &cache); err != nil {
		return nil, err
	}

	// Check if cache is expired
	if time.Since(cache.Timestamp) > f.cacheTTL {
		return nil, fmt.Errorf("cache expired")
	}

	return cache.Locations, nil
}

func (f *AzureLocationFetcher) saveToCache(locations LocationsMapSchema) error {
	cachePath := f.cacheFilePath()

	// Ensure cache directory exists
	cacheDir := filepath.Dir(cachePath)
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		return fmt.Errorf("failed to create cache directory: %w", err)
	}

	cache := azureLocationCache{
		Locations: locations,
		Timestamp: time.Now(),
	}

	data, err := json.MarshalIndent(cache, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal cache: %w", err)
	}

	if err := os.WriteFile(cachePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write cache file: %w", err)
	}

	return nil
}

// ApplyAliases applies location aliases to a location map
// The aliases map has the format: original_name -> new_name
func ApplyAliases(locations LocationsMapSchema, aliases map[string]string) LocationsMapSchema {
	if len(aliases) == 0 {
		return locations
	}

	result := make(LocationsMapSchema, len(locations))
	for key, value := range locations {
		// Check if there's an alias for this location
		if alias, ok := aliases[key]; ok {
			result[key] = alias
		} else {
			result[key] = value
		}
	}

	return result
}
