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

// ── objectPairs / duplicatePairs ────────────────────────────────────────────

func dupsOf(t *testing.T, raw string, label string) []string {
	t.Helper()
	pairs, err := objectPairs([]byte(raw))
	require.NoError(t, err)
	return duplicatePairs(pairs, label)
}

func TestDuplicatePairs_None(t *testing.T) {
	assert.Nil(t, dupsOf(t, `{"eastus":"eus","westeurope":"weu"}`, "location"))
}

func TestDuplicatePairs_Duplicate(t *testing.T) {
	msgs := dupsOf(t, `{"eastus":"eus","westeurope":"weu","eastus":"eus1"}`, "location")
	require.Len(t, msgs, 1)
	assert.Equal(t,
		`location "eastus" is defined 2 times (values "eus", "eus1"); the last value ("eus1") is used`,
		msgs[0])
}

func TestDuplicatePairs_NestedValuesAreSkipped(t *testing.T) {
	assert.Nil(t, dupsOf(t, `{"a":{"x":1,"x":2},"b":[1,2]}`, "key"))
}

func TestObjectPairs_Errors(t *testing.T) {
	_, err := objectPairs([]byte(`not json`))
	assert.Error(t, err)

	_, err = objectPairs([]byte(`["eastus"]`))
	assert.Error(t, err)
}

// ── locationPairs ────────────────────────────────────────────────────────────

func TestLocationPairs(t *testing.T) {
	tests := []struct {
		name string
		doc  string
		dups []string // keys expected to be reported
	}{
		{"v1 duplicate", `{"eastus":"a","eastus":"b"}`, []string{"eastus"}},
		{"v2 duplicate", `{"version":2,"locations":{"eastus":"a","eastus":"b"}}`, []string{"eastus"}},
		{"v2 envelope keys ignored", `{"version":2,"version":2,"locations":{"eastus":"a"}}`, nil},
		{"v2 locations missing", `{"version":2}`, nil},
		{"v2 locations null", `{"version":2,"locations":null}`, nil},
		{"v2 repeated locations merge", `{"version":2,"locations":{"eastus":"a","eastus":"b"},"locations":{"westus":"w"}}`, []string{"eastus"}},
		{"v2 repeated locations across objects", `{"version":2,"locations":{"eastus":"a"},"locations":{"eastus":"b"}}`, []string{"eastus"}},
		{"v2 null resets earlier objects", `{"version":2,"locations":{"eastus":"a"},"locations":null,"locations":{"eastus":"b"}}`, nil},
		{"v2 field name is case-insensitive", `{"version":2,"Locations":{"eastus":"a","eastus":"b"}}`, []string{"eastus"}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// The scan must accept everything loadLocations accepts.
			_, err := loadLocations([]byte(tc.doc))
			require.NoError(t, err)

			pairs, err := locationPairs([]byte(tc.doc))
			require.NoError(t, err)
			msgs := duplicatePairs(pairs, "location")

			require.Len(t, msgs, len(tc.dups))
			for i, k := range tc.dups {
				assert.Contains(t, msgs[i], `location "`+k+`"`)
			}
		})
	}
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
