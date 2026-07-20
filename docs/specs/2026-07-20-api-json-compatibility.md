# Public API and JSON compatibility

## Go API

The packages under `pkg/` are public. Releases follow Semantic Versioning.

- During `v0.x`, incompatible public API changes are allowed only when necessary and must be listed in release notes.
- Starting with `v1.0.0`, removing or renaming exported identifiers, changing exported signatures, or changing documented semantics requires a new major version.
- Additive exported APIs may be released in a minor version.
- Compatible fixes may be released in a patch version.

## JSON schema

JSON documents contain an integer `schemaVersion`. The current version is `2`.

Schema v2 adds `analysis.module_versions`. Each entry identifies a normalized dependency family, exact Go module paths, recursively requested versions, selected versions, replacements, and requester/origin observations. Requested versions describe dependency pressure; they do not mean Go links all versions simultaneously. Go Minimal Version Selection chooses one version per exact module path. Grouping `/vN` semantic-import paths is an analyzer advisory, while exact paths remain present in every summary.

Increment `schemaVersion` when a release removes or renames a field, changes a field type or meaning, changes required structural shape, or changes enum semantics incompatibly. An additive field may retain the current schema version only when consumers can safely ignore unknown fields.

Within one schema version:

- field meanings and provenance values remain stable;
- arrays that represent deterministic analysis output remain deterministically ordered;
- `go-tool-fact` remains distinct from `policy-heuristic`;
- consumers must not interpret an absent platform-specific edge as proof that a dependency is removable.

Schema changes must include regression tests and release notes. Consumers should reject unsupported `schemaVersion` values rather than silently reinterpret them.
