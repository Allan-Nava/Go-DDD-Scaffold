VERSION ?= dev

# `./...` e `gofmt .` scendono in extensions/vscode/node_modules, che dopo un
# `npm install` puo contenere codice Go di terze parti (flatted ne ha un
# package). Si restringe ai .go tracciati e ai package fuori da extensions/.
GO_FILES := $(shell git ls-files '*.go')
GO_PKGS  := $(shell go list ./... | grep -v '/extensions/')

LDFLAGS := -s -w -X main.version=$(VERSION)

.PHONY: build
build:
	CGO_ENABLED=0 go build -trimpath -ldflags="$(LDFLAGS)" -o bin/scaffold .

.PHONY: install
install:
	go install -trimpath -ldflags="$(LDFLAGS)" .

.PHONY: test
test:
	go test -race $(GO_PKGS)

.PHONY: check
check:
	test -z "$$(gofmt -l $(GO_FILES))" || (gofmt -l $(GO_FILES) && exit 1)
	go vet $(GO_PKGS)

# Stesso gate della CI, che pinna v2.12.2 in go.yml: installare la stessa
# versione, altrimenti i due gate possono divergere.
#   go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.12.2
.PHONY: lint
lint:
	golangci-lint run ./...

# Generates a project in a throwaway directory and compiles it for real: the
# only gate that proves the templates still emit buildable Go.
.PHONY: e2e
e2e: build
	@set -eu; \
	dir=$$(mktemp -d)/github.com/acme/myservice; \
	mkdir -p $$dir; \
	cd $$dir && $(CURDIR)/bin/scaffold init && \
	go mod tidy && go build ./... && go vet ./... && test -z "$$(gofmt -l .)" && \
	go test ./...; \
	echo "e2e OK: $$dir"

.PHONY: docker
docker:
	docker build --build-arg VERSION=$(VERSION) -t go-ddd-scaffold:$(VERSION) .

# --- backlog / milestone / issue -------------------------------------------------
# docs/backlog.md è la sorgente unica: le milestone sono i titoli dichiarati dagli
# item, non un elenco fisso, e il sync le crea su GitHub se mancano.

.PHONY: backlog-lint
backlog-lint:
	python3 docs/scripts/backlog-lint.py

.PHONY: roadmap
roadmap:
	python3 docs/scripts/generate-roadmap.py

# Gate CI: la pagina generata committata deve combaciare con il backlog.
.PHONY: generated-pages-check
generated-pages-check:
	python3 docs/scripts/generate-roadmap.py --check

.PHONY: backlog-sync
backlog-sync:
	python3 docs/scripts/sync-backlog-to-issues.py

.PHONY: backlog-sync-apply
backlog-sync-apply:
	python3 docs/scripts/sync-backlog-to-issues.py --apply

.PHONY: clean
clean:
	rm -rf bin
