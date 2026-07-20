# Multi-Version Dependency Diagnostic Implementation Plan

## Context

The analyzer currently records each module's selected version and only the root module's direct requirements. Its `version-divergence` logic compares a direct requirement with an order-dependent `map[path]selectedVersion`; it cannot recursively explain which dependencies requested different versions. It also accepts only `go.work`, while the requested workflow starts from one explicit `go.mod`.

This change adds a read-only single-module mode and uses `go mod graph` to collect recursive requested-version edges. The diagnostic reports version pressure rather than claiming that Go links multiple versions simultaneously: Minimal Version Selection (MVS) still chooses one version for each exact module path. Per user direction, semantic-import major paths such as `example.com/lib` and `example.com/lib/v2` are additionally grouped into one dependency family.

Implementation remains sequential in the main session. Do not commit, tag, push, publish, or use implementation subagents.

## Recommended Approach

### 1. Add explicit `go.mod` analysis mode

Modify `pkg/discovery/discovery.go` and `cmd/command.go`:

- Let `--workspace` accept either an exact `go.work` or `go.mod` file.
- Preserve existing `go.work` behavior and the current active-workspace lookup when the flag is omitted.
- For `go.mod`, canonicalize the file, use its containing directory as the root, and execute all module/package commands with `GOWORK=off`.
- Reject explicit paths whose basename is neither `go.work` nor `go.mod` with a usage error.
- Update flag help and README examples to call the input an analysis root rather than only a workspace.

Reuse the existing `Runner`, `ExecRunner`, canonical-path helpers, `go mod edit -json`, `go list -m -json all`, and `go list -deps -json ./...` flow.

### 2. Collect recursive requested-version evidence

Extend `pkg/discovery/discovery.go` with a public request record retained in `Snapshot`, containing:

- analysis origin/root module;
- requester module path and version;
- requested module path and version;
- whether it is a direct root `go.mod` requirement;
- normalized raw edge evidence from `go mod graph`.

Run `go mod graph` with `GOWORK=off` for the explicit root module. Parse `path@version` tokens without treating an absent version on the main module as an error. Continue using `go list -m -json all` as the authoritative selected build list and preserve replacement path/version/directory from that output.

For retained `go.work` support, run the same module graph collection once per workspace member under `GOWORK=off`, preserving that member as the origin. This avoids collapsing independent module observations.

Add discovery tests using the existing fake `Runner` for:

- explicit `go.mod` mode and command environments;
- existing `go.work` mode regression;
- recursive `go mod graph` parsing;
- direct/indirect request marking;
- replacement retention;
- malformed graph lines and unsupported input paths;
- deterministic request ordering.

### 3. Build deterministic dependency-version summaries

Extend `pkg/graph/model.go` and `graph.Analysis` with structured version summaries. Each summary contains:

- normalized dependency family;
- exact module paths in that family;
- distinct requested versions;
- selected version/effective replacement for each exact path and origin;
- request observations with requester, origin, direct flag, and evidence.

Normalize a semantic-import major suffix only when the final path segment is exactly `/vN` with `N >= 2`; for example, group `example.com/lib` and `example.com/lib/v2`, but do not strip arbitrary numeric path segments. Keep the exact module paths in output so consumers can see that Go treats them as distinct modules.

Trigger a multi-version summary when either:

- one exact module path has more than one recursively requested version; or
- one normalized family contains more than one exact major module path/version identity.

Do not treat one requested version upgraded to one selected version by MVS as two coexisting versions. Selected versions and replacements are explanatory evidence, not extra trigger counts.

Sort families, paths, versions, origins, requesters, replacements, and evidence. Deduplicate exact observations only; never discard a distinct requester or origin. Preserve existing path-level module/package graphs and graph algorithms.

Add graph tests for same-path requests, `/v2` family grouping, non-major numeric suffixes, selected versions, local/path replacements, origin preservation, deduplication, and byte-stable ordering.

### 4. Replace the old version diagnostic

Modify `pkg/policy/evaluate.go`:

- Replace the current order-dependent `version-divergence` loop with a `multi-version` diagnostic generated from graph version summaries.
- Use provenance `go-tool-fact` and default severity `warning`.
- Include the family, exact paths, requested versions, selected versions/replacements, requesters/origins, and raw graph edges in deterministic `Evidence`.
- Word the message as “multiple requested versions/module majors” and explicitly avoid saying that multiple versions are linked simultaneously.
- Allow `severity.multi-version` in existing policy JSON without adding new configuration shape.
- Keep cycle and unused-direct facts, plus configured forbidden/layer/heavy heuristics.

Modify `cmd/command.go` so `policy.Evaluate(a, policy.Config{})` always runs when no policy is supplied. When `--policy` is supplied, load it and use it for severity overrides and policy heuristics. Existing `--fail-on` applies to default intrinsic diagnostics as well.

Add policy/CLI tests proving:

- recursive conflicting requests emit one deterministic `multi-version` diagnostic without `--policy`;
- a normal single requested version selected at a higher MVS version does not emit it;
- `/v1` and `/v2` paths are grouped per the requested family semantics;
- replacements and requesters appear in evidence;
- configured severity overrides work;
- `--fail-on warning` returns exit code `1` for this intrinsic diagnostic;
- existing configured diagnostics remain additive.

### 5. Report structured results and bump JSON schema

Modify `pkg/report/report.go` and tests:

- Include version summaries under `analysis` through the new graph field.
- Bump `schemaVersion` from `1` to `2`; the additive structured analysis changes the published JSON contract and project rules require a schema bump.
- Enhance text output to print each diagnostic followed by indented evidence lines, so the CLI directly summarizes versions/requesters/replacements rather than requiring JSON inspection.
- Keep Mermaid path-level and unchanged; it must not imply simultaneous versioned module nodes.
- Preserve deterministic JSON rendering and add schema-v2 fixture/assertion tests.

Update `docs/specs/2026-07-20-api-json-compatibility.md` to document schema v2, the new version-summary fields, and the distinction between requested versions, selected versions, and exact major module paths.

### 6. Synchronize user and maintainer documentation

Update `README.md`, `CLAUDE.md`, and `README.todo` where applicable:

- document `--workspace /path/to/go.mod` and retained `go.work` support;
- explain recursive request comparison and Go MVS semantics;
- explain that `/v2` family grouping is an analyzer advisory layered over exact Go module identity;
- state that intrinsic diagnostics run without `--policy`;
- document `multi-version`, its default warning severity, evidence, and `--fail-on` behavior;
- change JSON references from schema v1 to schema v2;
- keep the tool generic, read-only, and stdlib-only.

Do not change build-tag/platform behavior, release publication, or unrelated static-analysis gaps in this feature.

## Critical Files

- `cmd/command.go`
- `pkg/discovery/discovery.go`
- `pkg/discovery/discovery_test.go`
- `pkg/graph/model.go`
- `pkg/graph/graph_test.go`
- `pkg/policy/evaluate.go`
- `pkg/policy/policy_test.go`
- `pkg/report/report.go`
- `pkg/report/report_test.go`
- `README.md`
- `CLAUDE.md`
- `docs/specs/2026-07-20-api-json-compatibility.md`

## Verification

Run focused red/green tests during each layer, then execute:

```bash
go test ./... -count=1
go vet ./...
go build ./...
```

Create a temporary fixture module graph with two transitive requesters asking for different versions of the same dependency, then verify end to end:

```bash
go run . --workspace /tmp/multi-version-fixture/go.mod --format text
go run . --workspace /tmp/multi-version-fixture/go.mod --format json --json-indent='  '
go run . --workspace /tmp/multi-version-fixture/go.mod --format json --fail-on warning
```

Expected results:

- text contains one `multi-version` warning plus requester/version evidence;
- JSON reports `schemaVersion: 2` and a deterministic structured version summary;
- `--fail-on warning` exits `1`;
- a control fixture with only one requested version emits no `multi-version` diagnostic;
- existing `go.work`, policy examples, text/JSON/Mermaid, release artifact, and checksum smoke tests remain green;
- `git diff --check` passes and no commit/tag/push/release is created.
