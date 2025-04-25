// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package tools

import "os"

const (
	standesamtSchemaDefaultCacheDir    = ".standesamt"
	standesamtSchemaDefaultCacheDirEnv = "SA_NAMING_DIR"
	standesamtSchemaGitUrl             = "github.com/c4a8-azure/Standesamt-Schema-Library"
	standesamtSchemaGitUrlEnv          = "SA_NAMING_GIT_URL"
)

func NamingSchemaCacheDir() string {
	dir := standesamtSchemaDefaultCacheDir
	if d := os.Getenv(standesamtSchemaDefaultCacheDirEnv); d != "" {
		dir = d
	}
	return dir
}

func NamingSchemaGitUrl() string {
	url := standesamtSchemaGitUrl
	if u := os.Getenv(standesamtSchemaGitUrlEnv); u != "" {
		url = u
	}
	return url
}
