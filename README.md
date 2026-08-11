# Go Domain Driven Design Scaffold

[![Go Report Card](https://goreportcard.com/badge/github.com/Allan-Nava/Go-DDD-Scaffold)](https://goreportcard.com/report/github.com/Allan-Nava/Go-DDD-Scaffold)
[![GoDoc](https://godoc.org/github.com/Allan-Nava/Go-DDD-Scaffold?status.svg)](https://godoc.org/github.com/Allan-Nava/Go-DDD-Scaffold)
[![Deploy Jekyll with GitHub Pages dependencies preinstalled](https://github.com/Allan-Nava/Go-DDD-Scaffold/actions/workflows/jekyll-gh-pages.yml/badge.svg?branch=main)](https://github.com/Allan-Nava/Go-DDD-Scaffold/actions/workflows/jekyll-gh-pages.yml)
[![Go](https://github.com/Allan-Nava/Go-DDD-Scaffold/actions/workflows/go.yml/badge.svg?branch=main)](https://github.com/Allan-Nava/Go-DDD-Scaffold/actions/workflows/go.yml)


Generate a Domain Driven Design project layout for Go.

The templates are embedded in the binary, so `scaffold` runs anywhere — no
`$GOPATH`, no repository checkout beside it.

The generated layout:

```
├── Dockerfile           # multi-stage build, alpine runtime
├── docker-compose.yml   # service + MySQL
├── Makefile
├── README.md
├── .dockerignore
├── cmd
│   ├── main.go          # HTTP entrypoint: config, middlewares, graceful shutdown
│   └── main_test.go     # smoke test of /health through app.Test (no port needed)
├── config
│   └── config.yml       # non-secret tunables
├── database
│   └── db.go            # GORM connection pool
└── env
    ├── env.go           # configuration from the environment
    ├── env_test.go      # pins the envDefault tags
    └── .env.local       # local values
```

Stack: [Fiber](https://github.com/gofiber/fiber) · [GORM](https://gorm.io) ·
[zap](https://github.com/uber-go/zap) · [caarlos0/env](https://github.com/caarlos0/env) ·
[godotenv](https://github.com/joho/godotenv).

## Installation

```sh
go install github.com/Allan-Nava/Go-DDD-Scaffold@latest
```

The binary is named after the module; rename it to `scaffold` if you prefer, or
build it directly:

```sh
git clone https://github.com/Allan-Nava/Go-DDD-Scaffold
cd Go-DDD-Scaffold
make build      # -> bin/scaffold
```

Or run it from the container image, generating into the current directory:

```sh
docker run --rm -v "$PWD:/work" ghcr.io/allan-nava/go-ddd-scaffold init
```

## Create a new project

```sh
mkdir -p ~/code/github.com/myorg/myservice
cd ~/code/github.com/myorg/myservice
scaffold init
```

Files are written to the **current directory**. The module path in `go.mod` is
derived from the directory: a path under `github.com/myorg/myservice` yields
`module github.com/myorg/myservice`.

Then:

```sh
go mod tidy
make run                        # listens on :8080
curl localhost:8080/health      # 200
```

The service starts without a database when `DB_HOST` is empty.

### Flags

| Flag | Effect |
| --- | --- |
| `scaffold init [dir]` | generate into `dir` instead of the current directory |
| `--force`, `-f` | overwrite files that already exist |
| `--debug` | trace what the generator is doing |

Without `--force`, existing files are left untouched and reported as skipped, so
re-running `scaffold init` in a live project is safe.

## Development

```sh
make check   # gofmt + go vet
make lint    # golangci-lint (errcheck & co., see .golangci.yml)
make test    # unit tests
make e2e     # generates a project in a temp dir and compiles it for real
make docker  # build the container image
```

`make e2e` is the gate that matters: it proves the templates still emit
buildable Go. See [CLAUDE.md](CLAUDE.md) for the working rules and
[docs/audit-2026-08-11.md](docs/audit-2026-08-11.md) for the technical audit.

### Backlog and milestones

[docs/backlog.md](docs/backlog.md) is the single source of truth for the open work.
An idempotent script opens, updates and closes one GitHub issue per item, creating any
milestone that does not exist yet — milestones are whatever the items declare, there is
no fixed list. [docs/roadmap.md](docs/roadmap.md) is generated from it.

```sh
make backlog-lint          # validate the backlog, no GitHub access
make roadmap               # regenerate docs/roadmap.md
make backlog-sync          # dry-run of the issue sync
make backlog-sync-apply    # apply it
```

The sync only ever touches issues carrying the `backlog-sync` label, so Renovate,
Dependabot and third-party reports are never read or closed.

## VSCode extension

`extensions/vscode/` holds a companion extension. Its generation command is
currently a stub — see the audit for details.
