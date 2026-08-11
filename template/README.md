# {{.ProjectName}}

Generated with [Go-DDD-Scaffold](https://github.com/Allan-Nava/Go-DDD-Scaffold).

Module path: `{{.ProjectPath}}` — check it matches your remote before the first push.

## Layout

```
├── cmd/main.go          # HTTP entrypoint: config, middlewares, graceful shutdown
├── database/db.go       # GORM connection pool
├── env/env.go           # configuration from the environment
├── env/.env.local       # local values (never commit real credentials)
├── config/config.yml    # non-secret tunables
├── Dockerfile           # multi-stage build, alpine runtime
├── docker-compose.yml   # service + MySQL
└── Makefile
```

## Run

```sh
make tidy   # resolve the dependencies, writes go.sum
make run    # APP_ENV=local, listens on :8080
make up     # service + MySQL through docker compose
```

Health check: `curl localhost:8080/health`.

The service starts without a database when `DB_HOST` is empty.
