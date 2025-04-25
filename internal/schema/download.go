// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package schema

import (
	"context"
	"fmt"
	"github.com/hashicorp/go-getter/v2"
	"io/fs"
	"net/url"
	"os"
	"path/filepath"
	"terraform-provider-standesamt/internal/tools"
)

func DownloadFromDefaultSource(ctx context.Context, path, ref, dstDir string) (fs.FS, error) {
	q := url.Values{}
	q.Add("ref", ref)

	gitUrl := tools.NamingSchemaGitUrl()

	u := fmt.Sprintf("git::%s//%s?%s", gitUrl, path, q.Encode())
	return DownloadFromCustomSource(ctx, u, dstDir)
}

func DownloadFromCustomSource(ctx context.Context, src, dstDir string) (fs.FS, error) {
	rootDir := tools.NamingSchemaCacheDir()
	dst := filepath.Join(rootDir, dstDir)
	client := getter.Client{
		DisableSymlinks: true,
	}

	wd, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("error getting working directory: %w", err)
	}
	if err := os.RemoveAll(dst); err != nil {
		return nil, fmt.Errorf("error cleaning destination directory %s: %w", dst, err)
	}

	req := &getter.Request{
		Src: src,
		Dst: dst,
		Pwd: wd,
	}

	_, err = client.Get(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("error downloading schema. source `%s`, destination `%s`, wd `%s`: %w", src, dst, wd, err)
	}

	return os.DirFS(dst), nil
}
