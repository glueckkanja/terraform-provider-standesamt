// Copyright glueckkanja AG 2025, 2026
// SPDX-License-Identifier: MPL-2.0

package schema

import (
	"bytes"
	"encoding/json"
	"fmt"
)

// maxSupportedSchemaVersion is the highest schema version this provider understands.
// Bump this when adding support for a new version.
const maxSupportedSchemaVersion = 2

// versionProbe is used to read only the "version" field from an envelope object
// without fully deserialising it.
type versionProbe struct {
	Version int `json:"version"`
}

// namingSchemaEnvelopeV2 is the versioned wrapper introduced in schema v2.
type namingSchemaEnvelopeV2 struct {
	Version     int                `json:"version"`
	GeneratedAt string             `json:"generatedAt"`
	Resources   []JsonNamingSchema `json:"resources"`
}

// locationsEnvelopeV2 is the versioned wrapper for locations introduced in schema v2.
type locationsEnvelopeV2 struct {
	Version     int                `json:"version"`
	GeneratedAt string             `json:"generatedAt"`
	Locations   LocationsMapSchema `json:"locations"`
}

// detectVersion peeks at the raw JSON bytes to determine the schema version.
//
// Rules:
//   - First non-whitespace byte is '[' → v1 (legacy raw array, frozen)
//   - First non-whitespace byte is '{' → probe for "version" integer field;
//     missing or zero is treated as v1 for forward-compatibility with any
//     old object-style schemas.
func detectVersion(data []byte) (int, error) {
	trimmed := bytes.TrimLeft(data, " \t\n\r")
	if len(trimmed) == 0 {
		return 0, fmt.Errorf("schema file is empty")
	}

	switch trimmed[0] {
	case '[':
		return 1, nil
	case '{':
		var probe versionProbe
		if err := json.Unmarshal(data, &probe); err != nil {
			return 0, fmt.Errorf("detectVersion: failed to probe version field: %w", err)
		}
		if probe.Version == 0 {
			return 1, nil
		}
		return probe.Version, nil
	default:
		return 0, fmt.Errorf("detectVersion: unexpected first byte %q — expected '[' or '{'", trimmed[0])
	}
}

// loadNamingSchemas is the version-dispatching entry point for naming schema files.
//
// v1 (raw JSON array)   → unmarshalled directly as []JsonNamingSchema
// v2 (versioned object) → envelope unwrapped, .Resources returned
//
// An explicit error is returned for any version beyond maxSupportedSchemaVersion
// so that users receive a clear message rather than a confusing parse failure.
func loadNamingSchemas(data []byte) ([]JsonNamingSchema, error) {
	version, err := detectVersion(data)
	if err != nil {
		return nil, fmt.Errorf("loadNamingSchemas: %w", err)
	}

	switch version {
	case 1:
		var schemas []JsonNamingSchema
		if err := json.Unmarshal(data, &schemas); err != nil {
			return nil, fmt.Errorf("loadNamingSchemas: v1: failed to unmarshal: %w", err)
		}
		return schemas, nil

	case 2:
		var envelope namingSchemaEnvelopeV2
		if err := json.Unmarshal(data, &envelope); err != nil {
			return nil, fmt.Errorf("loadNamingSchemas: v2: failed to unmarshal: %w", err)
		}
		return envelope.Resources, nil

	default:
		return nil, fmt.Errorf(
			"loadNamingSchemas: schema version %d is not supported by this provider (max supported: %d); upgrade the provider",
			version, maxSupportedSchemaVersion,
		)
	}
}

// loadLocations is the version-dispatching entry point for location schema files.
//
// v1 (raw JSON object / flat map) → unmarshalled directly as LocationsMapSchema
// v2 (versioned object)           → envelope unwrapped, .Locations returned
func loadLocations(data []byte) (LocationsMapSchema, error) {
	version, err := detectVersion(data)
	if err != nil {
		return nil, fmt.Errorf("loadLocations: %w", err)
	}

	switch version {
	case 1:
		var lm LocationsMapSchema
		if err := json.Unmarshal(data, &lm); err != nil {
			return nil, fmt.Errorf("loadLocations: v1: failed to unmarshal: %w", err)
		}
		return lm, nil

	case 2:
		var envelope locationsEnvelopeV2
		if err := json.Unmarshal(data, &envelope); err != nil {
			return nil, fmt.Errorf("loadLocations: v2: failed to unmarshal: %w", err)
		}
		return envelope.Locations, nil

	default:
		return nil, fmt.Errorf(
			"loadLocations: schema version %d is not supported by this provider (max supported: %d); upgrade the provider",
			version, maxSupportedSchemaVersion,
		)
	}
}
