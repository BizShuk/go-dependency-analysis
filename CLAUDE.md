# CLAUDE.md — go-dependency-analysis 技術脈絡

`go-dependency-analysis` 是獨立、read-only、stdlib-only 的 Go CLI。它以 Go tooling 建立 workspace module/package graph，並將工具可證明的 facts 與 policy heuristics 分開輸出。

## 技術基準

- Module：`github.com/bizshuk/go-dependency-analysis`
- Go：`1.26.0`
- Third-party dependencies：無
- Binary：`go-dependency-analysis`
- Outputs：text、JSON schema v1、Mermaid
- Exit codes：`0` success、`1` policy threshold、`2` usage/policy、`3` discovery/tool/output

## 結構

```text
go-dependency-analysis/
├── main.go                  # process exit boundary
├── cmd/command.go           # flags、orchestration、atomic output
├── internal/discovery/      # go.work/go.mod/go list facts
├── internal/graph/          # normalization、SCC、reachability、shortest path
├── internal/policy/         # strict JSON policy + diagnostics
├── internal/report/         # deterministic text/JSON/Mermaid
├── examples/agentsdk.json   # AgentSDK-specific optional policy
├── docs/memory/             # historical decisions
└── README.todo              # pending work
```

## 架構規則

- 不 import target workspace packages；只透過 `os/exec` 呼叫 Go tooling。
- 不加入被分析 repo 的 dependency graph；installed binary 或 explicit checkout 執行。
- 不修改 `go.mod`、`go.work`、policy、source 或 dependency versions。
- Self-exclusion 依 module identity `github.com/bizshuk/go-dependency-analysis`，不依 checkout directory 名稱。
- `go-tool-fact` 與 `policy-heuristic` 不可混用。
- 所有 node、edge、evidence 與 diagnostics 必須 deterministic sort。
- JSON schema 變動需升級 `schemaVersion`。
- Mermaid labels 使用雙引號，node ID 必須 collision-resistant。

## 驗證

```bash
go test ./... -count=1
go vet ./...
go build ./...

go run . --workspace /Users/shuk/projects/agentSDK/go.work --format text
go run . --workspace /Users/shuk/projects/agentSDK/go.work --format json --json-indent='  '
go run . --workspace /Users/shuk/projects/agentSDK/go.work --format mermaid
go run . --workspace /Users/shuk/projects/agentSDK/go.work --policy examples/agentsdk.json --fail-on error
```

## 注意事項

- Host Go version、GOOS/GOARCH、build tags、generated files 與 test selection 會影響 graph。
- `unused-direct-candidate` 只表示當次 `go list` 未觀察到，不代表可安全刪除。
- `examples/agentsdk.json` 是 domain-specific example，不是 universal default。
