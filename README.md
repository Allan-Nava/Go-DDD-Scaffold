# Go Domain Driven Design Scaffold

[![Go Report Card](https://goreportcard.com/badge/github.com/Allan-Nava/Go-DDD-Scaffold)](https://goreportcard.com/report/github.com/Allan-Nava/Go-DDD-Scaffold)
[![GoDoc](https://godoc.org/github.com/Allan-Nava/Go-DDD-Scaffold?status.svg)](https://godoc.org/github.com/Allan-Nava/Go-DDD-Scaffold)
[![Pages](https://github.com/Allan-Nava/Go-DDD-Scaffold/actions/workflows/pages.yml/badge.svg?branch=main)](https://allan-nava.github.io/Go-DDD-Scaffold/)
[![Go](https://github.com/Allan-Nava/Go-DDD-Scaffold/actions/workflows/go.yml/badge.svg?branch=main)](https://github.com/Allan-Nava/Go-DDD-Scaffold/actions/workflows/go.yml)


Generate a Domain Driven Design project layout for Go. Site: <https://allan-nava.github.io/Go-DDD-Scaffold/>

The templates are embedded in the binary, so `scaffold` runs anywhere — no
`$GOPATH`, no repository checkout beside it.

The generated layout:

```
├── Dockerfile           # multi-stage build, alpine runtime
├── docker-compose.yml   # service + MySQL
├── Makefile
├── README.md
├── .dockerignore
├── .gitignore           # bin/ and env/.env.* stay out of the first commit
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
    └── .env.local       # local values, git-ignored; DB_HOST empty = no database
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

Files are written to the **current directory**. The module path in `go.mod` is,
in order:

1. the git `origin` remote, when the directory is inside a clone
   (`git@github.com:myorg/myservice.git` → `module github.com/myorg/myservice`, a
   subfolder of the repository keeps its relative path);
2. a known code host in the directory path (`github.com`, `gitlab.com`,
   `bitbucket.org`, `codeberg.org`, …): `…/github.com/myorg/myservice` yields
   `module github.com/myorg/myservice`;
3. the legacy `…/src/<path>` GOPATH layout;
4. the directory name.

`--module` overrides all of them. The project name (binary, compose service,
README title) is the directory name lowercased, with spaces turned into `-`, or
`--name`. Both are validated before anything is written.

Then:

```sh
go mod tidy
make run                        # listens on :8080, no MySQL needed
curl localhost:8080/health      # 200
```

The generated `env/.env.local` leaves `DB_HOST` empty, so the service starts
without a database; `make up` brings it up together with MySQL. CORS is closed
until you list origins in `CORS_ALLOWED_ORIGINS`.

### Flags

| Flag | Effect |
| --- | --- |
| `scaffold init [dir]` | generate into `dir` instead of the current directory |
| `--module`, `-m` | Go module path for `go.mod` (default: inferred, see above) |
| `--name` | project name: lowercase letters, digits, `-`, `_`, `.` (default: the directory name, normalised) |
| `--force`, `-f` | overwrite files that already exist |
| `--debug` | trace what the generator is doing |

Without `--force`, existing files are left untouched and reported as skipped, so
re-running `scaffold init` in a live project is safe. Every file is written to a
temporary file first and moved into place only when complete: a failure never
truncates or deletes what was there, with or without `--force`.

## Development

```sh
make check   # gofmt + go vet
make lint    # golangci-lint (errcheck & co., see .golangci.yml)
make test    # unit tests
make e2e     # generates a project in a temp dir, compiles it, starts it like `make run`
make docker  # build the container image
```

`make e2e` is the gate that matters: it proves the templates still emit
buildable Go. See [CLAUDE.md](CLAUDE.md) for the working rules and
[docs/audit-2026-08-11.md](docs/audit-2026-08-11.md) for the technical audit and
[docs/audit-2026-11-06.md](docs/audit-2026-11-06.md) for the product audit and its milestones.

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

`extensions/vscode/` holds a companion extension that **drives this CLI** rather
than carrying its own copy of the templates — so the editor and the command line
can never generate different projects.

- Command palette: `Go DDD: New Scaffold Project`
- Explorer: right-click a folder → `New Scaffold Project`

It finds the CLI on the `PATH` and in `$GOBIN` / `$GOPATH/bin` / `~/go/bin`, and
offers to `go install` it when missing. See
[extensions/vscode/README.md](extensions/vscode/README.md).
