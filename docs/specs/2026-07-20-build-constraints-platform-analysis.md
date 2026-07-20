# Build constraints and multi-platform analysis

A `go-dependency-analysis` invocation is a snapshot of the package graph selected by the active Go toolchain. The result depends on:

- Go toolchain version;
- `GOOS` and `GOARCH`;
- build constraints and file suffixes;
- generated files present at analysis time;
- workspace selection and replacements;
- whether `--include-tests` is enabled.

The CLI does not currently expose `--goos`, `--goarch`, or `--tags` flags and does not union multiple snapshots.

## Platform matrix

Run one analysis per valid environment and keep each JSON result separate:

```bash
mkdir -p analysis
for target in linux/amd64 linux/arm64 darwin/amd64 darwin/arm64 windows/amd64; do
  GOOS=${target%/*}
  GOARCH=${target#*/}
  output="analysis/${GOOS}-${GOARCH}.json"
  GOOS="$GOOS" GOARCH="$GOARCH" go run . \
    --workspace /path/to/go.work \
    --format json \
    --output "$output"
done
```

Run a second matrix with `--include-tests` when test-only imports matter. Record the Go version with `go version` next to the snapshots.

## Interpretation

Do not automatically union snapshots. A union can contain edges that never coexist in one valid build and can create artificial reachability or cycles. Compare snapshots as separate evidence sets and report the environment with every finding.

An `unused-direct-candidate` means only that the selected invocation did not observe a package import. Check other platforms, build tags, tests, generated files, tools, and runtime loading before changing dependencies.

Custom build tags are selected by Go tooling configuration outside this CLI. Run the equivalent `go list` environment separately and preserve its configuration with the resulting snapshot.
