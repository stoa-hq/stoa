.PHONY: build run test lint migrate-up migrate-down docker-up docker-down clean admin-build admin-dev storefront-build storefront-dev

BINARY=stoa
VERSION?=dev

build: admin-build storefront-build
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

clean:
	rm -f $(BINARY) coverage.out coverage.html

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
