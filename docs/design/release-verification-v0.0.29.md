# v0.0.29 發布驗證

## 發布範圍

- 版本：`v0.0.29`，PATCH，日期：2026-11-15。
- 功能提交：`241b409f`，改善行動版管理介面、儀表板說明與憑證表單。
- 獨立 `chore(release): v0.0.29` 提交承載完整中文說明，標籤直接指向該提交。
- 排除 `docs/CODE_DIFF_REPORT.md`、`docs/CODE_WIKI.md`、`docs/database-capacity-investigation-2026-11-03.md`，保留本機文件。
- 無 API、資料庫結構、migration 或設定相容性變更。

## 發布前實測

在 `frontend` 以 `C:\Program Files\nodejs\node.exe` 執行：

- `node_modules/vitest/vitest.mjs run`：42 個測試檔、291 項測試通過。
- `node_modules/typescript/bin/tsc --noEmit --jsx preserve`：通過。
- `node_modules/eslint/bin/eslint.js . --max-warnings 0`：通過。
- `scripts/check-i18n-keys.mjs`：四語系 4,232 個鍵值通過。
- `git diff --check` 與提交前暫存差異檢查：通過。
- 靜態匯出：使用者提供此前已通過的結果，發布後另核對 Linux 建置。
- 測試含 Vite 設定、React act 與 jsdom 圖表尺寸警告，未造成測試失敗。

## 限制與後續核對

- `make format` 因 Go／Unix shell 工具缺失失敗；`make code-check` 因無法執行 Bash 架構檢查而中止，不宣稱完整品質門禁通過。
- 未執行真實瀏覽器的手機／桌面視覺驗證。
- 待標籤工作完成後補記五個 workflow、17 個附件 SHA256 與四個映像雙架構證據。
- 此次為 GitHub 產物發布，未手動部署正式服務、執行資料庫遷移或進行正式服務健康檢查；映像 workflow 的既有 webhook 行為另依實際日誌記錄。
