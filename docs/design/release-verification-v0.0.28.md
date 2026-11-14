# v0.0.28 發布驗證

## 版本

- 版本：`v0.0.28`
- 發布日期：2026-11-15
- 版本類型：PATCH
- 變更範圍：同步健康、任務重試、節點批次操作、WAF 模擬、代理分頁及行動版管理介面

## 已完成驗證

- GitHub Linux Integration Validation：後端與前端工作成功
- 後端相關套件回歸測試通過
- 前端 Cloudflare 與 dashboard 測試通過
- TypeScript、ESLint、i18n key 檢查通過
- 前端靜態嵌入建置通過
- Swagger 文件已重新生成
- PostgreSQL／SQLite migration 已納入版本控制，未在正式資料庫執行

## 發布限制

- 未執行正式部署、正式資料庫 migration 或服務健康檢查
- Windows 本機完整 Make 品質門禁受 Unix shell 與工具缺失限制；Linux CI 已完成對應驗證
- GitHub Actions 產物、SHA-256 校驗檔與容器雙架構結果，待正式 tag workflow 完成後核對
