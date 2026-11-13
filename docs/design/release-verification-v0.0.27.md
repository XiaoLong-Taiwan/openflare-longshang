# v0.0.27 發布驗證紀錄

## 來源與發布順序

- 儲存庫：`XiaoLong-Taiwan/openflare-longshang`。
- 功能提交：`c70026e103d535a315f5b5aef33514a05879300c`，先推送 `main`，包含本次程式、測試、四語文案、Swagger、中文文件與 `AGENTS.md`。
- 手動 [Linux Integration Validation 43460242234](https://github.com/XiaoLong-Taiwan/openflare-longshang/actions/runs/43460242234) 驗證上述功能提交，前後端工作均為 `completed/success`，成功後才建立發布提交。
- 獨立發布提交：`cd204f12b400a3b6418c917cea410b983f80030d`，標題 `chore(release): v0.0.27`，附完整中文更新說明；相對功能提交只將 Changelog 落版。
- `v0.0.27` 標籤指向發布提交；[GitHub Release](https://github.com/XiaoLong-Taiwan/openflare-longshang/releases/tag/v0.0.27) 名稱與標籤一致，`draft=false`、`prerelease=false`。
- Release 發布時間：`2026-11-13T16:17:16Z`，臺北／上海時間為 2026-11-14 00:17:16；Changelog 使用實際本地發布日期 2026-11-14。
- 排除並保留工作區中的 `docs/CODE_DIFF_REPORT.md`、`docs/CODE_WIKI.md`、`docs/database-capacity-investigation-2026-11-03.md`，沒有提交這三份調查文件。
- 全程新增提交及標籤，未 amend、force push 或重寫公開歷史。本紀錄於發布後補寫。

## 發布前驗證

- 補齊 Go 與 Swag 執行路徑後，實際執行 `make swagger` 成功，更新 `backend/docs/{docs.go,swagger.json,swagger.yaml}` 及 `docs/{swagger.json,swagger.yaml}`。生成器仍有既有常數解析與重複路由警告，非零結束的首次缺少 Go 嘗試不計為成功。
- 本機 `go test -count=1 -timeout=10m ./openflare/plugins/server/domain/dashboard ./openflare/plugins/server/domain/observability ./openflare/plugins/server/kernel/repository/...`：五個套件均通過。
- 翻譯生成、`pnpm exec tsc --noEmit --jsx preserve`、`pnpm exec eslint . --max-warnings 0`、`pnpm check:i18n`、`actionlint .github/workflows/integration-linux.yml` 與 `git diff --check` 通過。
- 本機 Windows 全量 Vitest 兩次均為 267 通過、1 項 Pages 測試超過原有 5 秒上限；同檔案單獨重跑為 17/17 通過。未提高逾時、略過測試或修改測試斷言。Linux 整合環境以原始 `pnpm exec vitest run` 完整通過 37 個檔案、268 項測試。
- Linux 整合流程亦通過 Go 模組下載與完整性驗證、工作流程所列相關後端套件測試、追蹤設定測試、前端型別／ESLint、翻譯檢查與靜態前端建置。
- `make format` 與 `make code-check` 在此 Windows GNU Make／shell 組合未完整成功；另直接執行對應 `goimports`、`pnpm format`、架構檢查與前端檢查。格式化造成的原本未修改檔案變更均還原，不納入功能提交。
- 全庫 `golangci-lint run` 在 Windows 因 Linux 系統呼叫型別與既有診斷失敗；以 `GOOS=linux` 執行 `golangci-lint run --new-from-rev=HEAD`（功能提交前）仍回報新增匯出函式 `SummarizeNodeAccessDaily` 缺少文件註解。未放寬 lint 設定，亦不宣稱全庫 lint 通過。架構檢查另直接執行成功。

## 資料庫與相容性

本次沒有 Schema、Migration、節點上報協定或正式資料操作。統計欄位採追加方式，保留既有 API、壓縮陣列索引與舊版資料回退。統計窗口、抽樣明細、平均值與資料新鮮度的判讀方式見 [儀表板統計口徑](./dashboard-metric-scope.md)。

## 驗證證據位置

本機證據目錄為 `C:\Users\CHIUYANQI\AppData\Local\Temp\openflare-v0.0.27-verification\`。Temp 可能被系統清理；遠端 Release、附件與工作流程網址為可重新核對的來源。
