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

# Generates a project in a throwaway directory, adds a second bounded context
# with `add module` (a multi-word entity, to catch gofmt drift), compiles it for
# real, then starts
# it the way `make run` does (APP_ENV=local, so env/.env.local is loaded) and
# checks /health and a module route: the only gate that proves the templates emit a service that
# builds *and* starts on a machine with no MySQL.
.PHONY: e2e
e2e: build
	@set -eu; \
	dir=$$(mktemp -d)/github.com/acme/myservice; \
	mkdir -p $$dir; \
	cd $$dir && $(CURDIR)/bin/scaffold init && \
	$(CURDIR)/bin/scaffold add module --entity InvoiceLine billing && \
	go mod tidy && go build ./... && go vet ./... && test -z "$$(gofmt -l .)" && \
	go test ./... && \
	go build -o $$dir/bin/svc ./cmd && \
	{ APP_ENV=local HTTP_ADDR=127.0.0.1:18080 $$dir/bin/svc >$$dir/svc.log 2>&1 & pid=$$!; \
	  ok=; for _ in 1 2 3 4 5 6 7 8 9 10; do \
	    curl -fsS --max-time 2 -o /dev/null http://127.0.0.1:18080/health && { ok=1; break; }; sleep 1; \
	  done; \
	  test -z "$$ok" || curl -fsS --max-time 2 -o /dev/null -H 'Content-Type: application/json' \
	    -d '{"name":"e2e"}' http://127.0.0.1:18080/api/billing || ok=; \
	  kill -TERM $$pid; wait $$pid || true; \
	  test -n "$$ok" || { echo "health check failed:"; cat $$dir/svc.log; exit 1; }; }; \
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
