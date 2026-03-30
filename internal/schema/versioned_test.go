// Copyright glueckkanja AG 2025, 2026
// SPDX-License-Identifier: MPL-2.0

package schema

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ── detectVersion ────────────────────────────────────────────────────────────

func TestDetectVersion_Array(t *testing.T) {
	v, err := detectVersion([]byte(`[{"resourceType":"foo"}]`))
	require.NoError(t, err)
	assert.Equal(t, 1, v)
}

func TestDetectVersion_ArrayWithLeadingWhitespace(t *testing.T) {
	v, err := detectVersion([]byte("  \n\t[\n]"))
	require.NoError(t, err)
	assert.Equal(t, 1, v)
}

func TestDetectVersion_ObjectNoVersionField(t *testing.T) {
	v, err := detectVersion([]byte(`{"eastus":"eus"}`))
	require.NoError(t, err)
	assert.Equal(t, 1, v, "object without version field should be treated as v1")
}

func TestDetectVersion_ObjectVersionZero(t *testing.T) {
	v, err := detectVersion([]byte(`{"version":0,"resources":[]}`))
	require.NoError(t, err)
	assert.Equal(t, 1, v, "version=0 should be treated as v1")
}

func TestDetectVersion_ObjectVersion2(t *testing.T) {
	v, err := detectVersion([]byte(`{"version":2,"generatedAt":"2026-01-01T00:00:00Z","resources":[]}`))
	require.NoError(t, err)
	assert.Equal(t, 2, v)
}

func TestDetectVersion_Empty(t *testing.T) {
	_, err := detectVersion([]byte(""))
	assert.Error(t, err)
}

func TestDetectVersion_InvalidFirstByte(t *testing.T) {
	_, err := detectVersion([]byte(`"just a string"`))
	assert.Error(t, err)
}

// ── loadNamingSchemas ────────────────────────────────────────────────────────

func TestLoadNamingSchemas_V1(t *testing.T) {
	data := []byte(`[
		{
			"resourceType": "azurerm_resource_group",
			"abbreviation": "rg",
			"minLength": 1,
			"maxLength": 90,
			"validationRegex": "^[a-z]+$",
			"configuration": {
				"useEnvironment": true,
				"useLowerCase": false,
				"useUpperCase": false,
				"useSeparator": true,
				"separator": "-",
				"denyDoubleHyphens": true,
				"namePrecedence": ["abbreviation","name"],
				"hashLength": 0
			}
		}
	]`)

	schemas, err := loadNamingSchemas(data)
	require.NoError(t, err)
	require.Len(t, schemas, 1)
	assert.Equal(t, "azurerm_resource_group", schemas[0].ResourceType)
	assert.Equal(t, "rg", schemas[0].Abbreviation)
	assert.Equal(t, 1, schemas[0].MinLength)
	assert.Equal(t, 90, schemas[0].MaxLength)
	// v2 fields should be zero-valued
	assert.False(t, schemas[0].Deprecated)
	assert.Empty(t, schemas[0].DeprecatedBy)
	assert.Empty(t, schemas[0].Tags)
}

func TestLoadNamingSchemas_V2(t *testing.T) {
	data := []byte(`{
		"version": 2,
		"generatedAt": "2026-01-01T00:00:00Z",
		"resources": [
			{
				"resourceType": "azurerm_resource_group",
				"abbreviation": "rg",
				"minLength": 1,
				"maxLength": 90,
				"validationRegex": "^[a-z]+$",
				"configuration": {
					"useEnvironment": true,
					"useLowerCase": false,
					"useUpperCase": false,
					"useSeparator": true,
					"separator": "-",
					"denyDoubleHyphens": true,
					"namePrecedence": ["abbreviation","name"],
					"hashLength": 0
				},
				"deprecated": false,
				"deprecatedBy": "",
				"tags": ["core","infrastructure"]
			},
			{
				"resourceType": "azurerm_storage_account",
				"abbreviation": "st",
				"minLength": 3,
				"maxLength": 24,
				"validationRegex": "^[a-z0-9]+$",
				"configuration": {
					"useEnvironment": false,
					"useLowerCase": true,
					"useUpperCase": false,
					"useSeparator": false,
					"denyDoubleHyphens": false,
					"namePrecedence": ["abbreviation","name"],
					"hashLength": 4
				},
				"deprecated": true,
				"deprecatedBy": "azurerm_storage_account_v2",
				"tags": ["storage"]
			}
		]
	}`)

	schemas, err := loadNamingSchemas(data)
	require.NoError(t, err)
	require.Len(t, schemas, 2)

	rg := schemas[0]
	assert.Equal(t, "azurerm_resource_group", rg.ResourceType)
	assert.Equal(t, "rg", rg.Abbreviation)
	assert.False(t, rg.Deprecated)
	assert.Equal(t, []string{"core", "infrastructure"}, rg.Tags)

	st := schemas[1]
	assert.Equal(t, "azurerm_storage_account", st.ResourceType)
	assert.True(t, st.Deprecated)
	assert.Equal(t, "azurerm_storage_account_v2", st.DeprecatedBy)
	assert.Equal(t, []string{"storage"}, st.Tags)
}

func TestLoadNamingSchemas_UnsupportedVersion(t *testing.T) {
	data := []byte(`{"version":99,"resources":[]}`)
	_, err := loadNamingSchemas(data)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "version 99 is not supported")
	assert.Contains(t, err.Error(), "upgrade the provider")
}

func TestLoadNamingSchemas_InvalidJSON(t *testing.T) {
	_, err := loadNamingSchemas([]byte(`not json`))
	assert.Error(t, err)
}

// ── loadLocations ─────────────────────────────────────────────────────────────

func TestLoadLocations_V1(t *testing.T) {
	data := []byte(`{"eastus":"eus","uksouth":"uks","westeurope":"weu"}`)

	lm, err := loadLocations(data)
	require.NoError(t, err)
	require.Len(t, lm, 3)
	assert.Equal(t, "eus", lm["eastus"])
	assert.Equal(t, "uks", lm["uksouth"])
	assert.Equal(t, "weu", lm["westeurope"])
}

func TestLoadLocations_V2(t *testing.T) {
	data := []byte(`{
		"version": 2,
		"generatedAt": "2026-01-01T00:00:00Z",
		"locations": {
			"eastus": "eus",
			"uksouth": "uks"
		}
	}`)

	lm, err := loadLocations(data)
	require.NoError(t, err)
	require.Len(t, lm, 2)
	assert.Equal(t, "eus", lm["eastus"])
	assert.Equal(t, "uks", lm["uksouth"])
}

func TestLoadLocations_UnsupportedVersion(t *testing.T) {
	data := []byte(`{"version":99,"locations":{}}`)
	_, err := loadLocations(data)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "version 99 is not supported")
}
