# Cloudflare 健康同步任務去重

`RegisterCron` 更新的是程序內的排程 registry，不會更新 `w_schedules`。目前 Cordis Asynq cron driver 的 `Start` 使用 registry；保留的 `ReloadScheduler` 使用資料庫。兩條路徑分別處理，不能只改註冊時的 cron 字串。

健康同步 `cloudflare:sync_all_groups` 啟用 singleton metadata；手動派送、手動重試、registry cron 及資料庫 cron 均使用相同的空 payload、default queue、固定 TaskID 與 Asynq Unique 選項。單例任務只接受空參數，避免忽略有效業務參數。此路徑不把 trace envelope 寫進 queue payload，避免不同控制面的 trace 產生不同 Unique 鍵。

固定 TaskID 會拒絕同 queue 內仍處於 pending、active 或 retry 的任務；Unique 有效期為 24 小時，提供跨控制面的補充防重。成功完成後任務可進入下一週期。最終失敗進入 archived 後，手動重試會先清理該終態任務，再以固定 TaskID 重新入隊；pending、running 或 retry 中不會被手動重試覆蓋。超過 24 小時仍未完成的任務仍不承諾由 Unique 提供無限期鎖定，固定 TaskID 仍是同 queue 的主要防線。

server migration 00026 同時辨識實際任務名稱 `cloudflare:sync_all_groups` 與 metadata 別名 `of_cloudflare_sync_all_groups`，依主鍵 ID 升序保留最小 ID 一列、刪除其餘同類列，並只將保留列的 task type 正規化為實際任務名稱。保留列的名稱、cron、payload（包含 NULL）、啟用狀態及建立／更新時間均不覆寫；沒有既有列時才以資料庫自增 ID 補建啟用的四小時預設排程。其他任務不受影響。這不會將既有自訂週期強制改成四小時，也不會同步改寫程序內 registry 的排程。

新舊 task type 混存時沒有類型優先權，啟停狀態衝突也不採用「任一啟用」或「任一停用」合併：一律採最小 ID 主列的完整設定。例如主列停用而重複列啟用時，結果停用；主列啟用而重複列停用時，結果啟用。非主列可能是使用者刻意建立的不同設定，SQL 無法判斷其用途；升級前必須比較兩種 task type 的全部資料，將希望採用的設定先整理到主列，確認其餘列可刪除後再升級。

**00026 是不可逆資料遷移。** `Down` 的 `SELECT 1` 只是保留資料的空操作，不會恢復刪除列、原 task type，也不會刪除本次新建的預設排程。Goose 降版成功只代表版本紀錄已回退，不能視為資料已還原。此修正也不能恢復已執行舊版 00026 時遭覆寫或刪除的設定；已套用環境不會因更新 SQL 而自動重跑，必須從升級前備份人工還原。

回復程序：

1. 升級前在維護窗口停止相關 scheduler／排程寫入，使用資料庫原生工具備份完整資料庫與兩種 task type 的全部列（包含 ID、名稱、cron、payload、啟停及時間欄位），記錄 Goose 版本，並驗證備份可讀取。
2. 在資料庫副本預演升級，核對最小 ID 主列的設定、其他 task type 不變及健康排程只剩一列，再於正式環境執行受控升級。
3. 需要回復時先停止 scheduler／排程寫入並另存升級後資料；優先將完整備份還原到隔離環境驗證，再於同一維護窗口還原。若升級後已有其他寫入，須人工比對後只還原這兩種 task type 的相關列，避免覆蓋新設定；還原原始 ID 前檢查衝突並校正自增序列，確認資料與 Goose 版本一致後才重新啟動。

測試使用既有 miniredis 與真正的 Asynq client/server，涵蓋不同 client、pending、active、完成後再次入隊、終態 archived 後手動重試、超過 Unique TTL 的長 pending，以及重複派送不保留額外執行紀錄。00026 遷移測試將 SQLite 與 PostgreSQL SQL 原文分別放在 SQLite 執行，涵蓋無既有列、自增 ID、單一新／舊類型、同類重複、新舊混存的兩種 ID 順序、所有啟停組合、自訂欄位保留、其他任務不變、重複 Up 與 Down 後再次 Up；並驗證 Down 不會還原資料。這是兩份 SQL 的 SQLite 行為測試，尚未在真實 PostgreSQL 執行，不代表 PostgreSQL 整合驗證通過。
