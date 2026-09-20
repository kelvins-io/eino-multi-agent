# eino-multi-agent

基于 CloudWeGo Eino 的长任务工作 Harness，对标「豆包工作任务」的核心路径：下达目标、拆解执行、确认风险操作、交付可下载产物。

## 功能概览

| 能力 | 说明 |
|---|---|
| 任务执行 | 创建目标、上传输入文件、选择技能，DeepAgent 在沙箱中迭代执行，SSE 实时推送事件 |
| 风险确认 | 策略 `on_risk` / `always` / `never`；覆盖、删除、危险 shell、对外 POST 等会暂停等待批准 |
| 产物交付 | 扫描 `output/`，支持列表、预览与下载 |
| 项目空间 | 共享工作区与文件上传，任务可挂靠项目 |
| 定时任务 | 一次性 / 每日 / 每周调度，可手动立刻跑一轮 |
| 连接器 | 任务成功后 Webhook 通知或复制产物到本地目录 |
| 审计日志 | 关键写操作与系统恢复动作落库，可在 UI 查看 |
| 评估报告 | 按内置用例（周报 / 调研 / 整理）统计历史任务完成率 |
| 崩溃恢复 | 进程重启后对 `queued` / `running` 任务按检查点续跑或重新执行 |
| API 鉴权 | 配置 `auth_token` 后要求 `Authorization: Bearer` 或 `?token=` |

内置技能（`skills/`）：

- `weekly-report`：根据表格/文本生成周报 → `output/weekly-report.md`
- `research`：公开资料检索与综述 → `output/research-report.md`
- `file-organize`：整理归类并生成目录 → `output/file-index.md`

Agent 侧工具包括文件读写、shell、DuckDuckGo 搜索、`fetch_url`、技能加载与 `confirm_action` 等。

## 技术栈

- 后端：Gin + GORM + PostgreSQL + Eino DeepAgent
- 前端：Vue 3 + Element Plus + Vite
- 配置：`config.yaml` + `.env` / 环境变量（环境变量优先）

## 快速开始

```bash
# 1. 启动 Postgres（宿主机端口 14432）
docker compose up -d

# 2. 配置
cp config.example.yaml config.yaml
cp .env.example .env
# 编辑 .env，至少设置 EINO_LLM_API_KEY / EINO_LLM_MODEL

# 3. 启动 API（默认 :8180，见 .env）
go run ./cmd/server

# 4. 启动前端（Vite :5273，代理 /api → :8180）
cd web && npm install && npm run dev
```

浏览器打开 http://localhost:5273

示例任务：上传 `testdata/sample/sales.csv`，目标填写「根据 input/sales.csv 生成本周销售周报」，技能勾选 `weekly-report`。

### CLI

```bash
# 默认请求 http://127.0.0.1:8180，可用 EINO_API_BASE 覆盖
go run ./cmd/cli create --goal "根据 input/sales.csv 生成周报" --file testdata/sample/sales.csv --skills weekly-report
go run ./cmd/cli list
go run ./cmd/cli show <task-id>
go run ./cmd/cli confirm <task-id> --approved=true
go run ./cmd/cli retry <task-id>
go run ./cmd/cli cancel <task-id>
```

### 评估 CLI

对历史任务按内置用例打分，输出 JSON：

```bash
go run ./cmd/eval -limit 200
```

也可在前端「评估」页或 `GET /api/v1/eval` 查看。

### 生产构建

前端构建后由 Gin 托管 `web/dist`：

```bash
cd web && npm run build
go run ./cmd/server
```

## Web 控制台

| 页面 | 路径 | 作用 |
|---|---|---|
| 任务 | `/` | 创建/列表、事件流、确认、重试、产物预览下载 |
| 项目 | `/projects` | 项目管理与共享文件上传 |
| 定时 | `/schedules` | 创建/启停/立刻执行定时任务 |
| 连接器 | `/connectors` | Webhook / 本地目录投递 |
| 审计 | `/audit` | 操作审计列表 |
| 评估 | `/eval` | 用例完成率与明细 |

## HTTP API（`/api/v1`）

| 方法 | 路径 | 说明 |
|---|---|---|
| GET | `/health` | 健康检查（鉴权开启时也放行） |
| GET | `/meta` | 模型就绪状态、技能列表、确认策略 |
| GET/POST | `/tasks` | 列表 / 创建（支持 multipart 上传） |
| GET | `/tasks/:id` | 详情（含事件与产物） |
| POST | `/tasks/:id/cancel` | 取消 |
| POST | `/tasks/:id/confirm` | `{"approved": true\|false}` |
| POST | `/tasks/:id/retry` | 失败后重试 |
| GET | `/tasks/:id/events` | SSE 事件流 |
| GET | `/tasks/:id/artifacts[/:aid[/preview]]` | 产物列表 / 下载 / 预览 |
| GET/POST | `/projects` | 项目列表 / 创建 |
| GET | `/projects/:id` | 项目详情、共享文件、关联任务 |
| POST | `/projects/:id/files` | 上传共享文件 |
| GET/POST | `/schedules` | 定时列表 / 创建 |
| POST | `/schedules/:id/run` | 立刻跑一轮 |
| POST | `/schedules/:id/toggle` | 启停 |
| GET/POST | `/connectors` | 连接器列表 / 创建 |
| POST | `/connectors/:id/toggle` | 启停 |
| POST | `/connectors/:id/test` | 测试投递 |
| POST | `/hooks/echo` | Webhook 调试回声（鉴权放行） |
| GET | `/audit` | 审计日志 |
| GET | `/eval` | 评估报告 |

任务状态：`queued` → `running` →（可选 `waiting_confirm`）→ `succeeded` / `failed` / `cancelled`。

## 配置项

复制 `config.example.yaml` / `.env.example`。常用环境变量：

| 变量 | 含义 |
|---|---|
| `EINO_CONFIG` | 配置文件路径，默认 `./config.yaml` |
| `EINO_DATABASE_DSN` | Postgres 连接串 |
| `EINO_SERVER_ADDR` | HTTP 监听地址，示例 `:8180` |
| `EINO_SERVER_MODE` | Gin 模式，如 `debug` / `release` |
| `EINO_AUTH_TOKEN` | 非空则开启 API Bearer 鉴权 |
| `EINO_LLM_PROVIDER` | `openai` / `ark` / `ollama` |
| `EINO_LLM_API_KEY` | 模型密钥 |
| `EINO_LLM_BASE_URL` | 兼容 OpenAI 的网关，可空 |
| `EINO_LLM_MODEL` | 主模型名 |
| `EINO_LLM_FALLBACK_MODELS` | 主模型失败时的备用模型（逗号分隔） |
| `EINO_LLM_TIMEOUT` | 单次 ChatModel 超时，推理模型建议 `10m` |
| `EINO_AGENT_MAX_ITERATION` | Agent 最大迭代次数 |
| `EINO_AGENT_RUN_TIMEOUT` | 单任务总超时 |
| `EINO_AGENT_LANGUAGE` | Agent 语言偏好，默认 `zh` |
| `EINO_WORKSPACE_ROOT` | 任务沙箱根目录 |
| `EINO_SKILLS_DIR` | 技能目录 |

YAML 中还可配置 `server.cors_origins`、`search.enabled` / `max_results`、`database` 连接池等，见 `config.example.yaml`。

LLM 示例见 `.env.example`（OpenAI 兼容、火山方舟 Ark、本地 Ollama）。

## 目录结构（简要）

```
cmd/server          # HTTP 服务入口
cmd/cli             # 任务 CLI
cmd/eval            # 评估 CLI
internal/api        # Gin 路由与鉴权
internal/harness    # 任务运行时、检查点恢复、事件总线
internal/agent      # DeepAgent 工厂、工具、技能
internal/confirm    # 风险分类与确认门闩
internal/scheduler  # 定时调度
internal/connector  # Webhook / 本地目录投递
internal/eval       # 用例评分
internal/store      # GORM 模型与持久化
internal/workspace  # 沙箱与项目共享目录
skills/             # 内置技能 SKILL.md
web/                # Vue 控制台
```
