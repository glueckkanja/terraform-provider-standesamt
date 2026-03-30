# terraform-provider-standesamt

Terraform/OpenTofu provider for generating resource names following naming conventions. No managed resources — only data sources and provider functions.

## Commands

```bash
make build       # Compile provider binary
make install     # Build + install to $GOPATH/bin
make test        # Unit tests (no infra needed)
make testacc     # Acceptance tests (requires Azure env vars + TF_ACC=1)
make lint        # golangci-lint
make fmt         # gofmt -s -w
make docs        # Generate docs via tfplugindocs
make generate    # cd tools; go generate ./... (regenerates provider code)
```

Default `make` runs: `fmt lint install generate`

## Architecture

```
internal/
  provider/   # All business logic: functions, data sources, name builder
  schema/     # Type definitions (JsonNamingSchema, NamingSchema, BuildNameSettingsModel)
  source/     # Schema library download via go-getter (default or custom URL)
  random/     # Hash generation for random name suffixes
  tools/      # String utilities
```

**Provider exposes:**
- Data sources: `standesamt_config`, `standesamt_locations`
- Functions: `provider::standesamt::name`, `provider::standesamt::validate`
- No managed resources

**Schema library** — downloaded at `Configure()` time via `go-getter`, cached by SHA224 hash of the source URL. Default: `github.com/glueckkanja/standesamt-schema-library` at path `azure/caf`, ref `2025.04`. Custom URL supported via `schema_reference.custom_url`.

## Environment Variables

Provider config can be set via env vars (only applied when the HCL attribute is null):

| Variable | Provider Attribute |
|---|---|
| `SA_ENVIRONMENT` | `environment` |
| `SA_CONVENTION` | `convention` (`default`\|`passthrough`) |
| `SA_SEPARATOR` | `separator` |
| `SA_RANDOM_SEED` | `random_seed` |
| `SA_HASH_LENGTH` | `hash_length` |
| `SA_LOWERCASE` | `lowercase` |

## Testing

**Unit tests** — no setup needed:
```bash
make test
```

**Acceptance tests** — require real Azure credentials:
```bash
export TF_ACC=1
export ARM_CLIENT_ID=...
export ARM_CLIENT_SECRET=...
export ARM_SUBSCRIPTION_ID=...
export ARM_TENANT_ID=...
make testacc
```

**Testing with OpenTofu** (instead of Terraform):
```bash
export TF_ACC_TERRAFORM_PATH="/path/to/opentofu"
export TF_ACC_PROVIDER_NAMESPACE="hashicorp"
export TF_ACC_PROVIDER_HOST="registry.opentofu.org"
```

## Local Development

Add to `~/.terraformrc` to use local binary instead of registry:
```hcl
provider_installation {
  dev_overrides {
    "glueckkanja/standesamt" = "/home/<user>/go/bin"
  }
  direct {}
}
```

## Docs Generation

Docs are auto-generated — do not edit `docs/` manually.

- Examples live in `examples/functions/<function-name>/function.tf`
- Guides go in `templates/guides/` — tfplugindocs copies them to `docs/guides/` on each run; **do not** place guides directly in `docs/guides/` as they will be deleted by the next `make generate`
- Every schema attribute/parameter needs a `MarkdownDescription` set

```bash
make docs  # or: go generate ./...
```

## Gotchas

- `settings` parameter in `name`/`validate` functions is `types.Dynamic` — HCL passes literal lists as tuples, not lists; `extractStringSlice` handles both
- `hash_length` in provider config overrides all per-schema configurations when set
- `convention = "passthrough"` bypasses all name building logic and returns the raw `name` argument unchanged
- Schema download happens once per provider configure; subsequent data source/function calls reuse `p.config`
- **Separator priority chain** (highest to lowest): per-call `settings.separator` > schema-level `separator` in JSON library (when `useSeparator=true` and non-empty) > provider-level `separator` (when `useSeparator=true`) > empty string (when `useSeparator=false`). The schema-level value flows via `NewNamingSchemaMap()` → `Configuration.Separator` → data source output → function parameter — no side channels needed.
- **Provider functions cannot access provider config in Terraform** — `Configure()` is not called before provider function `Run()` by design (terraform-plugin-framework#1093, closed won't-fix). Functions receive all needed data as explicit parameters. Do not attempt to pass a provider pointer to function structs.
- **Inline schema `configuration = {}` blocks require all attributes including `separator`** — always include `separator = ""` when writing inline schema objects (e.g. in tests). Users consuming data source output are unaffected.
