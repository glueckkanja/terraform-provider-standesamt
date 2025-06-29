---
# generated by https://github.com/hashicorp/terraform-plugin-docs
page_title: "standesamt Provider"
subcategory: ""
description: |-
  
---

# standesamt Provider





<!-- schema generated by tfplugindocs -->
## Schema

### Optional

- `convention` (String) Define the convention for naming results. Possible values are 'default' and 'passthrough'. Default 'default'
- `environment` (String) Define the environment for the naming schema. Normally this is the name of the environment, e.g. 'prod', 'dev', 'test'.
- `hash_length` (Number) Default hash length. Overrides all schema configurations.
- `lowercase` (Boolean) Control if the resulting name should be lower case. Default 'false'
- `random_seed` (Number) A random seed used by the random number generator. This is used to generate a random name for the naming schema. The default value is 1337. Make sure to update this value to avoid collisions for globally unique names.
- `schema_reference` (Attributes) A reference to a Naming schema library to use. The reference should either contain a `path` (e.g. `azure/caf`) and the `ref` (e.g. `2025.04`), or a `custom_url` to be supplied to go-getter.
    If this value is not specified, the default value will be used, which is:

    ```terraform

    schema_reference = {
      path = "azure/caf",
      ref = "2025.04"
    }

    ```

    The reference is using the [default standesamt library](https://github.com/glueckkanja/standesamt-schema-library). (see [below for nested schema](#nestedatt--schema_reference))
- `separator` (String) The separator to use for generating the resulting name. Default '-'

<a id="nestedatt--schema_reference"></a>
### Nested Schema for `schema_reference`

Optional:

- `custom_url` (String, Sensitive) A custom path/URL to the schema reference to use. Conflicts with `path` and `ref`. For supported protocols, see [go-getter](https://pkg.go.dev/github.com/hashicorp/go-getter/v2). Value is marked sensitive as may contain secrets.
- `path` (String) The path in the default schema library, e.g. `azure/caf`. Also requires `ref`. Conflicts with `custom_url`.
- `ref` (String) This is the version of the schema reference to use, e.g. `2025.04`. Also requires `path`. Conflicts with `custom_url`.
