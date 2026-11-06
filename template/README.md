# {{.ProjectName}}

Generated with [Go-DDD-Scaffold](https://github.com/Allan-Nava/Go-DDD-Scaffold).

Module path: `{{.ProjectPath}}` — check it matches your remote before the first push
(`scaffold init --module <path>` sets it explicitly).

## Layout

```
├── cmd/main.go          # HTTP entrypoint: config, middlewares, graceful shutdown
├── database/db.go       # GORM connection pool
├── env/env.go           # configuration from the environment
├── env/.env.local       # local values, git-ignored (never commit credentials)
├── config/config.yml    # non-secret tunables
├── Dockerfile           # multi-stage build, alpine runtime
├── docker-compose.yml   # service + MySQL
├── .gitignore           # bin/, env/.env.*
└── Makefile
```

## Run

```sh
make tidy   # resolve the dependencies, writes go.sum
make run    # APP_ENV=local, listens on :8080 — no database needed
make up     # service + MySQL through docker compose
```

Health check: `curl localhost:8080/health`.

`make run` needs no MySQL: `DB_HOST` is empty in `env/.env.local`, and the service then
starts without a database. Set `DB_HOST=localhost` there to use the MySQL started by `make up`.

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
