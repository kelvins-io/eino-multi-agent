# Docker Compose helpers for eino-multi-agent
# Services: postgres | api | web

COMPOSE ?= docker compose
COMPOSE_FILE ?= docker-compose.yml
COMPOSE_CMD = $(COMPOSE) -f $(COMPOSE_FILE)

.DEFAULT_GOAL := help

.PHONY: help up up-db down restart build rebuild pull ps logs logs-api logs-web logs-db \
	shell-api shell-db clean

help: ## Show available targets
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-14s\033[0m %s\n", $$1, $$2}'

up: ## Start full stack (postgres + api + web), build if needed
	$(COMPOSE_CMD) up -d --build

up-db: ## Start Postgres only (host port 14432, for local API/web)
	$(COMPOSE_CMD) up -d postgres

down: ## Stop and remove containers
	$(COMPOSE_CMD) down

restart: ## Restart all services
	$(COMPOSE_CMD) restart

build: ## Build api and web images
	$(COMPOSE_CMD) build

rebuild: ## Rebuild images without cache, then start
	$(COMPOSE_CMD) build --no-cache
	$(COMPOSE_CMD) up -d

pull: ## Pull base images (postgres, etc.)
	$(COMPOSE_CMD) pull

ps: ## Show service status
	$(COMPOSE_CMD) ps

logs: ## Follow logs for all services
	$(COMPOSE_CMD) logs -f --tail=200

logs-api: ## Follow API logs
	$(COMPOSE_CMD) logs -f --tail=200 api

logs-web: ## Follow web logs
	$(COMPOSE_CMD) logs -f --tail=200 web

logs-db: ## Follow Postgres logs
	$(COMPOSE_CMD) logs -f --tail=200 postgres

shell-api: ## Shell into API container
	$(COMPOSE_CMD) exec api sh

shell-db: ## psql into Postgres
	$(COMPOSE_CMD) exec postgres psql -U eino -d eino_work

clean: ## Stop stack and remove named volumes (pgdata/workspace/logs)
	$(COMPOSE_CMD) down -v --remove-orphans
