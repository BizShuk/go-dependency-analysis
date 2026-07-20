# Pure Go static-analysis package and release design

Date: 2026-07-20
Status: Approved

## Purpose

Keep `go-dependency-analysis` a generic, read-only, standard-library-only Go static-analysis CLI and library. Remove domain-specific material, expose the analysis packages under `pkg/`, document compatibility boundaries, add reusable layer-policy examples, and prepare release automation without publishing a release.

## Scope

- Move `internal/discovery`, `internal/graph`, `internal/policy`, and `internal/report` to matching `pkg/` paths.
- Treat all four packages as public Go APIs.
- Preserve their current package names, exported identifiers, behavior, deterministic ordering, and test coverage.
- Remove the existing domain-specific policy example and its references.
- Add generic examples for standard Go layout, Clean Architecture, and Hexagonal Architecture.
- Define Go API and JSON schema compatibility guarantees.
- Document how host `GOOS`, `GOARCH`, build constraints, and test selection affect results and how users can run a platform matrix externally.
- Add automation that builds cross-platform binaries and a checksum manifest.
- Do not create or push a tag and do not publish a GitHub Release.

## Architecture

```text
go-dependency-analysis/
├── cmd/                         CLI orchestration and process-facing behavior
├── pkg/discovery/               facts obtained from Go tooling
├── pkg/graph/                   normalized graph and graph algorithms
├── pkg/policy/                  strict generic policy loading and evaluation
├── pkg/report/                  deterministic text, JSON, and Mermaid output
├── examples/
│   ├── standard-layout.json
│   ├── clean-architecture.json
│   └── hexagonal-architecture.json
└── docs/specs/                  compatibility and analysis-strategy documents
```

The dependency direction remains:

```text
cmd → report → policy → graph → discovery
      report ─────────→ graph
```

No package imports analyzed workspace source. Discovery continues to invoke Go tooling through `os/exec`; analysis remains read-only.

## Public API and compatibility

The `pkg/*` paths are importable public APIs. During `v0.x`, incompatible API changes are permitted only when necessary and must be called out in release notes. Starting with `v1.0.0`, incompatible public Go API changes require a new semantic major version.

JSON uses an independent integer `schemaVersion`. Removing or renaming fields, changing field meaning or type, changing required structural shape, or changing enum semantics requires incrementing `schemaVersion`. Additive fields may retain the current schema version when older consumers can safely ignore them. Deterministic ordering and fact/provenance meanings are part of the documented contract.

## Generic policy examples

Each example is a strict JSON policy accepted by the existing loader and is accompanied by README documentation describing the assumed package layout and dependency direction:

- Standard layout: commands and transport entry points may depend on application/service and lower-level packages; lower layers must not depend on commands.
- Clean Architecture: entities are innermost, followed by use cases, adapters, and frameworks/drivers; dependencies point inward.
- Hexagonal Architecture: domain and application ports remain independent of adapters; inbound and outbound adapters depend on ports, not each other.

The examples illustrate policy configuration rather than impose a universal repository layout.

## Platform and build-constraint strategy

A single invocation reports the package graph selected by the active Go toolchain environment. Results may differ by Go version, `GOOS`, `GOARCH`, build tags, generated files, and test inclusion. This change documents those limits without adding CLI flags or changing the JSON schema.

Users needing broader evidence should invoke the CLI separately under explicit environment combinations and retain each result as a separate snapshot. Results are not automatically unioned because an aggregate graph could imply dependency edges that never coexist in one valid build.

## Release preparation

Add a GitHub Actions release workflow triggered by version tags. It builds deterministic named binaries for supported Linux, macOS, and Windows architecture combinations, then produces a checksum manifest covering every archive. The workflow may publish artifacts only when a maintainer later pushes an intentional version tag.

This task validates build commands locally where possible but does not create a tag, push refs, or publish a release. Consequently, “Publish the first tagged release” remains open in `README.todo`; cross-platform binary and checksum preparation may be archived when verified.

## Error handling

- Package relocation must not alter runtime error classification or CLI exit codes.
- Policy examples must pass strict JSON decoding; unknown fields remain errors.
- Release jobs fail on build or checksum errors and must not silently omit a target.
- Documentation must distinguish observed Go-tool facts from policy heuristics and platform-specific absence from safe dependency removal.

## Verification

Run:

```bash
go test ./... -count=1
go vet ./...
go build ./...
```

Additionally verify:

- no Go import contains `/internal/`;
- no product code, examples, or documentation contains domain-specific analysis material;
- all three generic policy examples load successfully through the CLI;
- JSON output remains deterministic and reports `schemaVersion: 1`;
- release configuration covers Linux, macOS, and Windows and generates checksums for all artifacts;
- `README.md`, `CLAUDE.md`, and `README.todo` reflect the final repository state.

## Implementation process

Implementation is sequential and keeps product code, examples, documentation, and release artifacts limited to generic Go static analysis.
