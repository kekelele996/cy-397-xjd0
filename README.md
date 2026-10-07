# 合同模板生成与法律工单 API 服务

一个纯后端 API 服务，提供常用合同模板管理、合同填充生成、法律咨询工单提交与跟踪，并支持将合同导出为 PDF。

## 快速启动（Docker Compose）

首次启动前请先复制环境变量模板：

```bash
cp .env.example .env
```

然后一键构建并启动（backend + MySQL 8.0）：

```bash
docker compose --env-file .env up -d --build --wait
```

启动完成后：

```bash
curl http://127.0.0.1:19412/healthz
```

停止并清理数据：

```bash
docker compose --env-file .env down -v --remove-orphans
```

## 项目主要功能

- 合同模板管理：租赁合同、劳动合同、借款合同、合作协议、保密协议。
- 合同生成：选择模板并填充变量，生成纯文本 / HTML 合同，支持 wkhtmltopdf 导出 PDF。
- 合同签署状态管理：草稿 → 待签署 → 已签署 → 已过期，记录签署时间与签署方信息。
- 法律工单系统：提交劳动纠纷 / 合同纠纷 / 房产纠纷 / 知识产权 / 其他类型工单。
- 工单流转：待处理 → 处理中 → 已回复 → 已关闭，支持文字与附件回复。
- 常见法律知识库：FAQ 分类检索与关键词搜索。
- 用户合同库：查看自己创建的全部合同，按状态筛选，支持模板收藏。

## 本地开发

要求本机安装 Go 1.22+ 与 MySQL 8.0，并准备数据库：

```bash
mysql -uroot -p < database/init.sql
cp .env.example .env
# 本地开发时将 .env 中的 DB_HOST 改为 127.0.0.1，DB_PORT 改为本机 MySQL 端口
export $(grep -v '^#' .env | xargs)
cd backend
go mod tidy
go run ./cmd/server
```

服务默认监听 `8080` 端口，健康检查地址为 `http://127.0.0.1:8080/healthz`。

## 技术栈

| 层级 | 技术 |
| --- | --- |
| 后端 | Go 1.22 + Gin + GORM |
| 数据库 | MySQL 8.0 |
| 认证 | JWT（github.com/golang-jwt/jwt/v5） |
| PDF 生成 | wkhtmltopdf |
| 配置 | caarlos0/env/v11 |
| 日志 | log/slog |
| 参数校验 | go-playground/validator/v10 |

## 项目目录结构

```text
.
├── backend/
│   ├── cmd/server/main.go
│   ├── internal/
│   │   ├── config/       # 环境变量配置
│   │   ├── constants/    # 错误码与状态枚举
│   │   ├── dto/          # 请求/响应结构体
│   │   ├── handler/      # HTTP 处理层
│   │   ├── middleware/   # JWT、CORS、日志、错误处理
│   │   ├── model/        # GORM 模型
│   │   ├── repository/   # 数据访问层
│   │   ├── router/       # 路由注册
│   │   └── service/      # 业务逻辑层
│   ├── templates/        # 合同模板文件
│   └── Dockerfile
├── database/init.sql     # 数据库初始化脚本
├── migrations/           # 数据库迁移脚本
├── api/                  # OpenAPI 文档
├── deploy/               # 部署说明
├── docker-compose.yml
├── .env.example
└── README.md
```

## 主要 API 列表

统一响应格式：

```json
{ "code": 0, "message": "ok", "data": {} }
```

| 方法 | 路径 | 说明 | 认证 |
| --- | --- | --- | --- |
| POST | /api/v1/auth/register | 注册 | 否 |
| POST | /api/v1/auth/login | 登录获取 JWT | 否 |
| GET | /api/v1/templates | 模板列表 | 否 |
| GET | /api/v1/templates/:id | 模板详情 | 否 |
| GET | /api/v1/templates/favorites | 我的模板收藏 | 是 |
| POST | /api/v1/templates/:id/favorite | 收藏模板 | 是 |
| DELETE | /api/v1/templates/:id/favorite | 取消收藏 | 是 |
| POST | /api/v1/admin/templates | 新建模板 | 是 |
| PUT | /api/v1/admin/templates/:id | 更新模板 | 是 |
| DELETE | /api/v1/admin/templates/:id | 删除模板 | 是 |
| POST | /api/v1/contracts | 填充生成合同 | 是 |
| GET | /api/v1/contracts | 用户合同库 | 是 |
| GET | /api/v1/contracts/:id | 合同详情 | 是 |
| POST | /api/v1/contracts/:id/submit | 提交待签署 | 是 |
| POST | /api/v1/contracts/:id/sign | 签署合同 | 是 |
| POST | /api/v1/contracts/:id/expire | 合同过期 | 是 |
| GET | /api/v1/contracts/:id/signers | 签署方列表 | 是 |
| GET | /api/v1/contracts/:id/export | 导出 PDF | 是 |
| POST | /api/v1/tickets | 提交法律工单 | 是 |
| GET | /api/v1/tickets | 工单列表 | 是 |
| GET | /api/v1/tickets/:id | 工单详情及回复 | 是 |
| POST | /api/v1/tickets/:id/replies | 添加回复 | 是 |
| GET | /api/v1/tickets/:id/replies | 回复列表 | 是 |
| PATCH | /api/v1/tickets/:id/status | 工单流转 | 是 |
| GET | /api/v1/faqs | FAQ 搜索 | 否 |
| GET | /api/v1/faqs/:id | FAQ 详情 | 否 |
| POST | /api/v1/admin/faqs | 新建 FAQ | 是 |
| PUT | /api/v1/admin/faqs/:id | 更新 FAQ | 是 |
| DELETE | /api/v1/admin/faqs/:id | 删除 FAQ | 是 |

## 环境变量说明

| 变量 | 默认值 | 说明 |
| --- | --- | --- |
| COMPOSE_PROJECT_NAME | contractapi | Compose 项目名 |
| DB_NAME | contractapi | MySQL 数据库名 |
| DB_USER | contract | MySQL 业务用户 |
| DB_PASSWORD | - | MySQL 业务用户密码 |
| DB_ROOT_PASSWORD | root_secret | MySQL root 密码 |
| DB_HOST | mysql | MySQL 主机（Compose 内为 mysql） |
| DB_PORT | 3306 | MySQL 端口 |
| JWT_SECRET | - | JWT 签名密钥，生产环境必须修改 |
| BACKEND_PORT | 19412 | 后端对外映射端口 |

## License

MIT
