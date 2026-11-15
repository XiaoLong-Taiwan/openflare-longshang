# 未發布變更

## 2026-11-16

- 範圍：v0.0.30 發布準備與 Linux 驗證覆蓋。
- 內容：補齊認證、快取、維護、節點觀測與 Cloudflare 發布說明；Linux 工作擴及 admin/auth/upload 全子套件及兩種快取實作，以詳細測試日誌保存新增案例證據；三份調查文件排除發布提交。
- 首輪 Linux 驗證發現 HTTP 中介層測試快取替身漏實作 Take，補齊介面後重新驗證；尚未建立發布標籤。

- 範圍：未提交前端差異審查與明確問題修復，排除三份調查文件。
- 內容：補齊資料管理分頁、最佳化選項與中文 host-network 提示；移除新增節點文案的重複鍵，分開請求結構與請求數；維護確認說明改為僅執行已勾選項目，保留既有功能修改及後端檔案。
- 驗證：新增九個回歸案例，先重現失敗後通過；完整 Vitest 44 檔案、300 測試通過，TypeScript 與 ESLint 零警告通過。四語翻譯鍵共 4,255 個且一致；靜態匯出 59/59 頁通過。環境未提供 Browser 外掛及 Playwright，未執行真實瀏覽器、OAuth 登入或 Docker Compose 驗收；未提交、推送或升版。
- 範圍：未提交後端 diff 審查，涵蓋快取契約、OAuth state/provider、管理維護及 Cloudflare health。
- 內容：補上 auth 快取測試替身的 Take；Redis Take 消耗後廣播失效；provider ID 使用精確數字解碼並拒絕缺值；維護參數區分省略與 false，舊確認請求維持全範圍，明確未選範圍會拒絕；既有佇列的空物件工作維持相容；恢復管理員 SQL API，前端控制台仍移除；Cloudflare 單一成員入隊失敗不阻擋後續成員。未修改 Schema 或發布版本。
- 驗證：使用 TEMP/openflare-go-1.25.10/go/bin 的 Go 1.25.10；cache/cache_memory、admin、auth service/model 與快取案例、upload、Cloudflare、fleet/agent、observability/chwriter、repository/logstore 相關測試通過；相關 core、infra、domain 與 server 的 go vet 通過，Swagger 重建成功，Cordis 架構檢查零違規，git diff --check 通過。完整 auth 測試遇到既有 SQLite 連線未關閉導致 Windows 暫存檔刪除失敗；chwriter 去重測試首輪失敗，單獨連跑五次及後續整包通過，仍保留不穩定風險。make code-check 受 Windows env bash 啟動失敗阻擋，環境亦未提供 golangci-lint；未宣稱完整品質門檻通過。
- 相容性：SQL API 保留原管理員權限；新維護請求仍須 confirmed=true，只有 Execute 接受既有已排隊的精確 {} payload。通用修改已登記 upstream-patches，尚未回流 Wavelet。未執行真實第三方 OAuth 或 Cloudflare DNS 變更。

## 2026-11-15

- 範圍：節點清單、節點詳情及行動裝置安全區。
- 內容：整理節點清單批次操作與網域摘要；節點觀測依節點、時間範圍、筆數及類型分隔快取；健康事件、請求結構與網域流量分區顯示，部署範例採 Linux host-network Compose。
- 驗證：相關 observability、Cloudflare、fleet/agent、logstore、admin 與 upload 共用套件測試通過；Compose 未在 Docker daemon 驗證。
- 範圍：資料管理與 OAuth provider。
- 內容：資料管理頁重排概覽、資料表、快取、維護與匯出，清理操作可選資料類別並確認；移除任意 SQL 控制台前端入口與無引用執行服務；OAuth UI 支援 Google、Discord、GitHub provider 預設與綁定保護。
- 驗證：前端 TypeScript、production build 通過；俄文目錄鍵一致性測試通過。全量 Vitest 於修復前發現俄文缺鍵，補齊後未重跑全量；ESLint 嚴格零警告因節點詳情 Hook 警告失敗，已移除該警告但尚未重跑。未完成登入後瀏覽器驗收。
- 範圍：資料管理與 OAuth 登入綁定。
- 內容：重排資料管理分區，SQL 控制台改為可選入口；OAuth 認證源支援 Google、Discord、GitHub，沿用既有 state/session 綁定並加入 PKCE 與一次性 state 取用。
- 驗證：待執行後端 Go 測試、前端 TypeScript 與 ESLint；未修改共享 Changelog。

- 範圍：代理重試、憑證任務追蹤、網站健康錯誤分類、節點批次操作、代理分頁搜尋、WAF 請求模擬。
- 內容：新增代理重試設定與分頁搜尋；顯示憑證續期任務；分類源站健康錯誤；提供節點批次同步與 Agent 更新；提供不發送外部請求的 WAF 離線模擬與管理介面。
- 驗證：相關 Go 回歸測試、前端 Cloudflare 與儀表板測試、TypeScript、ESLint 及 i18n 檢查通過；新增 PostgreSQL／SQLite migration，未在本機正式資料庫執行。
- 範圍：Cloudflare 同步事件、節點健康與排程。
- 內容：全量同步調整為每 4 小時；心跳只在節點 IP、在線狀態、OpenResty 健康狀態變化或離線恢復時派送；新增 30 秒本地漂移檢查，透過原子領取避免重複排隊，並保留節點故障防抖切換。
- 驗證：新增 Cloudflare 健康漂移、防抖切換、排隊失敗重試及節點心跳變更測試；Go 執行環境未安裝，尚未執行測試。
- 範圍：節點詳情健康事件、請求結構、網域流量與 Linux Compose 部署。
- 內容：新增節點詳情獨立區域；存取紀錄聚合嚴格限定 `node_id`；Docker 節點安裝指令改用 host network Compose，並說明非 privileged 與 Linux 限制。
