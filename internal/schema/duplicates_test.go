// Copyright glueckkanja AG 2025, 2026
// SPDX-License-Identifier: MPL-2.0

package schema

import (
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ── findDuplicateResourceTypes ──────────────────────────────────────────────

func rt(resourceType, abbreviation string) JsonNamingSchema {
	return JsonNamingSchema{ResourceType: resourceType, Abbreviation: abbreviation}
}

func TestFindDuplicateResourceTypes_None(t *testing.T) {
	assert.Nil(t, findDuplicateResourceTypes([]JsonNamingSchema{rt("a", "a"), rt("b", "b")}))
	assert.Nil(t, findDuplicateResourceTypes(nil))
}

func TestFindDuplicateResourceTypes_Conflicting(t *testing.T) {
	msgs := findDuplicateResourceTypes([]JsonNamingSchema{
		rt("azurerm_resource_group", "rg"),
		rt("azurerm_storage_account", "st"),
		rt("azurerm_resource_group", "rsg"),
	})
	require.Len(t, msgs, 1)
	assert.Equal(t,
		`resourceType "azurerm_resource_group" is defined 2 times (entries #1, #3); entries differ; the last entry (#3) is used`,
		msgs[0])
}

func TestFindDuplicateResourceTypes_Identical(t *testing.T) {
	msgs := findDuplicateResourceTypes([]JsonNamingSchema{rt("x", "cog"), rt("x", "cog")})
	require.Len(t, msgs, 1)
	assert.Contains(t, msgs[0], "entries are identical")
}

func TestFindDuplicateResourceTypes_ThreeCopiesOneDiffers(t *testing.T) {
	msgs := findDuplicateResourceTypes([]JsonNamingSchema{rt("x", "a"), rt("x", "a"), rt("x", "b")})
	require.Len(t, msgs, 1)
	assert.Contains(t, msgs[0], "defined 3 times (entries #1, #2, #3)")
	assert.Contains(t, msgs[0], "entries differ")
	assert.Contains(t, msgs[0], "the last entry (#3) is used")
}

func TestFindDuplicateResourceTypes_SortedOutput(t *testing.T) {
	msgs := findDuplicateResourceTypes([]JsonNamingSchema{
		rt("zeta", "z"), rt("alpha", "a"), rt("zeta", "z"), rt("alpha", "a"),
	})
	require.Len(t, msgs, 2)
	assert.Contains(t, msgs[0], `"alpha"`)
	assert.Contains(t, msgs[1], `"zeta"`)
}

// ── findDuplicateObjectKeys ─────────────────────────────────────────────────

func TestFindDuplicateObjectKeys_None(t *testing.T) {
	msgs, err := findDuplicateObjectKeys([]byte(`{"eastus":"eus","westeurope":"weu"}`), "location")
	require.NoError(t, err)
	assert.Nil(t, msgs)
}

func TestFindDuplicateObjectKeys_Duplicate(t *testing.T) {
	msgs, err := findDuplicateObjectKeys([]byte(`{"eastus":"eus","westeurope":"weu","eastus":"eus1"}`), "location")
	require.NoError(t, err)
	require.Len(t, msgs, 1)
	assert.Equal(t,
		`location "eastus" is defined 2 times (values "eus", "eus1"); the last value ("eus1") is used`,
		msgs[0])
}

func TestFindDuplicateObjectKeys_NestedValuesAreSkipped(t *testing.T) {
	// Duplicate keys inside nested values are not top-level duplicates.
	msgs, err := findDuplicateObjectKeys([]byte(`{"a":{"x":1,"x":2},"b":[1,2]}`), "key")
	require.NoError(t, err)
	assert.Nil(t, msgs)
}

func TestFindDuplicateObjectKeys_Errors(t *testing.T) {
	_, err := findDuplicateObjectKeys([]byte(`not json`), "location")
	assert.Error(t, err)

	_, err = findDuplicateObjectKeys([]byte(`["eastus"]`), "location")
	assert.Error(t, err)
}

// ── locationsRaw ─────────────────────────────────────────────────────────────

func TestLocationsRaw_V2IgnoresEnvelopeKeys(t *testing.T) {
	// "version" is duplicated at envelope level; only the locations object is returned.
	raw, err := locationsRaw([]byte(`{"version":2,"version":2,"locations":{"eastus":"eus","eastus":"eus1"}}`))
	require.NoError(t, err)

	msgs, err := findDuplicateObjectKeys(raw, "location")
	require.NoError(t, err)
	require.Len(t, msgs, 1)
	assert.Contains(t, msgs[0], `location "eastus"`)
}

// ── ProcessorClient integration ─────────────────────────────────────────────

func TestProcess_ReportsDuplicatesAndKeepsLastWins(t *testing.T) {
	fsys := fstest.MapFS{
		"lib/schema.naming.json": {Data: []byte(`{
			"version": 2,
			"resources": [
				{ "resourceType": "azurerm_resource_group", "abbreviation": "rg",  "configuration": { "namePrecedence": ["abbreviation", "name", "environment"] } },
				{ "resourceType": "azurerm_resource_group", "abbreviation": "rg",  "configuration": { "namePrecedence": ["name", "abbreviation"] } }
			]
		}`)},
		"lib/schema.locations.json": {Data: []byte(`{"eastus":"eus","eastus":"eus1"}`)},
	}

	res := Result{}
	require.NoError(t, NewProcessorClient(fsys).Process(&res))

	require.Len(t, res.NamingDuplicates, 1)
	assert.Contains(t, res.NamingDuplicates[0], "schema.naming.json: ")
	assert.Contains(t, res.NamingDuplicates[0], "entries differ")

	require.Len(t, res.LocationDuplicates, 1)
	assert.Contains(t, res.LocationDuplicates[0], "schema.locations.json: ")

	// Behaviour is unchanged: the last entry / value still wins.
	m := NewNamingSchemaMap(res.NamingSchemas)
	assert.Equal(t, `["name","abbreviation"]`, m["azurerm_resource_group"].Configuration.NamePrecedence.String())
	assert.Equal(t, "eus1", res.Locations["eastus"])
}

func TestProcess_NoDuplicates(t *testing.T) {
	fsys := fstest.MapFS{
		"schema.naming.json":    {Data: []byte(`[{"resourceType":"a"},{"resourceType":"b"}]`)},
		"schema.locations.json": {Data: []byte(`{"version":2,"locations":{"eastus":"eus"}}`)},
	}

	res := Result{}
	require.NoError(t, NewProcessorClient(fsys).Process(&res))
	assert.Nil(t, res.NamingDuplicates)
	assert.Nil(t, res.LocationDuplicates)
}
