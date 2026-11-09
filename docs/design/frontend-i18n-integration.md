# 前端依賴與俄文整合核對

核對日期：2026-11-09。上游基準：`217a7166`（OpenFlare v3.5.10）。本次不升版、不提交、不修改 changelog。

## 整合範圍

- 套用上游前端依賴與完整鎖檔：Next.js 16.4.1、Axios 1.20.0、Zod 4.6.6，以及上游 Vite、DOMPurify、Undici、follow-redirects 等間接更新。
- 保留原有 `zh-CN`、`zh-TW`、`en`，新增 `ru`；預設與不支援語言的 fallback 維持 `zh-TW`。
- 俄文支援 cookie、Accept-Language、瀏覽器語言、既有語言切換器、SSR 訊息載入、靜態輸出後 Provider 切換，以及共用日期／數字格式化。
- 未以整份上游頁面覆蓋本地客製。繁中及既有英文／簡中 fragment 不因俄文匯入而改寫。

## 翻譯缺口

依本地六組 fragment 的現有結構匯入 3,921 個上游俄文翻譯；235 個缺少上游對應鍵或參數不相容的鍵使用本地英文原文，確保介面不因缺鍵中斷。逐鍵來源、原因與 fallback 內容見 [frontend-russian-fallbacks.json](./frontend-russian-fallbacks.json)。這些 fallback 是已知翻譯待辦，不代表俄文翻譯完成。

上游另有 runtime、builtin-metadata、search-keywords 等訊息群組及大量呼叫端改造。本次沒有只加入無呼叫端的 catalog，也沒有直接覆蓋本地服務錯誤、WAF 驗證、內建 HTML、說明頁及搜尋邏輯。這些部分仍可能顯示中文。

runtime AST 掃描發現 718 個候選位置，分布於 78 個前端檔案；逐項路徑與行號見 [frontend-runtime-i18n-gaps.json](./frontend-runtime-i18n-gaps.json)。候選包含文案、搜尋別名、語言名稱與相容性識別字，不能等同 718 項確定缺陷。主要集中於 API／使用說明、條款與隱私頁、搜尋資料、WAF graph-validation、API client／錯誤處理及首頁客製文案。掃描不檢查後端，也不涵蓋所有英文硬編碼或動態產生文案，不能據此宣稱 runtime 完全國際化。

後續移植應逐域比對上游 runtime key 與本地客製語意，保留模板變數、持久化識別字及原有業務行為，再補齊四語翻譯與測試。

## 可重跑核對

在 `frontend` 執行：

```powershell
node scripts/import-upstream-russian.mjs 217a7166
node scripts/merge-i18n-fragments.mjs
node scripts/check-i18n-keys.mjs
node scripts/audit-runtime-i18n.mjs --write
node node_modules/vitest/vitest.mjs run --maxWorkers=2
npm run build
```

俄文匯入命令會重新產生六份俄文 fragment 與 fallback 清單；人工調整俄文前應先確認是否仍需重匯，避免覆蓋後續翻譯。runtime 掃描是盤點工具，只回報候選位置，不作為已完成翻譯的品質門禁。

整合 worktree 使用獨立 `node_modules`，原先指向主專案的 junction 保留在 worktree 同層的 `../openflare-integration-node_modules-original-20261109`。未刪除主專案依賴，未更新主專案的依賴安裝。

## 驗證結果

- `pnpm install --frozen-lockfile` 通過，鎖檔與上游相同。
- `npm run build` 通過 Next.js 16.4.1 Turbopack 正式編譯、TypeScript 檢查與 59 個頁面產生。
- `vitest run --maxWorkers=2`：33 個測試檔、239 項測試全部通過，包含俄文偵測、繁中預設、cookie 優先序、錯誤 cookie、數字格式及靜態頁面 Provider 切換。
- 四語 4,156 個訊息鍵一致，俄文與英文的命名 ICU 參數測試通過；全前端 ESLint 零警告門檻通過。
- 第一輪與 build 同時執行的測試有三項 Pages UI 超過 5 秒；未改測試時限或測試程式，build 結束後降低併行度重跑全套通過。
- 測試仍輸出 React `act`、jsdom 圖表尺寸及 Vite 未來 native config loader 相容性提醒；未將這些提醒宣稱為已修復。
- 未執行後端品質門禁、真實 Discord／資料庫／正式部署測試；本次僅修改前端與盤點文件。
