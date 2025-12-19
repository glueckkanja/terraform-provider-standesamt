// Copyright (c) glueckkanja AG
// SPDX-License-Identifier: MPL-2.0

package azure

import (
	"context"
	"fmt"
	"strings"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resources/armsubscriptions"
)

// Location represents an Azure location with its metadata
type Location struct {
	Name                string
	DisplayName         string
	RegionalDisplayName string
	Metadata            LocationMetadata
}

// LocationMetadata contains additional location metadata
type LocationMetadata struct {
	GeographyGroup   string
	Latitude         string
	Longitude        string
	PhysicalLocation string
	PairedRegion     []string
}

// LocationClient provides methods to fetch Azure locations
type LocationClient struct {
	config *Config
}

// NewLocationClient creates a new LocationClient with the given configuration
func NewLocationClient(config *Config) (*LocationClient, error) {
	if err := config.Validate(); err != nil {
		return nil, err
	}
	return &LocationClient{config: config}, nil
}

// GetLocations fetches all available Azure locations for the configured subscription
func (c *LocationClient) GetLocations(ctx context.Context) ([]Location, error) {
	cred, err := c.config.GetCredential(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get Azure credential: %w", err)
	}

	clientFactory, err := armsubscriptions.NewClientFactory(cred, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create subscriptions client factory: %w", err)
	}

	client := clientFactory.NewClient()

	var locations []Location
	pager := client.NewListLocationsPager(c.config.SubscriptionId, &armsubscriptions.ClientListLocationsOptions{
		IncludeExtendedLocations: nil, // Only include standard locations
	})

	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to list Azure locations: %w", err)
		}

		for _, loc := range page.Value {
			if loc == nil || loc.Name == nil {
				continue
			}

			// Only include physical locations (not logical/edge zones)
			if loc.Type != nil && *loc.Type != armsubscriptions.LocationTypeRegion {
				continue
			}

			location := Location{
				Name:                *loc.Name,
				DisplayName:         safeString(loc.DisplayName),
				RegionalDisplayName: safeString(loc.RegionalDisplayName),
			}

			if loc.Metadata != nil {
				location.Metadata = LocationMetadata{
					GeographyGroup:   safeString(loc.Metadata.GeographyGroup),
					Latitude:         safeString(loc.Metadata.Latitude),
					Longitude:        safeString(loc.Metadata.Longitude),
					PhysicalLocation: safeString(loc.Metadata.PhysicalLocation),
				}

				if loc.Metadata.PairedRegion != nil {
					for _, pr := range loc.Metadata.PairedRegion {
						if pr.Name != nil {
							location.Metadata.PairedRegion = append(location.Metadata.PairedRegion, *pr.Name)
						}
					}
				}
			}

			locations = append(locations, location)
		}
	}

	return locations, nil
}

// GetLocationsMap returns a map of location names to their short names (same as name by default)
// This is the format expected by the schema package (LocationsMapSchema)
func (c *LocationClient) GetLocationsMap(ctx context.Context) (map[string]string, error) {
	locations, err := c.GetLocations(ctx)
	if err != nil {
		return nil, err
	}

	result := make(map[string]string, len(locations))
	for _, loc := range locations {
		// By default, use the location name as both key and value
		// The key is the full name (e.g., "eastus"), value is the short name
		// Users can override these with location_aliases in the provider config
		result[loc.Name] = loc.Name
	}

	return result, nil
}

// safeString safely dereferences a string pointer
func safeString(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

// NormalizeLocationName normalizes a location name for comparison
func NormalizeLocationName(name string) string {
	return strings.ToLower(strings.ReplaceAll(name, " ", ""))
}
