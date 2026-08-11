---
layout: default
title: Home
nav_order: 1
description: "Go DDD Scaffold docs"
permalink: /
last_modified_date: 2026-08-11T00:00:00+0000
---

# Go Domain Driven Design Scaffold official documentation

Generate a Domain Driven Design project layout for Go.

The templates are embedded in the binary: `scaffold` runs anywhere, with no
`$GOPATH` and no repository checkout beside it.

The generated layout:

```
├── Dockerfile           # multi-stage build, alpine runtime
├── docker-compose.yml   # service + MySQL
├── Makefile
├── README.md
├── .dockerignore
├── cmd
│   └── main.go          # HTTP entrypoint: config, middlewares, graceful shutdown
├── config
│   └── config.yml       # non-secret tunables
├── database
│   └── db.go            # GORM connection pool
└── env
    ├── env.go           # configuration from the environment
    └── .env.local       # local values
```

Stack: Fiber · GORM (MySQL) · zap · caarlos0/env · godotenv.

## Install

```sh
go install github.com/Allan-Nava/Go-DDD-Scaffold@latest
```

## Create a new project

```sh
mkdir -p ~/code/github.com/myorg/myservice
cd ~/code/github.com/myorg/myservice
scaffold init

go mod tidy
make run                        # listens on :8080
curl localhost:8080/health      # 200
```

Files are written to the **current directory**, and the module path in `go.mod`
is derived from it: a project under `github.com/myorg/myservice` yields
`module github.com/myorg/myservice`.

The service starts without a database when `DB_HOST` is empty.

### Flags

| Flag | Effect |
| --- | --- |
| `scaffold init [dir]` | generate into `dir` instead of the current directory |
| `--force`, `-f` | overwrite files that already exist |
| `--debug` | trace what the generator is doing |

Without `--force` existing files are left untouched and reported as skipped, so
re-running `scaffold init` in a live project is safe.

## Audit

The technical audit of the generator, the applied fixes and the remaining debt
are in [audit-2026-08-11](audit-2026-08-11.md).
