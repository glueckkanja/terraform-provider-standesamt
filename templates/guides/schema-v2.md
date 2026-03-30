---
page_title: "Schema v2 Format"
subcategory: "Schema Library"
description: |-
  Guide to the v2 versioned envelope format for standesamt schema files and how to migrate from v1.
---

# Schema v2 Format

This guide covers the v2 schema file format introduced alongside provider v2 support.
It is primarily aimed at **schema library authors** — teams maintaining a `schema.naming.json`
and/or `schema.locations.json` file that is consumed by this provider.

## Background

The original (v1) schema format is a bare JSON array with no version marker:

```json
[
  {
    "resourceType": "azurerm_resource_group",
    "abbreviation": "rg",
    ...
  }
]
```

While this is simple to read, it provides no way for the provider to detect breaking changes
or new fields in a forward-compatible way. The v2 format wraps the payload in a versioned
envelope so that the provider can adapt its behaviour based on the schema version it receives.

## v2 File Format

### `schema.naming.json`

The resource array is moved into a `resources` key inside an envelope object:

```json
{
  "version": 2,
  "generatedAt": "2026-04-01T00:00:00Z",
  "resources": [
    {
      "resourceType": "azurerm_resource_group",
      "abbreviation": "rg",
      "minLength": 1,
      "maxLength": 90,
      "validationRegex": "^[a-zA-Z0-9-._()]{1,90}$",
      "configuration": {
        "useEnvironment": true,
        "useLowerCase": false,
        "useUpperCase": false,
        "useSeparator": true,
        "separator": "-",
        "denyDoubleHyphens": true,
        "namePrecedence": [
          "abbreviation",
          "prefixes",
          "name",
          "location",
          "environment",
          "hash",
          "suffixes"
        ],
        "hashLength": 0
      },
      "deprecated": false,
      "deprecatedBy": "",
      "tags": ["core", "infrastructure"]
    }
  ]
}
```

#### Envelope fields

| Field | Type | Required | Description |
|---|---|---|---|
| `version` | integer | yes | Schema format version. Must be `2` for v2 files. |
| `generatedAt` | string | no | ISO-8601 timestamp of when the file was generated. Informational only. |
| `resources` | array | yes | The full list of resource naming schemas (same structure as v1 array entries). |

#### New per-resource fields (v2+)

These fields are optional and may be omitted. When absent they default to their zero values.

| Field | Type | Default | Description |
|---|---|---|---|
| `deprecated` | boolean | `false` | Marks the resource type as deprecated. |
| `deprecatedBy` | string | `""` | The resource type that replaces this one, e.g. `"azurerm_resource_group_v2"`. |
| `tags` | string array | `[]` | Free-form category tags, e.g. `["core", "networking"]`. |

~> **Note on `separator`:** Only include `separator` inside `configuration` when the resource
requires a specific value (e.g. `"-"`). An explicit empty string and an omitted field are
semantically identical — both fall through to the provider-level separator. Omitting it keeps
the file concise.

### `schema.locations.json`

The flat location map is moved into a `locations` key inside the same envelope structure:

```json
{
  "version": 2,
  "generatedAt": "2026-04-01T00:00:00Z",
  "locations": {
    "eastus": "eus",
    "uksouth": "uks",
    "westeurope": "weu"
  }
}
```

#### Envelope fields

| Field | Type | Required | Description |
|---|---|---|---|
| `version` | integer | yes | Schema format version. Must be `2` for v2 files. |
| `generatedAt` | string | no | ISO-8601 timestamp of when the file was generated. Informational only. |
| `locations` | object | yes | Map of Azure region names to their short abbreviations. |

## Version Detection

The provider detects the schema version automatically — no provider configuration change is needed.

| File starts with | Detected version |
|---|---|
| `[` | v1 (legacy raw array) |
| `{` without a `version` field | v1 (treated as legacy) |
| `{"version": 2, ...}` | v2 |

Schema versions beyond the provider's maximum supported version produce a clear error:

```
schema version 3 is not supported by this provider (max supported: 2); upgrade the provider
```

## Provider Compatibility Matrix

| Schema format | Provider < v2 | Provider v2+ |
|---|---|---|
| v1 (raw array) | supported | supported (frozen, behaviour unchanged) |
| v2 (versioned envelope) | parse error | supported |
| v3+ (future) | parse error | explicit error message |

## Migrating from v1 to v2

A helper script is provided in the schema library repository to automate the conversion:

```shell
scripts/migrate-schema-v1-to-v2.sh azure/caf
```

See [Migration Script](#migration-script) below for details.

### Manual migration steps

1. Open `schema.naming.json`.
2. Wrap the existing array in the envelope object:
   ```json
   {
     "version": 2,
     "generatedAt": "<ISO-8601 timestamp>",
     "resources": <paste existing array here>
   }
   ```
3. Optionally add `"deprecated"`, `"deprecatedBy"`, and `"tags"` to individual resource entries.
4. Repeat for `schema.locations.json`, wrapping the flat map in `"locations": { ... }`.
5. Bump the schema ref tag (e.g. `2026.01`) and update `ref` in your provider configuration.

## Migration Script

The script `scripts/migrate-schema-v1-to-v2.sh` automates the conversion of both schema files
in a given library path. It is a **placeholder** — see the script file for the full implementation
plan and expected behaviour.

```shell
# Usage
scripts/migrate-schema-v1-to-v2.sh <path>

# Example — migrate the azure/caf library
scripts/migrate-schema-v1-to-v2.sh azure/caf
```

The script will:

1. Read `<path>/schema.naming.json` and `<path>/schema.locations.json`
2. Detect whether each file is already v2 (and skip if so)
3. Wrap v1 files in the v2 envelope, adding `"version": 2` and `"generatedAt"`
4. Write the result back in-place (pretty-printed, 2-space indentation)
5. Print a summary of what was changed
