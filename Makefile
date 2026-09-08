.PHONY: dev dev-backend dev-frontend test-backend check-frontend build build-backend build-frontend

ENV_FILE ?= .env
# ponytail: local env files use POSIX shell syntax; add dotenv tooling only if richer syntax becomes necessary.
LOAD_ENV = set -ae; if [ -f "$(ENV_FILE)" ]; then case "$(ENV_FILE)" in /*) . "$(ENV_FILE)" ;; *) . "./$(ENV_FILE)" ;; esac; fi; set +a;

dev:
	$(MAKE) -j2 dev-backend dev-frontend

dev-backend:
	$(LOAD_ENV) cd backend && go run ./cmd/server

dev-frontend:
	$(LOAD_ENV) npm --prefix frontend run dev

test-backend:
	cd backend && go test ./...

check-frontend:
	$(LOAD_ENV) npm --prefix frontend run check

build: build-backend build-frontend

build-backend:
	cd backend && go build -o bin/server ./cmd/server

build-frontend:
	$(LOAD_ENV) npm --prefix frontend run build
