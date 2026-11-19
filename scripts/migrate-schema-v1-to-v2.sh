#!/usr/bin/env bash
# Copyright glueckkanja AG 2025, 2026
# SPDX-License-Identifier: MPL-2.0
#
# Migrate a standesamt schema library directory from v1 to the v2 envelope, in place.
#
# Usage: scripts/migrate-schema-v1-to-v2.sh <path>
#   <path>  directory containing schema.naming.json and/or schema.locations.json
#
# Entries are left untouched, so generated names do not change. Files that are
# already v2 are skipped; v3+ files are skipped with a warning; invalid files are
# reported and left unchanged. Requires jq.

set -euo pipefail

usage() {
  echo "Usage: $0 <path>" >&2
  echo "  <path>  directory containing schema.naming.json and/or schema.locations.json" >&2
}

if [ "$#" -ne 1 ]; then
  usage
  exit 2
fi

dir="$1"
if [ ! -d "$dir" ]; then
  echo "ERROR: $dir is not a directory" >&2
  exit 1
fi

if ! command -v jq >/dev/null 2>&1; then
  echo "ERROR: jq is required (https://jqlang.org)" >&2
  exit 1
fi

generated_at=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
found=0
failed=0

# migrate_file <file> <payload key> <expected payload type: array|object>
migrate_file() {
  local file="$1" key="$2" type="$3" name
  name=$(basename "$file")

  [ -f "$file" ] || return 0
  found=1

  # Mirrors detectVersion: array → 1, object without version (or 0) → 1.
  local version
  if ! version=$(jq -r 'if type == "array" then 1
                        elif type == "object" then (.version // 0 | if . == 0 then 1 else . end)
                        else error("unexpected top-level \(type)") end' "$file" 2>&1); then
    echo "ERROR:    $name ($version)" >&2
    failed=1
    return 0
  fi

  case "$version" in
    1) ;;
    2)
      echo "Skipped:  $name (already v2)"
      return 0
      ;;
    *)
      echo "WARNING:  $name has schema version $version; skipping" >&2
      return 0
      ;;
  esac

  if [ "$(jq -r 'type' "$file")" != "$type" ]; then
    echo "ERROR:    $name (expected a JSON $type for v1)" >&2
    failed=1
    return 0
  fi

  local tmp
  tmp=$(mktemp "$file.XXXXXX")
  cp -p "$file" "$tmp" # keep the original file mode after mv
  if jq --arg ts "$generated_at" --arg key "$key" \
    '{version: 2, generatedAt: $ts} + {($key): .}' "$file" >"$tmp"; then
    mv "$tmp" "$file"
    echo "Migrated: $name (v1 → v2)"
  else
    rm -f "$tmp"
    echo "ERROR:    $name (jq failed)" >&2
    failed=1
  fi
}

migrate_file "$dir/schema.naming.json" resources array
migrate_file "$dir/schema.locations.json" locations object

if [ "$found" -eq 0 ]; then
  echo "ERROR: no schema.naming.json or schema.locations.json in $dir" >&2
  exit 1
fi

exit "$failed"
