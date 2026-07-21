# Context

目前 `--format text` 的 summary、diagnostics 與 evidence 都是純文字；在互動式 terminal 中不易快速辨識 severity 與區段。這次變更要在不破壞 pipe、redirect、`--output`、JSON schema v2、Mermaid，以及既有 public `report.RenderText` byte output 的前提下，加入可控 ANSI 色彩。

預期結果：CLI 預設只對可辨識的 terminal 自動上色；使用者可用 `--color always|never` 強制選擇；library caller 透過新增 options API 明確選擇色彩，既有 API 仍輸出 deterministic plain text。

## 實作計畫

1. **擴充 text renderer，但保留既有 public API 相容性**（`pkg/report/report.go`）
   - 保留 `RenderText(w, analysis, diagnostics)` signature，並讓它固定走無色模式，確保現有 caller 與 snapshot 的 bytes 不變。
   - 新增 exported `ColorMode`（`ColorNever`、`ColorAlways`）、`TextOptions`，以及 `RenderTextWithOptions(...)`；terminal/環境判斷不放進 public report package。
   - 將 ANSI 只包在既有顯示 token 周圍：summary labels 使用 bold cyan、`error` 使用 bold red、`warning` 使用 yellow、`info` 使用 blue、provenance/evidence 使用 dim；未知 severity 保持 plain。
   - 每個 colored span 立即 reset，diagnostic message、排序、換行與 evidence 內容維持原樣；writer error 照現有方式立即傳回。
   - 不改動 `RenderJSON*`、`Document`、schema version 或 `RenderMermaid`。

2. **在 CLI 層解析與決定色彩模式**（`cmd/command.go`，視可讀性新增 `cmd/color.go`）
   - 新增 `--color auto|always|never`，預設 `auto`；無效值在 discovery 前回報 usage error 並回傳 exit code `2`。
   - `auto` 僅在 `--format text`、未使用 `--output`、輸出 writer 是可辨識的 terminal 時啟用；其餘保守回退 plain text。
   - 以 stdlib-only `*os.File.Stat()` 與 `os.ModeCharDevice` 做 best-effort terminal 判斷；任何其他 `io.Writer`（包含 `bytes.Buffer`）或 `Stat` failure 都視為非 terminal。
   - `NO_COLOR` 只要存在，或 `TERM=dumb`，就停用 `auto`；明確 `--color always` 視為 opt-in 並覆蓋這兩個環境條件。
   - JSON/Mermaid 即使收到 `--color always` 也維持無 ANSI；text 分支以解析後的 `report.TextOptions` 呼叫新 renderer。
   - 維持既有 `--output` 的 `CreateTemp → render → Close → Rename` 原子寫入流程；`auto` 不對暫存檔誤判為 terminal，而 `always` 可依使用者明確要求寫入 ANSI。

3. **補齊 deterministic renderer 與 resolver 測試**（`pkg/report/report_test.go`、新增 `cmd/command_test.go`）
   - 鎖定 legacy `RenderText` 與 `ColorNever` 完全一致且不含 `\x1b[`。
   - 驗證 `ColorAlways` 對 summary、error/warning/info、provenance、evidence 套用固定 SGR/reset，同時原始文字、順序與換行不變；未知 severity 不誤套 severity color。
   - 以 table-driven tests 覆蓋 resolver：auto terminal、auto buffer、`--output`、`NO_COLOR` 空值、`TERM=dumb`、always override、never、未知 mode，以及 JSON/Mermaid always 仍為 plain。
   - 將 environment lookup 與 terminal detector 注入 resolver 測試，避免測試依賴執行機器是否為 TTY 或目前環境變數。
   - 保留既有 JSON schema v2 與 Mermaid deterministic tests，確認其他 renderer 沒有 regression。

4. **更新使用說明**（`README.md`）
   - 在 flags 表加入 `--color auto|always|never`。
   - 說明色彩只適用 text；auto 對 redirect/`--output`、`NO_COLOR`、`TERM=dumb` 的行為；`always` 的 override 語義；TTY detection 是 best-effort。
   - 加入互動 terminal、強制 ANSI consumer、CI/snapshot plain text 三種範例，並說明既有 `RenderText` 保持無色、新 API 才能 opt in。

## 關鍵重用與不變條件

- 重用 `cmd.Run` 現有 `io.Writer` 邊界與 format switch：`cmd/command.go`。
- 重用 `report.RenderText` 現有文字格式，不改資料模型：`pkg/report/report.go`。
- 保持 graph 與 diagnostics 上游 deterministic sorting，不在 renderer 重排資料。
- 維持 stdlib-only，不新增 `go.mod` dependency。

## 驗證

1. 執行完整品質檢查：
   - `go test ./... -count=1`
   - `go vet ./...`
   - `go build ./...`
2. 對單一 module 做 end-to-end smoke test：
   - `go run . --workspace ./go.mod --format text --color always`：應看見 ANSI 色彩。
   - `go run . --workspace ./go.mod --format text --color never`：輸出不含 ANSI。
   - redirect `--color auto` 至檔案：檔案不含 `\x1b[`。
   - `--format text --color auto --output <file>`：原子產生 plain-text 檔案。
3. 驗證非 text formats：
   - JSON 與 Mermaid 搭配 `--color always` 仍不含 ANSI；JSON 可正常 decode 且 `schemaVersion` 仍為 `2`。
4. 驗證 CLI contract：未知 `--color` 回傳 exit code `2`；色彩設定不改變 policy threshold exit codes。
