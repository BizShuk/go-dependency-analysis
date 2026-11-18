# CLAUDE.md — go-dependency-analysis 技術脈絡

`go-dependency-analysis` 是獨立、read-only、stdlib-only 的 Go CLI。它以 Go tooling 建立 workspace module/package graph，並將工具可證明的 facts 與 policy heuristics 分開輸出。

## 技術基準

- Module：`github.com/bizshuk/go-dependency-analysis`
- Go：`1.26.0`
- Third-party dependencies：無
- Binary：`go-dependency-analysis`
- Outputs：text、JSON schema v2、Mermaid
- Exit codes：`0` success、`1` policy threshold、`2` usage/policy、`3` discovery/tool/output

## 結構

```text
go-dependency-analysis/
├── main.go                  # process exit boundary
├── cmd/command.go           # flags、orchestration、atomic output
├── pkg/discovery/           # public Go tooling facts
├── pkg/graph/               # public normalization、SCC、reachability、shortest path
├── pkg/policy/              # public strict JSON policy + diagnostics
├── pkg/report/              # public deterministic text、JSON、Mermaid
├── examples/                # generic optional architecture policies
├── docs/memory/             # historical decisions
└── README.todo              # pending work
```

## 架構規則

- `pkg/discovery`、`pkg/graph`、`pkg/policy`、`pkg/report` 是 public API，遵循 Semantic Versioning。
- JSON compatibility 由獨立 `schemaVersion` 管理；breaking schema change 必須升版。
- Policy examples 必須保持 generic，不綁定特定產品或 repository。
- 不 import target workspace packages；只透過 `os/exec` 呼叫 Go tooling。
- 不加入被分析 repo 的 dependency graph；installed binary 或 explicit checkout 執行。
- 不修改 `go.mod`、`go.work`、policy、source 或 dependency versions。
- Self-exclusion 依 module identity `github.com/bizshuk/go-dependency-analysis`，不依 checkout directory 名稱。
- `go-tool-fact` 與 `policy-heuristic` 不可混用。
- `--workspace` 接受 `go.work` 或 `go.mod`；`go.mod` 模式以 `GOWORK=off` 遞迴分析 module requests。
- `multi-version` 比較 requested versions；Go MVS 對每個 exact module path 仍只選一個版本。
- `/vN` semantic major paths 會聚合成 dependency family advisory，但 exact paths 必須保留在 evidence。
- Intrinsic diagnostics 即使未提供 policy 也必須執行。
- 所有 node、edge、evidence 與 diagnostics 必須 deterministic sort。
- JSON schema 變動需升級 `schemaVersion`。
- Mermaid labels 使用雙引號，node ID 必須 collision-resistant。

## 驗證

```bash
go test ./... -count=1
go vet ./...
go build ./...

go run . --workspace /path/to/go.work --format text
go run . --workspace /path/to/go.work --format json --json-indent='  '
go run . --workspace /path/to/go.work --format mermaid
go run . --workspace /path/to/go.work --policy examples/standard-layout.json --fail-on error
```

## 注意事項

- Host Go version、GOOS/GOARCH、build tags、generated files 與 test selection 會影響 graph。
- `unused-direct-candidate` 只表示當次 `go list` 未觀察到，不代表可安全刪除。
- `examples/*.json` 是 generic examples，不是 universal default。
