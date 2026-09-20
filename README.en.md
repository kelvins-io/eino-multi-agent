# eino-multi-agent

[中文](./README.md) | **English**

A long-running work harness built on CloudWeGo Eino. It mirrors the core loop of “Doubao Work Tasks”: set a goal, plan and execute, confirm risky actions, and deliver downloadable artifacts.

## Features

| Capability | Description |
|---|---|
| User accounts | Register / login issues JWTs; tasks, projects, schedules, and connectors are scoped per user |
| Task execution | Create a goal, upload inputs, pick skills; DeepAgent iterates in a sandbox with SSE event streaming |
| Risk confirmation | Policies `on_risk` / `always` / `never`; overwrite, delete, dangerous shell, outbound POST, etc. pause for approval |
| Artifact delivery | Scan `output/` with list, preview, and download |
| Project workspaces | Shared workspace and file uploads; tasks can attach to a project |
| Schedules | Once / daily / weekly runs, plus manual trigger |
| Connectors | On success: webhook notify or copy artifacts to a local directory |
| Audit log | Critical writes and system recovery actions are persisted and viewable in the UI |
| Eval report | Score historical tasks against built-in cases (weekly report / research / organize) |
| Crash recovery | After restart, resume or re-run `queued` / `running` tasks from checkpoints |
| API auth | Business APIs require a JWT (`Authorization: Bearer` or `?token=`); optional static `auth_token` coexists with JWT |

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
# 1. Start Postgres only (host port 14432). Full stack: see Docker Compose below
docker compose up -d postgres

# 2. Configure
cp config.example.yaml config.yaml
cp .env.example .env
# Edit .env; set at least EINO_LLM_API_KEY / EINO_LLM_MODEL
# Change EINO_JWT_SECRET in production

# 3. Start API (default :8180, see .env)
go run ./cmd/server

# 4. Start frontend (Vite :5273, proxies /api → :8180)
cd web && npm install && npm run dev
```

Open http://localhost:5273 and register / sign in first.

Sample task: upload `testdata/sample/sales.csv`, set the goal to “Generate this week’s sales report from input/sales.csv”, and select skill `weekly-report`.

### CLI

Business APIs need a user JWT. Register or log in, then set `EINO_API_TOKEN`:

```bash
# Register (or use /auth/login)
curl -s -X POST http://127.0.0.1:8180/api/v1/auth/register \
  -H 'Content-Type: application/json' \
  -d '{"username":"demo","display_name":"Demo User","password":"secret1"}'
export EINO_API_TOKEN=<your-jwt>

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

Also available on the Eval page or `GET /api/v1/eval` (auth required).

### Production build

After building the frontend, Gin serves `web/dist`:

```bash
cd web && npm run build
go run ./cmd/server
```

### Docker Compose

One command starts Postgres, the API, and the frontend (Nginx serves the UI and proxies `/api`, including SSE, to the API). LLM keys and the JWT secret come from the repo `.env`. Compose overrides the in-container database DSN and workspace path, so you do not need to change the local DSN in `.env`.

```bash
cp .env.example .env
# Set at least EINO_LLM_API_KEY / EINO_LLM_MODEL; change EINO_JWT_SECRET in production
docker compose up -d --build
```

- Console: http://localhost:8080 (`EINO_WEB_PORT` changes the host port)
- API: http://localhost:8180 (`EINO_API_PORT` changes the host port; point the CLI here)
- Task sandboxes and logs live in volumes `workspace` and `logs`. `skills/` is mounted read-only; restart `api` after editing skills

For local development, `docker compose up -d postgres` starts only the database.

## Web console

| Page | Path | Purpose |
|---|---|---|
| Login / Register | `/login` · `/register` | Account auth; unauthenticated users are redirected here |
| Tasks | `/` | Create/list, event stream, confirm, retry, artifact preview/download |
| Projects | `/projects` | Project management and shared file uploads |
| Schedules | `/schedules` | Create/toggle/run schedules immediately |
| Connectors | `/connectors` | Webhook / local-directory delivery |
| Audit | `/audit` | Audit log list |
| Eval | `/eval` | Case completion rates and details |

## HTTP API (`/api/v1`)

Public (no token): `GET /health`, `POST /auth/register`, `POST /auth/login`, `POST /hooks/echo`. All other endpoints require a JWT (or optional static `auth_token`). Create/list and similar business APIs need a logged-in user; data is isolated by `user_id`.

| Method | Path | Description |
|---|---|---|
| GET | `/health` | Health check |
| POST | `/auth/register` | Register `{"username","display_name","password"}`, returns JWT |
| POST | `/auth/login` | Log in and receive a JWT |
| GET | `/auth/me` | Current user |
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
| POST | `/hooks/echo` | Webhook debug echo |
| GET | `/audit` | Audit logs |
| GET | `/eval` | Eval report |

Register/login body: `{"username":"...","password":"..."}` (username 3–64, letters/digits/`_`/`-` only; password 6–128). Success response includes `token`, `expires_at`, and `user`.

Task status: `queued` → `running` → (optional `waiting_confirm`) → `succeeded` / `failed` / `cancelled`.

## Configuration

Copy `config.example.yaml` / `.env.example`. Common environment variables:

| Variable | Meaning |
|---|---|
| `EINO_CONFIG` | Config file path, default `./config.yaml` |
| `EINO_DATABASE_DSN` | Postgres DSN |
| `EINO_SERVER_ADDR` | HTTP listen address, e.g. `:8180` |
| `EINO_SERVER_MODE` | Gin mode, e.g. `debug` / `release` |
| `EINO_LOG_LEVEL` | Log level: `debug` / `info` / `warn` / `error` |
| `EINO_LOG_FORMAT` | Log format: `console` / `json` (json by default in release) |
| `EINO_LOG_FILE` | If set, write daily files `<name>-YYYY-MM-DD.log` |
| `EINO_LOG_KEEP_DAYS` | Days to keep log files, default `14`; `0` keeps all |
| `EINO_JWT_SECRET` | JWT signing secret (change in production) |
| `EINO_JWT_EXPIRE` | Token lifetime, default `168h` |
| `EINO_AUTH_TOKEN` | Optional static Bearer alongside JWT |
| `EINO_API_TOKEN` | Bearer used by the CLI (JWT from login) |
| `EINO_API_BASE` | CLI API base URL |
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
Dockerfile          # API image
docker-compose.yml  # Postgres + API + frontend
cmd/server          # HTTP server entry
cmd/cli             # Task CLI
cmd/eval            # Eval CLI
internal/api        # Gin routes, JWT auth, per-user scoping
internal/harness    # Runtime, checkpoint recovery, event bus
internal/agent      # DeepAgent factory, tools, skills
internal/confirm    # Risk classification and confirm gate
internal/scheduler  # Scheduled runs
internal/connector  # Webhook / local-dir delivery
internal/eval       # Case scoring
internal/store      # GORM models and persistence (incl. users)
internal/workspace  # Sandbox and project shared dirs
skills/             # Built-in SKILL.md files
web/                # Vue console (Dockerfile / nginx.conf)
```
