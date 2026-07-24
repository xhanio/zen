# zen — developer convenience targets.
#
# Quick start (after `make deps` once):
#
#   make dev         # renders config + runs backend + Vite in one shot
#
# Or run them in separate terminals:
#
#   make backend     # terminal 1
#   make frontend    # terminal 2
#   make mcp         # optional terminal 3 (only when testing MCP tools)
#
# Override the gopro env if you ever need to:  GOPRO_ENV=prod make backend

GOPRO_ENV   ?= local

BACKEND_BIN := bin/zen-backend
MCP_BIN     := bin/zen-mcp
BACKEND_CFG := dist/$(GOPRO_ENV)/config/zen-backend/config.yaml
MCP_CFG     := dist/$(GOPRO_ENV)/config/zen-mcp/config.yaml

.DEFAULT_GOAL := help

.PHONY: help
help:
	@echo "zen developer targets (GOPRO_ENV=$(GOPRO_ENV))"
	@echo ""
	@echo "  make dev            config + backend + Vite frontend in parallel (Ctrl-C stops both)"
	@echo "  make backend        run zen-backend in the foreground"
	@echo "  make mcp            run zen-mcp in the foreground"
	@echo "  make frontend       run Vite dev server in the foreground"
	@echo ""
	@echo "  make config         gopro generate config -e $(GOPRO_ENV)"
	@echo "  make build          gopro build binary -e $(GOPRO_ENV)"
	@echo "  make deps           npm install in frontend/"
	@echo ""
	@echo "  make test           backend (go test) + frontend (vitest)"
	@echo "  make test-backend   go test ./..."
	@echo "  make test-frontend  cd frontend && npm test"
	@echo "  make e2e            Playwright (needs backend + Vite running)"
	@echo ""
	@echo "  make reset-db       delete /tmp/zen-local.db so migrations re-run"
	@echo "  make clean          wipe bin/, dist/, frontend/dist + reset-db"

.PHONY: config
config:
	gopro generate config -e $(GOPRO_ENV)

.PHONY: build
build:
	gopro build binary -e $(GOPRO_ENV)

.PHONY: backend
backend: config build
	./$(BACKEND_BIN) daemon -c $(BACKEND_CFG)

.PHONY: mcp
mcp: config build
	./$(MCP_BIN) daemon -c $(MCP_CFG)

.PHONY: frontend
frontend:
	cd frontend && npm run dev

# `make dev` runs backend + Vite together. The trap sends SIGTERM to the whole
# process group on Ctrl-C / exit so both children die together — no orphan
# backend left holding port 8080. `kill 0` also signals this shell, so the
# handler must disarm itself first: otherwise the TERM trap re-enters and
# recurses until the stack blows (dash segfaults).
.PHONY: dev
dev: config build
	@echo "==> starting zen-backend + Vite (Ctrl-C stops both)"
	@trap 'trap "" TERM; trap - EXIT INT; kill 0; exit 0' EXIT INT TERM; \
	  ( ./$(BACKEND_BIN) daemon -c $(BACKEND_CFG) ) & \
	  ( cd frontend && npm run dev ) & \
	  wait

.PHONY: deps
deps:
	cd frontend && npm install

.PHONY: test test-backend test-frontend
test: test-backend test-frontend

test-backend:
	go test ./...

test-frontend:
	cd frontend && npm test

.PHONY: e2e
e2e:
	cd frontend && npm run e2e

.PHONY: reset-db
reset-db:
	rm -f /tmp/zen-local.db /tmp/zen-local.db-journal /tmp/zen-local.db-wal /tmp/zen-local.db-shm

.PHONY: clean
clean: reset-db
	rm -rf bin dist frontend/dist
