// Copyright glueckkanja AG 2025, 2026
// SPDX-License-Identifier: MPL-2.0

// Command standesamt-migrate-schema migrates a schema library directory from v1 to v2.
// Usually run through scripts/migrate-schema-v1-to-v2.sh.
package main

import (
	"fmt"
	"os"
	"time"

	"terraform-provider-standesamt/internal/schema"
)

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintln(os.Stderr, "Usage: migrate-schema-v1-to-v2.sh <path>")
		fmt.Fprintln(os.Stderr, "  <path>  directory containing schema.naming.json and/or schema.locations.json")
		os.Exit(2)
	}
	dir := os.Args[1]
	if info, err := os.Stat(dir); err != nil || !info.IsDir() {
		fmt.Fprintf(os.Stderr, "ERROR: %s is not a directory\n", dir)
		os.Exit(1)
	}

	lines, err := schema.MigrateDir(dir, time.Now().UTC().Format(time.RFC3339))
	for _, l := range lines {
		fmt.Println(l)
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: %v\n", err)
		os.Exit(1)
	}
}
