# 上游整合逐項核對（2026-11-09）

## 核對範圍與判定方式

- 整合 worktree：`C:\Users\CHIUYANQI\AppData\Local\Temp\openflare-integration-20261109`。
- 分支：`integrate/openflare-3.5.10`；HEAD：`c0e4598c5aba111491d08cd1512e51d419c559a1`（`chore(release): v0.0.25`）。
- 上游：`Rain-kl/OpenFlare`；比較範圍為 `v3.5.4..upstream/main`，不包含 v3.5.4 本身。
- 起點：`9f79fb9969a3eb0c40105cb99b42ae86e370fa27`；終點：`217a71662fd80f169584f42150dfdd7aac8180e8`（v3.5.10）。本次使用現存遠端追蹤參照，沒有 fetch；結論固定於此終點。
- `git log --reverse` 共 74 筆，包含 merge、依賴、文件與發布提交；下表每筆均列出，不以功能數代替提交數。Merge 另列來源，避免把相同內容算成兩次移植。
- 核對對象為「目前工作目錄」，包含本次開始前已有的已修改及未追蹤檔案。下列「已移植」不代表已提交、已發布或測試通過。
- 本次只新增本文件；不修程式、不改設定、不執行 migration、不升版、不變更 changelog、不提交或部署。既存 changelog 與其他未提交內容屬於先前工作。

| 分類 | 意義 |
| --- | --- |
| 既有 | HEAD 已有對應能力；依實作語意判斷，非宣稱原提交已 cherry-pick。 |
| 已移植 | 本次開始前的工作目錄已有對應實作及相關檔案；仍需整體驗證與提交。 |
| 保留差異 | Cordis 分層、本地多節點／語言／版本／文件設計與上游不同；不直接覆蓋。未有明確取捨依據者仍列缺口。 |
| 缺口 | 已確認未覆蓋的上游行為，或僅有局部能力；列出證據及後續驗收條件。 |

## 路徑與證據索引

表內使用下列證據代號。完整路徑以 repository 根目錄為準；省略 `backend/openflare/plugins/` 的 `server/`、省略 `backend/openflare/` 的 `share/`，以及同列後續省略的共同目錄，均沿用該列上下文。上游 `internal/apps`、`internal/model`、`pkg/render` 不能直接對應本地同名目錄，需要比對 Cordis 插件中的實作。

| 代號 | 本地證據與核對重點 |
| --- | --- |
| E01 | `backend/plugins/drivers/driver_http/frontend.go`：`assetCandidates` 只有原路徑、`.html`、`index.html` 與根 shell，沒有 `cloudflare/groups/1.html`、`.txt`、`__next.*.txt` 動態模板映射。 |
| E02 | `backend/openflare/plugins/server/domain/cloudflare/logics.go`：`buildGroupItem` 對主、活動、備援及多節點查詢錯誤直接返回；`MoveMembers`、`RemoveMembers` 直接迭代輸入 ID，沒有上游 `uniqueIDs`。`routers.go`、`httpapi/register_cloudflare.go` 與 `kernel/repository/openflare_cloudflare.go` 已有批次 API／持久化。 |
| E03 | `frontend/app/(main)/cloudflare/groups/[id]/page-client.tsx`：已有勾選、目標群組、批次移動／移出與 pending 按鈕；採內嵌 Select，沒有上游獨立 `member-move-dialog.tsx`。`frontend/lib/services/openflare/cloudflare.service.ts` 已提供批次呼叫。 |
| E04 | `frontend/app/(main)/waf/components/ip-group-view-dialog.tsx`：`entries.map` 直接列出所有 IP，沒有查詢輸入及 filtered entries。WAF 其他位置的群組名稱搜尋不能替代此功能。 |
| E05 | `backend/plugins/infra/storage/objectstore/webdav.go`：邏輯 key 與遠端路徑分離，`key()` 避免重加 basePath；沿用平台 storage，不能複製上游 objectstore 另建入口。 |
| E06 | `backend/openflare/plugins/agent/agent/runner.go`、`state/state.go` 與相關測試：已有離線啟動、設定套用失敗重試及持久化。`nginx/manager.go` 的 `Reload` 仍是遇到 invalid PID／未執行訊息便直接 start，`CheckHealth` 只跑配置檢查；未有上游 `runtime_process*.go` 程序辨識及復原流程。 |
| E07 | `backend/openflare/plugins/server/kernel/model/openflare_proxy_route.go` 沒有 `EnableHTTP2`；`backend/openflare/share/render/openresty/render.go` 的 TLS 與 Pages 路徑固定輸出 `http2 on`。 |
| E08 | `backend/openflare/share/render/openresty/types.go` 及 `server/kernel/model/openflare_option.go` 預設模板沒有 `gzip_vary on`／擴充 `gzip_types`；`server/migrate/{postgres,sqlite}/00001_initial.sql` 的 request buffering seed 仍為 `false`。`frontend/app/(main)/performance/page.tsx` 已有開關，且路由 `proxy_config` 已可覆寫 buffering。 |
| E09 | `backend/openflare/plugins/server/kernel/model/openflare_cloudflare.go`、`domain/cloudflare/{logics,reconcile}.go`、`migrate/*/00002_cloudflare_multi_node.sql`：本地採多節點優先序與 managed records；沒有上游 `TargetMode`／`RecordType`／`RecordContent` 自訂目標欄位。離線切換與恢復已有本地實作。 |
| E10 | `backend/openflare/plugins/server/migrate/{postgres,sqlite}` 使用插件內 00001…00019 序號；`backend/plugins/domain/admin/migrations/*/00001_initial.sql` 已 seed `log_db_migration`，觀測域另有 upsert 及凍結旗標回歸測試。不能套用上游舊全域時間戳 migration 更名。 |
| E11 | `server/migrate/*/00012_trusted_proxy_cidrs.sql`、`server/kernel/model/openflare_option.go`、`share/render/openresty/render.go`：已有 Cloudflare 可信 CIDR 與本地追加／覆蓋／關閉能力。 |
| E12 | `backend/plugins/domain/auth/parse_userid_test.go` 及 auth 實作：已有字串／數值第三方識別處理，保持外部識別與本地 user ID 分離；本地 changelog v0.0.18 記錄 Casdoor 相容修復。 |
| E13 | `backend/openflare/plugins/server/domain/waf/ip_group_sync.go` 兩處以 `iputil.IsPublicString` 篩選，`ip_group_sync_test.go` 及 `share/geoip/iputil` 有對應測試。 |
| E14 | `backend/plugins/infra/database/postgres.go`、`sqlite_test.go`，`backend/plugins/domain/admin/{handler,service,repository}/db.go` 與 `repository/database_export_test.go`：工作目錄已有 WAL 與 `VACUUM INTO` 快照移植。通用平台補丁已登記於 `backend/openflare/upstream-patches.md`，仍待回流 Wavelet。 |
| E15 | `docs/design/frontend-i18n-integration.md`、`frontend-russian-fallbacks.json`、`frontend-runtime-i18n-gaps.json`：四語、俄文、繁中預設與已知缺口；`frontend/i18n/{config,client,format}.ts`、Provider、fragment、匯入／檢查腳本可對照。未完整導入上游 runtime、builtin-metadata、search-keywords 與後端 catalog。 |
| E16 | `backend/openflare/plugins/agent/{agent,config,heartbeat,httpclient,localfile,state}`、`server/domain/fleet/agent/registration.go`、`kernel/repository/openflare_node_registration.go`、`share/protocol/agent.go`：工作目錄已有相容註冊、持久 registration key、確定性身分、重試與原子寫檔及測試。 |
| E17 | `server/domain/fleet/node/list.go`、`kernel/repository/openflare_node_page.go`、`httpapi/register_node.go`、`server/migrate/*/00018_node_list_indexes.sql`；前端 nodes 控制列、詳情及 `node.service.ts`：已有分頁、搜尋、篩選與單筆詳情移植，保留舊列表及本地在線狀態判定。 |
| E18 | `backend/openflare/share/edge/heartbeat/autoupdate.go`、`updater/service.go` 及測試：工作目錄已有 `CanCheck`、自動更新節流與失敗退避，保留本地 GitHub 限流及校驗流程。 |
| E19 | `server/domain/cloudflare/{logics,reconcile,types}.go`、`ttl_test.go`、`ttl_multi_node_test.go`、`server/migrate/*/00019_cloudflare_group_ttl.sql`，前端 group dialog／detail、四語訊息：TTL 已適配本地多節點，橙雲自動、舊值 300 秒、省略值保留。 |
| E20 | `backend/cmd/{root,trace_config,trace_config_test}.go`、`backend/pkg/trace/{trace,trace_provider,trace_provider_test}.go`、`manifest/config/config.default.yaml`：工作目錄已有 HTTP protobuf exporter，協定由 ConfigSource／ConfigRegistry 解決；保留標準環境變數優先序及 SDK v1.45.0，補丁待回流 Wavelet。 |
| E21 | `server/domain/site/zone`、`server/domain/option/uptimekuma/sync.go`、`server/kernel/{model,repository}`、`share/render/openresty/wildcard_test.go`、Agent sync、Cloudflare 及 snapshot 的 wildcard 測試：工作目錄已有通配符、IDN 匯入、日誌篩選與 Kuma 適配，前端建立／選擇器同步。 |
| E22 | `frontend/package.json` 與 `frontend/pnpm-lock.yaml`：本地鎖檔對 `upstream/main` 無差異；Next 16.4.1、Axios 1.20.0、Zod 4.6.6、Vite 8.3.5、Sharp 0.35.5、js-yaml 4.3.2、DOMPurify 3.4.17、Undici 8.12.3、follow-redirects 1.16.2、source-map-js 1.2.2、brace-expansion 1.1.21 已覆蓋。 |
| E23 | `backend/go.mod`：gRPC v1.83.2 在 HEAD 已有；go-jose/v4 仍為 v4.1.4，未達上游 v4.1.6。`docs/pnpm-lock.yaml` 仍為 markdown-it 14.1.1、Vite 7.3.3、source-map-js 1.2.1。 |
| E24 | `manifest/docker/Dockerfile.agent`：仍用 `openresty/openresty:alpine`，perl 留在 runtime，MMDB 與 entrypoint 仍用分離 RUN 設定權限，未套用臨時 build-deps 清理及 COPY 權限合併。 |
| E25 | `README.md`、`docs/config.ts`、`docs/index.md`、`docs/design/index.md`、`docs/guide/index.md`、`docs/changelog/index.md`：保留 fork 品牌、Cordis 建置路徑與 v0.0.x 發布紀錄；上游原始發布說明不能直接視為本地已交付。 |

## 74 筆提交核對表

順序與 `git log --reverse --format="%h %s" v3.5.4..upstream/main` 一致。表內「缺口」的完整條件見後節；文件與 merge 提交仍保留獨立列。

| # | Commit | 上游內容 | 判定 | 本地結果／證據 |
| --- | --- | --- | --- | --- |
| 01 | `a0324dc6` | gRPC 1.83.2 | 既有 | HEAD 已為 v1.83.2，E23。 |
| 02 | `501c7543` | js-yaml 4.3.2 | 已移植 | 最終前端鎖檔已覆蓋，E22。 |
| 03 | `71a715a6` | Sharp 0.35.4 | 已移植 | 最終 0.35.5 覆蓋中間版，E22。 |
| 04 | `a2404a50` | 非 1 ID 的 Cloudflare 靜態詳情頁 | 缺口 | Cordis fallback 未做動態 HTML／RSC 模板映射；G01、E01。 |
| 05 | `1d4533f1` | 群組回應容忍已刪節點 | 缺口 | 主／活動／備援／多節點任一查無節點仍返回錯誤；G02、E02。 |
| 06 | `65bc7d9b` | v3.5.5 發布紀錄 | 保留差異 | 本地 v0.0.x；不複製發布版本，E25。 |
| 07 | `bf1e77a8` | Agent 映像縮層與權限最佳化 | 缺口 | build-deps 與 COPY 權限調整未套用；G03、E24。 |
| 08 | `1fe27630` | OpenResty alpine-slim | 缺口 | runtime 仍為 alpine；G03、E24。 |
| 09 | `a49d07e2` | 第二筆 v3.5.5 發布紀錄 | 保留差異 | 與 06 分開盤點，本地不消耗版本，E25。 |
| 10 | `e1753f86` | 合併 PR #30（gRPC） | 既有 | merge 對應 01，無須再次移植；E23。 |
| 11 | `fff4f425` | 合併 PR #31（js-yaml） | 已移植 | merge 對應 02，E22。 |
| 12 | `96abbf18` | 合併 PR #32（Sharp） | 已移植 | merge 對應 03，E22。 |
| 13 | `aad059ab` | WebDAV logical key／basePath | 既有 | 平台 storage 已處理 prefix 去重；E05。保留平台入口。 |
| 14 | `805eea5b` | Cloudflare 移動與 IP 搜尋設計 | 保留差異／缺口 | 本地批次 UI／多節點不同；設計不能當作 IP 內容搜尋已實作，E02–E04、G04。 |
| 15 | `3867841a` | 同功能實作計畫 | 保留差異／缺口 | 上游計畫未直接搬入；本地差異由本文件追蹤，E02–E04。 |
| 16 | `079fa7ee` | Cloudflare 移動與批次邏輯 | 既有 | `MoveMembers`／`RemoveMembers` 與 repository 已存在；去重缺口另列 21，E02。 |
| 17 | `b1626d06` | 移動與批次 API 路由 | 既有 | 本地插件 HTTP API 已註冊批次路由，E02。 |
| 18 | `4d52a5b0` | WAF 群組內 IP 搜尋與篩選 | 缺口 | 詳情仍全量 entries.map；群組名稱搜尋不等價，G04、E04。 |
| 19 | `11c8e5c7` | 群組詳情移動與批次 UI | 既有／保留差異 | 已有內嵌多選及目標 Select；未照搬獨立 dialog，E03。 |
| 20 | `2b3be6f3` | 批次操作／IP 搜尋 changelog | 保留差異／缺口 | 本地已記批次操作，但不能把名稱搜尋宣稱為 IP 內容搜尋完成，E04、E25。 |
| 21 | `c3606bc6` | 批次 ID 去重、pending 禁關閉 | 缺口／保留差異 | API 沒有 uniqueIDs；本地沒有 move dialog，該關閉防護不直接適用；G05、E02–E03。 |
| 22 | `737b3ae7` | Undici 8.11.2 | 已移植 | 最終 8.12.3 覆蓋，E22。 |
| 23 | `17ce9d05` | v3.5.6 發布紀錄 | 保留差異 | 本地版本及交付範圍獨立，E25。 |
| 24 | `5471a633` | 合併 PR #33（Undici） | 已移植 | 對應 22，E22。 |
| 25 | `de793010` | Axios 1.20.0 | 已移植 | package 與鎖檔已對齊，E22。 |
| 26 | `04615e24` | DOMPurify 3.4.16 | 已移植 | 最終 3.4.17 覆蓋，E22。 |
| 27 | `09469780` | 合併 PR #34（Axios） | 已移植 | 對應 25，E22。 |
| 28 | `588298c5` | 合併 PR #35（DOMPurify） | 已移植 | 對應 26，E22。 |
| 29 | `04cda060` | Next 16.3.6 | 已移植 | 最終 16.4.1 覆蓋，E22。 |
| 30 | `1db4c068` | Sharp 0.35.5 | 已移植 | 最終版本對齊，E22。 |
| 31 | `6c15c11e` | 前端 source-map-js 1.2.2 | 已移植 | 僅前端已對齊，docs 另列 66，E22–E23。 |
| 32 | `2960a4dd` | 設定套用失敗重試 | 既有 | 本地持久重試及離線啟動已有；不能等同 PID 完整復原，E06。 |
| 33 | `0bebca29` | brace-expansion 1.1.21 | 已移植 | 最終鎖檔覆蓋，E22。 |
| 34 | `7ad25f61` | 每條路由可停用 HTTP/2 | 缺口 | 沒有 EnableHTTP2 欄位，render 固定 on；G06、E07。 |
| 35 | `28b7f7d7` | docs markdown-it 14.3.2 | 缺口 | 仍 14.1.1；後續目標 14.3.3，G07、E23。 |
| 36 | `e49e0ad2` | gzip 類型與預設 request buffering | 缺口 | 模板缺 gzip_vary／gzip_types，seed false；已有 buffering 能力不等於預設行為對齊，G08、E08。 |
| 37 | `fd0e9acc` | Cloudflare 自訂目標與節點 failover | 既有／保留差異／缺口 | 多節點切換與回切已有；自訂 A／AAAA／CNAME 目標未覆蓋，G09、E09。 |
| 38 | `4e584d2e` | 修正重複 goose 版本 | 保留差異 | 本地插件序號制無同名時間戳衝突；不重命名已套用 migration，E10。 |
| 39 | `d1978a85` | seed 日誌遷移凍結旗標 | 既有 | admin seed 與觀測域 upsert 已覆蓋，E10。 |
| 40 | `5e86f63c` | v3.5.7 發布紀錄 | 保留差異 | 不改本地版本，E25。 |
| 41 | `d979b747` | 效能頁 request buffering 開關 | 既有 | performance 頁及路由覆寫已存在；預設值另列 36，E08。 |
| 42 | `7483897d` | 第二筆 v3.5.7 發布紀錄 | 保留差異 | 不重複移植發布紀錄，E25。 |
| 43 | `6f637151` | 預設信任 Cloudflare IP | 既有／保留差異 | CIDR 與 append／replace／off 客製已覆蓋，E11。 |
| 44 | `fec9bd18` | Casdoor 字串 user ID callback | 既有 | auth 插件已有字串／數值識別處理與測試，E12。 |
| 45 | `96a17ae3` | invalid PID runtime 完整復原 | 缺口 | 只有錯誤後直接 start，沒有辨識存活 master／listener 的程序復原；G10、E06。 |
| 46 | `7c2304d4` | v3.5.8 發布紀錄 | 保留差異 | 本地獨立發布紀錄，E25。 |
| 47 | `53f2e58c` | README 介紹與部署說明 | 保留差異 | 保留 fork 品牌與 backend／manifest 入口；文件逐字差異不代表功能缺失，E25。 |
| 48 | `cf7613da` | 文件首頁、設定與索引 | 保留差異 | 實際改 docs/config.ts、index.md、design/index.md、guide/index.md；不能只依 commit 標題當作 README 改動，E25。 |
| 49 | `0e1ce40c` | 整理 changelog | 保留差異 | 實際只改 docs/changelog/index.md；保留本地交付歷史，E25。 |
| 50 | `02a2b79c` | 自動 IP 群組排除內網 IP | 既有 | 自動累加與既有候選清理均有公開 IP 過濾，E13。 |
| 51 | `ada05505` | SQLite WAL 與一致快照 | 已移植 | 平台層已有適配，待回流 Wavelet；E14。 |
| 52 | `65a96670` | 英文 runtime 與系統 UI 國際化 | 已移植（部分）／缺口 | admin system 有局部調整；大量 runtime／後端 catalog／呼叫端未導入，G11、E15。 |
| 53 | `b4e94076` | 俄文與語系強化 | 已移植（部分）／保留差異／缺口 | 四語與俄文已導入，保留繁中預設；fallback 及 runtime 未齊，G11、E15。 |
| 54 | `e2acbecb` | go-jose/v4 4.1.6 | 缺口 | backend/go.mod 仍 v4.1.4；G12、E23。 |
| 55 | `6a2503cf` | docs markdown-it 14.3.3 | 缺口 | docs 鎖檔仍 14.1.1；G07、E23。 |
| 56 | `32cec2ac` | docs Vite 7.3.7 | 缺口 | docs 鎖檔仍 7.3.3；G07、E23。 |
| 57 | `47c47fea` | 前端 Vite 8.3.5 | 已移植 | 最終前端鎖檔對齊；不等於 docs Vite 已更新，E22。 |
| 58 | `bcdb2775` | DOMPurify 3.4.17 | 已移植 | 最終前端鎖檔對齊，E22。 |
| 59 | `5db05bd1` | Undici 8.12.3 | 已移植 | 最終前端鎖檔對齊，E22。 |
| 60 | `babead59` | Zod 4.6.6 | 已移植 | package 與鎖檔對齊，E22。 |
| 61 | `d7c7ccb2` | follow-redirects 1.16.2 | 已移植 | 最終前端鎖檔對齊，E22。 |
| 62 | `1ca03df0` | Next 16.4.1 | 已移植 | package 與鎖檔對齊，E22。 |
| 63 | `72d67c80` | discovery 相容與冪等註冊 | 已移植 | Agent／server／protocol／repository／原子寫檔與回歸測試已有，E16。 |
| 64 | `3c7b347d` | 節點分頁與單筆詳情 | 已移植／保留差異 | 前後端與雙方言索引已有；保留舊 API、本地狀態與客製資訊，E17。 |
| 65 | `5f7e6d89` | 合併 PR #52（英文預設與俄文） | 已移植（部分）／保留差異／缺口 | merge 含 52–53，不能再算一套完成；繁中預設保留，G11、E15。 |
| 66 | `8867c822` | docs source-map-js 1.2.2 | 缺口 | docs 仍 1.2.1；G07、E23。 |
| 67 | `b60546a9` | 自動更新節流與指數退避 | 已移植 | 共用 edge heartbeat／updater 已適配及有測試，E18。 |
| 68 | `18dee29b` | v3.5.9 發布紀錄 | 保留差異 | 不將未提交移植寫為本地已發布，E25。 |
| 69 | `e0ba758a` | 分組自動／手動 TTL | 已移植／保留差異 | 適配多節點、舊群組 300 秒及橙雲自動；migration 使用 00019，E19。 |
| 70 | `b281b65e` | OTLP/HTTP trace exporter | 已移植／保留差異 | 以平台 trace 實作、OTel 1.45.0 對齊既有 SDK，E20。 |
| 71 | `acd63ff4` | 啟動設定解析 exporter 協定 | 已移植／保留差異 | 使用 Cordis ConfigSource 與 YAML，保留 trace 專用環境變數優先，E20。 |
| 72 | `8d8b3e1e` | 合併 PR #64（OTLP） | 已移植 | merge 含 70–71；平台補丁待回流 Wavelet，E20。 |
| 73 | `9a51a682` | 通配符域名與相關適配 | 已移植 | 模型、API、匯入、快照、DNS、日誌、Kuma、render 與前端已有改動／測試，E21。 |
| 74 | `217a7166` | v3.5.10 發布紀錄 | 保留差異 | 只作上游核對終點，不把本地改為 v3.5.10，E25。 |

## 缺口與後續驗收

以下 12 組是初次靜態核對確認的差距；後續整合狀態更新於本表及文末各交付節。前面的證據索引與 74 筆提交核對表保留初次盤點狀態，不能當作最新程式狀態；最前方「本次只新增文件」及靜態驗證限制亦只適用初次盤點。

| 編號 | 缺口與影響 | 後續驗收條件 |
| --- | --- | --- |
| G01 | 已整合：Cordis HTTP driver 提供通用正整數 ID 靜態模板映射，群組詳情從 pathname 讀取 ID 並保留舊 query 入口。 | 17 個 HTTP 案例、driver 套件與 6 項前端測試通過；通用補丁已登記。真實 embed 產物與瀏覽器驗收受前端 build 阻擋，見文末。 |
| G02 | 已整合：主、活動、備援及優先序節點缺失時保留 ID 並回傳降級資料，同步略過缺失節點。 | 列表、詳情、真正 DB 錯誤及健康備援回歸通過；見 G02／G05／G09 交付節。 |
| G03 | 已適配 Dockerfile：alpine-slim、臨時 build-deps、清理 OPM 快取與 COPY 權限。 | 保留本地 Cordis 建置、MMDB、低權限使用者與 capabilities；Docker daemon 不可用，映像建置及啟動尚未驗收，見 G03／G06／G10 交付節。 |
| G04 | 已整合：WAF IP 群組詳情支援 IP／CIDR 文字搜尋，忽略大小寫與首尾空白，顯示符合數量。 | 9 項元件測試通過，涵蓋清空、無結果、外部關閉／切換群組重設、刪除後更新、自動群組欄位、訂閱提示與四語文案；保留原本刪除流程。 |
| G05 | 已整合：批次移動／移出按首次順序去重，統一用於清理、持久化、結果與派送；操作中鎖定選取及兩個批次動作。 | 重複 ID、派送次數、移動後重送拒絕、前端 pending 回歸通過；見交付節。 |
| G06 | 已整合：每條路由支援 HTTP/2 開關，模型、API、repository、00020 migration、快照、共用 Agent renderer 與四語 UI 對齊。 | CRUD、省略值／明確 false、舊快照、Pages／SW／逐域名代理覆蓋、HTTP/3 獨立及跨分區保存測試通過；真實 TLS 協商與瀏覽器操作未驗收。 |
| G07 | 已整合：docs 獨立鎖檔對齊 markdown-it 14.3.3、Vite 7.3.7、source-map-js 1.2.2，連帶 linkify-it 5.0.2。 | pnpm 10.10.0 frozen install 與 GITHUB_PAGES=true 文件 build 通過；只保留上游鎖檔的 libc／deprecated metadata 差異。 |
| G08 | 已整合／保留差異：伺服器與 Agent 內建模板加入 gzip_vary 與上游 MIME 清單；依使用者要求保留 request buffering=false，維持請求串流相容預設。 | 渲染與設定快照套件通過，覆蓋新／舊／自訂模板、Gzip 開關、全域 buffering 與路由繼承／覆寫；不重寫既存設定、不修改 seed、不需 migration（00022 未使用）。真實 OpenResty 串流與壓縮實測尚未執行。 |
| G09 | 已整合：群組可選多節點或自訂 A／AAAA／CNAME；保留多節點優先序與 managed records，以 node_id=0 追蹤單一自訂目標。 | 00021 雙方言 migration、模型／API／同步／清理／任務／四語 UI 與回歸已加入；真實 Cloudflare、PostgreSQL 與部署驗收未執行。 |
| G10 | 已整合程序辨識、PID 修復、HUP reload、受管 worker 排空後重啟；保留本地 runtimeMu、回滾與持久重試。 | 程序替身、合成 /proc／ps、安全拒絕與 PID reuse 測試通過，Linux／Darwin 編譯通過；真實 OpenResty、Linux 訊號及端口占用復原尚未驗收。 |
| G11 | 國際化部分完成：俄文有英文 fallback；runtime、內建 metadata／搜尋／HTML／錯誤、後端 task／push／response／PoW／腳本等未逐域覆蓋。 | 依既有 i18n 清單逐呼叫端移植，保留繁中與客製文案、ICU 參數及持久化識別字，補四語與後端驗證。不得以「可切俄文」宣稱完整英文 runtime。 |
| G12 | 已整合：go-jose/v4 升級 v4.1.6，保留既有平台依賴與 module。 | Auth service、go-oidc、go-jose／JWT 測試與 go mod verify 通過；完整品質門禁仍有限制，見文末。平台依賴補丁已登記待回流。 |

國際化既有盤點記錄 3,921 個上游俄文翻譯、235 個英文 fallback，以及 78 個檔案的 718 個 runtime 候選位置。數字引用自 [前端整合核對](./frontend-i18n-integration.md)，本輪未重跑掃描；候選不等同確定缺陷，亦不包含後端完整覆蓋。

## 必須保留的整合邊界

- Cordis 插件架構、`Wavelet` module、平台 storage／DB／設定／任務／trace 契約保持本地入口。SQLite 與 OTLP 通用修改已列 [待回流補丁](../../backend/openflare/upstream-patches.md)，尚不能宣稱 Wavelet 已完成回流。
- Cloudflare 使用多節點優先序與一成員多 managed records；自訂目標作為另一種群組模式接入相同受管記錄流程，TTL 與通配符保留多節點語意。
- 節點列表保留舊 API、客製資料及 WebSocket 連線狀態判斷。分頁與單筆查詢不能退回只靠過期 heartbeat 判離線。
- 語言保留 `zh-TW`、`zh-CN`、`en`、`ru`，預設與不支援語言 fallback 為 `zh-TW`；英文預設的上游產品決策不自動套用。
- 日誌每日摘要、七天明細、容量上限、outbox／checkpoint、清理及重送是本地既有能力；通配符日誌查詢適配不能以整份上游 repository 覆蓋。
- 上游時間戳 migration 轉為插件內序號；工作目錄目前新增 00018／00019 只代表已有 SQL，不表示資料庫已套用。發布紀錄與 fork 品牌沿用本地體系，不匯入 v3.5.x 作為本地版本。

## 本次驗證與限制

本次執行 Git 範圍／歷史／差異與檔案靜態核對，並驗證本文件的 74 列與固定比較範圍一一對應、無重複或遺漏，證據中明列的路徑存在。`frontend/pnpm-lock.yaml` 對上游無差異，以鎖檔為依賴覆蓋證據，不代表所有上游頁面／執行期邏輯相同。

```powershell
git rev-parse v3.5.4 upstream/main HEAD
git rev-list --count v3.5.4..upstream/main
git log --reverse --format="%h %s" v3.5.4..upstream/main
git show <commit> -- <upstream-path>
git diff --numstat upstream/main -- frontend/pnpm-lock.yaml
git diff --check -- docs/design/upstream-integration-2026-11-09.md
```

沒有執行 Go／前端全套測試、make code-check、Docker 建置、真實 Cloudflare DNS／Agent／OTLP／資料庫或正式部署測試。原因是本次為文件盤點且明確不修改程式；本文件的「既有／已移植」只陳述靜態證據。先前 i18n 文件中的 build、239 項測試等結果屬於該文件原有紀錄，未作為本次重新執行的結果。

此清單可作為後續補漏與整體驗收的依據；完成已移植項目的測試、處理或明確接受上述缺口之前，不能宣稱與 v3.5.10 完全等價或已可發布。

## G02／G05／G09 後續整合交付

於指定 integration worktree 對照同層上游 `1d4533f1`、`c3606bc6`、`fd0e9acc`，讀取 `new-api`、`database-migration`、`new-async-task`、`go-testing`、`shadcn` 技能。改動限定 Cloudflare 插件、前端、相關 API 產物與中文文件；不提交、不升版、不部署、不修改 changelog。

- G02：`domain/cloudflare/logics.go` 群組回應保留失聯節點 ID，真正 DB 錯誤仍回傳；repository 同步上下文略過已刪節點，健康備援仍可同步。
- G05：移動及移出入口先去重，再執行既有清理、repository 與派送。內嵌 UI 在任一批次操作 pending 時鎖定兩種操作、選取與目標群組；群組編輯 pending 不允許關閉或再次送出。
- G09：新增 `target_mode`、`record_type`、`record_content`；舊模式空值按 node，更新省略模式保留既有模式。custom 不需節點，A／AAAA 分家族驗證，CNAME 驗證主機名稱。沿用 TTL、橙雲、通配符記錄名、成員 managed records 與非受管記錄保護；任務摘要與列表／詳情顯示自訂目標。
- migration：新增 PostgreSQL／SQLite `00021_cloudflare_custom_target.sql`；PG 擴充兩個 desired_ip 欄位為 253 字元。SQLite Up／Down 與舊資料保留已實測；PG 僅核對 SQL，未連線執行。Down 保留擴充長度，避免截斷；降版前須先將 custom 群組轉回 node 並同步。
- 測試：新增 `integration_gaps_test.go`、`cloudflare_custom_target_test.go`，擴充 group dialog／detail 測試；涵蓋缺節點、DB 失敗、重複 ID 與派送、移動後重送、三種自訂目標、長 CNAME、更新冪等、模式雙向切換、遠端清理及 pending 鎖定。另驗證節點模式仍先建立新記錄再清理舊記錄；跨自訂模式先清理，以避免 CNAME 衝突。

實際檢查（Go 1.25.5 windows/amd64）：

```powershell
go test ./openflare/plugins/server/domain/cloudflare ./openflare/plugins/server/migrate ./openflare/plugins/server/kernel/repository
go test ./openflare/plugins/server/httpapi ./openflare/plugins/server/domain/fleet/integration
go vet ./openflare/plugins/server/domain/cloudflare ./openflare/plugins/server/migrate ./openflare/plugins/server/kernel/repository
npx vitest run tests/cloudflare
npx tsc --noEmit --jsx preserve
npx eslint 'app/(main)/cloudflare' tests/cloudflare/group-dialog.test.tsx tests/cloudflare/cloudflare-group-detail.test.tsx --max-warnings 0
node scripts/check-i18n-keys.mjs
make swagger
git diff --check
```

既有開發伺服器 `http://localhost:3012/cloudflare` 回傳 HTTP 200；嘗試另啟 3017 時被 Next.js 同工作目錄鎖阻擋，未停止既有程序。HTTP 回應僅代表頁面可載入，不等於真實 DNS 或登入後互動驗收。

上述檢查通過；Cloudflare 前端 7 檔共 40 項測試通過，四語共 4,167 個鍵一致。Swagger 產生器有既有常數解析與重複路由警告。`make code-check` 在架構腳本階段 exit 1，未進入完整 golangci-lint；已另行通過相關 Go vet、TypeScript 與 ESLint。未宣稱全庫品質門禁通過，未執行真實 Cloudflare／多實體節點／PostgreSQL 或正式環境測試。changelog SHA256 與任務開始相同（`CBEB8212ACF73051B763C495258E3F7C1E8B39AB836403E506F7517C04571171`），保留原有未提交修改。

## G04／G08 後續整合交付

本節更新初次盤點的 G04／G08 結論。工作於指定 integration worktree 進行，對照同層 `openflare-upstream-20261109` 的 `4d52a5b0` 與 `e49e0ad2`；讀取 `new-api`、`new-setting`、`shadcn`、`go-testing` 技能。此次沒有新增 API、修改平台框架或資料庫結構，沒有提交、升版、部署或修改 changelog。

- G04：修改 `frontend/app/(main)/waf/components/ip-group-view-dialog.tsx`，新增搜尋與符合數量；以既有 Input／Button 並排組合，保留手動刪除。搜尋是 IP／CIDR 字串包含比對，並非判斷輸入 IP 是否落在某 CIDR。同群組資料更新保留查詢；關閉或切換群組重設。四語文案寫入 `security.*.json` fragments，再以合併腳本產生語系檔。
- G08：修改 `backend/openflare/share/render/openresty/types.go` 與 `backend/openflare/plugins/server/kernel/model/openflare_option.go`，兩處內建模板同步加入 `gzip_vary on` 與上游完整 MIME 清單，不加入 SSE 的 `text/event-stream`。已保存的舊模板、自訂模板與既有版本快照保留原樣；下次建立並發布新配置版本才套用未自訂的新版預設模板。
- request buffering：保留 initial seed、快照 fallback 與前端預設 `false`，不套上游強制開啟的 migration。既有顯式 `true` 及路由覆寫仍生效。理由與請求／回應緩衝區別、chunked／WAF 限制及自訂模板合併方式已寫入 [反代指南](../guide/proxy-config.md#请求缓冲响应串流与-gzip)。
- 測試新增 `frontend/tests/waf/ip-group-view-dialog.test.tsx`、`backend/openflare/share/render/openresty/compression_test.go`、`backend/openflare/plugins/server/domain/site/config_version/compression_snapshot_test.go`。資料庫只用測試環境；無正式資料變更，不需 migration，保留 00022 未使用。

實際驗證結果（命令於對應 backend／frontend 目錄執行；Go 使用本機 Temp 下 1.25.10 工具鏈）：

| 命令 | 結果 |
| --- | --- |
| `node node_modules/vitest/vitest.mjs run tests/waf/ip-group-view-dialog.test.tsx` | 9 項測試通過；Vite 顯示既有 CommonJS 設定提示。 |
| `go test ./openflare/share/render/openresty ./openflare/plugins/server/domain/site/config_version ./openflare/plugins/agent/sync` | 前兩個套件通過；Agent sync 有 4 項失敗，見下列限制。 |
| `go vet ./openflare/share/render/openresty ./openflare/plugins/server/domain/site/config_version` | 通過；上述兩套件另外重跑 go test 亦通過。 |
| `node node_modules/eslint/bin/eslint.js 'app/(main)/waf/components/ip-group-view-dialog.tsx' tests/waf/ip-group-view-dialog.test.tsx --max-warnings 0` | 通過。 |
| `node scripts/merge-i18n-fragments.mjs`、`node scripts/check-i18n-keys.mjs` | 合併成功，四語 4,160 keys 一致。 |
| `node_modules/.bin/tsc.cmd --noEmit --jsx preserve` | 失敗：範圍外 Cloudflare group-dialog 使用的 target_mode／record_type／record_content 尚未對齊型別，共 4 筆錯誤。 |
| `make code-check` | 未完成：Windows make 啟動 `env bash scripts/check_cordis_architecture.sh` 失敗，未進入後續全域 lint。 |
| `git diff --check -- <本次相關路徑>` | 通過；changelog 檔案雜湊維持開始時的 `a5dfddcbe049bb1fd82bfb1859012a4f3fa50c8e`，HEAD 維持 `c0e4598c5aba111491d08cd1512e51d419c559a1`。 |

Agent sync 失敗項目：`TestSwitchPagesProjectCurrentDirRenameFailureKeepsPreviousCurrent` 與 `TestPromoteSameHashReleaseFailureRestoresPreviousCurrent` 未觸發預期注入錯誤；`TestSyncOnceSuccess` 與 `TestSyncPagesDeploymentEnsuresWorkerReadAccess` 因 Windows 找不到 `openflare` 使用者而失敗。本次未改動這些測試及相關實作，不以針對性通過宣稱整個套件通過。

未執行真實 OpenResty／Nginx 壓縮協商、上傳／SSE 串流、瀏覽器登入後操作、Docker 或正式部署驗證。渲染測試只證明指令與繼承語意，不能代替真實網路串流驗收。

本機 Next 開發服務曾於 `http://localhost:3012` 成功啟動；開啟 WAF 預覽時無法取得瀏覽器日誌，因此不列為瀏覽器驗收通過。

## G01／G07／G12 後續整合交付

本節為初次盤點的後續更新；前面 E01／E23 與 74 筆提交核對表保留初次狀態，以本節和 G01／G07／G12 列為準。修改位於指定 integration worktree，對照同層 `openflare-upstream-20261109`（`217a7166`）的 `a2404a50`、`28b7f7d7`、`6a2503cf`、`32cec2ac`、`8867c822` 與 `e2acbecb`，未 fetch、提交、升版、發布或變更 changelog。

- G01：修改 `backend/plugins/drivers/driver_http/frontend.go`，精確檔案優先，再將正整數 ID 請求映射至同目錄的 `1.html`／`1.txt`／`1/__next.*.txt`；模板不存在維持 SPA shell，後端命名空間維持 404。與上游固定 Cloudflare／websites 前綴不同，平台採通用 ID 模板，沒有把業務名稱寫入 driver。`frontend_test.go` 覆蓋 17 個案例，包括非 1 ID、尾斜線、RSC、HEAD、精確檔案優先、websites、無效 ID 與後端路徑。
- 群組詳情修改 `frontend/app/(main)/cloudflare/groups/[id]/page-client.tsx`，pathname ID 優先，保留 `/cloudflare/group?id=7` 入口；掛載前固定顯示 loading，避免模板 ID 與瀏覽器路徑造成 hydration 差異。前端新增非模板 ID 與無效路徑回歸測試。
- G07：僅更新 `docs/pnpm-lock.yaml` 中 markdown-it 14.3.3、Vite 7.3.7、source-map-js 1.2.2 與其 linkify-it 5.0.2 依賴。`docs/package.json` 與同層上游相同；鎖檔餘下差異僅上游額外 libc 與 deprecated metadata，沒有套入無關套件版本。
- G12：`go get github.com/go-jose/go-jose/v4@v4.1.6` 更新 `backend/go.mod`／`go.sum`，保留既有 OTLP 等未提交變更。driver 與平台 OIDC 依賴的通用補丁均登記於 `backend/openflare/upstream-patches.md`；尚未回流 Wavelet。

實際驗證（Go 1.25.10、Node 24.18.0、pnpm 10.10.0；workflow 設定使用 Node 22，本機版本不同）：

| 命令／檢查 | 結果 |
| --- | --- |
| backend：`go test ./plugins/drivers/driver_http -run TestFrontendStaticExport -v` | 17 個案例通過。 |
| backend：`go test ./plugins/drivers/driver_http ./plugins/domain/auth/service/... github.com/coreos/go-oidc/v3/oidc github.com/go-jose/go-jose/v4/...` | 全部通過，包括 OIDC 簽章／JWT 套件；未執行真實第三方登入。 |
| backend：`go mod verify` | 全部 modules checksum 通過。 |
| backend：`GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o NUL ./...` | 全後端編譯通過，包含 server／agent／relay／flared 入口；尚未編譯 embed_frontend 產物與其餘平台矩陣。 |
| docs：`pnpm install --frozen-lockfile`、`GITHUB_PAGES=true pnpm build` | 成功安裝並完成 VitePress client／SSR、頁面與 sitemap；有 VueUse PURE annotation、chunk size 與略過 esbuild install script 提示，未阻擋 build。 |
| `actionlint v1.7.7 -shellcheck=` | 全部 workflow YAML、表達式與 job 依賴檢查通過；未執行 shellcheck 或遠端 Actions。 |
| frontend：`pnpm exec vitest run tests/cloudflare/cloudflare-group-detail.test.tsx` | 首次 6 項通過；同 worktree 其他整合新增測試後，最後一輪 8 項中 7 項通過，批次移除測試讀取不存在的 `zhCN.cloudflare.groupDetail.removeSelected` 失敗。本次兩項路由回歸均通過。 |
| frontend：`pnpm exec tsc --noEmit --jsx preserve` | 最後一輪失敗：上述 groupDetail 屬性兩處不存在；`tests/cloudflare/group-dialog.test.tsx:82` 的 getByRole options 含不支援的 exact。 |
| frontend：針對 page-client 與 group-detail test 執行 ESLint `--max-warnings 0` | 通過。 |
| frontend：以 `NEXT_STANDALONE_EXPORT=true` 執行 `next build` | JS 編譯成功，TypeScript 被範圍外 group-dialog test 的 exact 選項阻擋；未產生可驗收的新 embed 匯出。 |
| `make code-check` | Windows make 在 Makefile:42 啟動 shell 命令失敗，沒有進入架構掃描與全域 lint，不列為通過。 |
| `go test ./plugins/domain/auth/...` | service／model／pow 通過；根套件四項測試因 Windows 暫存 SQLite 檔案仍被占用而清理失敗。 |

發布 workflow 結論：`build-release.yml` 的 backend 入口、Wavelet ldflags、embed 目錄與前端 `build:embed` 均對應現有路徑；`deploy-docs.yml` 使用 docs 獨立鎖檔及 Pages 建置，已本機驗證。`build-image.yml` 是刻意拒絕執行的 Wavelet 隔離流程，OpenFlare 使用 `build-image-openflare*.yml`／`build-image-openflared.yml`。目前不能宣稱發布端到端可成功：前端型別檢查仍阻擋 build；手動輸入版本只變更標籤 metadata，checkout 仍用觸發 ref，發布前須選擇正確 ref；create-release job 早於 binaries build，建置失敗可能留下尚無完整產物的 Release。這些 workflow 本次只檢查，未改動或觸發。

未執行 Docker build／push、GitHub Release、Pages 部署、其餘 OS／CPU 矩陣、真實瀏覽器群組導航與 RSC hydration。沒有資料庫 Schema 或 API 契約變更，不需 migration 或 Swagger 更新。changelog 雜湊維持 `a5dfddcbe049bb1fd82bfb1859012a4f3fa50c8e`，HEAD 維持 `c0e4598c5aba111491d08cd1512e51d419c559a1`，沒有暫存提交內容。

## G03／G06／G10 後續整合交付

本節補充指定 integration worktree 的 G03／G06／G10，對照同層上游 `bf1e77a8`、`1fe27630`、`7ad25f61`、`96a17ae3`，比較終點仍為 `217a7166`。已讀 database-migration、new-api、go-testing、shadcn 技能。驗證跨越本機午夜；以下記錄包含 2026-11-10 凌晨的重跑結果，前節的早期失敗紀錄保留作歷史，不能視為最新前端狀態。

- G06：模型、請求／回應、repository update map、設定快照、政策差異、共用 OpenResty renderer 與四語 UI 新增 `enable_http2`。建立省略時預設 true；更新省略保留既有值，明確 false 能持久化。舊快照缺欄位在 Server 差異比較及 Agent renderer 均視為 true。保留逐域名 ProxyConfig／headers／timeout、Pages、SW 與獨立 HTTP/3。
- migration 使用 PostgreSQL／SQLite `00020_proxy_route_http2.sql`，新增 NOT NULL 且預設開啟的欄位。SQLite Up／Down 已在隔離測試資料庫執行，舊資料保留；PostgreSQL 僅核對 SQL，未連線執行。Down 會移除 HTTP/2 選項，回到舊程式固定開啟的行為；降版前應確認停用需求。沒有對正式資料庫執行 migration。
- G10：PathExecutor 先辨識受管 master，修復 PID 後 HUP；restart 發 QUIT 並等待原 master／workers 排空，逾時不啟動第二個 master。Linux 使用 /proc 的 binary、有效 UID、設定路徑及 start time，其他 Unix 使用 ps；signal 前再核身。多 master、無法辨識與非 Unix 會安全拒絕。保留 Manager runtimeMu、套用／回滾／兜底及 Agent 持久重試。沒有新增任意 kill listener 行為；其他服務占用端口時由啟動命令回報錯誤。
- G03：Dockerfile 改 alpine-slim，Perl／curl 作臨時 .build-deps 並清理 OPM 快取，MMDB／entrypoint 使用 COPY chown／chmod。保留 backend/Cordis 建置路徑、Wavelet 版本注入、低權限 openflare 使用者、su-exec、libmaxminddb 及 CAP_NET_BIND_SERVICE。
- 測試新增 HTTP/2 CRUD／省略值、快照與舊值解碼、Pages／SW／HTTP3／ProxyConfig 渲染、跨分區 payload 保存、PID 修復、/proc 與 ps fixture、不同執行檔／UID／設定、PID reuse、worker 排空與逾時拒絕。/proc／ps 是合成資料，signal 使用測試替身，不能當作真實 Linux OpenResty 驗收。中文指南已同步，Swagger 已重新產生。

實際驗證（Go 1.25.10 windows/amd64、Node 24.18.0；命令於相應 backend／frontend 目錄執行）：

| 命令／檢查 | 結果 |
| --- | --- |
| `go test ./openflare/plugins/server/domain/site/proxy_route ./openflare/plugins/server/domain/site/config_version ./openflare/plugins/server/migrate ./openflare/share/render/openresty` | 四套件通過，包括 HTTP/2 及雙向 SQLite migration 測試。 |
| `go test ./openflare/plugins/agent/nginx -run 'Test(PathExecutor\|ReloadRepairs\|RecoveryRefuses\|RestartWaits\|RestartDeadline\|RuntimeDescendants\|ParsePSRuntime\|InspectProcRuntime)' -v` | 12 個頂層測試通過，含空／invalid／stale PID 子案例。 |
| `go vet` 上述五個相關套件 | 通過。 |
| `GOOS=linux GOARCH=amd64 go build ./...`；`GOOS=darwin go build ./openflare/plugins/agent/nginx` | 均通過；屬交叉編譯，沒有執行 Linux／Darwin 測試。 |
| `go test ./...`；Windows `go build ./...` | 未通過：Linux syscall 程式在 Windows 無法編譯；另有 PoW embed 路徑、openflare 系統帳號、SQLite 暫存檔鎖及程序／檔案操作測試限制。全套原始日誌在本機 Temp 的 `openflare-g06-g10-full-go.log`。 |
| `go test ./openflare/plugins/agent/nginx` | 整套未通過：Windows pow_static 路徑與 openflare 帳號相關測試失敗；本次 PID 專項另行通過。 |
| `node node_modules/vitest/vitest.mjs run` | 最新全套 35 檔、261 項通過；有既有 act／chart size／Vite CommonJS 提示。新增 payload 測試型別修正後另重跑 4 項通過。 |
| `tsc --noEmit --jsx preserve`；ESLint 全庫 `--max-warnings 0` | 均通過；早期 Cloudflare 測試錯誤在同 worktree 後續整合後已消失。 |
| `node scripts/check-i18n-keys.mjs` | 四語 4,167 個鍵一致。 |
| `NEXT_STANDALONE_EXPORT=true npm run build` | Next 靜態匯出成功；沒有執行登入後瀏覽器操作。 |
| `make swagger` | 成功；有既有常數解析與重複路由警告。 |
| `make code-check` | 未完成：Windows shell 啟動階段 Makefile:42 exit 1，未進入全域 golangci-lint；不列為通過。 |
| `docker version` | Client 可用，DockerDesktopLinuxEngine named pipe 不存在；未執行映像建置、容器啟動、MMDB Lua／低權限 capabilities／真實 TLS 與 HTTP/2 協商。 |

本次不修改 changelog、不提交、不升版、不發布或部署。工作目錄中既有 changelog 修改保留，SHA256 為 `CBEB8212ACF73051B763C495258E3F7C1E8B39AB836403E506F7517C04571171`；HEAD 仍為 `c0e4598c5aba111491d08cd1512e51d419c559a1`。全庫品質門禁與真實 OpenResty／Docker／PostgreSQL 驗收仍未完成，不宣稱可正式發布。

## 整合最終驗證（2026-11-10）

本輪使用 Git Bash、既有 Go 1.25.10、Node 24.18.0 與 Corepack pnpm 12.4.2。檢查工具安裝於工作目錄外：goimports v0.38.0、golangci-lint v2.5.0。原始日誌、初始檔案快照與 HEAD 基準位於 `C:\Users\CHIUYANQI\AppData\Local\Temp\openflare-final-validation-20261109`，未納入版本庫。

- `make SHELL=C:/PROGRA~1/Git/bin/bash.exe format`：exit 0。Windows 全庫 Go 參數超長，以暫存 goimports shell 包裝分批執行；Makefile 不變。格式化後還原 18 個任務開始時未修改的檔案，僅保留整合檔案的格式差異；不代表原全庫已通過格式零差異檢查。
- Cloudflare 自訂目標抽出共用驗證、集中模式常數並同步任務摘要；修正新增錯誤文案的 staticcheck 命名警示。HTTP driver 的路徑長度判斷移除魔術數字。Unreleased 補記既有整合的自訂 DNS 目標能力；沒有 API、Schema 或 migration 變更。
- 前端 `pnpm exec tsc --noEmit --jsx preserve`、`pnpm exec eslint . --max-warnings 0`、`pnpm check:i18n` 與 `pnpm exec vitest run`：全部 exit 0；35 檔、261 項測試通過，四語 4,167 keys 一致。測試仍有 act、圖表尺寸與 Vite CommonJS 提示。
- 前端設 `NEXT_STANDALONE_EXPORT=true` 後執行 `pnpm build`：exit 0，完成 Next 靜態匯出。`pnpm build:embed` 在 Windows 預設 script shell 無法解析行內環境變數，因此改用等價環境與 build 指令，未修改 package.json；未執行瀏覽器驗收。
- Go 相關測試：Cloudflare、fleet 全部子套件、site 全部子套件、Kuma、repository 全部子套件、migrate、httpapi、protocol、OpenResty renderer、updater、heartbeat、localfile、state、trace、HTTP driver、admin repository 與 auth service，共 31 套件全部通過（部分使用 Go cache）。程序恢復專項加 `-count=1` 通過；相關 Go vet 通過。
- 較廣測試 `go test -json ./cmd/... ./openflare/... ./pkg/trace/... ./plugins/drivers/driver_http/... ./plugins/infra/database/... ./plugins/domain/admin/... ./plugins/domain/auth/service/...`：exit 1；60 套件通過、28 套件無測試／skip、23 套件失敗，其中 17 套件受 Linux syscall Windows 編譯錯誤阻擋，6 套件共有 34 個失敗測試事件（含父子案例）。獨立 HEAD 快照重跑該六套件，34 個失敗事件全部重現；包括 SQLite 檔案鎖、openflare 帳號、PoW embed 路徑、檔案注入及 Unix 程序測試。未藉此宣稱整套通過。
- `GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build ./...` 與 `go mod verify`：exit 0。此為交叉編譯，沒有執行 Linux 測試。
- 原生 Windows `make code-check`：exit 2，架構檢查 0 違規，但 lint 受 Linux syscall 編譯限制。最終以 `GOOS=linux GOARCH=amd64 CGO_ENABLED=0` 重跑：exit 2，架構檢查通過，golangci-lint 列 139 項，前端階段未執行（已獨立通過）。相同工具檢查 HEAD 基準列 130 項，數量差不能等同新增問題數。`--new-from-rev HEAD` 列 22 項：gosec 3、revive 16、unused 3；此過濾不涵蓋所有既有函式因整合增加複雜度的診斷。剩餘整合 lint 尚未清零，沒有新增註解、關閉規則或刪除跨平台解析器來繞過門禁。

HEAD 維持 `c0e4598c5aba111491d08cd1512e51d419c559a1`，未提交、升版、推送、套用正式 migration 或部署。品質門禁尚未通過，真實 PostgreSQL、Cloudflare、OpenResty、Docker 與瀏覽器驗收仍未完成。

## 整合新增 lint 清理與 Linux workflow（2026-11-10）

- 補齊本次整合導出符號與 localfile 套件文件；不修改 lint 設定。程序 PID／proc 狀態與資料庫匯出改以 `os.OpenRoot` 限制讀取範圍，沒有新增 gosec 忽略。新增逃逸 proc status 符號連結的拒絕測試。proc 的 exe 仍使用原有 inode 比對，以支援 Linux `/proc/<pid>/exe` 特殊連結。
- ps 解析器與其測試限制 `!linux`，保留非 Linux 程序辨識及測試；Linux 使用 proc 解析器。Cloudflare 拆分目標驗證、記錄輸入、區域查找、清理及同步，保留更新／重建失敗處理與切換清理順序。舊域名匯入改用提前 continue 降低巢狀；HTTP2 整合造成孤立的 Pages 舊包裝移除，原 SW 測試改測實際使用的 HTTP2 renderer。
- 完整 lint 使用同一 golangci-lint v2.5.0、Go 1.25.10、Linux amd64、CGO=0 與相同 `.golangci.yml`；加 `--max-issues-per-linter=0 --max-same-issues=0` 只取消診斷輸出截斷。HEAD 基準 230 項、目前 228 項，逐項多重集合比較新增 0 項；Cloudflare 原有 cyclop 34 與 nestif 6 診斷消除，其餘基準診斷保留。鍵為路徑、linter、完整文字，忽略行號；revive package 命名／package 文件診斷按目錄定位，避免分析器選擇不同檔案造成假差異。完整 JSON、文字報告與 `compare-lint.ps1`、`lint-comparison.log` 留於既有外部驗證目錄。前輪 22 項與整合增加的複雜度已清除。
- `make format` exit 0；還原無關格式差異。`GOOS=linux CGO_ENABLED=0 make SHELL=C:/PROGRA~1/Git/bin/bash.exe code-check` 仍 exit 2，架構檢查 0 違規，預設診斷上限列 128 項既有 lint，前端階段未到達；沒有把基準比較成功說成全庫 lint 通過。
- Go 相關 31 套件重新以 `-count=1` 執行全部通過；JSON 含 712 個 pass、8 個 skip 測試事件（含子測試）。nginx 程序恢復／proc／ps 專項、cmd 的兩個 OTLP 設定測試及相關 go vet 通過；新增 proc status 符號連結逃逸案例在 Windows 因缺少建立 symlink 權限而 skip，尚待 Linux 執行。Linux 全後端交叉編譯通過。
- 前端獨立執行 tsc、ESLint 零警告、i18n、Vitest 與靜態 build，全部 exit 0；35 檔 261 項測試通過。build 使用 `NEXT_STANDALONE_EXPORT=true pnpm build` 的等價環境方式。
- `.github/workflows/integration-linux.yml`：手動 `workflow_dispatch` 或推送 `integrate/**` 觸發，唯讀 contents 權限，Ubuntu 24.04，後端／前端各 30 分鐘。Go 按 go.mod 選版本、固定相關套件清單並加 `-count=1`，另單獨跑 OTLP 設定；前端 Node 24、pnpm 10.10.0、frozen lockfile，包含型別、ESLint、翻譯、全前端測試及 build:embed。沒有發布或全後端測試。actionlint v1.7.7 驗證通過（本機無 shellcheck／pyflakes，未執行這兩項可選檢查）。

本機沒有可用 WSL 發行版，Docker Linux daemon 未啟動，故新增 workflow 尚未在 GitHub/Linux 實際執行，也沒有推送或遠端觸發。HEAD 不變、沒有提交或修改正式資料／版本；後續 Linux CI 的測試結果仍待執行。
