#!/usr/bin/env bash
# Copyright glueckkanja AG 2025, 2026
# SPDX-License-Identifier: MPL-2.0
#
# Migrate a standesamt schema library directory from v1 to the v2 envelope, in place.
#
# Usage: scripts/migrate-schema-v1-to-v2.sh <path>
#   <path>  directory containing schema.naming.json and/or schema.locations.json
#
# Runs internal/cmd/standesamt-migrate-schema, which uses the provider's own parsing,
# so migrated files decode exactly like the originals. Requires Go.

set -euo pipefail

if ! command -v go >/dev/null 2>&1; then
  echo "ERROR: Go is required (https://go.dev/dl/)" >&2
  exit 1
fi

repo="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
bin="$(mktemp -d)"
trap 'rm -rf "$bin"' EXIT

# Build instead of `go run`, which hides the tool's exit code.
go -C "$repo" build -o "$bin/migrate" ./internal/cmd/standesamt-migrate-schema

rc=0
"$bin/migrate" "$@" || rc=$?
exit "$rc"
