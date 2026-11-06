# {{.ProjectName}}

Generated with [Go-DDD-Scaffold](https://github.com/Allan-Nava/Go-DDD-Scaffold).

Module path: `{{.ProjectPath}}` — check it matches your remote before the first push
(`scaffold init --module <path>` sets it explicitly).

## Layout

```
├── cmd/
│   ├── main.go              # HTTP entrypoint: config, middlewares, graceful shutdown
│   ├── modules.go           # composition root: mounts every module under /api
│   └── module_<name>.go     # one per bounded context: store → service → handler
├── internal/
│   ├── <name>/              # one bounded context per folder (catalog is the example)
│   │   ├── domain/          # aggregate, value objects, errors, repository port — stdlib only
│   │   ├── application/     # use cases: load → ask the aggregate → save
│   │   ├── infrastructure/  # GORM store, in-memory store, row ⇄ aggregate mapping
│   │   └── transport/http/  # HTTP ⇄ use cases, domain errors ⇄ status codes
│   └── archtest/            # fails go test when domain/application import what they must not
├── database/db.go           # GORM connection pool
├── env/env.go               # configuration from the environment
├── env/.env.local           # local values, git-ignored (never commit credentials)
├── config/config.yml        # non-secret tunables
├── Dockerfile               # multi-stage build, alpine runtime
├── docker-compose.yml       # service + MySQL
├── .gitignore               # bin/, env/.env.*
└── Makefile
```

## Where the rules go

Dependencies point inwards: `transport → application → domain ← infrastructure`.

- A **business rule** ("an archived item cannot be renamed") goes in the aggregate, in
  `domain/`. Its fields are unexported, so nothing can change state without passing the rule.
- A **use case** in `application/` loads the aggregate through the repository port, calls one of
  its methods, saves it. No `if` on business state there.
- **Adapters** (`infrastructure/`, `transport/`) translate and hold no rule.
- `internal/archtest` enforces it: a `domain` package that imports GORM, Fiber or another layer
  fails `go test`.

Guide with the rich vs anemic trade-off:
[the generated DDD layout](https://github.com/Allan-Nava/Go-DDD-Scaffold/blob/main/docs/ddd-guide.md).

## Add a bounded context

```sh
scaffold add module billing --entity Invoice   # internal/billing + cmd/module_billing.go
go test ./...
```

No existing file is edited: the new `cmd/module_billing.go` registers itself. To remove a
module, delete its `internal/<name>/` folder and its `cmd/module_<name>.go`.

## Run

```sh
make tidy   # resolve the dependencies, writes go.sum
make run    # APP_ENV=local, listens on :8080 — no database needed
make up     # service + MySQL through docker compose
```

Health check: `curl localhost:8080/health`. The example module:

```sh
curl -X POST localhost:8080/api/catalog -H 'Content-Type: application/json' -d '{"name":"Laptop"}'
curl localhost:8080/api/catalog
```

| Route | Use case | Errors |
|---|---|---|
| `POST /api/<module>` `{"name"}` | create | 422 invalid name |
| `GET /api/<module>` | list | |
| `GET /api/<module>/:id` | get | 404 |
| `PATCH /api/<module>/:id` `{"name"}` | rename | 404, 409 archived, 422 |
| `POST /api/<module>/:id/archive` | archive | 404, 409 already archived |

`make run` needs no MySQL: `DB_HOST` is empty in `env/.env.local`, and the service then
starts without a database — each module then keeps its data in memory (lost on restart). Set
`DB_HOST=localhost` there to use the MySQL started by `make up`: each module creates its table
(`<module>_<entity>`) on startup.

## Configuration

Everything comes from the environment; `env/.env.<APP_ENV>` is loaded when present
(`APP_ENV` defaults to `local`). Variables already set in the environment win.

| Variable | Default | |
|---|---|---|
| `APP_ENV` | `local` | picks `env/.env.<APP_ENV>` |
| `LOG_LEVEL` | `info` | `debug`, `info`, `warn`, `error`; an invalid value stops the service |
| `HTTP_ADDR` | `0.0.0.0:8080` | listen address |
| `CORS_ALLOWED_ORIGINS` | empty | comma-separated browser origins; empty = no cross-origin access |
| `DB_HOST` | empty | empty = start without a database |
| `DB_PORT`, `DB_NAME`, `DB_USERNAME`, `DB_PASSWORD` | `3306`, — | MySQL connection |
| `DB_IDLE_CONN`, `DB_MAX_CONN` | `10`, `100` | pool sizes |
