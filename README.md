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

Analyze an explicit workspace:

```bash
go-dependency-analysis \
  --workspace /path/to/go.work \
  --format json \
  --json-indent='  '
```

Other output formats:

```bash
go-dependency-analysis --workspace /path/to/go.work --format mermaid
go-dependency-analysis --workspace /path/to/go.work --output dependency.json --format json
```

Apply a JSON policy:

```bash
go-dependency-analysis \
  --workspace /Users/shuk/projects/agentSDK/go.work \
  --policy examples/agentsdk.json \
  --fail-on error
```

[`examples/agentsdk.json`](examples/agentsdk.json) is an AgentSDK-specific example, not a default policy.

## Analysis model

The CLI uses `go env GOWORK`, `go work edit -json`, `go mod edit -json`, and `go list`. It distinguishes:

- `go-tool-fact`: workspace membership, selected versions, import edges, cycles, version divergence, and direct requirements not observed by the invocation.
- `policy-heuristic`: configured layer violations, forbidden edges, and heavy dependency paths.

An `unused-direct-candidate` is not an instruction to remove a dependency. Check test-only imports, build tags, platform files, generated code, and tool dependencies first.

The analyzer excludes its own module from a target workspace by module identity. Use `--include-tool-module` to include it. Use `--exclude PATH[,PATH...]` for explicit workspace module exclusions.

## Flags

| Flag | Purpose |
| --- | --- |
| `--workspace PATH` | Explicit `go.work`; defaults to the active workspace. |
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
