# Module Type Schema (module\_types.json)

Soong can generate a JSON file describing all registered module types and their
properties. This is intended for use with LSPs and other tooling that needs
structured knowledge of Android.bp file syntax.

## Generating

```bash
$ m soong_module_type_schema
```

The output is written to `$OUT_DIR/soong/docs/module_types.json`.

## Schema Overview

The file has the following top-level structure:

```json
{
  "version": 1,
  "arch": { ... },
  "property_groups": { ... },
  "module_types": { ... }
}
```

### `version`

Integer schema version. Consumers should check this and warn on unknown
versions.

### `arch`

Static listing of valid keys for `arch`, `multilib`, and `target` blocks in
Android.bp files. Only properties with `tags.arch_variant == true` can appear
inside these blocks.

```json
{
  "arch": {
    "arm": {
      "arch_variants": ["armv7-a-neon", "armv8-a", ...],
      "cpu_variants": ["cortex-a7", "cortex-a53", ...],
      "features": ["soft_ceil_floor"]
    },
    "arm64": { ... },
    "x86": { ... },
    "x86_64": { ... },
    "riscv64": { ... }
  },
  "multilib": ["lib32", "lib64"],
  "target": ["host", "android", "linux_glibc", "android_arm64", ...]
}
```

Architecture names are the first-level keys inside `arch { }` blocks.
`arch_variants`, `cpu_variants`, and `features` are second-level keys
(e.g., `arch { arm { cortex_a53 { ... } } }`).

`multilib` keys are the first-level keys inside `multilib { }` blocks.

`target` keys are the first-level keys inside `target { }` blocks,
including OS names, OS+arch combinations, and special targets like `host`,
`bionic`, `linux`, etc.

### `property_groups`

Shared property definitions used by multiple module types. Each group
corresponds to a Go property struct in the build system. This avoids
repeating common properties (like `enabled`, `visibility`, `srcs`) on every
module type.

```json
{
  "BaseCompilerProperties": {
    "documentation": "...",
    "properties": {
      "srcs": {
        "type": "string[]",
        "configurable": true,
        "documentation": "list of source files...",
        "tags": {
          "path": true,
          "arch_variant": true,
          "product_variables": ["debuggable", "malloc_low_memory", ...]
        }
      },
      "cflags": { ... },
      ...
    }
  }
}
```

### `module_types`

All registered module types. Each entry has a documentation string, the Go
package that defines it, a list of property groups it includes, and any
module-type-specific properties not covered by shared groups.

```json
{
  "cc_library": {
    "documentation": "cc_library creates both static and/or shared libraries...",
    "package": "android/soong/cc",
    "groups": [
      "BaseProperties",
      "VendorProperties",
      "BaseCompilerProperties",
      ...
    ],
    "properties": {
      "buildstubs": { "type": "bool" }
    }
  }
}
```

To get the full set of properties for a module type, merge all properties
from the referenced groups with the module type's own `properties`.
Per-module-type properties take precedence over group properties.

## Property Fields

Each property has the following fields:

| Field | Type | Description |
|---|---|---|
| `type` | string | **Required.** One of the type tokens below. |
| `configurable` | bool | If true, the property also accepts `select()` expressions. |
| `documentation` | string | Plain-text description from Go source comments. |
| `default` | string | Default value, if one is set. |
| `required` | bool | If true, the property must be specified (only `name`). |
| `tags` | object | Additional metadata (see below). |
| `properties` | object | For `struct` and `struct[]` types: nested property definitions. For `struct[]`, the properties describe each element of the list. |

### Type Tokens

| Token | Blueprint syntax | Go type |
|---|---|---|
| `string` | `"value"` | `string` / `*string` |
| `bool` | `true` / `false` | `bool` / `*bool` |
| `int` | `42` | `int64` / `*int64` |
| `string[]` | `["a", "b"]` | `[]string` |
| `bool[]` | `[true, false]` | `[]bool` |
| `int[]` | `[1, 2]` | `[]int64` |
| `struct` | `{ key: value }` | nested struct |
| `struct[]` | `[{...}, ...]` | slice of struct |
| `any` | varies | `interface{}` |

### Configurable Properties

When `configurable` is true, the property accepts either a literal value or a
`select()` expression:

```
srcs: select(arch(), {
    "arm": ["arm.c"],
    "x86": ["x86.c"],
    default: ["generic.c"],
}),
```

Available select conditions include `arch()`, `os()`,
`soong_config_variable(namespace, variable)`, and `release_flag(flag)`.

### Tags

| Tag | Type | Description |
|---|---|---|
| `path` | bool | Property contains file paths. Useful for path completion. |
| `arch_variant` | bool | Property can appear inside `arch`, `multilib`, and `target` blocks. |
| `product_variables` | string[] | Product variable names under which this property can be conditionally set. |

#### `arch_variant`

When true, the property is valid both at the top level and inside
architecture-specific blocks:

```
cc_library {
    srcs: ["common.c"],
    arch: {
        arm: { srcs: ["arm.c"] },
    },
    target: {
        android: { srcs: ["android.c"] },
    },
}
```

The valid block keys are listed in the top-level `arch` section.

#### `product_variables`

Lists which product variables support conditionally setting this property.
In Android.bp syntax:

```
cc_library {
    product_variables: {
        debuggable: {
            cflags: ["-DDEBUG"],
        },
    },
}
```

A property is valid inside `product_variables { X { ... } }` if its
`tags.product_variables` list includes `"X"`.

## Resolving the Full Property Set

To determine all valid properties for a module type:

1. Start with the module type's `properties` map.
2. For each group name in `groups`, merge in `property_groups[name].properties`.
3. The union of all these properties is the full set.

To determine valid completions inside `arch { arm { ... } }`:
- Filter to properties where `tags.arch_variant == true`.

To determine valid completions inside `product_variables { debuggable { ... } }`:
- Filter to properties where `tags.product_variables` includes `"debuggable"`.

## Future Work

### Resolving `any` type properties

A small number of properties have type `any`, meaning the schema cannot
describe their valid values. These arise from Go `interface{}` fields backed
by dynamically generated property structs whose fields are created at runtime
via Go reflection. Examples include SDK member type and trait lists
(`java_libs`, `native_libs`, etc.), APK import DPI/architecture variant
properties, and genrule extension properties.

The schema generator already has access to the concrete runtime types at
generation time (the module factories populate these `interface{}` fields
before property structs are registered). Future work could resolve these to
their actual struct definitions, emitting concrete `struct` or `string[]`
types with full sub-property descriptions instead of `any`.
