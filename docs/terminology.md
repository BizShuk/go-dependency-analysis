# go-dependency-analysis — 術語表 (Terminology)

本檔是領域名詞、狀態值與縮寫的單一定義來源。CLI 輸出、JSON schema 與文件使用同一組正名。

## 事實與判讀 (Facts and Judgements)

本工具最重要的區分是`工具可證明的事實`與`政策推論`。兩者在輸出中永遠分開標示。

| 術語 (Term) | 英文 (English) | 定義 (Definition) | 出處 (Source) |
| --- | --- | --- | --- |
| 工具事實 | `go-tool-fact` | 由 Go tooling 直接得出、不含判斷的資料：workspace 成員、選定版本、import 邊、環、未被觀察到的直接需求 | `pkg/discovery`、`pkg/graph` |
| 政策推論 | `policy-heuristic` | 依使用者提供的政策判定出的結論：分層違規、禁止邊、過重依賴路徑 | `pkg/policy` |
| 診斷 | Diagnostic | 一則帶 severity 與證據的判定結果 | `pkg/policy`、`pkg/report` |
| 證據 | Evidence | 支撐某則診斷的原始事實，JSON 輸出中保留精確模組路徑 | JSON schema v2 |
| 內建診斷 | Intrinsic Diagnostic | 不需 `--policy` 即會產生的診斷；政策可覆寫其 severity | `README.md` Analysis model |

## 圖模型 (Graph Model)

| 術語 (Term) | 英文 (English) | 定義 (Definition) | 出處 (Source) |
| --- | --- | --- | --- |
| 工作區 | Workspace | 由 `go.work` 或單一 `go.mod` 界定的分析範圍 | `--workspace` |
| 模組 | Module | 一個 `go.mod` 所定義的版本化單位 | `pkg/discovery` |
| 套件 | Package | 模組內的一個 import 單位，是 import 圖的節點 | `pkg/discovery` |
| 依賴家族 | Dependency Family | 語意化 major 路徑歸為一組的視角，例如 `example.com/lib` 與 `example.com/lib/v2` | `pkg/graph` |
| 強連通元件 | SCC | 互相可達的節點集合；用來偵測依賴環 | `pkg/graph` |
| 可達性 | Reachability | 從某節點出發能否抵達另一節點 | `pkg/graph` |
| 最短路徑 | Shortest Path | 兩節點間邊數最少的依賴路徑，用於解釋「為什麼會依賴到它」 | `pkg/graph` |

## 診斷類型 (Diagnostic Types)

| 術語 (Term) | 英文 (English) | 定義 (Definition) | 注意事項 |
| --- | --- | --- | --- |
| 多版本 | `multi-version` | 同一依賴被遞迴請求多個版本的彙整 | Go 的 MVS 仍只會為每個精確模組路徑選一個版本；這是`供審閱用`的訊號，不是錯誤 |
| 未使用直接依賴候選 | `unused-direct-candidate` | 宣告為直接需求但未觀察到 import 的模組 | `不是`移除指示；先確認 test-only import、build tag、平台檔、產生碼與 tool dependency |
| 分層違規 | Layer Violation | 政策宣告的分層被跨越 | 僅在提供 `--policy` 時出現 |
| 禁止邊 | Forbidden Edge | 政策明文禁止的依賴關係 | 僅在提供 `--policy` 時出現 |

## 輸出與契約 (Output and Contract)

| 術語 (Term) | 英文 (English) | 定義 (Definition) | 出處 (Source) |
| --- | --- | --- | --- |
| Schema 版本 | Schema Version | JSON 輸出的相容性版本號；breaking change 必須升版 | `schemaVersion`、`docs/specs/2026-07-20-api-json-compatibility.md` |
| 確定性輸出 | Deterministic Output | 相同輸入必定產生逐位元組相同的輸出 | `pkg/report` |
| 原子寫入 | Atomic Output | `--output` 走 `CreateTemp → render → Close → Rename` | `cmd/command.go` |
| 失敗門檻 | Failure Threshold | `--fail-on none\|warning\|error` 決定何種 severity 會讓 exit code 變 `1` | `--fail-on` |

## 結束碼 (Exit Codes)

| 值 | 意義 |
| --- | --- |
| `0` | 分析完成且未達失敗門檻 |
| `1` | 分析完成但 `--fail-on` 命中 |
| `2` | CLI 參數或政策無效 |
| `3` | discovery、Go tooling、render 或輸出失敗 |

## 硬性約束 (Hard Constraints)

- 不 import 被分析的 workspace packages；只透過 `os/exec` 呼叫 Go tooling。
- 無第三方相依 (stdlib-only)，且不進入被分析 repo 的依賴圖。
- 分析時以模組識別排除自身；需納入時用 `--include-tool-module`。
