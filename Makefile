.DEFAULT_GOAL := test

.PHONY: check-frontend-tools install-frontend dev-backend dev-frontend test test-backend test-frontend check check-frontend build build-backend build-frontend e2e audit

ENV_FILE ?= .env
# ponytail: local env files use POSIX shell syntax; add dotenv tooling only if richer syntax becomes necessary.
LOAD_ENV = set -ae; if [ -f "$(ENV_FILE)" ]; then case "$(ENV_FILE)" in /*) . "$(ENV_FILE)" ;; *) . "./$(ENV_FILE)" ;; esac; fi; set +a;

check-frontend-tools:
	@expected=$$(tr -d '\n' < .nvmrc); actual=$$(node -p 'process.versions.node'); test "$$actual" = "$$expected" || { printf 'Node %s required, found %s\n' "$$expected" "$$actual" >&2; exit 1; }
	@expected=$$(node -p "require('./frontend/package.json').engines.npm"); actual=$$(npm --version); test "$$actual" = "$$expected" || { printf 'npm %s required, found %s\n' "$$expected" "$$actual" >&2; exit 1; }

install-frontend: check-frontend-tools
	npm --prefix frontend ci

dev-backend:
	$(LOAD_ENV) cd backend && go run ./cmd/server

dev-frontend: check-frontend-tools
	$(LOAD_ENV) npm --prefix frontend run dev

test-backend:
	cd backend && go test ./...

test-frontend: check-frontend-tools
	npm --prefix frontend run check

check-frontend: test-frontend

test:
	+$(MAKE) test-backend
	+$(MAKE) test-frontend

check:
	+$(MAKE) test
	cd backend && go vet ./...

build:
	+$(MAKE) build-backend
	+$(MAKE) build-frontend

build-backend:
	mkdir -p backend/bin
	cd backend && go build -o bin/server ./cmd/server

build-frontend: check-frontend-tools
	npm --prefix frontend run build

e2e:
	@printf '%s\n' 'E2E is not implemented yet; no browser test runner has been added.' >&2; exit 2

audit:
	@printf '%s\n' 'Compatibility audit is not implemented yet; do not use a production database.' >&2; exit 2
