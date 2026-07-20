# Pure Go Static Analysis Packages and Release Preparation Implementation Plan

> Execution requirement: implement this plan task-by-task in the main session. Track each checkbox while preserving a reviewable working-tree diff.

**Goal:** Publish the four analysis libraries under `pkg/`, remove domain-specific material, add generic policies and compatibility/platform documentation, and prepare verified cross-platform release artifacts without tagging or publishing.

**Architecture:** Preserve behavior while changing package import boundaries from `internal/*` to public `pkg/*`. Keep analysis generic and read-only: discovery obtains facts only from Go tooling, graph normalizes those facts, policy adds explicit heuristics, and report renders deterministic output. Release preparation is isolated in a tag-triggered GitHub Actions workflow and never runs an external release operation during implementation.

**Tech Stack:** Go 1.26.0 standard library, JSON policy files, Markdown, GitHub Actions, POSIX shell, `sha256sum`/`shasum`.

## Global Constraints

- The repository remains a generic, read-only, standard-library-only Go static-analysis CLI and library.
- Public packages are exactly `pkg/discovery`, `pkg/graph`, `pkg/policy`, and `pkg/report`.
- Preserve existing exported identifiers, behavior, deterministic ordering, JSON `schemaVersion: 1`, and exit codes `0`, `1`, `2`, and `3`.
- Do not import analyzed workspace packages or modify target `go.mod`, `go.work`, policies, source, or dependency versions.
- Remove domain-specific examples and references; examples must be reusable across Go repositories.
- Build-tag and multi-platform support is documentation-only; do not add CLI flags or union platform graphs.
- Prepare release automation only. Do not create or push a tag, push commits, or publish a GitHub Release.
- Execute sequentially in the main session without commits.

---

## File Structure

- Move `internal/discovery/{discovery.go,discovery_test.go}` to `pkg/discovery/`: Go tooling discovery and its tests.
- Move `internal/graph/{model.go,graph_test.go}` to `pkg/graph/`: normalized graph model and algorithms.
- Move `internal/policy/{evaluate.go,policy_test.go}` to `pkg/policy/`: strict policy decoding and evaluation.
- Move `internal/report/{report.go,report_test.go}` to `pkg/report/`: deterministic text, JSON, and Mermaid rendering.
- Modify `cmd/command.go`: consume public package paths.
- Delete the existing domain-specific JSON policy from `examples/`.
- Create `examples/standard-layout.json`: generic standard layout policy.
- Create `examples/clean-architecture.json`: inward-only Clean Architecture policy.
- Create `examples/hexagonal-architecture.json`: ports-and-adapters policy.
- Create `examples/README.md`: explain each policy's assumptions and invocation.
- Create `docs/specs/2026-07-20-api-json-compatibility.md`: public Go API and JSON schema contract.
- Create `docs/specs/2026-07-20-build-constraints-platform-analysis.md`: snapshot and matrix strategy.
- Create `.github/workflows/release.yml`: tag-triggered build, archive, checksum, and GitHub Release upload.
- Create `scripts/build-release.sh`: locally testable cross-platform artifact builder used by CI.
- Modify `README.md`: generic usage, public libraries, examples, compatibility, platform caveats, release build instructions.
- Modify `CLAUDE.md`: replace `internal/*` architecture with `pkg/*` and remove domain-specific wording.
- Modify `README.todo`: archive only verified completed work; keep first tagged release open.

---

### Task 1: Establish a Green Baseline

**Files:**
- Inspect: `go.mod`
- Test: all existing `*_test.go`

**Interfaces:**
- Consumes: current repository state.
- Produces: recorded green baseline against which relocation is compared.

- [ ] **Step 1: Confirm the working tree contains only the approved design document**

Run:

```bash
git status --short
```

Expected: the new `docs/specs/2026-07-20-pure-static-analysis-packages-release-design.md` is untracked and no unexpected user changes are present. If unexpected changes appear, stop and report them rather than overwriting them.

- [ ] **Step 2: Run the existing test suite**

Run:

```bash
go test ./... -count=1
```

Expected: every package reports `ok` or `[no test files]` and the command exits `0`.

- [ ] **Step 3: Run static and build checks**

Run:

```bash
go vet ./...
go build ./...
```

Expected: both commands produce no diagnostics and exit `0`.

---

### Task 2: Relocate Discovery and Graph to Public Packages

**Files:**
- Move: `internal/discovery/discovery.go` → `pkg/discovery/discovery.go`
- Move: `internal/discovery/discovery_test.go` → `pkg/discovery/discovery_test.go`
- Move: `internal/graph/model.go` → `pkg/graph/model.go`
- Move: `internal/graph/graph_test.go` → `pkg/graph/graph_test.go`
- Modify: `pkg/graph/model.go:4`
- Modify: `pkg/graph/graph_test.go:4`

**Interfaces:**
- Consumes: existing `discovery.Snapshot`, `discovery.Options`, `discovery.Runner`, and `discovery.Discover` behavior.
- Produces: importable `github.com/bizshuk/go-dependency-analysis/pkg/discovery` and `github.com/bizshuk/go-dependency-analysis/pkg/graph`; `graph.Build(discovery.Snapshot) graph.Analysis` remains unchanged.

- [ ] **Step 1: Move the two package directories without altering source**

Run:

```bash
mkdir -p pkg
git mv internal/discovery pkg/discovery
git mv internal/graph pkg/graph
```

Expected: Git records four renames and `internal/discovery` and `internal/graph` no longer exist.

- [ ] **Step 2: Update graph imports to the public discovery path**

Replace every exact import:

```go
"github.com/bizshuk/go-dependency-analysis/internal/discovery"
```

with:

```go
"github.com/bizshuk/go-dependency-analysis/pkg/discovery"
```

in `pkg/graph/model.go` and `pkg/graph/graph_test.go`.

- [ ] **Step 3: Verify the moved packages**

Run:

```bash
gofmt -w pkg/discovery/*.go pkg/graph/*.go
go test ./pkg/discovery ./pkg/graph -count=1
```

Expected: both packages report `ok`.

---

### Task 3: Relocate Policy and Report to Public Packages

**Files:**
- Move: `internal/policy/evaluate.go` → `pkg/policy/evaluate.go`
- Move: `internal/policy/policy_test.go` → `pkg/policy/policy_test.go`
- Move: `internal/report/report.go` → `pkg/report/report.go`
- Move: `internal/report/report_test.go` → `pkg/report/report_test.go`
- Modify: imports in all moved files and `cmd/command.go:7-10`

**Interfaces:**
- Consumes: `graph.Analysis` from Task 2.
- Produces: `policy.Load(string) (policy.Config, error)`, `policy.Evaluate(graph.Analysis, policy.Config) []policy.Diagnostic`, and existing `report.Render*` signatures at public paths.

- [ ] **Step 1: Move policy and report directories**

Run:

```bash
git mv internal/policy pkg/policy
git mv internal/report pkg/report
```

Expected: `internal/` becomes empty and all four implementation packages are under `pkg/`.

- [ ] **Step 2: Rewrite all production and test imports**

Use these exact public imports wherever their internal counterparts occur:

```go
"github.com/bizshuk/go-dependency-analysis/pkg/discovery"
"github.com/bizshuk/go-dependency-analysis/pkg/graph"
"github.com/bizshuk/go-dependency-analysis/pkg/policy"
"github.com/bizshuk/go-dependency-analysis/pkg/report"
```

Apply them to `pkg/policy/evaluate.go`, `pkg/policy/policy_test.go`, `pkg/report/report.go`, `pkg/report/report_test.go`, and `cmd/command.go`.

- [ ] **Step 3: Format and run package-focused tests**

Run:

```bash
gofmt -w cmd/command.go pkg/policy/*.go pkg/report/*.go
go test ./pkg/... ./cmd -count=1
```

Expected: all public packages pass; `cmd` reports `[no test files]` unless tests are later added.

- [ ] **Step 4: Prove no Go source imports the old boundary**

Run:

```bash
if grep -RIn 'github.com/bizshuk/go-dependency-analysis/internal/' --include='*.go' .; then
  echo 'old internal import remains' >&2
  exit 1
fi
```

Expected: no matches and exit `0`.

---

### Task 4: Add Generic Policy Examples with Load Tests

**Files:**
- Delete: the pre-existing domain-specific JSON file under `examples/`
- Create: `examples/standard-layout.json`
- Create: `examples/clean-architecture.json`
- Create: `examples/hexagonal-architecture.json`
- Create: `examples/README.md`
- Create: `pkg/policy/examples_test.go`

**Interfaces:**
- Consumes: `policy.Load(path string) (Config, error)`.
- Produces: three strict JSON policies that decode through the public policy API.

- [ ] **Step 1: Write a failing test that requires exactly the three generic examples**

Create `pkg/policy/examples_test.go`:

```go
package policy_test

import (
    "path/filepath"
    "testing"

    "github.com/bizshuk/go-dependency-analysis/pkg/policy"
)

func TestGenericExamplesLoad(t *testing.T) {
    t.Parallel()

    examples := []string{
        "standard-layout.json",
        "clean-architecture.json",
        "hexagonal-architecture.json",
    }
    for _, name := range examples {
        name := name
        t.Run(name, func(t *testing.T) {
            t.Parallel()
            path := filepath.Join("..", "..", "examples", name)
            if _, err := policy.Load(path); err != nil {
                t.Fatalf("Load(%q): %v", path, err)
            }
        })
    }
}
```

- [ ] **Step 2: Verify the test fails because generic examples do not exist**

Run:

```bash
go test ./pkg/policy -run TestGenericExamplesLoad -count=1
```

Expected: FAIL with an `open ../../examples/...: no such file or directory` error.

- [ ] **Step 3: Remove the domain-specific policy and create the standard layout policy**

Delete the pre-existing domain-specific JSON policy. Create `examples/standard-layout.json`:

```json
{
  "layers": {},
  "forbidden": [
    {
      "from": "example.com/project/internal/domain*",
      "to": "example.com/project/cmd*"
    },
    {
      "from": "example.com/project/internal/service*",
      "to": "example.com/project/cmd*"
    }
  ],
  "heavy": {},
  "thresholds": {},
  "severity": {
    "layer-forbidden": "error"
  }
}
```

- [ ] **Step 4: Create the Clean Architecture policy**

Create `examples/clean-architecture.json`:

```json
{
  "layers": {},
  "forbidden": [
    {
      "from": "example.com/project/internal/entity*",
      "to": "example.com/project/internal/usecase*"
    },
    {
      "from": "example.com/project/internal/entity*",
      "to": "example.com/project/internal/adapter*"
    },
    {
      "from": "example.com/project/internal/usecase*",
      "to": "example.com/project/internal/adapter*"
    },
    {
      "from": "example.com/project/internal/adapter*",
      "to": "example.com/project/cmd*"
    }
  ],
  "heavy": {},
  "thresholds": {},
  "severity": {
    "layer-forbidden": "error"
  }
}
```

- [ ] **Step 5: Create the Hexagonal Architecture policy**

Create `examples/hexagonal-architecture.json`:

```json
{
  "layers": {},
  "forbidden": [
    {
      "from": "example.com/project/internal/domain*",
      "to": "example.com/project/internal/adapter*"
    },
    {
      "from": "example.com/project/internal/application*",
      "to": "example.com/project/internal/adapter*"
    },
    {
      "from": "example.com/project/internal/adapter/inbound*",
      "to": "example.com/project/internal/adapter/outbound*"
    },
    {
      "from": "example.com/project/internal/adapter/outbound*",
      "to": "example.com/project/internal/adapter/inbound*"
    }
  ],
  "heavy": {},
  "thresholds": {},
  "severity": {
    "layer-forbidden": "error"
  }
}
```

- [ ] **Step 6: Document how to adapt and run every example**

Create `examples/README.md` with:

```markdown
# Policy examples

These policies illustrate dependency directions for common Go architectures. They are examples, not defaults. Replace `example.com/project` and package prefixes with the module paths used by the analyzed workspace.

## Standard layout

`standard-layout.json` prevents domain and service packages from depending on command entry points.

```bash
go-dependency-analysis --workspace /path/to/go.work --policy examples/standard-layout.json --fail-on error
```

## Clean Architecture

`clean-architecture.json` keeps entities independent from use cases and adapters, and keeps use cases independent from adapters. Dependencies point inward.

```bash
go-dependency-analysis --workspace /path/to/go.work --policy examples/clean-architecture.json --fail-on error
```

## Hexagonal Architecture

`hexagonal-architecture.json` keeps domain and application ports independent from adapters and prevents inbound and outbound adapters from depending on each other.

```bash
go-dependency-analysis --workspace /path/to/go.work --policy examples/hexagonal-architecture.json --fail-on error
```

Policy matching uses exact module paths or a trailing `*` prefix match. The current evaluator applies `forbidden` rules to module edges, so use module paths rather than filesystem-only directory names.
```

- [ ] **Step 7: Verify all examples load**

Run:

```bash
gofmt -w pkg/policy/examples_test.go
go test ./pkg/policy -run TestGenericExamplesLoad -count=1
```

Expected: PASS.

---

### Task 5: Document Public API and JSON Compatibility

**Files:**
- Create: `docs/specs/2026-07-20-api-json-compatibility.md`
- Modify: `pkg/report/report_test.go`

**Interfaces:**
- Consumes: `report.Document` and `report.RenderJSONWithMetadata`.
- Produces: a documented contract preserving JSON `schemaVersion: 1` and public API versioning rules.

- [ ] **Step 1: Add an explicit schema-version regression assertion**

Add this test to `pkg/report/report_test.go`:

```go
func TestRenderJSONSchemaVersion(t *testing.T) {
    var out bytes.Buffer
    if err := RenderJSONWithMetadata(&out, graph.Analysis{}, nil, "", false); err != nil {
        t.Fatal(err)
    }

    var doc Document
    if err := json.Unmarshal(out.Bytes(), &doc); err != nil {
        t.Fatal(err)
    }
    if doc.SchemaVersion != 1 {
        t.Fatalf("SchemaVersion = %d, want 1", doc.SchemaVersion)
    }
}
```

Ensure `bytes` and `encoding/json` are present in the import block alongside existing imports.

- [ ] **Step 2: Run the schema regression test**

Run:

```bash
gofmt -w pkg/report/report_test.go
go test ./pkg/report -run TestRenderJSONSchemaVersion -count=1
```

Expected: PASS with `SchemaVersion == 1`.

- [ ] **Step 3: Write the compatibility contract**

Create `docs/specs/2026-07-20-api-json-compatibility.md` with these exact normative points:

```markdown
# Public API and JSON compatibility

## Go API

The packages under `pkg/` are public. Releases follow Semantic Versioning.

- During `v0.x`, incompatible public API changes are allowed only when necessary and must be listed in release notes.
- Starting with `v1.0.0`, removing or renaming exported identifiers, changing exported signatures, or changing documented semantics requires a new major version.
- Additive exported APIs may be released in a minor version.
- Compatible fixes may be released in a patch version.

## JSON schema

JSON documents contain an integer `schemaVersion`. The current version is `1`.

Increment `schemaVersion` when a release removes or renames a field, changes a field type or meaning, changes required structural shape, or changes enum semantics incompatibly. An additive field may retain the current schema version only when consumers can safely ignore unknown fields.

Within one schema version:

- field meanings and provenance values remain stable;
- arrays that represent deterministic analysis output remain deterministically ordered;
- `go-tool-fact` remains distinct from `policy-heuristic`;
- consumers must not interpret an absent platform-specific edge as proof that a dependency is removable.

Schema changes must include regression tests and release notes. Consumers should reject unsupported `schemaVersion` values rather than silently reinterpret them.
```

- [ ] **Step 4: Check the document for forbidden placeholders**

Run:

```bash
if grep -nE '\b(TBD|TODO)\b' docs/specs/2026-07-20-api-json-compatibility.md; then
  exit 1
fi
```

Expected: no output and exit `0`.

---

### Task 6: Document Build Constraints and Platform Analysis

**Files:**
- Create: `docs/specs/2026-07-20-build-constraints-platform-analysis.md`

**Interfaces:**
- Consumes: current CLI behavior (`--include-tests`, host Go tooling environment).
- Produces: a reproducible external matrix strategy without new CLI surface or schema changes.

- [ ] **Step 1: Write the platform snapshot contract**

Create `docs/specs/2026-07-20-build-constraints-platform-analysis.md`:

```markdown
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
```

- [ ] **Step 2: Verify the strategy does not advertise unsupported flags**

Run:

```bash
if grep -nE -- '--goos|--goarch|--tags' docs/specs/2026-07-20-build-constraints-platform-analysis.md | grep -v 'does not currently expose'; then
  echo 'unsupported CLI flag presented as usable' >&2
  exit 1
fi
```

Expected: no output and exit `0`.

---

### Task 7: Add a Locally Testable Cross-Platform Release Builder

**Files:**
- Create: `scripts/build-release.sh`

**Interfaces:**
- Consumes: optional positional version, default `dev`; environment variable `DIST_DIR`, default `dist`.
- Produces: `.tar.gz` archives for Unix targets, `.zip` archives for Windows targets, and `checksums.txt` containing one SHA-256 line per archive.

- [ ] **Step 1: Create the release build script**

Create executable `scripts/build-release.sh`:

```bash
#!/usr/bin/env bash
set -euo pipefail

version="${1:-dev}"
dist_dir="${DIST_DIR:-dist}"
name="go-dependency-analysis"
targets=(
  "linux/amd64"
  "linux/arm64"
  "darwin/amd64"
  "darwin/arm64"
  "windows/amd64"
)

rm -rf "$dist_dir"
mkdir -p "$dist_dir"

for target in "${targets[@]}"; do
  goos="${target%/*}"
  goarch="${target#*/}"
  base="${name}_${version}_${goos}_${goarch}"
  stage="$dist_dir/$base"
  binary="$name"
  if [[ "$goos" == "windows" ]]; then
    binary+=".exe"
  fi

  mkdir -p "$stage"
  CGO_ENABLED=0 GOOS="$goos" GOARCH="$goarch" go build \
    -trimpath \
    -ldflags="-s -w" \
    -o "$stage/$binary" .

  if [[ "$goos" == "windows" ]]; then
    (
      cd "$dist_dir"
      zip -q -X -r "$base.zip" "$base"
    )
  else
    tar -C "$dist_dir" -czf "$dist_dir/$base.tar.gz" "$base"
  fi
  rm -rf "$stage"
done

(
  cd "$dist_dir"
  if command -v sha256sum >/dev/null 2>&1; then
    sha256sum ./*.tar.gz ./*.zip > checksums.txt
  else
    shasum -a 256 ./*.tar.gz ./*.zip > checksums.txt
  fi
)
```

Run:

```bash
chmod +x scripts/build-release.sh
```

- [ ] **Step 2: Execute the release builder in a temporary output directory**

Run:

```bash
DIST_DIR="$(mktemp -d)" scripts/build-release.sh test
```

Expected: five cross-platform archives and one `checksums.txt`; command exits `0`.

- [ ] **Step 3: Verify archive and checksum counts deterministically**

Run in one shell so `DIST_DIR` remains available:

```bash
DIST_DIR="$(mktemp -d)"
export DIST_DIR
scripts/build-release.sh test
archive_count="$(find "$DIST_DIR" -maxdepth 1 -type f \( -name '*.tar.gz' -o -name '*.zip' \) | wc -l | tr -d ' ')"
checksum_count="$(wc -l < "$DIST_DIR/checksums.txt" | tr -d ' ')"
test "$archive_count" = 5
test "$checksum_count" = 5
rm -rf "$DIST_DIR"
```

Expected: all assertions pass and exit `0`.

---

### Task 8: Add Tag-Triggered GitHub Release Automation

**Files:**
- Create: `.github/workflows/release.yml`

**Interfaces:**
- Consumes: tag `v*` and `scripts/build-release.sh` from Task 7.
- Produces: release archives and `checksums.txt` attached to a GitHub Release when a maintainer intentionally pushes a matching tag.

- [ ] **Step 1: Create the release workflow**

Create `.github/workflows/release.yml`:

```yaml
name: release

on:
  push:
    tags:
      - "v*"

permissions:
  contents: write

jobs:
  release:
    runs-on: ubuntu-latest
    steps:
      - name: Check out source
        uses: actions/checkout@v4

      - name: Set up Go
        uses: actions/setup-go@v5
        with:
          go-version-file: go.mod
          cache: false

      - name: Install archive tools
        run: sudo apt-get update && sudo apt-get install -y zip

      - name: Test
        run: go test ./... -count=1

      - name: Build release artifacts
        env:
          DIST_DIR: dist
        run: scripts/build-release.sh "${GITHUB_REF_NAME#v}"

      - name: Publish GitHub Release
        uses: softprops/action-gh-release@v2
        with:
          generate_release_notes: true
          files: |
            dist/*.tar.gz
            dist/*.zip
            dist/checksums.txt
```

- [ ] **Step 2: Statically verify tag gating and artifact paths**

Run:

```bash
grep -F 'tags:' .github/workflows/release.yml
grep -F 'scripts/build-release.sh' .github/workflows/release.yml
grep -F 'dist/checksums.txt' .github/workflows/release.yml
```

Expected: all three patterns are printed. Do not trigger the workflow, create a tag, or call `gh release`.

---

### Task 9: Synchronize Repository Documentation

**Files:**
- Modify: `README.md`
- Modify: `CLAUDE.md`

**Interfaces:**
- Consumes: final public paths, generic examples, compatibility docs, platform docs, and release script.
- Produces: user-facing and maintainer-facing documentation matching repository reality.

- [ ] **Step 1: Replace the README policy example and add public library documentation**

In `README.md`:

- Replace the domain-specific policy command with:

```bash
go-dependency-analysis \
  --workspace /path/to/go.work \
  --policy examples/standard-layout.json \
  --fail-on error
```

- Replace the sentence describing the old example with:

```markdown
The policies under [`examples/`](examples/) illustrate common Go architecture boundaries and are not universal defaults.
```

- Add after `Analysis model`:

```markdown
## Public packages

The analysis pipeline is available as four standard-library-only Go packages:

- `pkg/discovery`: collect workspace, module, requirement, replacement, and package facts from Go tooling.
- `pkg/graph`: normalize facts and calculate cycles, reachability, and shortest paths.
- `pkg/policy`: strictly load JSON policies and evaluate diagnostics.
- `pkg/report`: render deterministic text, JSON schema v1, and Mermaid output.

See [`docs/specs/2026-07-20-api-json-compatibility.md`](docs/specs/2026-07-20-api-json-compatibility.md) for compatibility guarantees.
```

- Add after the existing host-environment caveat:

```markdown
For platform matrices and build-constraint interpretation, see [`docs/specs/2026-07-20-build-constraints-platform-analysis.md`](docs/specs/2026-07-20-build-constraints-platform-analysis.md). Keep each platform result as a separate snapshot; do not treat an automatic union as one valid build graph.

To validate release artifacts locally without publishing:

```bash
scripts/build-release.sh dev
```
```

- [ ] **Step 2: Update the technical structure in CLAUDE.md**

Replace the four `internal/*` tree entries with:

```text
├── pkg/discovery/           # public Go tooling facts
├── pkg/graph/               # public normalization, SCC, reachability, shortest path
├── pkg/policy/              # public strict JSON policy and diagnostics
├── pkg/report/              # public deterministic text, JSON, Mermaid
```

Replace the example entry with:

```text
├── examples/                # generic optional architecture policies
```

Add these architecture rules:

```markdown
- `pkg/discovery`、`pkg/graph`、`pkg/policy`、`pkg/report` 是 public API，遵循 Semantic Versioning。
- JSON compatibility 由獨立 `schemaVersion` 管理；breaking schema change 必須升版。
- Policy examples 必須保持 generic，不綁定特定產品或 repository。
```

Replace the policy verification command with:

```bash
go run . --workspace /path/to/go.work --policy examples/standard-layout.json --fail-on error
```

- [ ] **Step 3: Verify documentation references real paths**

Run:

```bash
test -f examples/README.md
test -f docs/specs/2026-07-20-api-json-compatibility.md
test -f docs/specs/2026-07-20-build-constraints-platform-analysis.md
test -x scripts/build-release.sh
```

Expected: exit `0`.

---

### Task 10: Update TODO State Based on Verified Deliverables

**Files:**
- Modify: `README.todo`

**Interfaces:**
- Consumes: verified work from Tasks 4–9.
- Produces: a truthful backlog where publication remains open and completed preparation/documentation moves to Archive.

- [ ] **Step 1: Rewrite TODO state after verification**

Set `README.todo` to:

```markdown
# TODO

- [ ] Publish the first tagged release after repository remote setup

## Archive

- [x] Extract analyzer into a standalone stdlib-only repository
- [x] Expose discovery, graph, policy, and report as public packages
- [x] Add cross-platform release binaries and checksums automation
- [x] Define JSON schema compatibility guarantees
- [x] Add documented layer-policy examples for common Go architectures
- [x] Add build-tag and multi-platform analysis strategy
```

- [ ] **Step 2: Confirm no unverified publication claim exists**

Run:

```bash
grep -F -- '- [ ] Publish the first tagged release' README.todo
if grep -F -- '- [x] Publish the first tagged release' README.todo; then
  exit 1
fi
```

Expected: the open publication item is printed and exit `0`.

---

### Task 11: Run Repository-Wide Verification

**Files:**
- Verify: entire repository

**Interfaces:**
- Consumes: all preceding tasks.
- Produces: evidence that the relocation, generic scope, examples, schema contract, and release preparation are complete without external publication.

- [ ] **Step 1: Format and test all Go code**

Run:

```bash
gofmt -w cmd/*.go pkg/**/*.go
go test ./... -count=1
```

Expected: all packages pass.

- [ ] **Step 2: Run vet and build checks**

Run:

```bash
go vet ./...
go build ./...
```

Expected: no diagnostics and exit `0`.

- [ ] **Step 3: Smoke-test all three generic policies**

Run against a temporary workspace whose `use` path remains valid from the temporary directory, while including the tool module explicitly:

```bash
tmpdir="$(mktemp -d)"
module_dir="$(pwd)"
(
  cd "$tmpdir"
  go work init "$module_dir"
)
for policy in examples/standard-layout.json examples/clean-architecture.json examples/hexagonal-architecture.json; do
  go run . \
    --workspace "$tmpdir/go.work" \
    --include-tool-module \
    --policy "$policy" \
    --format json \
    --fail-on none >/dev/null
done
rm -rf "$tmpdir"
```

Expected: all three invocations exit `0`; no `go.work` is created in the repository.

- [ ] **Step 4: Verify release artifacts and checksums again**

Run:

```bash
DIST_DIR="$(mktemp -d)"
export DIST_DIR
scripts/build-release.sh verify
archive_count="$(find "$DIST_DIR" -maxdepth 1 -type f \( -name '*.tar.gz' -o -name '*.zip' \) | wc -l | tr -d ' ')"
checksum_count="$(wc -l < "$DIST_DIR/checksums.txt" | tr -d ' ')"
test "$archive_count" = 5
test "$checksum_count" = 5
rm -rf "$DIST_DIR"
```

Expected: five archives, five checksum entries, and exit `0`.

- [ ] **Step 5: Enforce the clean generic scope**

Run:

```bash
if find . -path ./.git -prune -o -path ./docs/memory -prune -o -type f -print0 | xargs -0 grep -nEi 'domain-specific product marker'; then
  echo 'domain-specific product content remains' >&2
  exit 1
fi
if grep -RIn '/internal/' --include='*.go' .; then
  echo 'old internal import remains' >&2
  exit 1
fi
```

Expected: no matches and exit `0`. The approved design and this plan must use generic wording so they do not cause false positives.

- [ ] **Step 6: Validate the final diff and repository state**

Run:

```bash
git diff --check
git status --short
git diff --stat
```

Expected: `git diff --check` exits `0`; status shows only this approved implementation; no tag, commit, push, or release was created.

- [ ] **Step 7: Report exact outcomes**

Report:

- package relocation count and public import paths;
- test, vet, and build outcomes;
- number of policy examples loaded;
- release target and checksum counts;
- remaining open TODO (`Publish the first tagged release`);
- any skipped or failed step with its exact command output.
