// Copyright glueckkanja AG 2025, 2026
// SPDX-License-Identifier: MPL-2.0

package schema

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testTS = "2026-01-01T00:00:00Z"

func TestMigrateV1ToV2_DecodesLikeOriginal(t *testing.T) {
	tests := []struct {
		name, file, doc string
	}{
		{"naming", schemaNamingFileName, `[
			{ "resourceType": "azurerm_resource_group", "abbreviation": "rg", "maxLength": 90,
			  "configuration": { "useSeparator": true, "namePrecedence": ["abbreviation", "name"], "hashLength": 0 } },
			{ "resourceType": "azurerm_resource_group", "abbreviation": "rsg" }
		]`},
		// encoding/json merges repeated object fields; re-serialising would drop useSeparator.
		{"naming repeated configuration", schemaNamingFileName,
			`[{"resourceType":"x","configuration":{"useSeparator":true},"configuration":{"hashLength":4}}]`},
		{"locations", schemaLocationFileName, `{"westeurope":"weu","eastus":"eus"}`},
		{"locations duplicate key kept", schemaLocationFileName, `{"eastus":"a","eastus":"b"}`},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			out, outcome, _, err := MigrateV1ToV2(tc.file, []byte(tc.doc), testTS)
			require.NoError(t, err)
			assert.Equal(t, MigrateMigrated, outcome)

			v, err := detectVersion(out)
			require.NoError(t, err)
			assert.Equal(t, 2, v)

			if tc.file == schemaNamingFileName {
				want, _ := loadNamingSchemas([]byte(tc.doc))
				got, err := loadNamingSchemas(out)
				require.NoError(t, err)
				assert.Equal(t, want, got)
			} else {
				want, _ := loadLocations([]byte(tc.doc))
				got, err := loadLocations(out)
				require.NoError(t, err)
				assert.Equal(t, want, got)
			}
		})
	}
}

func TestMigrateV1ToV2_RepeatedConfigurationPreserved(t *testing.T) {
	out, _, _, err := MigrateV1ToV2(schemaNamingFileName,
		[]byte(`[{"resourceType":"x","configuration":{"useSeparator":true},"configuration":{"hashLength":4}}]`), testTS)
	require.NoError(t, err)
	got, err := loadNamingSchemas(out)
	require.NoError(t, err)
	assert.True(t, got[0].Configuration.UseSeparator)
	assert.Equal(t, 4, got[0].Configuration.HashLength)
}

func TestMigrateV1ToV2_Skips(t *testing.T) {
	tests := []struct {
		name, doc string
		outcome   MigrateOutcome
	}{
		{"v2", `{"version":2,"locations":{}}`, MigrateAlreadyV2},
		// encoding/json matches field names case-insensitively, like detectVersion.
		{"v2 capitalised Version", `{"Version":2,"locations":{"eastus":"eus"}}`, MigrateAlreadyV2},
		{"v3", `{"version":3}`, MigrateNewerVersion},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			out, outcome, _, err := MigrateV1ToV2(schemaLocationFileName, []byte(tc.doc), testTS)
			require.NoError(t, err)
			assert.Equal(t, tc.outcome, outcome)
			assert.Nil(t, out)
		})
	}
}

func TestMigrateV1ToV2_Errors(t *testing.T) {
	tests := []struct {
		name, file, doc string
	}{
		{"empty", schemaLocationFileName, ``},
		{"whitespace only", schemaLocationFileName, " \n\t"},
		{"two values", schemaNamingFileName, `[] []`},
		{"invalid json", schemaLocationFileName, `{"eastus": `},
		{"string version", schemaLocationFileName, `{"version":"2","locations":{}}`},
		{"array as locations", schemaLocationFileName, `["eastus"]`},
		{"object as naming", schemaNamingFileName, `{"eastus":"eus"}`},
		{"unknown file", "other.json", `[]`},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, _, _, err := MigrateV1ToV2(tc.file, []byte(tc.doc), testTS)
			assert.Error(t, err)
		})
	}
}

func TestMigrateDir(t *testing.T) {
	dir := t.TempDir()
	naming := filepath.Join(dir, schemaNamingFileName)
	locations := filepath.Join(dir, schemaLocationFileName)
	require.NoError(t, os.WriteFile(naming, []byte(`[{"resourceType":"x"}]`), 0o640))
	require.NoError(t, os.WriteFile(locations, []byte(`{"eastus":"eus"}`), 0o644))

	lines, err := MigrateDir(dir, testTS)
	require.NoError(t, err)
	assert.Equal(t, []string{
		"Migrated: schema.naming.json (v1 → v2)",
		"Migrated: schema.locations.json (v1 → v2)",
	}, lines)

	info, err := os.Stat(naming)
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0o640), info.Mode().Perm(), "file mode kept")

	first, err := os.ReadFile(naming)
	require.NoError(t, err)

	// A second run skips both files and leaves them byte-identical.
	lines, err = MigrateDir(dir, "2027-01-01T00:00:00Z")
	require.NoError(t, err)
	assert.Equal(t, []string{
		"Skipped:  schema.naming.json (already v2)",
		"Skipped:  schema.locations.json (already v2)",
	}, lines)
	again, err := os.ReadFile(naming)
	require.NoError(t, err)
	assert.Equal(t, first, again)
}

func TestMigrateDir_InvalidFileUnchanged(t *testing.T) {
	dir := t.TempDir()
	naming := filepath.Join(dir, schemaNamingFileName)
	locations := filepath.Join(dir, schemaLocationFileName)
	require.NoError(t, os.WriteFile(naming, []byte(`[{"resourceType":"x"}]`), 0o644))
	require.NoError(t, os.WriteFile(locations, []byte(`{"eastus": `), 0o644))

	lines, err := MigrateDir(dir, testTS)
	require.Error(t, err)
	require.Len(t, lines, 2)
	assert.Contains(t, lines[0], "Migrated: schema.naming.json")
	assert.Contains(t, lines[1], "ERROR:    schema.locations.json")

	got, err := os.ReadFile(locations)
	require.NoError(t, err)
	assert.Equal(t, `{"eastus": `, string(got))

	entries, err := os.ReadDir(dir)
	require.NoError(t, err)
	assert.Len(t, entries, 2, "no temp files left behind")
}

func TestMigrateDir_NoSchemaFiles(t *testing.T) {
	_, err := MigrateDir(t.TempDir(), testTS)
	assert.ErrorContains(t, err, "no schema.naming.json or schema.locations.json")
}
