# 上游本地补丁清单

## 待回流 Wavelet

- `backend/plugins/domain/admin/service/cleanup.go`：移除对 admin repository 中不存在的 `CleanupRetainedVersions` 调用。OpenFlare 配置与 Pages 版本清理由 `backend/openflare/plugins/server/plugin.go` 订阅 `SystemCleanup` 事件处理，上游 admin 不依赖下游业务。
- `backend/plugins/domain/admin/plugin.go`、`backend/plugins/domain/admin/plugin_test.go`：移除每日 03:30 的 `database:maintenance` 默认排程，保留任务注册供管理员手动执行；每日 03:00 仅执行 `system:cleanup`，避免自动触发 PostgreSQL `VACUUM FULL`。测试验证完整维护不注册默认排程且仍可手动执行。

- `backend/plugins/domain/admin/repository/task.go`、`db.go`、`task_cleanup_test.go`：将任务执行记录清理的 `Limit().Delete()` 改为按主键排序、每批最多 1000 笔的 ID 子查询删除，共用批次逻辑并检查未初始化的数据库；子查询及外层删除均限制终态与过期时间，保留 pending/running。回归测试覆盖逐批删除笔数、保留期限边界、活动任务、重复清理及 nil DB，并验证预览计数与成功 3 天、失败 30 天的实际清理笔数一致。
- `backend/core/contracts/log_storage.go`、`backend/plugins/domain/admin/`：擴充上游日誌儲存契約與管理工作，支援必填 YYYY-MM 月份、每批最多 1,000 筆、每工作最多 1,000,000 筆，並透過同範圍存在查詢回報真實 has_more；容量估值與實際刪除進度分開呈現。工作名稱、參數與執行文案統一為「月份日誌清理與彙總壓縮」，說明先保留摘要再刪過期明細，不執行 VACUUM 縮檔；保留 `force_log_cleanup` 與 `system:force-log-cleanup` 標識相容。需回流 Wavelet 後確認下游仍只透過契約呼叫。

- `backend/plugins/domain/upload/plugin.go`、`task/cleanup.go`、`plugin_test.go` 與 `backend/plugins/domain/admin/plugin_test.go`：upload 孤兒檔案清理只訂閱系統清理事件，不再註冊或覆蓋 admin 的 `system:cleanup`；保留的內部工作標識改用 upload 前綴，並驗證兩種插件註冊順序均保留 admin handler 與每日排程。

- `backend/plugins/infra/database/postgres.go`、`sqlite_test.go` 与 `backend/plugins/domain/admin/{handler,service,repository}/db.go`、`repository/database_export_test.go`：移植 OpenFlare `ada05505` 的 SQLite WAL 与一致性快照到 Cordis 平台层；文件库启动启用 WAL，内存库保留 memory，管理端导出通过既有 DBService 执行 `VACUUM INTO`，使用私有临时目录并在流式下载后清理，避免遗漏 WAL 已提交数据。保留原导出文件名及旧文件访问接口，不更动 Schema；需回流 Wavelet 后移除此登记。
  本輪補上匯出介面的文件註解，並以 `os.OpenRoot` 限定快照讀取於私有暫存目錄，避免讀取逃逸目錄的符號連結；此修正一併待回流。

- `backend/cmd/{root.go,trace_config.go,trace_config_test.go}`、`backend/pkg/trace/{trace.go,trace_provider.go,trace_provider_test.go}`、`backend/go.mod`、`backend/go.sum` 与 `manifest/config/config.default.yaml`：移植 OpenFlare `8d8b3e1e` 的 OTLP/HTTP protobuf 导出，保留 Cordis ConfigRegistry、Wavelet module 与既有 Tracer 名称；协议依序采用非空的 trace 专用环境变量、通用环境变量、YAML 配置及默认 gRPC，地址与请求头继续由标准 SDK 处理。HTTP exporter 与现有 OTel SDK 对齐至 v1.45.0，不降级既有依赖；测试覆盖协议优先级、HTTP 路径、请求头、资源属性、非法协议及 gRPC 相容性。此通用能力需回流 Wavelet 后移除此登记。

- `backend/plugins/drivers/driver_http/frontend.go`、`frontend_test.go`：對照 OpenFlare `a2404a50`，在 Cordis HTTP driver 加入通用 Next.js 靜態匯出正整數 ID 模板 fallback；精確資產優先，其次對應同目錄 `1.html`、`1.txt`、`1/__next.*.txt`，模板不存在時維持既有 SPA shell。平台不硬編碼 Cloudflare／websites 業務前綴；無效 ID、額外子路徑與 API／檔案／Swagger 命名空間不套模板。17 個 HTTP 回歸案例及 driver 套件通過。此通用能力尚未回流 Wavelet，回流並合併後確認仍在，再移除此登記。

- `backend/go.mod`、`backend/go.sum`：對齊 OpenFlare `e2acbecb` 的 `github.com/go-jose/go-jose/v4 v4.1.6`，保留 Wavelet module 與既有 OTel 版本；Auth service、go-oidc 及 go-jose／JWT 套件測試與 module checksum 驗證通過。此平台共用 OIDC 間接依賴升級尚待回流 Wavelet；本次沒有宣稱真實第三方登入已驗證。

以上为本地待回流补丁，尚未同步至 Wavelet；回流并合并上游后确认修复仍在，再移除此登记。
