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
	@go tool templ generate && go build -ldflags="-s -w" -o bin/k2 cmd/k2/main.go

.PHONY: build-linux-amd64
build-linux-amd64:
	@go tool templ generate && CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o bin/k2-linux-amd64 cmd/k2/main.go

.PHONY: lint
lint:
	@go tool templ generate && go fmt ./... && golangci-lint run ./...

.PHONY: check
check:
	@go tool templ generate && go fmt ./... && golangci-lint run ./... && go test ./...

.PHONY: dev
dev:
	@go tool air
