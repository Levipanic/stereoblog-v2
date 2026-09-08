.PHONY: dev dev-backend dev-frontend test-backend check-frontend build build-backend build-frontend

API_PORT ?= 8080

dev:
	$(MAKE) -j2 dev-backend dev-frontend

dev-backend:
	cd backend && PORT=$(API_PORT) go run ./cmd/server

dev-frontend:
	npm --prefix frontend run dev

test-backend:
	cd backend && go test ./...

check-frontend:
	npm --prefix frontend run typecheck

build: build-backend build-frontend

build-backend:
	cd backend && go build -o bin/server ./cmd/server

build-frontend:
	npm --prefix frontend run build
