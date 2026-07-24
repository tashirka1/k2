include .env

.PHONY: up
up:
	@docker compose -p k2 up -d --remove-orphans --force-recreate

.PHONY: down
down:
	@docker compose -p k2 down

.PHONY: build
build:
	@docker compose -p k2 build

.PHONY: build-bin
build-bin:
	@go tool templ generate && go build -tags fts5 -ldflags="-s -w" -o bin/k2 cmd/k2/main.go

.PHONY: lint
lint:
	@go tool templ generate && go fmt ./... && golangci-lint run ./...

.PHONY: check
check:
	@go tool templ generate && go fmt ./... && golangci-lint run ./... && go test -tags fts5 ./...

.PHONY: dev
dev:
	@go tool air
