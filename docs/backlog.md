---
layout: default
title: Backlog
nav_order: 7
description: "Operational backlog — the source of truth for the GitHub issues"
---

# Operational backlog — the source of truth for the GitHub issues

This file is the repository's **single source of truth** for todos. An idempotent script
([`scripts/sync-backlog-to-issues.py`](scripts/sync-backlog-to-issues.py)) reads it and
**opens/updates/closes** one GitHub issue per item, creating any missing milestone on the fly.

## How it works (flow)

```
  docs/backlog.md  ──parse(id)──▶  sync-backlog-to-issues.py  ──GitHub API──▶  Issues
   ### `id` — Title                   (idempotent)                            (label backlog-sync
   - status / labels / milestone      fingerprint in the body:                  + fingerprint + milestone)
   - priority / ref                   <!-- backlog-id: id | hash: sha -->
                                                │
   open item  + no issue        → CREATE        │  milestone declared by the item
   open item  + open issue      → SKIP          └─ looked up by exact title on the repo
                  hash differs  → UPDATE           └─ if missing → CREATE (dynamic)
   open item  + closed issue    → REOPEN
   done / removed item          → CLOSE (with a comment + an excerpt from the backlog)
          │
          └──▶  generate-roadmap.py  ──▶  docs/roadmap.md  (generated page, one section
                                                            per milestone, open/done counts)
```

Safe re-runs: the match is by stable `id` (fingerprint in the issue body), **not** by text → editing
a title or description creates no duplicates. The script touches **only** issues labelled
`backlog-sync`: those from Renovate/Dependabot or opened by others are never read or closed.

## Dynamic milestones

**There is no list of milestones anywhere.** There are only the titles written on the items:

- adding a milestone = writing `- **milestone**: <title>` on an item. On the first sync the script
  looks it up by exact title on the repo and, if it is missing, **creates it**;
- `docs/roadmap.md` is **generated** by `generate-roadmap.py`: it discovers the milestones from the
  items, sorts them and counts open/done. Do not edit it by hand;
- the title is part of the fingerprint hash → changing or removing it realigns the issue on the next
  sync. A milestone assigned by hand is never removed if the item declares none;
- items of the same milestone → **identical** title, character by character. `backlog-lint` warns
  when it finds case/space variants of the same title.

## How to write an item

- An item starts with `### \`<stable-id>\` — <Title>` (the `id` is in backticks, kebab-case, **never change it**).
- Metadata as `- **key**: value` bullets:
  - **status**: `open` (default) | `done` → `done` closes the issue.
  - **labels**: comma-separated list (the `backlog-sync` label is added automatically; labels
    missing on the repo are created).
  - **priority**: `low` | `medium` | `high` (optional).
  - **milestone**: the milestone title (optional, created when missing — see above).
  - **owner**: GitHub login to assign the issue to (optional).
  - **ref**: link/path to the reference doc (optional).
- Everything else in the block (prose) becomes the issue **body**.

> To "close" a todo: set `status: done` (history) **or** remove the item. Either way the issue is
> closed on the next sync. Prefer `done` for items **already pushed** (the issue closes with the
> trace of why); removing is fine only for items that were never synced.

## Usage

```sh
make backlog-lint             # validates the backlog, does not touch GitHub
make roadmap                  # regenerates docs/roadmap.md
make backlog-sync             # dry-run: prints the plan
make backlog-sync-apply       # applies: creates/updates/closes the issues and the milestones
```

⚠️ If you change this file **regenerate `roadmap.md`** and commit it, otherwise the CI
`generated-pages-check` gate fails.

---

## Active items

### `config-yml-double-source` — `config/config.yml` is not read: two sources of configuration

- **status**: open
- **priority**: low
- **labels**: enhancement, template, documentation
- **milestone**: Generated project
- **ref**: [audit-2026-08-11.md](audit-2026-08-11.md)

The generated project reads its configuration **only** from the environment (`env/env.go`,
`caarlos0/env` + `godotenv`), but it also ships a `config/config.yml` that no line of code opens.
Two sources of configuration, one of them fake: whoever generates the project edits `config.yml` and
cannot understand why nothing changes. The generated `Dockerfile` depends on it
(`COPY --from=builder /src/config`), so it cannot be removed without touching that too.

Options, to be decided:

1. **remove it** and drop the `COPY` from the Dockerfile — the environment stays the only source (more
   consistent with twelve-factor and with what the code already does);
2. **actually use it** for the non-secret parameters (timeouts, `logMode`), reading it in `env.go` as a
   default layer under the environment variables.

Today the file exists with comments stating that the real values live in the environment: it is a
stopgap, not a decision.

### `release-drop-thirdparty-action` — release.yml: consider dropping the third-party action entirely

- **status**: open
- **priority**: low
- **labels**: github_actions, enhancement
- **milestone**: Hardening CI/CD
- **ref**: [audit-2026-08-11.md](audit-2026-08-11.md)

`release-yml-pin-and-labels` pinned `Allan-Nava/go-release.action` to a SHA: the risk is reduced, not
removed. Publishing the binaries still depends entirely on an external action that runs in a
container, receives `GITHUB_TOKEN` with `contents: write` and has non-obvious logic (it uses the
calling repo's `build.sh` and interprets its stdout — see that item's note).

`go-release.yml` already does the same thing natively: `build.sh` + `softprops/action-gh-release`.
So today there are **two** release paths publishing the same artefact in different ways.

To decide:

1. **drop the action**: a native `goos/goarch` matrix (`sh build.sh` + `softprops/action-gh-release`),
   a single release path, zero third-party dependencies in the step that holds the token;
2. **keep it**, and then retire `go-release.yml`, so the two paths stop overlapping.

Either way **one** of the two workflows has to be chosen: today's duplication is the real problem.

### `pages-config-stale` — GitHub Pages: stale config and a build never seen green

- **status**: done
- **priority**: low
- **labels**: documentation, github_actions
- **milestone**: Hardening CI/CD
- **ref**: [audit-2026-08-11.md](audit-2026-08-11.md)

The Pages workflow was broken for a reason unrelated to the content: `upload-pages-artifact@v1`
uses `actions/upload-artifact@v3` internally, which GitHub now **fails by default**. The job died in
*"Set up job"*, before running a single step — for **all** 6 runs visible in the history.
Fixed (`configure-pages@v5`, `upload-pages-artifact@v3`, `deploy-pages@v4`, `checkout@v4`) and
`JEKYLL_GITHUB_TOKEN` added to the build step.

**A consequence to keep in mind**: since the job never got as far as building the site, the Jekyll
build of these docs has **never been seen green**. I could not reproduce it locally: the
`ghcr.io/actions/jekyll-build-pages` container always loads `jekyll-github-metadata`, which queries
`GET /repos/:nwo/pages` and fails fatally without valid credentials; the token available locally
reads that endpoint through `curl` (HTTP 200) but is rejected inside the container, for reasons that
cannot be observed from outside. The first run after the fix is therefore the first real check.

**Closed on 2026-11-06**: Jekyll was removed. The site is now a static page (`docs/index.html`)
published by `pages.yml` with no build, like the other projects; `_config.yml`, `index.md` and
`jekyll-gh-pages.yml` were deleted, so the points below no longer apply.

To fix in `docs/_config.yml`, independently:

- [ ] `remote_theme: pmarsceill/just-the-docs` — the repo was **renamed** to
      `just-the-docs/just-the-docs`; today it works through a redirect, which is not guaranteed
      forever. To change **after** seeing the build green, so two variables are not mixed up;
- [ ] `ga_tracking: UA-2709176-10` — Universal Analytics has been retired since 2023: the property no
      longer collects anything. Remove it or move to GA4 (`G-…`);
- [ ] consider an explicit `repository: Allan-Nava/Go-DDD-Scaffold`, so `jekyll-github-metadata`
      does not have to infer the nwo from the context.

## Milestones from the 2026-11-06 product audit

Full plan and reasoning: [audit-2026-11-06.md](audit-2026-11-06.md).

### `first-run-needs-mysql` — The generated project's first `make run` dies without MySQL

- **status**: open
- **priority**: high
- **labels**: bug, template
- **milestone**: v0.9 — Smooth first run
- **ref**: [audit-2026-11-06.md](audit-2026-11-06.md)

`make run` sets `APP_ENV=local`, which loads `env/.env.local` with `DB_HOST=localhost`: with no MySQL
running the service exits with `fatal` ("connection refused"). The generated README promises the
opposite ("listens on :8080", "starts without a database when `DB_HOST` is empty").

CI does not see it: the smoke test runs with `APP_ENV: runner`, i.e. without the dotenv file.

- [ ] empty `DB_HOST` in `.env.local` (the database comes up with `make up`), or a README that matches
- [ ] the CI smoke test and `make e2e` run with `APP_ENV=local`, like a user

### `module-path-dotted-dirs` — Wrong module path when the path contains a segment with a dot

- **status**: open
- **priority**: high
- **labels**: bug, go
- **milestone**: v0.9 — Smooth first run
- **ref**: [audit-2026-11-06.md](audit-2026-11-06.md)

`ModulePath` takes the first "host-like" segment: `/Users/john.doe/code/svc` → `module john.doe/code/svc`.
It hits every user whose username is `first.last` (common on macOS and Windows).

- [ ] recognise as a host only known domains (`github.com`, `gitlab.com`, `bitbucket.org`, …) or a
      segment followed by at least an owner and a repo, or read the git `origin` remote when present
- [ ] regression test with `/Users/john.doe/…`

### `module-and-name-flags` — No `--module` and `--name`: the inferred path cannot be corrected

- **status**: open
- **priority**: medium
- **labels**: enhancement, go
- **milestone**: v0.9 — Smooth first run
- **ref**: [audit-2026-11-06.md](audit-2026-11-06.md)

When inference is wrong (see `module-path-dotted-dirs`) the only way out is editing `go.mod` and every
import by hand. `scaffold init --module github.com/acme/svc --name svc` turns inference into a default,
not a constraint. Propagate it to the VS Code extension (argv in `cli.ts`).

### `project-name-validation` — Invalid directory names produce a broken project

- **status**: open
- **priority**: high
- **labels**: bug, template
- **milestone**: v0.9 — Smooth first run
- **ref**: [audit-2026-11-06.md](audit-2026-11-06.md)

The directory name ends up unchecked in `go.mod`, in the `docker-compose.yml` service and in the
`Dockerfile` binary name: `/work/Service Name` → `module Service Name` (invalid).

- [ ] validate the module path (`golang.org/x/mod/module.CheckPath`) and the name; clear error before writing
- [ ] normalise the name for compose and the binary (lowercase, `-` instead of spaces)

### `atomic-file-writes` — `--force` can delete the user's file; a failed flush leaves partial files

- **status**: open
- **priority**: medium
- **labels**: bug, go
- **milestone**: v0.9 — Smooth first run
- **ref**: [audit-2026-11-06.md](audit-2026-11-06.md)

With `--force` the file is opened with `O_TRUNC`; if rendering fails, the error path calls `os.Remove`:
the original file is truncated first and then deleted. A `Flush` error leaves a partial file instead.

- [ ] write to a temporary file in the same directory, `rename` only after a successful write
- [ ] test with an `fs.FS` that fails halfway, with and without `--force`

### `app-env-default-mismatch` — `APP_ENV` has two different defaults in the generated project

- **status**: open
- **priority**: low
- **labels**: bug, template
- **milestone**: v0.9 — Smooth first run
- **ref**: [audit-2026-11-06.md](audit-2026-11-06.md)

`Configuration.AppEnv` has `envDefault:"local"`, but `loadDotEnv` reads `os.Getenv("APP_ENV")`: without
the variable the configuration says `local` but `.env.local` is not loaded. One default, in one place.

### `generated-gitignore` — The generated project has no `.gitignore`

- **status**: open
- **priority**: medium
- **labels**: enhancement, template
- **milestone**: v0.9 — Smooth first run
- **ref**: [audit-2026-11-06.md](audit-2026-11-06.md)

The first `git add .` commits `bin/` and `env/.env.local` (with `API_SECRET` and the DB password).
Generate a `.gitignore` (in `static/`) with `bin/`, `env/.env.*` except an `.env.example`, `*.test`, `*.out`.

### `cors-closed-by-default` — CORS open to everyone by default in the generated project

- **status**: open
- **priority**: medium
- **labels**: security, template
- **milestone**: v0.9 — Smooth first run
- **ref**: [audit-2026-11-06.md](audit-2026-11-06.md)

`AllowOrigins: "*"` with `Authorization` among the headers: a new service's default must be closed.
`CORS_ALLOWED_ORIGINS` from the environment, empty = no external origin, with a test.

### `unused-config-fields` — Configuration declared but unused (`LOG_LEVEL`, `RUNNING_MODE`, `API_SECRET`)

- **status**: open
- **priority**: low
- **labels**: enhancement, template
- **milestone**: v0.9 — Smooth first run
- **ref**: [audit-2026-11-06.md](audit-2026-11-06.md)

`LOG_LEVEL` changes nothing (`zap.NewProduction()` is always at `info`); `RUNNING_MODE` and `API_SECRET`
are read by no code. Wire `LOG_LEVEL` to the logger and remove the other two (or actually use them).
To be decided together with `config-yml-double-source`.

### `ddd-layout-generated` — The generated project contains no DDD layer

- **status**: open
- **priority**: high
- **labels**: enhancement, template
- **milestone**: v1.0 — Real DDD in the generated project
- **ref**: [audit-2026-11-06.md](audit-2026-11-06.md)

The name promises a DDD layout; the result has `cmd/`, `config/`, `database/`, `env/` and no domain.
It is the product's main gap.

Generate a module layout (one bounded context per folder) with a **complete example module**:

```
internal/<module>/domain/          entities, value objects, errors, repository interface
internal/<module>/application/     use cases, depending only on domain
internal/<module>/infrastructure/  GORM store implementing the repository
internal/<module>/transport/http/  HTTP handlers ⇄ use cases
```

- [ ] example module with tests at every layer (domain without a database, store with a fake or sqlite)
- [ ] composition root in `cmd/` wiring the layers (manual DI; Wire as an option in `di-wiring`)
- [ ] README and `docs/index.html` updated with the tree and "where the rules go"

### `scaffold-add-module` — `scaffold add module <name>` command to generate a new bounded context

- **status**: open
- **priority**: high
- **labels**: enhancement, go
- **milestone**: v1.0 — Real DDD in the generated project
- **ref**: [audit-2026-11-06.md](audit-2026-11-06.md)

After `init` there must be a way to add a domain without copying the example module by hand:
`scaffold add module billing` generates the four layers, the tests and the registration in the
composition root. Idempotent like `init` (skip without `--force`); also exposed in the VS Code extension.

### `architecture-test` — Architecture test in the generated project: the domain does not import infrastructure

- **status**: open
- **priority**: medium
- **labels**: enhancement, template
- **milestone**: v1.0 — Real DDD in the generated project
- **ref**: [audit-2026-11-06.md](audit-2026-11-06.md)

An `internal/archtest` that reads imports with `go/packages` (or `go list -json`) and fails if `domain`
imports `infrastructure`/`transport` or a third-party package (GORM, Fiber). It makes the DDD rule
checkable in CI instead of relying on review.

### `di-wiring` — Dependency injection: explicit composition root, Wire as an option

- **status**: open
- **priority**: low
- **labels**: enhancement, template
- **milestone**: v1.0 — Real DDD in the generated project
- **ref**: [audit-2026-11-06.md](audit-2026-11-06.md)

The default stays manual DI in `cmd/` (readable, no tooling). `--di wire` generates `wire.go` and
`wire_gen.go` for those who prefer it.

### `ddd-guide-docs` — Guide to the generated DDD layout: where the rules go, anemic vs rich model

- **status**: open
- **priority**: medium
- **labels**: documentation
- **milestone**: v1.0 — Real DDD in the generated project
- **ref**: [audit-2026-11-06.md](audit-2026-11-06.md)

A page in `docs/` with the layer diagram, the inward dependency rule, an example of a rule in the
aggregate versus the same rule in the service (anemic model), and when each one fits.

### `health-readiness` — `/livez` and `/readyz`: today `/health` does not check the database

- **status**: open
- **priority**: medium
- **labels**: enhancement, template
- **milestone**: v1.1 — Production ready
- **ref**: [audit-2026-11-06.md](audit-2026-11-06.md)

`/livez` answers when the process is alive; `/readyz` checks the dependencies (database ping with a
timeout) and answers 503 until they are ready. Health checks in the `Dockerfile` and in compose too.

### `observability` — Observability in the generated project: metrics, request id, tracing

- **status**: open
- **priority**: medium
- **labels**: enhancement, template
- **milestone**: v1.1 — Production ready
- **ref**: [audit-2026-11-06.md](audit-2026-11-06.md)

- [ ] Prometheus `/metrics` with RED metrics per route
- [ ] request id propagated in logs and responses
- [ ] optional OpenTelemetry (`OTEL_EXPORTER_OTLP_ENDPOINT` empty = off)

### `db-migrations` — Database migrations in the generated project

- **status**: open
- **priority**: medium
- **labels**: enhancement, template
- **milestone**: v1.1 — Production ready
- **ref**: [audit-2026-11-06.md](audit-2026-11-06.md)

No schema management today (neither `AutoMigrate` nor migration files). A `migrations/` folder with
`goose` or `golang-migrate`, `make migrate-up/down`, and the example module with its first migration.

### `generated-ci-and-lint` — CI and `golangci-lint` for the generated project

- **status**: open
- **priority**: medium
- **labels**: enhancement, template, github_actions
- **milestone**: v1.1 — Production ready
- **ref**: [audit-2026-11-06.md](audit-2026-11-06.md)

The generated project has no pipeline. Generate `.github/workflows/ci.yml` (gofmt, vet, lint, test,
image build) and a `.golangci.yml` with the same rules this repository uses.

### `integration-tests` — Integration tests against a real database in the generated project

- **status**: open
- **priority**: low
- **labels**: enhancement, template
- **milestone**: v1.1 — Production ready
- **ref**: [audit-2026-11-06.md](audit-2026-11-06.md)

The example module's store tested against MySQL (or PostgreSQL) with `testcontainers-go`, behind an
`integration` build tag, so `go test ./...` stays fast and Docker-free.

### `toolchain-current` — Go 1.22 and `alpine:3.20` are end of life: move to supported versions

- **status**: open
- **priority**: medium
- **labels**: go, template, github_actions
- **milestone**: v1.1 — Production ready
- **ref**: [audit-2026-11-06.md](audit-2026-11-06.md)

Go 1.22 is unsupported since February 2025; the CI matrix (1.22–1.24) is entirely unsupported;
`alpine:3.20` is since April 2026. Align the 4 places of the Go version (`go.mod`, `Dockerfile`,
`template/Dockerfile`, matrix in `go.yml`) plus `go.mod.tpl` and the runtime images, and teach Renovate
to read `go.mod.tpl` and the `Dockerfile`s inside `template/` (custom manager), or the drift comes back.

### `http-router-option` — Router choice: `--http fiber|std`

- **status**: open
- **priority**: medium
- **labels**: enhancement, go, template
- **milestone**: v1.2 — Stack choices
- **ref**: [audit-2026-11-06.md](audit-2026-11-06.md)

Since 1.22 the `net/http` router handles methods and path parameters: many teams prefer zero
dependencies. `--http std` generates handlers and tests with `net/http` and `httptest`. Evaluate Fiber v3 too.

### `db-option` — Database choice: `--db mysql|postgres|none`

- **status**: open
- **priority**: medium
- **labels**: enhancement, go, template
- **milestone**: v1.2 — Stack choices
- **ref**: [audit-2026-11-06.md](audit-2026-11-06.md)

MySQL only today. `postgres` (`pgx` driver) and `none` (no database layer, for stateless services)
change `db.go`, compose, `.env.local` and the example store. `make e2e` must exercise every combination.

### `dry-run` — `--dry-run`: show what will be generated without writing

- **status**: open
- **priority**: low
- **labels**: enhancement, go
- **milestone**: v1.2 — Stack choices
- **ref**: [audit-2026-11-06.md](audit-2026-11-06.md)

The list of files with `Create`/`Skip`/`Overwrite` as the real command would print it, exit 0, no
writes. Also useful to the VS Code extension for a preview.

### `golden-tests` — Golden tests of the generated tree

- **status**: open
- **priority**: medium
- **labels**: enhancement, go
- **milestone**: Distribution and adoption
- **ref**: [audit-2026-11-06.md](audit-2026-11-06.md)

`TestEmbeddedAssetsProduceTheDocumentedLayout` checks file names, not content. A golden test
(`testdata/golden/…`, refreshed with `-update`) captures the whole generated project: every template
change shows up as a diff in the PR. One combination per stack option.

### `binary-name` — `go install` produces a binary called `Go-DDD-Scaffold`, not `scaffold`

- **status**: open
- **priority**: medium
- **labels**: enhancement, go
- **milestone**: Distribution and adoption
- **ref**: [audit-2026-11-06.md](audit-2026-11-06.md)

`main` sits at the module root, so the binary takes the module's name. Move the entrypoint to
`cmd/scaffold/` (`go install github.com/Allan-Nava/Go-DDD-Scaffold/cmd/scaffold@latest`), keeping a root
`main` for compatibility for a few versions; update binary discovery in the VS Code extension.

### `goreleaser` — Releases with GoReleaser: binaries, checksums, SBOM, signing, Homebrew

- **status**: open
- **priority**: medium
- **labels**: enhancement, github_actions
- **milestone**: Distribution and adoption
- **ref**: [audit-2026-11-06.md](audit-2026-11-06.md)

Replaces the two current paths (see `release-drop-thirdparty-action`) with one, started by the **tag**
rather than by a release published by hand: today tags reach `v0.8.4` but the latest GitHub Release is
`v0.8.2`. Multi-platform binaries, checksums, SBOM, cosign signature, Homebrew tap.

### `product-readme` — Product README: 60-second quickstart, demo, comparison

- **status**: open
- **priority**: medium
- **labels**: documentation
- **milestone**: Distribution and adoption
- **ref**: [audit-2026-11-06.md](audit-2026-11-06.md)

The README explains installation but not why to pick it. A quickstart (`install → init → make run` in
three lines), a recorded demo (asciinema/GIF), the generated tree, a comparison with the alternatives
(go-blueprint, `gonew`), and a public example repository generated by the CLI and refreshed by CI.

### `community-files` — Contributor files: CONTRIBUTING, SECURITY, issue templates, English docs

- **status**: open
- **priority**: low
- **labels**: documentation
- **milestone**: Distribution and adoption
- **ref**: [audit-2026-11-06.md](audit-2026-11-06.md)

`CONTRIBUTING.md`, `SECURITY.md`, issue and PR templates are missing. Documentation is in **English**
(the whole repository was translated on 2026-11-06: backlog, audits, scripts, workflows, agent files).
The CLI `Dockerfile` has a work address as `maintainer`: a personal project wants a personal address
or the repository link.

## History (closed items)

Items with `status: done`: they stay here as the record of why, and the sync closes the matching
issue with an excerpt of the reason. Do not remove them: removing them loses the history.

### `audit-2026-08-11` — Audit 2026-08-11: repairing the generator (32 findings)

- **status**: done
- **priority**: high
- **labels**: documentation, go, enhancement
- **milestone**: Audit 2026-08-11
- **ref**: [audit-2026-08-11.md](audit-2026-08-11.md)

Full audit of the CLI, the generated project's templates, the VS Code extension, CI/CD and Docker.
The generator was **broken on any modern installation** and, when it managed to run, **produced a
project that did not compile**. Every finding was reproduced before being fixed and re-verified
after. Full report with evidence and measurements:
[`docs/audit-2026-08-11.md`](audit-2026-08-11.md).

The debt **not** resolved in this pass is tracked in the open items of this backlog
(`vscode-extension-rewrite`, `vscode-extension-publish-metadata`, `release-yml-pin-and-labels`,
`golangci-lint`, `generated-project-has-no-tests`, `config-yml-double-source`).

#### Blocking (5)

- [x] The `scaffold` package's `init()` did `panic("cannot find $GOPATH environment variable")`: the CLI died at startup on every module-based setup. The tests passed only because the repo was cloned in `~/go/src/github.com/…` → removed, assets in `embed.FS`
- [x] The generation target was `filepath.Dir(os.Args[0])`, i.e. **the binary's** directory: reproduced, `scaffold init` created the files next to the executable and left the launch directory empty → `os.Getwd()`
- [x] The templates were read from disk from a hardcoded GOPATH path: a release binary / `go install` / Docker image never found them → `//go:embed all:template` + `all:static`, injected as `fs.FS` into `scaffold.Options`
- [x] The generated project **did not compile**, 9 errors verified with `go build ./...`: `"log" imported and not used`, `undefined: fmt` ×3, `undefined: env`, `undefined: strconv` ×2, `"os" imported and not used`, `undefined: http` → templates rewritten, it now compiles, passes `go vet` and is `gofmt`-clean
- [x] `make up` and the CLI's final message pointed to a `docker-compose.yml` that no template generated → added `template/docker-compose.yml.tpl` (service + MySQL with a health check)

#### Correctness (10)

- [x] `html/template` used to generate **Go code**: HTML-escaping of `&&`, `<-` and struct tags. Hence the `unescaped` helper and the `data.Quit = "<-quit"` field, workarounds for the bug rather than features → `text/template`, helper and field removed, `TestGenerateDoesNotEscapeGoOperators` as the regression test
- [x] The return value of `filepath.Walk` was **discarded** in `getTemplateSets()`: every I/O error on the templates was silent → propagated
- [x] `defer src.Close()` / `defer dist.Close()` inside the walk callback: the defers pile up until the end of the whole walk, not per file → closed per file
- [x] `os.MkdirAll(..., os.ModePerm)` = `0777` on every generated directory → `0755` dirs / `0644` files, with permission tests
- [x] Unconditional `os.Create`: `scaffold init` in an existing project **overwrote** `cmd/main.go`, `go.mod`, `Dockerfile` without asking → `O_EXCL` (atomic skip) + overwriting only with `--force`
- [x] A `tmpl.Execute` error halfway through writing → left a truncated `.go` on disk → the partial file is removed, with a test
- [x] `template/db.tmpl`: `log.Fatalf("error connection on %s", dsn)` — the DSN contains the **DB password** → error with host/port/name, never the DSN
- [x] `log.Infof("Configuration loaded ", cfg.AppEnv)`: `Infof` with no format verbs, argument discarded → structured `Infow`
- [x] `test/a_main_test.go`: empty `TestMain`, zero tests, not `gofmt`-clean → removed; real tests in `scaffold/scaffold_test.go` (with `fstest.MapFS`) and `main_test.go` (real embedded assets)
- [x] `go build -o scaffold` writes **inside** `scaffold/`: Go reads `-o <existing-dir>` as "put the binary there" and produces `scaffold/Go-DDD-Scaffold`; the release tarball would have contained the sources → output in `bin/`, and never a `scaffold` line in `.gitignore`/`.dockerignore`

#### Security (5)

- [x] DB password in the logs through the DSN
- [x] Docker image with no runtime stage: it shipped the whole Go toolchain and ran as **root** → final `alpine:3.20` stage + unprivileged user `app` (uid 10001)
- [x] The Docker template copied the whole context, dotenv files included, into the final image → binary + `config/` only, and `.dockerignore` excludes `env/.env.*`
- [x] Generated directories at `0777`
- [x] `template/env/.env.local` contained sample credentials with no warning → explicit "never commit real credentials" comment

#### CI/CD (12)

- [x] `go-test.yml` duplicated `go.yml`: every push started 4 jobs doing the same build → workflow removed
- [x] The test job ran `go test` (only the root package, which had no tests) instead of `go test ./...`: **the suite never ran in CI** → `go test -race ./...`
- [x] `echo ::set-output`, syntax **disabled** by GitHub → `docker/metadata-action@v5`
- [x] `actions/github-script@v3` with `github.repos.get`, an API removed from the supported versions → step deleted, `metadata-action` generates the labels
- [x] Obsolete actions: `checkout@v3`, `setup-go@v3`, `upload-artifact@v3` (retired), `build-push-action@v4`, `login-action@v2` → v4/v5/v6
- [x] `vscode-publish.yml`: `- run: cd extensions/vscode` is a **no-op** (every step opens a new shell), so `npm ci` ran in the root with no `package.json` → `defaults.run.working-directory` + `packagePath`
- [x] `dependabot.yml` pointed to `/tests`, a directory that does not exist; the npm and docker ecosystems were missing → entries fixed
- [x] **No gate checked that the generated project compiled**: that is why the 9 compilation errors survived the whole CI → `generated-project-builds` job (generate, `go mod tidy`, `go build`, `go vet`, `gofmt`, HTTP smoke test) + `make e2e` locally
- [x] Inconsistent Go versions: `go.mod` 1.19, `Dockerfile` 1.18, `template/Dockerfile` 1.20, CI matrix 1.18/1.19 → 1.22 everywhere, matrix 1.22-1.24
- [x] No `permissions:` block and no `concurrency` → added, with `cancel-in-progress`
- [x] Version hardcoded as `v1.0.1` in `main.go` while the last tag was `v0.7.7` and the extension `0.0.1` → version from `-ldflags "-X main.version"`, taken from `github.ref_name` at release
- [x] A **4.8 MB** `scaffold-cli-macos` binary committed to the repo → removed from tracking, `.gitignore` updated

#### Performance (measured, not estimated)

- [x] Assets: `filepath.Walk` on disk + `ParseFiles` (reopens and rereads every file) + two walks with an intermediate slice → in-memory `embed.FS`, `fs.WalkDir`, `Parse` on the bytes already read, one pass per tree: **4.7 ms** per project (average over 50 generations)
- [x] `os.MkdirAll` called for **every** file (11 times) → cache of created dirs: 5 calls (-55%)
- [x] "File exists" check with an unconditional `os.Create` → `O_EXCL`: atomic skip, one syscall less per file
- [x] Direct writes to the file → `bufio.Writer`
- [x] Binary: **5.41 MB → 3.74 MB** (-31%) with `-trimpath -ldflags="-s -w"`
- [x] CLI Docker image: single-stage `golang:1.18-bullseye` (~1 GB, root, `go mod tidy` in the build, which needs the network and invalidates the cache) → multi-stage with `go mod download` in its own layer: **12.2 MB** measured
- [x] Docker build context: no `.dockerignore`, the whole repo was uploaded including `.git` and the 4.8 MB binary → **15.3 kB**
- [x] CI: 4 duplicate jobs per push, no cache, no cancel → 1 workflow, Go cache, `cancel-in-progress`, GHA cache for the Docker layers (-50% jobs)
- [x] Generated project runtime: no HTTP timeouts, pool without `ConnMaxLifetime`, no graceful shutdown, compose without a health check on the DB → `ReadTimeout`/`WriteTimeout` 15s, `ConnMaxLifetime` 1h + `ConnMaxIdleTime` 10m, `ShutdownWithTimeout(10s)` on `SIGINT`/`SIGTERM`, MySQL health check with `depends_on: service_healthy`

#### Final verification

- [x] `make check` (gofmt + go vet) clean · `make test -race` ok
- [x] `make e2e`: 11 files generated, `go mod tidy && go build ./... && go vet ./... && gofmt -l` clean
- [x] Functional smoke test: `GET /health` = 200, `SIGTERM` = graceful shutdown
- [x] Idempotency: a second `scaffold init` = 0 created / 11 skipped; `--force` = 11 rewritten
- [x] `docker build` = 12.2 MB, `scaffold version` prints the version; `docker run -v $PWD:/work` generates into the bind mount
- [x] Module path inferred from the directory: `~/code/github.com/acme/myservice` → `module github.com/acme/myservice` (removed the `module TODO/<name>` placeholder, which `go get` cannot resolve)

### `golangci-lint` — No linter beyond gofmt + go vet

- **status**: done
- **priority**: medium
- **labels**: enhancement, go, github_actions
- **milestone**: Hardening CI/CD
- **ref**: [audit-2026-08-11.md](audit-2026-08-11.md)

> **CLOSED**: `.golangci.yml` (v2) with `errcheck`, `nilerr`, `bodyclose`, `revive`, `misspell`,
> `unconvert`, `wastedassign`, `copyloopvar` on top of the defaults. It immediately found **7**
> ignored return values in `scaffold/scaffold.go` (4 `Close`/`Remove` on error paths, 3 progress
> `Fprintf`s): no bug, but a bare `f.Close()` reads like an oversight, so they are now `_ = ...` with
> the reason next to them. `check-blank` stays OFF on purpose, so `_ =` is the accepted form for
> "ignored on purpose" instead of a `//nolint` that says less.
> Gate in `make lint` + `golangci-lint` job in `go.yml` (action pinned to `v2.12.2`). `0 issues`.

CI runs `gofmt -l` and `go vet ./...`, which do not cover the most frequent class of error in this
repo: ignored return values. The audit's most expensive historical bug was exactly that — the return
value of `filepath.Walk` discarded in `getTemplateSets()`, which hid every I/O error on the
templates. `errcheck` would have reported it.

To do:

- [ ] add a `.golangci.yml` with at least `errcheck`, `ineffassign`, `misspell`, `revive`;
- [ ] a `golangci-lint` job in `go.yml` (action `golangci/golangci-lint-action`, pinned);
- [ ] a `make lint` target for the same gate locally;
- [ ] consider extending it to the **generated project** too (a linter on the templates would make
      explicit the conventions the template teaches its users).

### `generated-project-has-no-tests` — The generated project contains no tests

- **status**: done
- **priority**: medium
- **labels**: enhancement, go, template
- **milestone**: Generated project
- **ref**: [audit-2026-08-11.md](audit-2026-08-11.md)

> **CLOSED**: the generated project goes from 11 to **13 files**, with two real tests.
> `cmd/main_test.go` drives the router through `app.Test(httptest.NewRequest(...))` - no port to
> open, no database: it checks `/health` = 200 and that a route that does not exist stays 404 (a
> catch-all added by mistake would show up immediately). To do that `main.tmpl` now extracts
> `newApp(name) *fiber.App`, separate from `main()`.
> `env/env_test.go` pins the `envDefault` tags - the ones that keep `make run` working with no
> configuration - using `goenv.ParseWithOptions` with an explicit environment: the tests are
> **hermetic**, the shell that runs them cannot make them pass or fail.
> They are not vacuous: changing an `envDefault` makes the test fail with the right message.
> `make e2e` and the CI job now also run `go test ./...` on the generated project.

`scaffold init` produces 11 files and **zero** `_test.go`. The generated `Makefile` has a `test`
target that runs `go test ./...` on a project with no tests: it always passes, so it says nothing. A
scaffold teaches the conventions of the project it generates: with no example test it teaches that
tests are optional.

To do:

- [ ] `template/cmd/main_test.tmpl` with a smoke test of the `/health` endpoint through
      `app.Test(httptest.NewRequest(...))` (Fiber supports it natively, no port needed);
- [ ] `template/env/env_test.tmpl` on the `Configuration` defaults (`envDefault` is easy to break and
      nobody notices);
- [ ] update the exact assertion of `TestEmbeddedAssetsProduceTheDocumentedLayout` in `main_test.go`
      and the tree in `README.md` + `docs/index.md`;
- [ ] extend the `generated-project-builds` job with a `go test ./...` on the generated project.

### `release-yml-pin-and-labels` — `release.yml`: third-party action not pinned to a SHA, jobs mislabelled

- **status**: done
- **priority**: medium
- **labels**: github_actions, documentation
- **milestone**: Hardening CI/CD
- **ref**: [audit-2026-08-11.md](audit-2026-08-11.md)

> **CLOSED**, and the finding went deeper than I had written. Reading the action's code
> (`go-release.action`'s `build.sh`) it turned out that **if the repo has an executable `./build.sh`
> the action uses it and captures its stdout as the list of files to archive**
> (`OUTPUT=$(./build.sh "$CMD_PATH")`), ignoring `BUILD_ARGS`; and `PROJECT_NAME` is overwritten with
> the repo's basename anyway. Our `build.sh` printed logs to stdout, so the release archive was
> built from file names that did not exist. Now `build.sh` has an explicit contract: **stdout = only
> the binary's path**, everything else on stderr, plus the `.exe` suffix on windows (it was missing:
> the Windows artefacts were not executable).
> The workflow: 8 copy-pasted jobs -> one **matrix**, so the names can no longer lie (they were wrong
> on 3 jobs, `darwin/amd64` was duplicated and `darwin/arm64` was not published at all); action
> pinned to SHA `ad97966` instead of the mutable tag `v1.5.01`; `permissions: write-all` ->
> `contents: write` per job; `actions/checkout@master` -> `@v4`; `on: release` -> `types: [published]`
> (before, it also restarted on edited/deleted); the commented-out block removed.
> `release-drop-thirdparty-action` stays open: pinning reduces the risk, it does not remove it.

`.github/workflows/release.yml` uses `Allan-Nava/go-release.action@v1.5.01` on **every** release job,
with `permissions: write-all`. A third-party action referenced by a mutable tag and with full
permissions on the token is the typical supply-chain attack surface in CI: whoever controls that tag
can rewrite the release assets.

To do:

- [ ] pin the action to the commit **SHA** (`@<sha40>`), not to the tag;
- [ ] narrow `permissions` to the minimum per job (`contents: write` on the jobs that publish);
- [ ] fix the job labels: `release-macos-amd64` is named `release macos/32` but builds amd64;
- [ ] remove the commented-out blocks (macos/386) or re-enable them, not leave them half done.

Not touched in the 2026-08-11 audit: the workflow cannot be run locally, and changing it blind risks
breaking the publication of the binaries.

### `vscode-extension-rewrite` — VS Code extension: the command generates nothing, it needs a rewrite

- **status**: done
- **priority**: high
- **labels**: bug, vscode, javascript
- **milestone**: VS Code extension
- **ref**: [audit-2026-08-11.md](audit-2026-08-11.md)

> **CLOSED**: rewritten as a **wrapper of the CLI**. Of the two options I took the first (the
> extension invokes the `scaffold` binary): it removes the drift between the two template sets by
> construction, because the second set no longer exists.
>
> Structure: `src/scaffold/cli.ts` is **pure** (it does not import `vscode`) and holds all the logic -
> binary discovery, argv, output parsing; the editor API lives only in `extension.ts` and
> `commands/init.ts`. That separation is what makes the risky part testable without a VS Code host:
> **15 tests** with `node --test`, which run in CI too.
>
> Details the old code got wrong that are now covered: `go install` produces a binary called
> `Go-DDD-Scaffold` (after the module), not `scaffold`, so discovery looks for both; it also looks in
> `$GOBIN`/`$GOPATH/bin`/`~/go/bin`, because a GUI editor does not read the shell profile and a CLI
> that works in the terminal looks absent from the editor; a **same-named** program that does not
> answer `scaffold version` is discarded instead of being run with `init`. When it is missing, the
> extension offers the `go install` in a terminal.
>
> Removed: `src/templates/` (second set, never read), `src/utils/get-selected-text.ts` (Dart code),
> the placeholder `helloWorld` command, the broken reference to `assets/logo.png`, and the **5
> runtime deps** (`change-case`, `lodash`, `mkdirp`, `node-fetch`, `semver`) that only served the
> dead code: the extension now has **zero runtime dependencies**. `tsconfig` moves to `strict: true`.
>
> Verified against the **real** binary, not a mock: discovery OK, 13 files created, re-run 0
> created / 13 skipped, `--force` 13 rewritten, `module github.com/acme/svc` correct; with a clean
> PATH an unrelated binary is rejected with `BinaryNotFoundError` and the GOBIN fallback finds it.
> New `vscode extension` CI job (lint + test + production bundle): before, nothing checked it on every
> push, and `vscode-publish.yml` only compiled it at publish time.

The `extension.new-scaffold` command writes **no** file: it is the leftover of a Flutter/BLoC
generator badly adapted. Verified by reading `extensions/vscode/src/`:

- `createScaffoldTemplate()` computes a `…_bloc.dart` target (**Dart**, not Go) and the `writeFile`
  is **commented out**: the function only does a `console.log` and a `throw` if the file exists;
- `getScaffoldTemplate()` (in `src/templates/scaffold.template.ts`) is imported nowhere;
- the `src/templates/**/*.tpl` files are **read by no code**: they are dead documentation;
- `workspace.getConfiguration("bloc").get("newBlocTemplate.createDirectory")` reads **another**
  extension's configuration namespace;
- `createDirectoryV2()` does not await `mkdirp` → race on the directory creation;
- the logic is duplicated between `extension.ts` and `commands/new-scaffold.command.ts`; the latter is
  never registered and uses `mkdirp` with a callback, an API removed in mkdirp v1+;
- a placeholder `go-ddd-scaffold.helloWorld` command remains.

It needs a **rewrite**, not a patch. A preliminary decision is needed: rewrite the extension as a
wrapper of the `scaffold` binary (a single source of truth, the extension invokes the CLI) or
reimplement the generation in TypeScript (two implementations to keep twins). The first removes the
drift between the two template sets by construction.

In this pass only the twin `.tpl` files were fixed (compilation errors; `database/db.tpl` was an
identical duplicate of `config/config.tpl`), to respect the twins rule in `CLAUDE.md`.

### `vscode-extension-publish-metadata` — VS Code extension: `publisher: "None"` and version 0.0.1 → publishing fails

- **status**: done
- **priority**: medium
- **labels**: bug, vscode, github_actions
- **milestone**: VS Code extension
- **ref**: [audit-2026-08-11.md](audit-2026-08-11.md)

> **CLOSED**: fixed by taking `nomad-lens`, which already publishes on the Marketplace, as the model.
> It also provided the fact I said I could not guess: the publisher is **`allannava95`** (verified:
> it has `nomad-lens v1.4.0` published). `go-ddd-scaffold` was not published on either the
> Marketplace or Open VSX, so the first one will be a create.
>
> Metadata: `publisher: allannava95`, `version: 0.1.0` (0.0.1 belonged to an extension that generated
> nothing), explicit `main`. The `.vsix` declares `allannava95.go-ddd-scaffold` 0.1.0 and really
> contains the file `main` points to.
>
> Workflow rewritten on the nomad-lens model: three jobs `verify` -> `package` -> `publish`, with a
> **check that the tag matches `version`**, `vsce`/`ovsx` from the CLI instead of the third-party
> action, and publish steps **conditional on the secrets being present** (skipped, not failed).
>
> Two choices different from nomad-lens, for this repo's context: the trigger is the **`vscode-v*`**
> namespace and not `v*`, because here every commit has a tag and otherwise a version would be
> published on the marketplaces on every commit; and the `.vsix` goes into an **artifact** instead of
> a GitHub Release, because creating a release would trigger `release.yml`/`go-release.yml` (which
> react to `release: published`) and publish the Go binaries inside an extension release.

`extensions/vscode/package.json` has `"publisher": "None"` and `"version": "0.0.1"` while the repo is
at `v0.7.7`. `vscode-publish.yml` fires on every `v*` tag and publishes to Open VSX + the Visual Studio
Marketplace: with these metadata the publish fails (non-existent publisher) and, even if it
succeeded, it would publish a version that does not match the tag.

To do:

- [ ] set the real `publisher` (the publisher ID on marketplace.visualstudio.com);
- [ ] align `version` with the tag, or derive it from the tag in the workflow (`npm version --no-git-tag-version`);
- [x] ~~`description` is empty~~ — filled in during the rewrite;
- [ ] consider gating the publish (only tags that touch `extensions/`), so the extension is not
      published on every CLI release.

No longer blocked by `vscode-extension-rewrite` (closed: the extension now works). Still to do because
it needs **your publisher ID** on the Marketplace, which I cannot guess.

Meanwhile **packaging was unblocked and verified**: `vsce package` failed with *"Couldn't detect the
repository where this extension is published"* because `package.json` had no `repository`, and the
README's relative links could not be resolved — the publish on tag would have broken there, before
even reaching the publisher. Added `repository`, `bugs`, `homepage`, `license: MIT` and the `LICENSE`
file; removed `tslint.json` (deprecated, eslint is used) which ended up inside the `.vsix`. It now
produces a clean package: **7 files, 8.5 KB, zero warnings**.

### `vscode-marketplace-secret` — The VSCE_PAT secret is missing: publishing is skipped instead of done

- **status**: done
- **priority**: medium
- **labels**: github_actions, vscode
- **milestone**: VS Code extension
- **ref**: [audit-2026-08-11.md](audit-2026-08-11.md)

> **CLOSED**: the `VSCE_PAT` secret was added and the extension is **published**.
> `allannava95.go-ddd-scaffold` **v0.1.0**, online since 2026-08-12 09:39 UTC:
> https://marketplace.visualstudio.com/items?itemName=allannava95.go-ddd-scaffold
>
> The run passed on all three jobs, and did exactly what it was designed for:
> `Publish to VS Code Marketplace` **success**, `Publish to Open VSX` **skipped** (no `OVSX_PAT`)
> without failing the job.
>
> After publishing, the trigger was simplified on request: now **any tag** triggers it and the
> decision belongs to the `package.json` `version`, compared with the public gallery. No separate tag
> namespace any more: to release, raise the version; the next tag publishes it and the ones after do
> nothing. `OVSX_PAT` remains optional.

The repo has **no secrets** configured (`gh secret list` is empty): the old `OPEN_VSX_TOKEN` /
`VS_MARKETPLACE_TOKEN` the workflow referred to never existed. That is the underlying reason why the
publish could not work, beyond the metadata.

Now `vscode-publish.yml` **skips** the publish steps when the token is missing, instead of failing:
the `.vsix` is produced and uploaded as an artifact anyway, and the run summary says which
marketplace was skipped and why. So nothing is broken — it simply does not publish.

To do (it needs an action from you, I cannot create secrets):

- [ ] add the **`VSCE_PAT`** secret to the repo — an Azure DevOps Personal Access Token for the
      `allannava95` publisher, scope *Marketplace → Manage*. It is the same kind of token already used
      in `nomad-lens`;
- [ ] optional: **`OVSX_PAT`** for Open VSX. `nomad-lens` does not have it either, so today neither
      extension is on Open VSX: if it is not of interest, the step stays skipped and that is fine;
- [ ] consider keeping the tokens in a GitHub **environment** (`marketplace`) with a protection rule,
      as `nomad-lens` does, instead of as a repo secret. In that case `environment: marketplace` must be
      added to the `publish` job, otherwise it does not see them.

First publication: tag `vscode-v0.1.0` (it must match `version` in `package.json`).
