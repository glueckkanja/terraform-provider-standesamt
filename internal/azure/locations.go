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
	RegionType       string
	RegionCategory   string
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

// GetLocations fetches all available Azure locations for the configured subscription.
// Only physical locations are returned (regionType == "Physical").
// Logical regions and edge zones are filtered out.
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

			// Filter by regionType - only include "Physical" regions
			// This excludes logical regions and edge zones
			if loc.Metadata == nil || loc.Metadata.RegionType == nil {
				continue
			}
			if *loc.Metadata.RegionType != armsubscriptions.RegionTypePhysical {
				continue
			}

			location := Location{
				Name:                *loc.Name,
				DisplayName:         safeString(loc.DisplayName),
				RegionalDisplayName: safeString(loc.RegionalDisplayName),
			}

			location.Metadata = LocationMetadata{
				GeographyGroup:   safeString(loc.Metadata.GeographyGroup),
				Latitude:         safeString(loc.Metadata.Latitude),
				Longitude:        safeString(loc.Metadata.Longitude),
				PhysicalLocation: safeString(loc.Metadata.PhysicalLocation),
				RegionType:       string(*loc.Metadata.RegionType),
				RegionCategory:   safeRegionCategory(loc.Metadata.RegionCategory),
			}

			if loc.Metadata.PairedRegion != nil {
				for _, pr := range loc.Metadata.PairedRegion {
					if pr.Name != nil {
						location.Metadata.PairedRegion = append(location.Metadata.PairedRegion, *pr.Name)
					}
				}
			}

			locations = append(locations, location)
		}
	}

	return locations, nil
}

// GetLocationsMap returns a map of location names to their short geo-codes.
// This is the format expected by the schema package (LocationsMapSchema).
// By default, it applies the official Microsoft Azure Backup geo-code mappings.
// Users can override these with location_aliases in the provider config.
func (c *LocationClient) GetLocationsMap(ctx context.Context) (map[string]string, error) {
	locations, err := c.GetLocations(ctx)
	if err != nil {
		return nil, err
	}

	result := make(map[string]string, len(locations))
	for _, loc := range locations {
		// Apply the default geo-code mapping if available,
		// otherwise use the location name as the value
		result[loc.Name] = GetGeoCode(loc.Name)
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

// safeRegionCategory safely converts a RegionCategory pointer to string
func safeRegionCategory(rc *armsubscriptions.RegionCategory) string {
	if rc == nil {
		return ""
	}
	return string(*rc)
}

// NormalizeLocationName normalizes a location name for comparison
func NormalizeLocationName(name string) string {
	return strings.ToLower(strings.ReplaceAll(name, " ", ""))
}
