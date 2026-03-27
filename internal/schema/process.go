// Copyright glueckkanja AG 2025, 2026
// SPDX-License-Identifier: MPL-2.0

package schema

import (
	"fmt"
	"io/fs"
	"path/filepath"
	"slices"
	"strings"
)

const (
	schemaNamingFileName   = "schema.naming.json"
	schemaLocationFileName = "schema.locations.json"
)

var supportedFileTypes = []string{".json"}

type Result struct {
	NamingSchemas []JsonNamingSchema
	Locations     LocationsMapSchema
}

type unmarshaler struct {
	d   []byte
	ext string
}

func newUnmarshaler(data []byte, ext string) unmarshaler {
	if !strings.HasPrefix(ext, ".") {
		ext = "." + ext
	}
	return unmarshaler{
		d:   data,
		ext: ext,
	}
}


type processFunc func(result *Result, data unmarshaler) error

// ProcessorClient is the client that is used to process the library files.
type ProcessorClient struct {
	fs fs.FS
}

func NewProcessorClient(fs fs.FS) *ProcessorClient {
	return &ProcessorClient{
		fs: fs,
	}
}

func (client *ProcessorClient) Process(res *Result) error {
	if err := fs.WalkDir(client.fs, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return fmt.Errorf("ProcessorClient.Process: error walking directory %s: %w", path, err)
		}
		// Skip directories
		if d.IsDir() {
			return nil
		}
		// Skip files that are not json or yaml
		if !slices.Contains(supportedFileTypes, strings.ToLower(filepath.Ext(path))) {
			return nil
		}
		file, err := client.fs.Open(path)
		if err != nil {
			return fmt.Errorf("ProcessorClient.Process: error opening file %s: %w", path, err)
		}
		return identifyFile(res, file, d.Name())
	}); err != nil {
		return err
	}
	return nil
}

func identifyFile(res *Result, file fs.File, name string) error {
	err := error(nil)

	switch n := strings.ToLower(name); {
	case schemaNamingFileName == n:
		err = readAndProcessFile(res, file, processNamingSchema)
	case schemaLocationFileName == n:
		err = readAndProcessFile(res, file, processLocationsMapSchema)
	}
	if err != nil {
		err = fmt.Errorf("classifyLibFile: error processing file: %w", err)
	}

	return err
}

func processNamingSchema(res *Result, unmar unmarshaler) error {
	schemas, err := loadNamingSchemas(unmar.d)
	if err != nil {
		return fmt.Errorf("processNamingSchema: %w", err)
	}
	res.NamingSchemas = schemas
	return nil
}

func processLocationsMapSchema(res *Result, unmar unmarshaler) error {
	lm, err := loadLocations(unmar.d)
	if err != nil {
		return fmt.Errorf("processLocationsMapSchema: %w", err)
	}
	res.Locations = lm
	return nil
}

func readAndProcessFile(res *Result, file fs.File, processFn processFunc) error {
	s, err := file.Stat()
	if err != nil {
		return err
	}
	data := make([]byte, s.Size())
	defer file.Close() // nolint: errcheck
	if _, err := file.Read(data); err != nil {
		return err
	}

	ext := filepath.Ext(s.Name())
	// create a new unmarshaler
	unmar := newUnmarshaler(data, ext)

	// pass the  data to the supplied process function
	if err := processFn(res, unmar); err != nil {
		return err
	}
	return nil
}
