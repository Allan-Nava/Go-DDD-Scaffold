APP_ENV ?= local

.PHONY: tidy
tidy:
	go mod tidy

.PHONY: run
run:
	APP_ENV=$(APP_ENV) go run ./cmd

.PHONY: build
build:
	CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o bin/app ./cmd

.PHONY: test
test:
	go test ./...

.PHONY: check
check:
	gofmt -l .
	go vet ./...

.PHONY: up
up:
	docker compose up -d --build

.PHONY: down
down:
	docker compose down

.PHONY: logs
logs:
	docker compose logs -f
