// Copyright glueckkanja AG 2025, 2026
// SPDX-License-Identifier: MPL-2.0

package schema

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const migrateScript = "../../scripts/migrate-schema-v1-to-v2.sh"

const v1NamingFixture = `[
  { "resourceType": "azurerm_resource_group", "abbreviation": "rg", "minLength": 1, "maxLength": 90,
    "validationRegex": "^[a-zA-Z0-9-._()]{0,89}[a-zA-Z0-9-_()]$",
    "configuration": { "useSeparator": true, "denyDoubleHyphens": false, "namePrecedence": ["abbreviation", "name", "environment"], "hashLength": 0, "useLowerCase": false, "useEnvironment": true } },
  { "resourceType": "azurerm_storage_account", "abbreviation": "stcustom", "maxLength": 24,
    "configuration": { "useSeparator": false, "namePrecedence": ["abbreviation", "name", "hash"], "hashLength": 4, "useLowerCase": true } },
  { "resourceType": "azurerm_resource_group", "abbreviation": "rsg", "maxLength": 90 }
]`

const v1LocationsFixture = `{"westeurope": "weu", "eastus": "eus", "germanywestcentral": "gwc"}`

func runMigrate(t *testing.T, dir string) (string, error) {
	t.Helper()
	out, err := exec.Command("bash", migrateScript, dir).CombinedOutput()
	return string(out), err
}

func requireMigrateTools(t *testing.T) {
	t.Helper()
	for _, tool := range []string{"bash", "jq"} {
		if _, err := exec.LookPath(tool); err != nil {
			t.Skipf("%s not available: %v", tool, err)
		}
	}
}

func TestMigrateScript_BehaviourNeutral(t *testing.T) {
	requireMigrateTools(t)

	dir := t.TempDir()
	namingPath := filepath.Join(dir, schemaNamingFileName)
	locationsPath := filepath.Join(dir, schemaLocationFileName)
	require.NoError(t, os.WriteFile(namingPath, []byte(v1NamingFixture), 0o644))
	require.NoError(t, os.WriteFile(locationsPath, []byte(v1LocationsFixture), 0o644))

	wantNaming, err := loadNamingSchemas([]byte(v1NamingFixture))
	require.NoError(t, err)
	wantLocations, err := loadLocations([]byte(v1LocationsFixture))
	require.NoError(t, err)

	out, err := runMigrate(t, dir)
	require.NoError(t, err, out)
	assert.Contains(t, out, "Migrated: "+schemaNamingFileName)
	assert.Contains(t, out, "Migrated: "+schemaLocationFileName)

	naming, err := os.ReadFile(namingPath)
	require.NoError(t, err)
	locations, err := os.ReadFile(locationsPath)
	require.NoError(t, err)

	v, err := detectVersion(naming)
	require.NoError(t, err)
	assert.Equal(t, 2, v)
	v, err = detectVersion(locations)
	require.NoError(t, err)
	assert.Equal(t, 2, v)

	gotNaming, err := loadNamingSchemas(naming)
	require.NoError(t, err)
	gotLocations, err := loadLocations(locations)
	require.NoError(t, err)
	assert.Equal(t, wantNaming, gotNaming)
	assert.Equal(t, wantLocations, gotLocations)

	// A second run skips both files and leaves them byte-identical.
	out, err = runMigrate(t, dir)
	require.NoError(t, err, out)
	assert.Contains(t, out, "Skipped:  "+schemaNamingFileName)
	assert.Contains(t, out, "Skipped:  "+schemaLocationFileName)
	again, err := os.ReadFile(namingPath)
	require.NoError(t, err)
	assert.Equal(t, naming, again)
}

func TestMigrateScript_InvalidFileUnchanged(t *testing.T) {
	requireMigrateTools(t)

	dir := t.TempDir()
	path := filepath.Join(dir, schemaLocationFileName)
	require.NoError(t, os.WriteFile(path, []byte(`{"eastus": `), 0o644))

	out, err := runMigrate(t, dir)
	require.Error(t, err, out)

	got, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Equal(t, `{"eastus": `, string(got))

	entries, err := os.ReadDir(dir)
	require.NoError(t, err)
	assert.Len(t, entries, 1, "no temp files left behind")
}

func TestMigrateScript_NoSchemaFiles(t *testing.T) {
	requireMigrateTools(t)

	out, err := runMigrate(t, t.TempDir())
	require.Error(t, err, out)
	assert.Contains(t, out, "no schema.naming.json or schema.locations.json")
}
