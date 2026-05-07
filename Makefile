.PHONY: build run test lint vet tools generate migrate

TOOLBIN := $(shell go env GOPATH)/bin

build:
	go build -o bin/board ./cmd/board

run: build
	./bin/board

test:
	go test ./...

vet:
	go vet ./...

lint:
	$(TOOLBIN)/golangci-lint run

tools:
	GOBIN=$(TOOLBIN) go install github.com/pressly/goose/v3/cmd/goose@latest
	GOBIN=$(TOOLBIN) go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest
	GOBIN=$(TOOLBIN) go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

generate:
	$(TOOLBIN)/sqlc generate

migrate:
	$(TOOLBIN)/goose -dir internal/db/schema sqlite3 ./yaama.db up
