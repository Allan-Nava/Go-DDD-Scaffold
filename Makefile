VERSION ?= dev

# `./...` and `gofmt .` descend into extensions/vscode/node_modules, which after an
# `npm install` can contain third-party Go code (flatted ships a package).
# So this narrows to the tracked .go files and the packages outside extensions/.
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

# Same gate as CI, which pins v2.12.2 in go.yml: install the same version,
# otherwise the two gates can diverge.
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
# docs/backlog.md is the single source: milestones are the titles the items
# declare, not a fixed list, and the sync creates them on GitHub when missing.

.PHONY: backlog-lint
backlog-lint:
	python3 docs/scripts/backlog-lint.py

.PHONY: roadmap
roadmap:
	python3 docs/scripts/generate-roadmap.py

# CI gate: the committed generated page must match the backlog.
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
