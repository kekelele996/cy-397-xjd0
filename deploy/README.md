# 部署说明

本项目使用根目录的 `docker-compose.yml` 作为唯一部署编排入口。

```bash
cp .env.example .env
docker compose --env-file .env up -d --build --wait
```

- backend 容器内监听 `8080` 端口，对外映射 `${BACKEND_PORT:-19412}`。
- MySQL 8.0 数据持久化在命名卷 `contractapi_mysql_data`。
- 首次启动时 MySQL 自动执行 `database/init.sql` 创建表结构并写入模板与 FAQ 示例数据。
- 后端启动时还会执行 GORM AutoMigrate 兜底补齐表结构（含 `legal_tickets.contract_id` 等工单-合同联动字段）。

## 升级：工单-合同联动回填

从不含合同编号字段的旧版本升级时，AutoMigrate 会补列，历史工单的 `contract_id` 默认为 0。
升级后执行一次回填命令，按问题描述中的合同编号把合同现状快照登记到工单；回填不出的工单列在 `unresolved`：

```bash
docker compose --env-file .env exec backend /app/ticket-contract-backfill
# 或只处理指定工单
docker compose --env-file .env exec backend /app/ticket-contract-backfill -tickets=12,18
```

也可在应用启动后调用管理接口：`POST /api/v1/admin/tickets/backfill-contract`（空 body 扫描全部）。
对应的纯 SQL 迁移见 `migrations/000002_ticket_contract_link.sql`。

## 生产环境注意事项

- 请务必修改 `.env` 中的 `JWT_SECRET`、`DB_PASSWORD`、`DB_ROOT_PASSWORD`。
- 如需对外暴露 MySQL，可在 `docker-compose.yml` 中为 mysql 服务增加 ports 映射。
- PDF 导出依赖容器内已安装的 `wkhtmltopdf`。
