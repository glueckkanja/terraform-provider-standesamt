#!/usr/bin/env bash
# Copyright glueckkanja AG 2025, 2026
# SPDX-License-Identifier: MPL-2.0
#
# migrate-schema-v1-to-v2.sh
# ─────────────────────────────────────────────────────────────────────────────
# PLACEHOLDER — this script is not yet implemented.
#
# PURPOSE
#   Migrate standesamt schema library files from the v1 format (raw JSON array /
#   flat map) to the v2 versioned envelope format in-place.
#
# USAGE
#   scripts/migrate-schema-v1-to-v2.sh <path>
#
#   <path>  Path to the schema library directory that contains
#           schema.naming.json and/or schema.locations.json.
#           Example: azure/caf
#
# WHAT IT SHOULD DO (when implemented)
#   1. Validate that <path> exists and contains at least one schema file.
#   2. For schema.naming.json:
#      a. Peek at the first non-whitespace byte.
#      b. If '[', the file is v1 — wrap it:
#           {
#             "version": 2,
#             "generatedAt": "<current UTC timestamp in ISO-8601>",
#             "resources": <existing array>
#           }
#      c. If '{' and "version" >= 2, print "already v2, skipping" and continue.
#      d. Write the result back in-place, pretty-printed with 2-space indentation.
#   3. For schema.locations.json:
#      a. Peek at the first non-whitespace byte.
#      b. If '{' without a "version" field (or version == 0), wrap it:
#           {
#             "version": 2,
#             "generatedAt": "<current UTC timestamp in ISO-8601>",
#             "locations": <existing object>
#           }
#      c. If "version" >= 2, print "already v2, skipping" and continue.
#      d. Write the result back in-place, pretty-printed with 2-space indentation.
#   4. Print a summary:
#        Migrated: schema.naming.json   (v1 → v2)
#        Skipped:  schema.locations.json (already v2)
#
# DEPENDENCIES (suggested)
#   - jq (https://jqlang.org) for JSON manipulation
#   - bash >= 4.0
#
# EXAMPLE IMPLEMENTATION SKETCH (jq-based)
#
#   NAMING="$1/schema.naming.json"
#   GENERATED_AT=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
#
#   first_byte=$(python3 -c "
#   import json, sys
#   with open('$NAMING') as f:
#       data = f.read().lstrip()
#   print(data[0])
#   ")
#
#   if [ "$first_byte" = "[" ]; then
#     jq --arg ts "$GENERATED_AT" \
#       '{version: 2, generatedAt: $ts, resources: .}' \
#       "$NAMING" > "${NAMING}.tmp" && mv "${NAMING}.tmp" "$NAMING"
#     echo "Migrated: $NAMING (v1 → v2)"
#   else
#     echo "Skipped:  $NAMING (already v2 or unknown format)"
#   fi
#
# ─────────────────────────────────────────────────────────────────────────────

set -euo pipefail

echo "ERROR: migrate-schema-v1-to-v2.sh is not yet implemented." >&2
echo "       See the comments in this file for the intended behaviour." >&2
exit 1
