# eino-multi-agent

[中文](./README.md) | **English**

A long-running work harness built on CloudWeGo Eino. It mirrors the core loop of “Doubao Work Tasks”: set a goal, plan and execute, confirm risky actions, and deliver downloadable artifacts.

## Features

| Capability | Description |
|---|---|
| Task execution | Create a goal, upload inputs, pick skills; DeepAgent iterates in a sandbox with SSE event streaming |
| Risk confirmation | Policies `on_risk` / `always` / `never`; overwrite, delete, dangerous shell, outbound POST, etc. pause for approval |
| Artifact delivery | Scan `output/` with list, preview, and download |
| Project workspaces | Shared workspace and file uploads; tasks can attach to a project |
| Schedules | Once / daily / weekly runs, plus manual trigger |
| Connectors | On success: webhook notify or copy artifacts to a local directory |
| Audit log | Critical writes and system recovery actions are persisted and viewable in the UI |
| Eval report | Score historical tasks against built-in cases (weekly report / research / organize) |
| Crash recovery | After restart, resume or re-run `queued` / `running` tasks from checkpoints |
| API auth | When `auth_token` is set, require `Authorization: Bearer` or `?token=` |

Built-in skills (`skills/`):

- `weekly-report`: build a weekly report from tables/text → `output/weekly-report.md`
- `research`: public-web research and synthesis → `output/research-report.md`
- `file-organize`: classify files and produce an index → `output/file-index.md`

Agent tools include file I/O, shell, DuckDuckGo search, `fetch_url`, skill loading, and `confirm_action`.

## Stack

- Backend: Gin + GORM + PostgreSQL + Eino DeepAgent
- Frontend: Vue 3 + Element Plus + Vite
- Config: `config.yaml` + `.env` / environment variables (env wins)

## Quick start

```bash
# 1. Start Postgres (host port 14432)
docker compose up -d

# 2. Configure
cp config.example.yaml config.yaml
cp .env.example .env
# Edit .env; set at least EINO_LLM_API_KEY / EINO_LLM_MODEL

# 3. Start API (default :8180, see .env)
go run ./cmd/server

# 4. Start frontend (Vite :5273, proxies /api → :8180)
cd web && npm install && npm run dev
```

Open http://localhost:5273

Sample task: upload `testdata/sample/sales.csv`, set the goal to “Generate this week’s sales report from input/sales.csv”, and select skill `weekly-report`.

### CLI

```bash
# Defaults to http://127.0.0.1:8180; override with EINO_API_BASE
go run ./cmd/cli create --goal "Generate a weekly report from input/sales.csv" --file testdata/sample/sales.csv --skills weekly-report
go run ./cmd/cli list
go run ./cmd/cli show <task-id>
go run ./cmd/cli confirm <task-id> --approved=true
go run ./cmd/cli retry <task-id>
go run ./cmd/cli cancel <task-id>
```

### Eval CLI

Score historical tasks against built-in cases and print JSON:

```bash
go run ./cmd/eval -limit 200
```

Also available on the Eval page or `GET /api/v1/eval`.

### Production build

After building the frontend, Gin serves `web/dist`:

```bash
cd web && npm run build
go run ./cmd/server
```

## Web console

| Page | Path | Purpose |
|---|---|---|
| Tasks | `/` | Create/list, event stream, confirm, retry, artifact preview/download |
| Projects | `/projects` | Project management and shared file uploads |
| Schedules | `/schedules` | Create/toggle/run schedules immediately |
| Connectors | `/connectors` | Webhook / local-directory delivery |
| Audit | `/audit` | Audit log list |
| Eval | `/eval` | Case completion rates and details |

## HTTP API (`/api/v1`)

| Method | Path | Description |
|---|---|---|
| GET | `/health` | Health check (allowed even when auth is on) |
| GET | `/meta` | Model readiness, skills, confirm policies |
| GET/POST | `/tasks` | List / create (multipart upload supported) |
| GET | `/tasks/:id` | Detail (events + artifacts) |
| POST | `/tasks/:id/cancel` | Cancel |
| POST | `/tasks/:id/confirm` | `{"approved": true\|false}` |
| POST | `/tasks/:id/retry` | Retry after failure |
| GET | `/tasks/:id/events` | SSE event stream |
| GET | `/tasks/:id/artifacts[/:aid[/preview]]` | List / download / preview artifacts |
| GET/POST | `/projects` | List / create projects |
| GET | `/projects/:id` | Project detail, shared files, related tasks |
| POST | `/projects/:id/files` | Upload shared files |
| GET/POST | `/schedules` | List / create schedules |
| POST | `/schedules/:id/run` | Run one cycle now |
| POST | `/schedules/:id/toggle` | Enable/disable |
| GET/POST | `/connectors` | List / create connectors |
| POST | `/connectors/:id/toggle` | Enable/disable |
| POST | `/connectors/:id/test` | Test delivery |
| POST | `/hooks/echo` | Webhook debug echo (auth bypass) |
| GET | `/audit` | Audit logs |
| GET | `/eval` | Eval report |

Task status: `queued` → `running` → (optional `waiting_confirm`) → `succeeded` / `failed` / `cancelled`.

## Configuration

Copy `config.example.yaml` / `.env.example`. Common environment variables:

| Variable | Meaning |
|---|---|
| `EINO_CONFIG` | Config file path, default `./config.yaml` |
| `EINO_DATABASE_DSN` | Postgres DSN |
| `EINO_SERVER_ADDR` | HTTP listen address, e.g. `:8180` |
| `EINO_SERVER_MODE` | Gin mode, e.g. `debug` / `release` |
| `EINO_AUTH_TOKEN` | Non-empty enables API Bearer auth |
| `EINO_LLM_PROVIDER` | `openai` / `ark` / `ollama` |
| `EINO_LLM_API_KEY` | Model API key |
| `EINO_LLM_BASE_URL` | OpenAI-compatible gateway; optional |
| `EINO_LLM_MODEL` | Primary model name |
| `EINO_LLM_FALLBACK_MODELS` | Comma-separated fallbacks when primary fails |
| `EINO_LLM_TIMEOUT` | Per ChatModel call timeout; reasoning models often need `10m` |
| `EINO_AGENT_MAX_ITERATION` | Max agent iterations |
| `EINO_AGENT_RUN_TIMEOUT` | Per-task overall timeout |
| `EINO_AGENT_LANGUAGE` | Agent language preference, default `zh` |
| `EINO_WORKSPACE_ROOT` | Task sandbox root |
| `EINO_SKILLS_DIR` | Skills directory |

YAML also supports `server.cors_origins`, `search.enabled` / `max_results`, database pool settings, and more — see `config.example.yaml`.

LLM examples are in `.env.example` (OpenAI-compatible, Volcengine Ark, local Ollama).

## Layout (brief)

```
cmd/server          # HTTP server entry
cmd/cli             # Task CLI
cmd/eval            # Eval CLI
internal/api        # Gin routes and auth
internal/harness    # Runtime, checkpoint recovery, event bus
internal/agent      # DeepAgent factory, tools, skills
internal/confirm    # Risk classification and confirm gate
internal/scheduler  # Scheduled runs
internal/connector  # Webhook / local-dir delivery
internal/eval       # Case scoring
internal/store      # GORM models and persistence
internal/workspace  # Sandbox and project shared dirs
skills/             # Built-in SKILL.md files
web/                # Vue console
```
