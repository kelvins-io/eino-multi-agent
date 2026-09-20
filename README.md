# eino-multi-agent

基于 CloudWeGo Eino 的长任务工作 Harness，对标「豆包工作任务」的核心路径：下达目标、拆解执行、确认风险操作、交付可下载产物。

## 技术栈

- 后端：Gin + GORM + PostgreSQL + Eino DeepAgent
- 前端：Vue 3 + Element Plus + Vite
- 配置：`config.yaml` + 环境变量

## 快速开始

```bash
# 1. 启动 Postgres
docker compose up -d

# 2. 配置
cp config.example.yaml config.yaml
cp .env.example .env
# 编辑 .env，至少设置 EINO_LLM_API_KEY / EINO_LLM_MODEL

# 3. 启动 API
go run ./cmd/server

# 4. 启动前端
cd web && npm install && npm run dev
```

浏览器打开 http://localhost:5173

示例任务：上传 `testdata/sample/sales.csv`，目标填写「根据 input/sales.csv 生成本周销售周报」，技能勾选 `weekly-report`。

CLI：

```bash
go run ./cmd/cli create --goal "根据 input/sales.csv 生成周报" --file testdata/sample/sales.csv --skills weekly-report
```

## 配置项

| 变量 | 含义 |
|---|---|
| `EINO_DATABASE_DSN` | Postgres 连接串 |
| `EINO_LLM_PROVIDER` | `openai` / `ark` / `ollama` |
| `EINO_LLM_API_KEY` | 模型密钥 |
| `EINO_LLM_BASE_URL` | 兼容 OpenAI 的网关地址，可空 |
| `EINO_LLM_MODEL` | 模型名 |
| `EINO_SERVER_ADDR` | 默认 `:8080` |
| `EINO_WORKSPACE_ROOT` | 任务沙箱根目录 |

生产构建前端后，Gin 会托管 `web/dist`：

```bash
cd web && npm run build
go run ./cmd/server
```
