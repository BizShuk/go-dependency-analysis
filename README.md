# go-dependency-analysis

`go-dependency-analysis` is a read-only, standard-library-only CLI for inspecting Go workspace module and package dependency graphs. It discovers topology from Go tooling instead of hard-coding a repository layout.

## Install

```bash
go install github.com/bizshuk/go-dependency-analysis@latest
```

For a local checkout:

```bash
go build -o go-dependency-analysis .
```

## Usage

Analyze the active workspace:

```bash
go-dependency-analysis --format text
```

Analyze an explicit workspace or a single module:

```bash
go-dependency-analysis \
  --workspace /path/to/go.work \
  --format json \
  --json-indent='  '

go-dependency-analysis \
  --workspace /path/to/go.mod \
  --format text
```

Other output formats:

```bash
go-dependency-analysis --workspace /path/to/go.work --format mermaid
go-dependency-analysis --workspace /path/to/go.work --output dependency.json --format json
```

Apply a JSON policy:

```bash
go-dependency-analysis \
  --workspace /path/to/go.work \
  --policy examples/standard-layout.json \
  --fail-on error
```

The policies under [`examples/`](examples/) illustrate common Go architecture boundaries and are not universal defaults.

## Analysis model

The CLI uses `go env GOWORK`, `go work edit -json`, `go mod edit -json`, and `go list`. It distinguishes:

- `go-tool-fact`: workspace membership, selected versions, recursive requested-version pressure, import edges, cycles, and direct requirements not observed by the invocation.
- `policy-heuristic`: configured layer violations, forbidden edges, and heavy dependency paths.

An `unused-direct-candidate` is not an instruction to remove a dependency. Check test-only imports, build tags, platform files, generated code, and tool dependencies first.

A `multi-version` diagnostic summarizes recursive requests for multiple versions of the same dependency. Go Minimal Version Selection still selects one version per exact module path. For review purposes, the analyzer also groups semantic-import major paths such as `example.com/lib` and `example.com/lib/v2` into one dependency family while preserving their exact paths in JSON evidence. Intrinsic diagnostics run without `--policy`; a policy may override their severity.

The analyzer excludes its own module from a target workspace by module identity. Use `--include-tool-module` to include it. Use `--exclude PATH[,PATH...]` for explicit workspace module exclusions.

## Public packages

The analysis pipeline is available as four standard-library-only Go packages:

- `pkg/discovery`: collect workspace, module, requirement, replacement, and package facts from Go tooling.
- `pkg/graph`: normalize facts and calculate cycles, reachability, and shortest paths.
- `pkg/policy`: strictly load JSON policies and evaluate diagnostics.
- `pkg/report`: render deterministic text, JSON schema v2, and Mermaid output.

See [`docs/specs/2026-07-20-api-json-compatibility.md`](docs/specs/2026-07-20-api-json-compatibility.md) for compatibility guarantees.

## Flags

| Flag | Purpose |
| --- | --- |
| `--workspace PATH` | Explicit `go.work` or `go.mod`; defaults to the active workspace. |
| `--format text\|json\|mermaid` | Output format; defaults to text. |
| `--output PATH` | Write atomically to a file. |
| `--policy PATH` | Load a strict JSON policy. |
| `--include-tests` | Add test imports. |
| `--show-stdlib` | Keep standard-library package nodes. |
| `--include-tool-module` | Include this analyzer if it belongs to the target workspace. |
| `--exclude PATH[,PATH...]` | Exclude module directories; relative paths resolve from the workspace directory. |
| `--fail-on none\|warning\|error` | Set the diagnostic failure threshold. |
| `--json-indent STRING` | Set JSON indentation; pass `''` for compact output. |
| `--version` | Print the version. |

Exit codes:

- `0`: analysis completed and the failure threshold was not met.
- `1`: analysis completed and `--fail-on` matched.
- `2`: invalid CLI arguments or policy.
- `3`: discovery, Go tooling, rendering, or output failed.

## Development

```bash
go test ./... -count=1
go vet ./...
go build ./...
```

The graph reflects the host Go version, OS/architecture, build constraints, and workspace selection. The CLI never modifies `go.mod`, `go.work`, policy files, or dependency versions.

For platform matrices and build-constraint interpretation, see [`docs/specs/2026-07-20-build-constraints-platform-analysis.md`](docs/specs/2026-07-20-build-constraints-platform-analysis.md). Keep each platform result as a separate snapshot; do not treat an automatic union as one valid build graph.

To validate release artifacts locally without publishing:

```bash
scripts/build-release.sh dev
```
