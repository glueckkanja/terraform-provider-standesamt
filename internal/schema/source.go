// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package schema

import (
	"context"
	"fmt"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"io/fs"
)

type SourceValue struct {
	Path      basetypes.StringValue `tfsdk:"path"`
	Ref       basetypes.StringValue `tfsdk:"ref"`
	CustomUrl basetypes.StringValue `tfsdk:"custom_url"`
}

type Source interface {
	fmt.Stringer
	Download(ctx context.Context, destinationDirectory string) (fs.FS, error)
	Dst() fs.FS
}

type DefaultSource struct {
	path string
	ref  string
	dst  fs.FS
}

func NewDefaultSource(path, ref string) *DefaultSource {
	return &DefaultSource{
		path: path,
		ref:  ref,
	}
}

func (r *DefaultSource) Download(ctx context.Context, destinationDirectory string) (fs.FS, error) {
	f, err := DownloadFromDefaultSource(ctx, r.path, r.ref, destinationDirectory)
	if err != nil {
		return nil, err
	}
	r.dst = f
	return f, nil
}

func (r *DefaultSource) String() string {
	return fmt.Sprintf("%s@%s", r.path, r.ref)
}

func (r *DefaultSource) Path() string {
	return r.path
}

func (r *DefaultSource) Ref() string {
	return r.ref
}

func (r *DefaultSource) Dst() fs.FS {
	return r.dst
}

type CustomSource struct {
	url string
	dst fs.FS
}

func NewCustomSource(url string) *CustomSource {
	return &CustomSource{
		url: url,
	}
}

func (r *CustomSource) Download(ctx context.Context, destinationDirectory string) (fs.FS, error) {
	f, err := DownloadFromCustomSource(ctx, r.url, destinationDirectory)
	if err != nil {
		return nil, err
	}
	r.dst = f
	return f, nil
}

func (r *CustomSource) String() string {
	return r.url
}

func (r *CustomSource) Url() string {
	return r.url
}

func (r *CustomSource) Dst() fs.FS {
	return r.dst
}
