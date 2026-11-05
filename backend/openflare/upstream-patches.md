# 上游本地补丁清单

## 待回流 Wavelet

- `backend/plugins/domain/admin/service/cleanup.go`：移除对 admin repository 中不存在的 `CleanupRetainedVersions` 调用。OpenFlare 配置与 Pages 版本清理由 `backend/openflare/plugins/server/plugin.go` 订阅 `SystemCleanup` 事件处理，上游 admin 不依赖下游业务。
- `backend/plugins/domain/admin/plugin.go`、`backend/plugins/domain/admin/plugin_test.go`：移除每日 03:30 的 `database:maintenance` 默认排程，保留任务注册供管理员手动执行；每日 03:00 仅执行 `system:cleanup`，避免自动触发 PostgreSQL `VACUUM FULL`。测试验证完整维护不注册默认排程且仍可手动执行。

- `backend/plugins/domain/admin/repository/task.go`、`db.go`、`task_cleanup_test.go`：将任务执行记录清理的 `Limit().Delete()` 改为按主键排序、每批最多 1000 笔的 ID 子查询删除，共用批次逻辑并检查未初始化的数据库；子查询及外层删除均限制终态与过期时间，保留 pending/running。回归测试覆盖逐批删除笔数、保留期限边界、活动任务、重复清理及 nil DB，并验证预览计数与成功 3 天、失败 30 天的实际清理笔数一致。
- `backend/core/contracts/log_storage.go`、`backend/plugins/domain/admin/`：擴充上游日誌儲存契約與管理工作，支援必填 YYYY-MM 月份、每批最多 1,000 筆、每工作最多 1,000,000 筆，並透過同範圍存在查詢回報真實 has_more；容量估值與實際刪除進度分開呈現。工作名稱、參數與執行文案統一為「月份日誌清理與彙總壓縮」，說明先保留摘要再刪過期明細，不執行 VACUUM 縮檔；保留 `force_log_cleanup` 與 `system:force-log-cleanup` 標識相容。需回流 Wavelet 後確認下游仍只透過契約呼叫。

- `backend/plugins/domain/upload/plugin.go`、`task/cleanup.go`、`plugin_test.go` 與 `backend/plugins/domain/admin/plugin_test.go`：upload 孤兒檔案清理只訂閱系統清理事件，不再註冊或覆蓋 admin 的 `system:cleanup`；保留的內部工作標識改用 upload 前綴，並驗證兩種插件註冊順序均保留 admin handler 與每日排程。

以上为本地待回流补丁，尚未同步至 Wavelet；回流并合并上游后确认修复仍在，再移除此登记。
