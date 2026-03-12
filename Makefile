.PHONY: build run test lint migrate-up migrate-down docker-up docker-down clean admin-build admin-dev storefront-build storefront-dev mcp-store-build mcp-admin-build mcp-store-run mcp-admin-run install

BINARY=bin/stoa
VERSION?=dev

build: admin-build storefront-build
	@mkdir -p bin
	go build -ldflags="-s -w -X main.version=$(VERSION)" -o $(BINARY) ./cmd/stoa

run: build
	./$(BINARY) serve

test:
	go test ./internal/... -v

test-race:
	go test ./internal/... -race

test-coverage:
	go test ./internal/... -coverprofile=coverage.out
	go tool cover -html=coverage.out -o coverage.html

lint:
	golangci-lint run
	go vet ./...

migrate-up:
	go run ./cmd/stoa migrate up

migrate-down:
	go run ./cmd/stoa migrate down

docker-up:
	docker compose up -d

docker-down:
	docker compose down

docker-build:
	docker compose build

INSTALL_DIR?=$(HOME)/.local/bin

install: build
	@mkdir -p $(INSTALL_DIR)
	install -m 755 $(BINARY) $(INSTALL_DIR)/stoa
	@echo "Installed to $(INSTALL_DIR)/stoa — ensure $(INSTALL_DIR) is in your PATH"

clean:
	rm -rf bin/ coverage.out coverage.html

mcp-store-build:
	@mkdir -p bin
	go build -ldflags="-s -w" -o bin/stoa-store-mcp ./cmd/stoa-store-mcp

mcp-admin-build:
	@mkdir -p bin
	go build -ldflags="-s -w" -o bin/stoa-admin-mcp ./cmd/stoa-admin-mcp

mcp-store-run: mcp-store-build
	./bin/stoa-store-mcp

mcp-admin-run: mcp-admin-build
	./bin/stoa-admin-mcp

admin-build:
	cd admin && npm install && npm run build

admin-dev:
	cd admin && npm install && npm run dev

storefront-build:
	cd storefront && npm install && npm run build

storefront-dev:
	cd storefront && npm install && npm run dev

admin-create:
	go run ./cmd/stoa admin create --email $(EMAIL) --password $(PASSWORD)

seed:
	go run ./cmd/stoa seed --demo
