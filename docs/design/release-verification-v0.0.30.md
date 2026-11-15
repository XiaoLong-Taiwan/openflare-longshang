# v0.0.30 發布驗證

## 發布範圍

- 版本：`v0.0.30`，PATCH，日期：2026-11-16（UTC+8）。
- 功能提交：`2269b7f8d3ae4793e1610b54ad8c27ea2db1fe4d`。
- Linux 首輪發現 HTTP 中介層測試替身缺少一次性快取介面，追加修正提交：`f1d1305d5bd532619744db904f2da3e59c0e5b7b`。
- 獨立中文 release commit：`3e82ea4e00ba2e1da6e0fbb672441e6cb78cb69f`，僅將 Changelog 落版並整理發布說明，未修改功能程式。
- `git ls-remote origin refs/heads/main refs/tags/v0.0.30` 在發布時確認 main 與標籤均為該 release commit。後續 main 只追加驗證文件，標籤不移動。
- 排除 `docs/CODE_DIFF_REPORT.md`、`docs/CODE_WIKI.md`、`docs/database-capacity-investigation-2026-11-03.md`，三份文件保留本機，未加入提交。
- 無 Schema 或 migration 變更；管理員查詢介面、舊版維護確認請求與既有已排隊工作保留相容。

## 發布前驗證

- [Linux 首輪 43774341135](https://github.com/XiaoLong-Taiwan/openflare-longshang/actions/runs/43774341135)：失敗，HTTP 測試快取替身缺少 `Take`；未在失敗狀態建立標籤。
- [Linux 通過 43774663060](https://github.com/XiaoLong-Taiwan/openflare-longshang/actions/runs/43774663060)：對應 `f1d1305d5bd532619744db904f2da3e59c0e5b7b`，backend 與 frontend 均 success。
- 後端執行 `go mod download && go mod verify`、`go test -v -count=1 -timeout=10m`（相關 Agent、server、fleet、site、repository、admin/auth/upload 全子套件與 cache/cache_memory）、指定 OTLP 設定回歸。
- 詳細日誌確認 `TestProviderIdentityPrecisionAndMissingID`、`TestTokenCache_GetSetInvalidate`、`TestUserCache_GetSetInvalidate`、`TestTakeAcrossNodes`、`TestTakeOnceAndExpired`、`TestMaintenanceSelectionCompatibility`、`TestSQLAPIServiceCompatibility` 均 PASS。
- Cloudflare 五個新增健康檢查案例、`TestCloudflareNodeChanged`、兩個節點觀測快取鍵案例均 PASS；涵蓋防抖切換、恢復、失敗重試、單次領取及成員失敗隔離。
- Linux 前端 TypeScript、ESLint 零警告、四語 4,255 鍵、Vitest 44 檔案 300 項測試、靜態匯出 59/59 頁均通過。
- 本機以 Go 1.25.10 執行相關 core、cache、auth、admin、Cloudflare、fleet/agent 與 observability 的 `go vet`；相關 service/cache/Cloudflare 測試通過，HTTP 修正案例單獨通過。auth/admin/upload 全子套件執行 `go test -run '^$'` 編譯檢查通過，此項不計為實際案例執行。
- 本機前端以 Node 執行 Vitest、TypeScript、ESLint 與翻譯鍵檢查，結果同為 44 檔案、300 項測試通過；含既有 React act、jsdom 圖表尺寸與 Vite 設定警告。
- `swag init -o docs --parseDependency --parseInternal` 成功重建 Swagger，並同步根文件目錄產物；產生器回報既有常數解析及重複路由警告。
- `git diff --check` 與提交前暫存差異檢查通過；變更 Go 檔案 `gofmt -l` 無輸出。

## 遠端發布

- [GitHub Release](https://github.com/XiaoLong-Taiwan/openflare-longshang/releases/tag/v0.0.30)：名稱與標籤均為 `v0.0.30`，非草稿、非預覽版，完整中文正文與 release commit 一致。
- Release 嵌入前端建置日誌確認 `NEXT_PUBLIC_APP_VERSION: v0.0.30`，59/59 頁匯出成功。
- 全程新增提交、一般推送及新標籤，未 force、amend、移動既有標籤或改寫公開歷史。

| Workflow | Run | 結果 |
| --- | --- | --- |
| Build Release | [43775070616](https://github.com/XiaoLong-Taiwan/openflare-longshang/actions/runs/43775070616) | success |
| Build Image (openflare) | [43775070587](https://github.com/XiaoLong-Taiwan/openflare-longshang/actions/runs/43775070587) | success |
| Build Image (openflare-agent) | [43775070519](https://github.com/XiaoLong-Taiwan/openflare-longshang/actions/runs/43775070519) | success |
| Build Image (openflare-relay) | [43775070581](https://github.com/XiaoLong-Taiwan/openflare-longshang/actions/runs/43775070581) | success |
| Build Image (openflared) | [43775070513](https://github.com/XiaoLong-Taiwan/openflare-longshang/actions/runs/43775070513) | success |

五個工作均對應 `3e82ea4e00ba2e1da6e0fbb672441e6cb78cb69f`；額外 CodeQL、ESLint 與 Copilot Setup Steps 亦成功。

## 附件完整性

執行 `gh release download v0.0.30 -R XiaoLong-Taiwan/openflare-longshang`，將 17 個產物下載至系統暫存目錄。以 Python `hashlib.sha256` 逐一比對 GitHub Release asset digest 與大小，17/17 相符。解開五個主站壓縮檔並檢查其二進位內容，加上 12 個節點二進位檔，17/17 均含 `v0.0.30` 標記。此為靜態檢查，不代表跨平台執行測試。

已補上 [SHA256SUMS.txt](https://github.com/XiaoLong-Taiwan/openflare-longshang/releases/download/v0.0.30/SHA256SUMS.txt)，上傳後重新下載並以 Python 及 PowerShell `Get-FileHash -Algorithm SHA256` 依清單核對，均為 17/17 通過。Release 共 18 個附件：17 個產物及 1 個校驗檔。校驗檔自身 SHA256：`291ac79c3facb110130374f92b513c64fbb581fbd148c752c5fe4fcccc6f6325`。

```text
662b23a993144eb5402a38757ca4174f6997961917ca25f589eb8896ef778fb5  openflare-agent-darwin-amd64
889064ab8e57ca093c9f089212879a8260bd0dfc84b5a5acc9fa827beb763743  openflare-agent-darwin-arm64
8040b9930eefd41d785b14a1d9cb4e805b88280feb54bbee974b41cde8c82333  openflare-agent-linux-amd64
27507a806c65bbc32b6ad9df7bd0a8680d373649b7a2373e14f1f6924565d9f6  openflare-agent-linux-arm64
c06a96bdbe37947024cf4bf82d0f4776d3718603bfa232a4b228db52a6a5d2d8  openflare-relay-darwin-amd64
6d37d63e2f4fa5a748cd1dd146152b81e3f3ac91036218a807cc2729f37158d8  openflare-relay-darwin-arm64
043123049918825354d4254acfce077d38eea9aede473e5afda53e42a1566bc0  openflare-relay-linux-amd64
87c1c1ffed756397cc871c8228a3fad6ac2e74bb629b147edc38bc687e6dfbc0  openflare-relay-linux-arm64
535c304fd3801c41a0054b02ca98f5e9ad9f11989c357c1cf45447bbe7794f86  openflare-server_v0.0.30_darwin_amd64.tar.gz
1224e324ac92c6fba57524d18ad5c69d2b4b2a185dd92b8164383240edecd354  openflare-server_v0.0.30_darwin_arm64.tar.gz
c4b218e790f07ace768b1e375f8fc87fba35e8aec5b7d8661e2a049957e39ba9  openflare-server_v0.0.30_linux_amd64.tar.gz
2c39b92499d0fa77b471ed043bad04daa858f248ae2ef1d154481c3afdd9ec95  openflare-server_v0.0.30_linux_arm64.tar.gz
564141670588b78414ed46891713b3e69be251a075cc09b5ec71dd142100d7f9  openflare-server_v0.0.30_windows_amd64.zip
529a22cd90dde3b129dd9a247f58e94066d41d587ee9c6e337a30304e0224dce  openflared-darwin-amd64
1775b149ba6be2c866ff91063c92c6a7a01a8be1de66e0e21d3c41002516b4e4  openflared-darwin-arm64
3003817de8a2df9f875d006e78337c743161e15fee6145ed0cdb140260958341  openflared-linux-amd64
dda8ba314db3ce4697688d0a3ea94727098c2161035a65cd4b1d326e8a83fcc9  openflared-linux-arm64
```

## 映像驗證

執行 `docker buildx imagetools inspect ghcr.io/xiaolong-taiwan/<image>:v0.0.30` 與 `--raw`，四個映像均包含 `linux/amd64` 與 `linux/arm64`。另查詢 `latest`，四個 index digest 均與版本標籤一致。額外 `unknown/unknown` 項為 attestation，不計為執行架構。

| 映像 | Index SHA256 |
| --- | --- |
| openflare | `d0982b472be8e71185d3ef9e058eaa41c3d39321e42a82a096c2e02660d5d847` |
| openflare-agent | `014871f905e6923f8f0b6fb0f07de12c9a1e16791abfcf67b6762366703925c8` |
| openflare-relay | `ca4da76a7a663154d61c46421bcc2fc35da332f397b50ea4d8b4ce556323197e` |
| openflared | `f86bcaf5d317b02acd0164a34a94e04361a3e51b9dcb621fb14163fab17a8354` |

| 映像 | amd64 manifest SHA256 | arm64 manifest SHA256 |
| --- | --- | --- |
| openflare | `1eb81c298a0e5191b0a1aae358cfaad38d0c495b98349d85059108d07a01c12d` | `b78b0637bec25007957f86ad8ff959c890a9132035eec446089038a20e4c0789` |
| openflare-agent | `b07a924a6feebe52ea8eb85fd09e8cfa252540770ad7cbb16d03e9e0c9607aca` | `0dcc1ee12735b8712a78b67fdee1726b183dcfd76b880d4b1e68426b5988f014` |
| openflare-relay | `ed535e1e0c8a5ce490bd8619654c3625c7d1058209d26d016634ebad87c33198` | `6c0bc03d727930569d58651ab47914d730911363a2ae660dddcfe4ca1dec3efb` |
| openflared | `0c5a8ea50179d31af3a972134df2b1068b7fc51aed583b0c83a1e4a7ebb8b91a` | `5f3fef884007e6521d6f662d0db50a47ae9f926513ba9f70bddb00d1139c4c89` |

## 限制與部署狀態

- Windows 的 `make format`、`make code-check`、`make swagger` 因 shell 啟動問題失敗；Swagger 已直接呼叫既有產生器完成。未宣稱完整 make 品質門檻或 golangci-lint 通過。
- 未執行真實第三方 OAuth、Cloudflare DNS 寫入、瀏覽器視覺驗收、Compose 啟動或跨平台二進位執行。
- 通用 auth/admin/cache 修改已登記 `backend/openflare/upstream-patches.md`，尚未回流 Wavelet。
- 既有日誌寫入去重測試曾有不穩定紀錄，本次 Linux 未以完整後端全套測試取代指定套件驗證。
- 未主動執行應用程式部署、服務重啟或資料庫遷移，未進行正式服務健康檢查。
- 主站映像工作實際日誌為 `Webhook URL is not set, skipping.`，未呼叫部署 webhook。
- main 推送會觸發既有文件站自動發布；這是儲存庫預設工作流程，與應用程式部署分開記錄。
