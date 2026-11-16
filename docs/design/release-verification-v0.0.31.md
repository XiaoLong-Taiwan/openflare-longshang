# v0.0.31 發布驗證

## 發布範圍

- 版本：`v0.0.31`，PATCH，日期：2026-11-16（UTC+8）。
- 範圍：Cloudflare 全分組健康同步任務與排程去重、資料管理緊湊總覽、OAuth 回呼網址顯示與複製、手機／平板／桌面響應式介面及四語文案。
- 功能提交：`6bf76c0e45603f9c478d98d439b18dea7d508160`。
- 獨立 release commit：`08232b8c2f796f239d6e236da36a51e4a09b6c46`，僅修改 Changelog，包含完整中文四分類發布說明。
- 發布時 `git ls-remote origin refs/heads/main refs/tags/v0.0.31` 確認兩者均直接指向 release commit；後續驗證文件只追加新提交，標籤不移動。
- 排除 `docs/CODE_DIFF_REPORT.md`、`docs/CODE_WIKI.md`、`docs/database-capacity-investigation-2026-11-03.md`，保留本機不提交。
- 新增 server migration 00026（PostgreSQL／SQLite）：保留最小 ID 排程設定並移除重複列；這是不可逆資料遷移，Down 不會復原刪除列。升級前須備份並於資料庫副本預演，詳見 `cloudflare-health-queue-dedup.md`。本次發布不執行正式資料庫遷移。
- 通用任務能力已登記 `backend/openflare/upstream-patches.md`，尚未回流 Wavelet。

## 發布前驗證

- 使用者交接紀錄：相關後端測試及 vet、前端 327 項測試與 static build、migration 42 cases 已完成；此項為交接資訊，本次以 Linux CI 保存獨立驗證證據。
- Linux 驗證新增 Asynq worker 與 cron 套件，確保單例任務回歸案例納入；既有流程亦包含 migration、Cloudflare、前端型別／ESLint／四語鍵檢查、全部前端測試及靜態匯出。
- 本機 `make format` 因 Go 未在 PATH 與 Windows shell 不相容失敗；`make code-check` 因 `env bash` 啟動失敗。不得視為完整 make 品質門檻通過。
- [Linux CI 43879727130](https://github.com/XiaoLong-Taiwan/openflare-longshang/actions/runs/43879727130)：對應功能提交，backend／frontend 均 success。後端模組驗證、相關整合套件與 OTLP 設定案例通過；包含三個 singleton lifecycle／manual retry／long pending 案例。
- Linux 前端 TypeScript、ESLint 零警告、四語 4,271 鍵、Vitest 49 檔案／327 測試及靜態匯出 59/59 頁均通過。
- Linux 詳細日誌確認 `TestCloudflareHealthScheduleMigrationDeduplicatesRows` 的 42 個子案例全部 PASS，涵蓋兩份 SQL、主列設定保留、重複升級及不可逆 Down 行為。
- 本機直接呼叫 Go 1.25.10 對 core/contracts、core/extpoints、Asynq worker／cron、Cloudflare 與 migration 執行 `go vet` 通過；暫存 Go 檔案 `gofmt -l` 無輸出，`git diff --cached --check` 通過。
- 暫存 79 份檔案，檢查排除清單及新增差異中的常見私鑰、GitHub／AWS token、Discord webhook 與長字串機密賦值模式，無命中；這是差異掃描，不是完整 Git 歷史機密稽核。

## 遠端發布

- [GitHub Release](https://github.com/XiaoLong-Taiwan/openflare-longshang/releases/tag/v0.0.31)：名稱與標籤均為 `v0.0.31`，非草稿、非預覽版；中文正文與 release commit 一致。
- 五項發布工作全部 success，均對應 release commit `08232b8c2f796f239d6e236da36a51e4a09b6c46`。

| Workflow | Run |
| --- | --- |
| Build Release | [43880377680](https://github.com/XiaoLong-Taiwan/openflare-longshang/actions/runs/43880377680) |
| Build Image (openflare) | [43880377845](https://github.com/XiaoLong-Taiwan/openflare-longshang/actions/runs/43880377845) |
| Build Image (openflare-agent) | [43880378008](https://github.com/XiaoLong-Taiwan/openflare-longshang/actions/runs/43880378008) |
| Build Image (openflare-relay) | [43880377815](https://github.com/XiaoLong-Taiwan/openflare-longshang/actions/runs/43880377815) |
| Build Image (openflared) | [43880377939](https://github.com/XiaoLong-Taiwan/openflare-longshang/actions/runs/43880377939) |

## 附件完整性

- 使用 `gh release download v0.0.31 -R XiaoLong-Taiwan/openflare-longshang` 將產物下載至系統 TEMP；Python `hashlib.sha256` 比對 GitHub asset digest 與大小，17/17 相符。
- 解開五個主站壓縮檔並檢查二進位內容，加上十二個節點二進位，17/17 含 `v0.0.31` 標記；此為靜態檢查，不代表跨平台執行測試。
- Release 前端日誌確認 `NEXT_PUBLIC_APP_VERSION: v0.0.31` 及 59/59 頁成功匯出。
- 已上傳 [SHA256SUMS.txt](https://github.com/XiaoLong-Taiwan/openflare-longshang/releases/download/v0.0.31/SHA256SUMS.txt)，重新下載後以 PowerShell `Get-FileHash -Algorithm SHA256` 再核對 17/17 通過。Release 共十八個附件：十七個產物及一個校驗檔。
- 校驗檔自身 SHA256：`2e5b1b08ff89996bcea4a89d745098c6b9c09ae9ca5af1cf2ad7691bdcc8c907`。

```text
20f67c1e5f1be58a076eaf606d52edfa606204a4610439d4daa578ceed879b46  openflare-agent-darwin-amd64
0175f173ea4539d2c61a01573b7842ecb96632148963950da2344a875e0fde94  openflare-agent-darwin-arm64
ff8c796612cb2a7e006ca771ab2d757f06d9d971264378563c9ab2ac03122272  openflare-agent-linux-amd64
28697513d66cccd374b98ab94aa79d08f9521ec66a6d6568080fc45caadf1de7  openflare-agent-linux-arm64
a4ef23b9cd881d2815eca56fc1e723bc368c5337861baf4936e9b830af1a0034  openflare-relay-darwin-amd64
645b1d0c84deb3323ae7dffd995959cd932e8a25f662d65d5c8d2f7f7a8c358a  openflare-relay-darwin-arm64
8d29042458708c0f12c1502497b5cf4f65a951a15a1fd204be9eedb1c853defb  openflare-relay-linux-amd64
1468482fd65cf02e3e1be4792e70b741a5a532e6391a5d499bd92c9c6e1b5ca8  openflare-relay-linux-arm64
92fa593c42ed751f4c3c029f81d85dd1108f58d7eaaaa293f70456b6894de2c8  openflare-server_v0.0.31_darwin_amd64.tar.gz
671a8000f5e49afa88663705941047eca1f0d9af63e796f0c6ad79b2c1f14425  openflare-server_v0.0.31_darwin_arm64.tar.gz
a641b1d293e790dd8273e4cc1dba886c6ff049bb43d30ef4ae7062cc6f9fb23d  openflare-server_v0.0.31_linux_amd64.tar.gz
1ad48ec84ed534fff3b7547f3d9ebeca3c3b65f20f2f8ce8b1e147206bc3cad2  openflare-server_v0.0.31_linux_arm64.tar.gz
a6ae3bdff2d7ab8c38f96213f448418f2d4aea5a655cefe6dc94cf92112e42c4  openflare-server_v0.0.31_windows_amd64.zip
3dfc23f005949d704500846e1d2a15231d5df6cf3a49bd15562885bd94271312  openflared-darwin-amd64
c5144bd03a3008843edfcf2140194be89c0533d5c5766982281b5c80377bde45  openflared-darwin-arm64
2cf99386138e57fde1899cac6eb1edef58131131df749293c1763139a19729ab  openflared-linux-amd64
ed11db2674d0369f1beb4106650de153189c91616b3dffc0c1fa2b15e024e33c  openflared-linux-arm64
```

## 映像驗證

透過 `docker buildx imagetools inspect` 與 `--raw` 查詢 `ghcr.io/xiaolong-taiwan/<image>:v0.0.31`，四個映像均含 `linux/amd64` 與 `linux/arm64`，各自的 `latest` index digest 與版本標籤一致。`unknown/unknown` 為 attestation，不計為執行架構。

| 映像 | Index SHA256 |
| --- | --- |
| openflare | `439c1003bb70512bbea013f37012940e3e02b99a8605dbfa878c47d96e8c7ae6` |
| openflare-agent | `90ef29764074262050411335d31b2a9ba05040cf49b8fa84793f3b8f5f97c9a9` |
| openflare-relay | `4db64c6636b848311e9e6b1e508c2908db6ffe42615c432b0e68e688114e8f7c` |
| openflared | `7c22c7337748c38b6852461dedea95235bc22fed83e93efa378ce8d7c70fe48c` |

| 映像 | amd64 manifest SHA256 | arm64 manifest SHA256 |
| --- | --- | --- |
| openflare | `ee92f1386fd31f6e1baf912ef3f26afa5a35623cb7b642c622759836bda04180` | `db3cf47bcf70f1053aaf89771d8ae35154fe1b72b523de12d99ebbcb640955d4` |
| openflare-agent | `d99513326bdd9c01945fe8be460f090d8b5075ec11ae24675ba5481fd6215889` | `c9342ff02d0381379cc8974d1b3f2f5bece60dbed9a0684540daa02e29e68f44` |
| openflare-relay | `04a6f6469627a1d635ab058baa845804d3532570c503c0ec825e62df3a26edc6` | `a5731ecf536ed0bdfc3ed22471eb27239ddb9d2fcd49a8264c638b880bfd0879` |
| openflared | `bbc26a1249d363fe298959ec424f0518ee2070c5f00ac0b6f16c8495f5c93d95` | `70a79a3f32a167ab3a5145218174ab35311b8e3499d84195c6a9d0fd76b96cfc` |

## 限制與部署狀態

- migration 測試將兩份 SQL 分別於 SQLite 執行，不代表真實 PostgreSQL 整合驗證。
- 不執行 force、amend、歷史改寫、既有標籤移動、應用程式部署或服務重啟。
- 發布前查詢儲存庫 secret 名稱清單為空；主站映像工作日誌確認 `Webhook URL is not set, skipping.`，未呼叫部署 webhook。
- main 推送已觸發既有文件站發布，功能提交的該工作成功；此為儲存庫既有自動化，與應用程式部署分開記錄。
- 尚未驗證跨平台二進位實際執行、正式登入流程、真實 Cloudflare DNS 寫入與正式服務健康狀態。
