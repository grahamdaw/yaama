.PHONY: build run test lint vet tools generate migrate release-check

TOOLBIN := $(shell go env GOPATH)/bin
GOLANGCI_LINT_CACHE ?= $(CURDIR)/.golangci-cache

build:
	go build -o bin/yaama ./cmd/yaama

run: build
	./bin/yaama

test:
	go test ./...

vet:
	go vet ./...

lint:
	mkdir -p "$(GOLANGCI_LINT_CACHE)"
	GOLANGCI_LINT_CACHE="$(GOLANGCI_LINT_CACHE)" $(TOOLBIN)/golangci-lint run

tools:
	GOBIN=$(TOOLBIN) go install github.com/pressly/goose/v3/cmd/goose@latest
	GOBIN=$(TOOLBIN) go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest
	GOBIN=$(TOOLBIN) go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

generate:
	$(TOOLBIN)/sqlc generate

migrate:
	$(TOOLBIN)/goose -dir internal/db/schema sqlite3 ./yaama.db up

release-check:
	rm -rf bin/release-check && mkdir -p bin/release-check
	GOOS=darwin GOARCH=arm64 go build -o bin/release-check/yaama-darwin-arm64 ./cmd/yaama
	GOOS=linux GOARCH=amd64 go build -o bin/release-check/yaama-linux-amd64 ./cmd/yaama
