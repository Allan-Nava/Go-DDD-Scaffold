VERSION ?= dev
LDFLAGS := -s -w -X main.version=$(VERSION)

.PHONY: build
build:
	CGO_ENABLED=0 go build -trimpath -ldflags="$(LDFLAGS)" -o bin/scaffold .

.PHONY: install
install:
	go install -trimpath -ldflags="$(LDFLAGS)" .

.PHONY: test
test:
	go test -race ./...

.PHONY: check
check:
	test -z "$$(gofmt -l .)" || (gofmt -l . && exit 1)
	go vet ./...

# Generates a project in a throwaway directory and compiles it for real: the
# only gate that proves the templates still emit buildable Go.
.PHONY: e2e
e2e: build
	@set -eu; \
	dir=$$(mktemp -d)/github.com/acme/myservice; \
	mkdir -p $$dir; \
	cd $$dir && $(CURDIR)/bin/scaffold init && \
	go mod tidy && go build ./... && go vet ./... && test -z "$$(gofmt -l .)"; \
	echo "e2e OK: $$dir"

.PHONY: docker
docker:
	docker build --build-arg VERSION=$(VERSION) -t go-ddd-scaffold:$(VERSION) .

.PHONY: clean
clean:
	rm -rf bin
