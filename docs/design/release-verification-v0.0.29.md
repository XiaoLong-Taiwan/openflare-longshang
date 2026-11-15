# v0.0.29 發布驗證

## 發布範圍

- 版本：`v0.0.29`，PATCH，日期：2026-11-15。
- 功能提交：`241b409f`，改善行動版管理介面、儀表板說明與憑證表單。
- 獨立 `chore(release): v0.0.29` 提交：`79473b2a3b9334c2b35eb5ec8b3ea65b30b1cd56`，承載完整中文說明，標籤直接指向該提交。
- 排除 `docs/CODE_DIFF_REPORT.md`、`docs/CODE_WIKI.md`、`docs/database-capacity-investigation-2026-11-03.md`，保留本機文件。
- 無 API、資料庫結構、migration 或設定相容性變更。

## 發布前實測

在 `frontend` 以 `C:\Program Files\nodejs\node.exe` 執行：

- `node_modules/vitest/vitest.mjs run`：42 個測試檔、291 項測試通過。
- `node_modules/typescript/bin/tsc --noEmit --jsx preserve`：通過。
- `node_modules/eslint/bin/eslint.js . --max-warnings 0`：通過。
- `scripts/check-i18n-keys.mjs`：四語系 4,232 個鍵值通過。
- `git diff --check` 與提交前暫存差異檢查：通過。
- 靜態匯出：使用者提供此前已通過的結果；發布工作中的 Linux `Build Embedded Frontend` 亦成功。
- 測試含 Vite 設定、React act 與 jsdom 圖表尺寸警告，未造成測試失敗。

## 遠端發布結果

- [GitHub Release](https://github.com/XiaoLong-Taiwan/openflare-longshang/releases/tag/v0.0.29)：正式發布，非草稿、非預覽版，名稱與標籤均為 `v0.0.29`。
- `git ls-remote origin refs/heads/main refs/tags/v0.0.29` 在發布時確認兩者均為 release commit；後續 main 僅追加驗證文件提交，標籤維持不動。
- 中文發布說明與 release commit 一致，未遭 merge 訊息覆蓋，無須修補 Release 正文。
- 全程使用新增提交、一般推送與新標籤，未 force、amend 或重寫歷史。

| Workflow | Run | 結果 |
| --- | --- | --- |
| Build Release | [43723915649](https://github.com/XiaoLong-Taiwan/openflare-longshang/actions/runs/43723915649) | success |
| Build Image (openflare) | [43723915655](https://github.com/XiaoLong-Taiwan/openflare-longshang/actions/runs/43723915655) | success |
| Build Image (openflare-agent) | [43723915717](https://github.com/XiaoLong-Taiwan/openflare-longshang/actions/runs/43723915717) | success |
| Build Image (openflare-relay) | [43723915718](https://github.com/XiaoLong-Taiwan/openflare-longshang/actions/runs/43723915718) | success |
| Build Image (openflared) | [43723915708](https://github.com/XiaoLong-Taiwan/openflare-longshang/actions/runs/43723915708) | success |

五個工作均對應 `79473b2a3b9334c2b35eb5ec8b3ea65b30b1cd56`；額外的 Copilot Setup Steps 亦成功，不列入五個發布工作計數。

## 附件完整性

執行 `gh release download v0.0.29 -R XiaoLong-Taiwan/openflare-longshang`，將 17 個產物下載至系統暫存目錄，以 Python `hashlib.sha256` 計算內容摘要，逐一與 GitHub Release asset digest、檔案大小比對，全部相符。解開五個 Server 壓縮檔並檢查二進位內容，加上其餘 12 個二進位檔，均含 `v0.0.29` 標記；此為靜態內容檢查，不代表跨平台執行測試。

原 workflow 未產生校驗清單，已補上 [SHA256SUMS](https://github.com/XiaoLong-Taiwan/openflare-longshang/releases/download/v0.0.29/SHA256SUMS)。上傳後重新下載該檔，以 PowerShell `Get-FileHash -Algorithm SHA256` 按清單再次比對，17/17 通過。Release 現有 18 個附件：17 個產物與 1 個校驗檔。校驗檔自身 SHA256：`fdb03380572f2d34f15ae1cb9a135fb4f8fbdc4df35b16d751f6f61850fdab2a`。

```text
d5e72b1dd391f14cb1ab5d8e646fedd4193fa68eea1c7e84f45e15718dbb1588  openflare-agent-darwin-amd64
77dd9ad038ff251af552cde208483ae028aa0e567bf98a26b7d50d2dea65dabd  openflare-agent-darwin-arm64
9d01a7d5bf03f2b98b8b2f959d00d2023844082b42a0a039d7c2204418ccfdb9  openflare-agent-linux-amd64
bcff973d2f8f6eeb194f47598b6bea83e3b811f40aecdcf1d9b5a57042119e8a  openflare-agent-linux-arm64
70aac6774afae7ba12f656c98af289b700b6ececc6d79469a59387bbfdd59714  openflare-relay-darwin-amd64
972e9b0f0c3f9b6b64cbb893ad9303b250f2137355f33670b086b2059b85126e  openflare-relay-darwin-arm64
757e7c9006bca9d811768e373c8fe6e49a15ae18af5063297f395893243479a0  openflare-relay-linux-amd64
f375a9e96d290bc13c2ba3eed4f81cc21a4496c4c2cfe8d50dae644f22aa8ad2  openflare-relay-linux-arm64
f993f3499d0434d09925df04502daf242582fd7f3e617cd709885f64f1004082  openflare-server_v0.0.29_darwin_amd64.tar.gz
2d3fecccd9030e28f3ff1fe1ac57d0006e568b0d3e58648ec6985e00cdbbaaa4  openflare-server_v0.0.29_darwin_arm64.tar.gz
768ccb88f127a162f12aa301d97f99b7b74a97d85be65c3eaf9860d90c6efa63  openflare-server_v0.0.29_linux_amd64.tar.gz
7638093d18a1f1fd7574f73b6cd27997b363295b201b063c7c460d75a6e1c548  openflare-server_v0.0.29_linux_arm64.tar.gz
fd85b2acf05d3932fa3b1862ddf899474859d3f86a35f83cc0d0afac320e650d  openflare-server_v0.0.29_windows_amd64.zip
8e3d7aa0d6c90d74d398e54906dc606b8d76834e8c1a894f1d12ebcfd59b3b34  openflared-darwin-amd64
d79e72696204b24b25cf78d4367ac54c284b47f64b57bd9b54d07f12b4f851a3  openflared-darwin-arm64
65b429766d849d048e02f17ca169e7f4ee89feed3ccc9ce23329356cc0a49674  openflared-linux-amd64
4532942ab3a103f7ba640fa658010d73806c86c4cdcf9ef50b24bff8370326b5  openflared-linux-arm64
```

## 映像驗證

以 `docker buildx imagetools inspect ghcr.io/xiaolong-taiwan/<image>:v0.0.29` 實際查詢四個映像。以下均含 `linux/amd64` 與 `linux/arm64`；`unknown/unknown` 為額外 attestation manifest，不列為執行架構。

| 映像 | Index SHA256 |
| --- | --- |
| openflare | `b11b25d3d808a1feedeae8bf956b4506dd66a56c2ec5db0eb3bfd9f9ae974c63` |
| openflare-agent | `367c53b48b3700e338684955eb0c5dbe3a8be8cb737d9c9fc75016cd03364df5` |
| openflare-relay | `906dcd6e4895b1849936d3d148bed826a229c8ab458d8b78c9dcdc90314b3a3c` |
| openflared | `2c8baf35e99e9a9d0151d9598df93acae481b900d50f55c7cfc7526384b038da` |

| 映像 | amd64 manifest SHA256 | arm64 manifest SHA256 |
| --- | --- | --- |
| openflare | `3ec8cddd270ca43f2418785ffea7a94e2b026ceb4af8fd58523298611fd3aa2e` | `003d8cd4101c43a663facf9367454705fadc77d70320130517855e126ee23395` |
| openflare-agent | `15955a12fcf1881697b5dcbfa686dcb122f2ed17ca7a424b093c7174c0587c9b` | `83da6c45bb73abd3e27c72cb9ca6f2ae6466729252cec1ef50df13e7e28f551e` |
| openflare-relay | `9fbc0da40407d991b164d41c7965e82d36c2c49464fd6cb1dae76e9d9197f18c` | `8ae71d7c07feed1ce6b0f98c9f2baef939da340695b683a25625d2f1f7adc206` |
| openflared | `3dc10ea4d3d7f586d0fc0afd35849adc43ba131cd6df96488504ed81425720ec` | `d8d6c7c661320e6592341985bab6c124271102c39f133241f6b4991ab55f065b` |

## 驗證限制與部署狀態

- `make format` 因 Go／Unix shell 工具缺失失敗；`make code-check` 因無法執行 Bash 架構檢查而中止，不宣稱完整品質門禁通過。
- 未執行真實瀏覽器的手機／桌面視覺驗證。
- 此次為 GitHub 產物發布，未部署正式服務、執行資料庫遷移或進行正式服務健康檢查。
- 主映像 workflow 日誌確認 `Webhook URL is not set, skipping.`，未透過該 webhook 觸發部署；Docker Hub 登入亦因未設定而略過，本次驗證範圍為 GHCR。
