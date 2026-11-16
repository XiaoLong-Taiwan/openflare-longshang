# v0.0.31 發布驗證

## 發布範圍

- 版本：`v0.0.31`，PATCH，日期：2026-11-16（UTC+8）。
- 範圍：Cloudflare 全分組健康同步任務與排程去重、資料管理緊湊總覽、手機／平板／桌面響應式介面及四語文案。
- 排除 `docs/CODE_DIFF_REPORT.md`、`docs/CODE_WIKI.md`、`docs/database-capacity-investigation-2026-11-03.md`，保留本機不提交。
- 新增 server migration 00026（PostgreSQL／SQLite）：保留最小 ID 排程設定並移除重複列；這是不可逆資料遷移，Down 不會復原刪除列。升級前須備份並於資料庫副本預演，詳見 `cloudflare-health-queue-dedup.md`。本次發布不執行正式資料庫遷移。
- 通用任務能力已登記 `backend/openflare/upstream-patches.md`，尚未回流 Wavelet。

## 發布前驗證

- 使用者交接紀錄：相關後端測試及 vet、前端 327 項測試與 static build、migration 42 cases 已完成；此項為交接資訊，本次以 Linux CI 保存獨立驗證證據。
- Linux 驗證新增 Asynq worker 與 cron 套件，確保單例任務回歸案例納入；既有流程亦包含 migration、Cloudflare、前端型別／ESLint／四語鍵檢查、全部前端測試及靜態匯出。
- 本機 `make format` 因 Go 未在 PATH 與 Windows shell 不相容失敗；`make code-check` 因 `env bash` 啟動失敗。不得視為完整 make 品質門檻通過。
- 功能提交、Linux run、release commit、五項發布工作、附件與映像核驗結果於實際完成後補記。

## 限制與部署狀態

- migration 測試將兩份 SQL 分別於 SQLite 執行，不代表真實 PostgreSQL 整合驗證。
- 不執行 force、amend、歷史改寫、既有標籤移動、應用程式部署或服務重啟。
- 發布前查詢儲存庫 secret 名稱清單為空；發布後仍須確認映像工作的 webhook 實際略過。
- main 推送可能觸發既有文件站發布，須與應用程式部署分開記錄。
- 尚未驗證跨平台二進位實際執行、正式登入流程、真實 Cloudflare DNS 寫入與正式服務健康狀態。
