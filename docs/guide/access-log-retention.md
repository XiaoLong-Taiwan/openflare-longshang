---
title: 存取記錄保留與清理
---

# 存取記錄保留與清理

系統清理排程每天執行一次，會依目前使用中的日誌資料庫與管理台設定清理過期資料：

- 節點存取明細固定保留七天；使用者存取記錄依 `log_retention_days_postgres`、`log_retention_days_sqlite` 或 `log_retention_days_clickhouse` 清理。
- CPU、記憶體、磁碟、網路等效能資料依 `metric_retention_days` 清理。
- PostgreSQL 的完整過期月份會優先以分區清理，邊界月份再逐筆刪除，減少大量資料刪除造成的鎖定與 WAL 壓力。
- PostgreSQL、ClickHouse 與 SQLite 的節點明細使用相同七天 cutoff。

管理員仍可從管理台手動執行日誌清理。清理失敗會回傳任務錯誤並記錄原因，不會靜默忽略，方便確認資料庫連線、分區或權限問題。

若 PostgreSQL 節點存取日誌佔用大量空間，可在「工作管理」執行「月份日誌清理與彙總壓縮」，填入 `YYYY-MM` 月份並確認清理與彙總壓縮。月份由參數指定，不固定於某一月份；既有管理類型 `force_log_cleanup` 與任務 ID `system:force-log-cleanup` 保持相容。工作先按 UTC 日期、節點與 HTTP 狀態碼保留請求數、回應流量及請求流量摘要，再刪除指定 UTC 月份且早於工作開始時七天 cutoff 的明細；摘要失敗會阻止刪除。每次 DELETE 最多 1,000 筆，每工作最多 1,000,000 筆；選取與刪除均檢查月份及 cutoff，使用實體列鍵避免重複 ID 擴大刪除範圍。

工作記錄提供啟動時分區統計估值與實際累計刪除筆數，結果的 `has_more` 由相同月份及 cutoff 的存在查詢決定；若為 true，可重新執行繼續清理與彙總壓縮。估計筆數不是精確剩餘數量，`has_more` 反映查詢當下狀態。「彙總壓縮」指保留摘要後刪除過期明細，不是產生壓縮檔，也不執行 `VACUUM` 或 `VACUUM FULL` 縮檔。此操作不會刪除整個月份分區，且 `DELETE` 後的空間通常先供 PostgreSQL 重用，不代表資料庫檔案立即縮小；如需回收檔案空間，須另行備份並安排 `VACUUM FULL` 維護窗。

每日 03:00 的系統清理透過領域事件觸發 OpenFlare 日誌與版本保留清理，不執行資料庫完整維護。完整維護僅供管理員手動觸發，PostgreSQL 會執行 `REINDEX`、`ANALYZE` 與 `VACUUM FULL`；`VACUUM FULL` 會鎖定資料表，應安排於維護時段執行。系統不再註冊每日 03:30 的完整維護排程；若管理員曾另建自訂完整維護排程，需自行停用。

Pages 版本排程透過 Pages 公開清理服務，逐一鎖定專案並保留最多 5 個部署，包含目前啟用的部署，其餘名額由最新部署補足。刪除部署與檔案清單後，重用既有 artifact 移除流程檢查上傳引用，只有不再被任何部署引用的部署包才會移除。上傳與啟用後的即時清理仍遵循 Pages 歷史保留數設定。

建議依流量設定保留天數。高流量服務可先使用較短的存取記錄保留期，再依實際稽核需求調整；效能資料通常只需保留數天。

## 節點每日配額與完整日期摘要

PostgreSQL、SQLite 與 ClickHouse 每個 UTC 日期、每個節點最多持久化 1 GB（十進位 1,000,000,000 bytes）明細。邏輯大小為固定欄位 64 bytes 加上 node_id、remote_addr、region、host、path、user_agent、cache_status 的 UTF-8 位元組數，不含資料庫索引、WAL、壓縮與實體列開銷。以記錄內容的 SHA-256 排序，ID 為同內容的次排序，保留配額內的優先序前綴；排除建立時間、追蹤標記及 ID 對內容雜湊的影響。後到高優先序資料會替換既存低優先序明細；持久化拒絕邊界避免可變長度記錄因重新分批而補入原先邊界以後的記錄。固定內容及 ID 下不依賴到達順序或分批方式，並非等機率均勻抽樣。超額記錄及抵達時已超過七天的記錄只寫摘要。

摘要按 UTC 日期、節點、HTTP 狀態碼保留請求數、回應流量與請求流量，所有成功進入持久化流程的記錄均參與摘要。配額、摘要與批次帳本屬 server 插件，保存於 PostgreSQL／SQLite 業務主庫；切換 ClickHouse 日誌明細後端不會重設配額或複製、累加摘要。明細的 summary_tracked 欄位會隨遷移保留。完整 UTC 日期、無 host/path/IP 篩選的流量總計與狀態碼查詢改讀摘要；其他視窗和維度仍讀抽樣明細。摘要無法還原 UV，摘要查詢的 UniqueIPCount 為 0。

歷史摘要按未處理索引分頁，每頁最多 1,000 筆；分組增量與列級 summary_tracked checkpoint 在同一交易提交，使用實體列鍵避免重複 ID 漏計。月份刪除限定該月範圍，後續每 1,000 筆刪除不再聚合整張表，且只刪除已 checkpoint 明細。月份日誌清理與彙總壓縮不執行抽樣淘汰或每日清理，因此每工作百萬筆預算涵蓋該工作所有明細刪除；獨立的寫入或每日清理不計入此工作。刪除後晚到歷史資料仍可累積。初版已有 legacy 摘要時，第一次按分組轉換基線，以同一查詢取得統計與最大 ID，在鎖內重查 checkpoint，使用既存最大值保留初版已刪除資料的累積，再開始增量 checkpoint；初版已混入晚到與刪除的損失無法精確還原。摘要失敗會阻止刪除。舊版寫入端須先停止，初次基線轉換及整月分區移除不保證與舊版旁路寫入併發安全。

三庫共用主庫明細索引、摘要、配額及 outbox。新批次的原始明細、摘要與待送記錄同交易提交；抽樣由資料庫對 sample_key 排序，以 SUM 視窗求位元組邊界，不把整日明細載入 Go。超額明細每次最多刪除 500 筆，各頁獨立交易；日期節點清單每頁 100 組、舊明細補鍵每頁 500 筆。SQL 排序仍需資料庫 CPU、暫存與 IO，不能保證固定執行時間；大量歷史資料宜先在維護窗評估。

### PostgreSQL 大型明細表索引升級

新增 migration 16／17 在分區表上僅以 `ON ONLY` 建立 `idx_node_access_pending`（`summary_tracked, logged_at, id`）與 `idx_node_access_sample`（`node_id, logged_at, sample_key`）的父索引定義，不遞迴掃描既有分區。既有普通表會略過這兩個索引的自動建立；SQLite 維持原有建索引行為。批次帳本的 `created_at` 索引仍由 migration 16 建立。

父索引在既有分區尚未補齊時可能顯示 `indisvalid = false`，這是預期狀態。既有分區須另行並行建索引及掛接；新建空分區會繼承父表的索引定義。未補齊前，歷史摘要、清理與抽樣仍可執行，但可能需要大量掃描及排序。這項修正避免升級時的同步全分區索引建置，不保證完全無鎖；新增欄位、父索引定義及掛接仍需取得 DDL 鎖，應避開長交易並設定適當的 `lock_timeout`。

先完成 migration 17，再由維運連線針對每個既有葉分區執行以下步驟。將範例 schema、分區名稱與索引名稱替換為實際值；每次僅處理一個分區，同一表的並行索引依序建立。`CREATE INDEX CONCURRENTLY` 必須獨立執行，不可包在 `BEGIN`、`DO` 或應用程式的遷移交易內。

```sql
SET lock_timeout = '5s';
CREATE INDEX CONCURRENTLY of_nal_202611_pending
    ON public.of_node_access_logs_202611 (summary_tracked, logged_at, id);
CREATE INDEX CONCURRENTLY of_nal_202611_sample
    ON public.of_node_access_logs_202611 (node_id, logged_at, sample_key);
ALTER INDEX public.idx_node_access_pending
    ATTACH PARTITION public.of_nal_202611_pending;
ALTER INDEX public.idx_node_access_sample
    ATTACH PARTITION public.of_nal_202611_sample;
```

以上適用於父表直接包含月份葉分區的配置；若有多層分區，需在中間分區以 `ON ONLY` 建立相同定義，再逐層由下往上掛接。所有既有分區完成後，PostgreSQL 會自動將父索引標記為有效。普通表則獨立執行下列兩句，無須掛接：

```sql
CREATE INDEX CONCURRENTLY idx_node_access_pending
    ON public.of_node_access_logs (summary_tracked, logged_at, id);
CREATE INDEX CONCURRENTLY idx_node_access_sample
    ON public.of_node_access_logs (node_id, logged_at, sample_key);
```

執行前與重試前應查驗 `pg_index`、`pg_get_indexdef()` 及 `pg_inherits`，確認索引定義、有效性與掛接關係，避免重複建立。並行建索引失敗可能留下無效索引；不要以 `IF NOT EXISTS` 略過同名物件就視為完成。僅對確認未掛接且無效的葉索引使用 `DROP INDEX CONCURRENTLY` 清理後重試；已有效且定義相同的索引可直接重用及掛接。掛接遇到鎖逾時可稍後重試。並行建立仍會消耗 IO、CPU、磁碟與 WAL，應監控負載及 `pg_stat_progress_create_index`。

若開發或測試環境已執行舊版 16／17，更新 SQL 不會重跑已記錄版本，既有有效索引可保留，不應重設遷移版本或刪除重建。回滾會刪除父索引及已掛接子索引，且仍需 DDL 鎖；正式環境優先前滾。

### 帳本保留與跨庫重送

每日清理以 500 筆分頁回收八天以前的批次帳本與日期 quota；摘要永久保留。同一 ID 集合在帳本保留窗口內的成功批次重試不重複記帳，窗口外重送、重新分批或重新分配 ID 不保證事件級去重。清理未執行時帳本仍會持續增長，待送 outbox 不因過期而丟棄。

ClickHouse 正常寫入經持久化 outbox 逐批送出；先同步刪除同批 ID，再批量插入，因此在送達後確認失敗可重播。淘汰明細與歷史 summary_tracked 標記也經 outbox 重試；跨庫網路 IO 不放入主庫交易。主庫保留七天抽樣明細供配額與重送索引，故啟用 CH 仍占用主庫空間。CH 斷線時保留 outbox 並回傳實際錯誤，下一次寫入、摘要讀取、每日清理或切換檢查會重送，尚無獨立秒級 outbox 排程。PG 使用 session advisory lock 序列化 flush，取得鎖、讀取 outbox 與確認刪除均使用同一條專用連線，支援 MaxOpenConns=1；取消後使用獨立五秒逾時 context 解鎖，取得鎖結果不明或解鎖失敗時丟棄連線，避免帶鎖回池。同進程等待 flush 可由 context 取消；SQLite 不支援跨進程 flush 鎖，應維持單一寫入進程。

歷史 CH 明細每頁 500 筆，摘要與 checkpoint 待送記錄同交易保存，CH 標記成功後才讀下一頁。已摘要的 CH 明細也以持久化 ID cursor 分頁補入主庫索引。此流程假設雪花 ID 穩定且唯一、禁止舊版旁路寫入；歷史重複 ID、較小 ID 事後插入或初版已刪除而未保存的資訊無法完整復原。同步 mutation 較昂貴，需真實 CH 壓測確認吞吐；重播期間同批明細可能短暫不可見。切換前會先補摘要、排空 outbox，再清空目標與複製。

清空主庫節點明細時，同交易重設 CH 明細索引游標，保留摘要與配額；CH 切回主庫若複製失敗，恢復使用 CH 後可從頭補回 shadow 索引，匯入已追蹤明細不再累加摘要。

Agent 一般上報每批最多 1 MiB、1000 條存取記錄與 4 個緩衝記錄，最短間隔為 3 秒。主站接收上限為 8 MiB；收到 413 時 Agent 會縮小批次，收到 429 時依退避時間與 Retry-After 延後重送。主站成功回應表示資料已接受進入非同步寫入佇列，並不代表資料庫已完成持久化。

ClickHouse 原始節點日誌與可觀測表移除固定 TTL，依清理任務的實際 cutoff 保留資料，需確保每日系統清理持續執行。PostgreSQL 刪除資料後的空間通常先供資料庫重用，需要在維護窗執行完整維護才能縮減實體檔案；操作前應確認備份與可用空間。程式限額不等於現場網路流量上限，應在主站與節點升級後比較接收位元組、拒收次數及各表容量的增長率。
