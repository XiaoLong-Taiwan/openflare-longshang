# v0.0.26 發布驗證紀錄

## 來源與時間

- 儲存庫：`XiaoLong-Taiwan/openflare-longshang`。
- [Release v0.0.26](https://github.com/XiaoLong-Taiwan/openflare-longshang/releases/tag/v0.0.26)：名稱與標籤均為 `v0.0.26`，`draft=false`、`prerelease=false`。
- 標籤指向 commit：`fc5fce61f38274da4232bc4ce4d737664d0ddc84`，與本紀錄所列發布及映像 CI 的 `headSha` 相同。
- Release 建立時間：`2026-11-09T17:14:58Z`（臺北時間 2026-11-10 01:14:58）。
- 本次核驗及 checksum 補件日期：臺北時間 2026-11-10；GitHub UTC 日期為 2026-11-09。
- 本紀錄於發布後補寫，未包含在上述發布 commit 內。

## 實際 CI 結果

| 工作流程 | Run | 結果與範圍 |
| --- | --- | --- |
| Build Release | [42662679656](https://github.com/XiaoLong-Taiwan/openflare-longshang/actions/runs/42662679656) | `completed/success`，20/20 個工作成功，包含前端、5 個 Server 平台及 12 個 Agent／Relay／Flared 平台建置。 |
| Build Image (openflare) | [42662679569](https://github.com/XiaoLong-Taiwan/openflare-longshang/actions/runs/42662679569) | `completed/success`，Prepare metadata、Build (amd64)、Build (arm64)、Merge multi-arch manifest 全部成功。 |
| Build Image (openflare-agent) | [42662679876](https://github.com/XiaoLong-Taiwan/openflare-longshang/actions/runs/42662679876) | `completed/success`，amd64、arm64 建置及多架構 manifest 全部成功。 |
| Build Image (openflare-relay) | [42662679431](https://github.com/XiaoLong-Taiwan/openflare-longshang/actions/runs/42662679431) | `completed/success`，amd64、arm64 建置及多架構 manifest 全部成功。 |
| Build Image (openflared) | [42662679799](https://github.com/XiaoLong-Taiwan/openflare-longshang/actions/runs/42662679799) | `completed/success`，amd64、arm64 建置及多架構 manifest 全部成功。 |

Build Release 完成時間為 `2026-11-09T17:19:07Z`（臺北時間 2026-11-10 01:19:07）。四映像成功係依 GitHub Actions 的工作與步驟結論確認；建置推送及 manifest 建立成功不代表本機已拉取或部署映像。

同 commit 的 Linux Integration Validation（[42662664690](https://github.com/XiaoLong-Taiwan/openflare-longshang/actions/runs/42662664690)）、ESLint（[42662672069](https://github.com/XiaoLong-Taiwan/openflare-longshang/actions/runs/42662672069)）、CodeQL（[42662672283](https://github.com/XiaoLong-Taiwan/openflare-longshang/actions/runs/42662672283)）亦由 API 查得 `completed/success`；本次未重新執行這些流程或逐項重驗其測試內容。

## 附件與產物驗證

發布流程預期 17 個產物：Server 5 個壓縮包（Linux／macOS amd64、arm64，以及 Windows amd64），Agent、Relay、Flared 各 4 個裸執行檔（Linux／macOS amd64、arm64）。補件前實際附件正好 17 個，沒有缺漏或額外產物。

所有 17 個產物均下載至系統 Temp，而非僅下載代表產物，實際完成下列檢查：

- 逐檔重算 SHA-256，與 GitHub Release asset 的 `digest` 相同；檔案大小符合 API，狀態為 `uploaded`。
- Linux ELF、macOS Mach-O、Windows PE 標頭與機器架構均符合附件檔名。
- 每個執行檔均檢出以 NUL 結尾的內嵌 `v0.0.26` 字串；Server 內嵌前端亦檢出 `v0.0.26` 與建置時間 `2026-11-09T17:14:54Z`。
- 5 個 Server 壓縮包皆可讀取；Windows ZIP 通過 CRC 檢查。各包包含執行檔、`LICENSE`、`README.md`、`README_zh.md`、`config.default.yaml`。
- Server 執行檔均檢出 `index.html` 與 `/_next/` 前端資源標記；此為靜態檢查，未啟動 HTTP 服務驗證網頁。
- Windows Server 代表產物解壓後實際執行 `openflare-server.exe --version`，結束碼為 `0`，輸出 `wavelet version v0.0.26`。

本機證據目錄：

```text
C:\Users\CHIUYANQI\AppData\Local\Temp\openflare-v0.0.26-run42662679656-93ab2060\
```

該目錄保留原 17 個附件、`verification.json`、`release-metadata.json`、`verify_release.py`、`SHA256SUMS.txt`、重新下載的 `checksum-downloaded/SHA256SUMS.txt` 與 Windows 解壓產物。Temp 可被系統清理；此路徑是本機證據位置，不是永久保存承諾。

## SHA256SUMS.txt 補件與回讀校驗

補件前確認 Release 沒有同名附件；依已驗證的 17 個檔案建立清單，格式為小寫十六進位 SHA-256、兩個空白、附件原名，共 17 行。上傳前再次確認每行與本機檔案內容及當下 GitHub digest 一致，且檔名集合完全等於原 17 個附件。

以以下命令新增附件，未使用 `--clobber`：

```powershell
gh release upload v0.0.26 C:\Users\CHIUYANQI\AppData\Local\Temp\openflare-v0.0.26-run42662679656-93ab2060\SHA256SUMS.txt --repo XiaoLong-Taiwan/openflare-longshang
```

- [下載 SHA256SUMS.txt](https://github.com/XiaoLong-Taiwan/openflare-longshang/releases/download/v0.0.26/SHA256SUMS.txt)。
- 新附件 ID：`728333447`；大小：`1665` bytes；狀態：`uploaded`。
- 建立時間：`2026-11-09T17:25:29Z`（臺北時間 2026-11-10 01:25:29）。
- 清單本身 SHA-256：`ea598a65a3f0894aa8a406377bb7e8dc1a690403ac66f18680523f24ef30594a`，符合 GitHub asset digest。
- 上傳後重新從 Release 下載清單，與本機原稿逐位元組完全相同。
- Release 附件數由 17 增至 18；與先前保存的 metadata 比對，原 17 個附件的 `id`、`size`、`digest`、`updated_at` 全部不變。
- 清單只涵蓋原 17 個發布產物，不列自身，也不列 GitHub 自動產生的原始碼壓縮包或容器映像。

可重現的下載及校驗方式（Linux，先下載完整 17 個產物與清單至同一目錄）：

```bash
gh release download v0.0.26 --repo XiaoLong-Taiwan/openflare-longshang --dir ./release-v0.0.26
cd ./release-v0.0.26
sha256sum --check SHA256SUMS.txt
```

本次 Windows 環境實際使用 Python `hashlib.sha256` 重算 17 個檔案，並以 GitHub CLI 下載及 Python 比對回讀位元組；未宣稱在 Windows 執行過上述 Linux 指令。

## 限制與既有落差

- 本機為 Windows，Linux／macOS 產物只做靜態架構、版本字串及 checksum 核驗，未在對應系統實際執行；版本字串存在不等於已驗證各平台的執行期版本查詢或完整功能。
- 四映像成功僅代表上述 CI 結果；本次未另行拉取映像、獨立驗證 registry manifest／OCI 版本標籤或簽章，也未執行容器健康檢查。
- 工作流程的 `EXTRA_FILES` 列有 `DEPLOYMENT_zh.md`，但發布標籤中不存在該檔案；打包腳本只複製存在的檔案，因此 5 個 Server 包均未包含此文件。此次只記錄落差，未重打包或覆蓋原附件。
- 此次 checksum 是發布後對既有附件計算的完整性清單，未提供獨立簽章，也未修改 CI 以自動產生未來版本清單。
- 本次沒有進行正式部署、資料庫操作、Migration 或服務啟停；未建立新版本、Git commit 或推送。主工作區僅新增本文件，保留使用者原有未提交變更；純文件變更依專案規範不新增 Changelog。
