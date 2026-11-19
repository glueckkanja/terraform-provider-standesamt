// Copyright glueckkanja AG 2025, 2026
// SPDX-License-Identifier: MPL-2.0

package schema

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
)

// MigrateOutcome describes what MigrateV1ToV2 did with a file.
type MigrateOutcome int

const (
	MigrateMigrated MigrateOutcome = iota
	MigrateAlreadyV2
	MigrateNewerVersion
)

// MigrateV1ToV2 wraps a v1 schema document in the v2 envelope. The payload bytes are
// embedded unchanged, so the result decodes exactly like the input.
// file is schemaNamingFileName or schemaLocationFileName.
func MigrateV1ToV2(file string, data []byte, generatedAt string) ([]byte, MigrateOutcome, int, error) {
	var key string
	var load func([]byte) (any, error)
	switch file {
	case schemaNamingFileName:
		key = "resources"
		load = func(b []byte) (any, error) { return loadNamingSchemas(b) }
	case schemaLocationFileName:
		key = "locations"
		load = func(b []byte) (any, error) { return loadLocations(b) }
	default:
		return nil, 0, 0, fmt.Errorf("unknown schema file %q", file)
	}

	// json.Valid rejects empty input and multiple top-level values.
	if !json.Valid(data) {
		return nil, 0, 0, fmt.Errorf("not a single valid JSON document")
	}
	version, err := detectVersion(data)
	if err != nil {
		return nil, 0, 0, err
	}
	switch {
	case version == 2:
		return nil, MigrateAlreadyV2, version, nil
	case version > 2:
		return nil, MigrateNewerVersion, version, nil
	}

	want, err := load(data)
	if err != nil {
		return nil, 0, version, err
	}

	ts, err := json.Marshal(generatedAt)
	if err != nil {
		return nil, 0, version, err
	}
	var doc bytes.Buffer
	fmt.Fprintf(&doc, `{"version":2,"generatedAt":%s,%q:%s}`, ts, key, bytes.TrimSpace(data))

	// json.Indent only reformats: duplicate keys and number literals are kept.
	var out bytes.Buffer
	if err := json.Indent(&out, doc.Bytes(), "", "  "); err != nil {
		return nil, 0, version, err
	}
	out.WriteByte('\n')

	got, err := load(out.Bytes())
	if err != nil || !reflect.DeepEqual(want, got) {
		return nil, 0, version, fmt.Errorf("migrated document does not decode like the original (%v)", err)
	}
	return out.Bytes(), MigrateMigrated, version, nil
}

// MigrateDir migrates the schema files in dir in place and returns one summary line
// per file. It returns an error if dir has no schema files or any file failed.
func MigrateDir(dir, generatedAt string) ([]string, error) {
	var lines []string
	found, failed := false, false

	for _, name := range []string{schemaNamingFileName, schemaLocationFileName} {
		path := filepath.Join(dir, name)
		info, err := os.Stat(path)
		if os.IsNotExist(err) {
			continue
		}
		found = true
		if err == nil && !info.Mode().IsRegular() {
			err = fmt.Errorf("not a regular file")
		}
		if err == nil {
			var line string
			if line, err = migrateFile(path, name, info.Mode().Perm(), generatedAt); err == nil {
				lines = append(lines, line)
			}
		}
		if err != nil {
			lines = append(lines, fmt.Sprintf("ERROR:    %s (%v)", name, err))
			failed = true
		}
	}

	switch {
	case !found:
		return nil, fmt.Errorf("no %s or %s in %s", schemaNamingFileName, schemaLocationFileName, dir)
	case failed:
		return lines, fmt.Errorf("migration failed for at least one file; failed files were left unchanged")
	}
	return lines, nil
}

func migrateFile(path, name string, perm os.FileMode, generatedAt string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	out, outcome, version, err := MigrateV1ToV2(name, data, generatedAt)
	if err != nil {
		return "", err
	}
	switch outcome {
	case MigrateAlreadyV2:
		return fmt.Sprintf("Skipped:  %s (already v2)", name), nil
	case MigrateNewerVersion:
		return fmt.Sprintf("Skipped:  %s (schema version %d is newer than v2)", name, version), nil
	}

	// Write a temp file next to the original and rename it, so a failure never leaves a partial file.
	tmp, err := os.CreateTemp(filepath.Dir(path), name+".*")
	if err != nil {
		return "", err
	}
	defer os.Remove(tmp.Name()) // no-op after a successful rename
	if _, err := tmp.Write(out); err != nil {
		tmp.Close()
		return "", err
	}
	if err := tmp.Close(); err != nil {
		return "", err
	}
	if err := os.Chmod(tmp.Name(), perm); err != nil {
		return "", err
	}
	if err := os.Rename(tmp.Name(), path); err != nil {
		return "", err
	}
	return fmt.Sprintf("Migrated: %s (v1 → v2)", name), nil
}
