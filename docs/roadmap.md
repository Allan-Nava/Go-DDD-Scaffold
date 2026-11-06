---
layout: default
title: Roadmap
nav_order: 8
description: "Milestone del backlog — pagina generata da docs/backlog.md"
---

# Roadmap — milestone del backlog

<!-- GENERATO da docs/scripts/generate-roadmap.py — NON editare a mano. -->
> Pagina **generata** da [`docs/scripts/generate-roadmap.py`](scripts/generate-roadmap.py) leggendo [`docs/backlog.md`](backlog.md) (unica sorgente). Le milestone sono quelle dichiarate dagli item: non c'è un elenco fisso. Affianca la [board issue di GitHub](https://github.com/Allan-Nava/Go-DDD-Scaffold/milestones).

_9 milestone · 38 item pianificati (30 open · 8 done)._

## Audit 2026-08-11

_0 open · 1 done_

| id | Titolo | Priorità | Status |
|----|--------|----------|--------|
| `audit-2026-08-11` | Audit 2026-08-11: risanamento del generatore (32 finding) | high | done |

## Distribution and adoption

_5 open · 0 done_

| id | Titolo | Priorità | Status |
|----|--------|----------|--------|
| `binary-name` | `go install` produces a binary called `Go-DDD-Scaffold`, not `scaffold` | medium | open |
| `golden-tests` | Golden tests of the generated tree | medium | open |
| `goreleaser` | Releases with GoReleaser: binaries, checksums, SBOM, signing, Homebrew | medium | open |
| `product-readme` | Product README: 60-second quickstart, demo, comparison | medium | open |
| `community-files` | Contributor files: CONTRIBUTING, SECURITY, issue templates, English docs | low | open |

## Estensione VSCode

_0 open · 3 done_

| id | Titolo | Priorità | Status |
|----|--------|----------|--------|
| `vscode-extension-rewrite` | Estensione VSCode: il comando non genera nulla, serve una riscrittura | high | done |
| `vscode-extension-publish-metadata` | Estensione VSCode: `publisher: "None"` e versione 0.0.1 → il publish fallisce | medium | done |
| `vscode-marketplace-secret` | Manca il secret VSCE_PAT: il publish si salta invece di pubblicare | medium | done |

## Hardening CI/CD

_1 open · 3 done_

| id | Titolo | Priorità | Status |
|----|--------|----------|--------|
| `release-drop-thirdparty-action` | release.yml: valutare se togliere del tutto la action di terze parti | low | open |
| `golangci-lint` | Nessun linter oltre gofmt + go vet | medium | done |
| `release-yml-pin-and-labels` | `release.yml`: action di terze parti non pinnata a SHA, job etichettati male | medium | done |
| `pages-config-stale` | GitHub Pages: config datata e build mai osservata verde | low | done |

## Progetto generato

_1 open · 1 done_

| id | Titolo | Priorità | Status |
|----|--------|----------|--------|
| `config-yml-double-source` | `config/config.yml` non è letto: doppia fonte di configurazione | low | open |
| `generated-project-has-no-tests` | Il progetto generato non contiene nessun test | medium | done |

## v0.9 — Smooth first run

_9 open · 0 done_

| id | Titolo | Priorità | Status |
|----|--------|----------|--------|
| `first-run-needs-mysql` | The generated project's first `make run` dies without MySQL | high | open |
| `module-path-dotted-dirs` | Wrong module path when the path contains a segment with a dot | high | open |
| `project-name-validation` | Invalid directory names produce a broken project | high | open |
| `atomic-file-writes` | `--force` can delete the user's file; a failed flush leaves partial files | medium | open |
| `cors-closed-by-default` | CORS open to everyone by default in the generated project | medium | open |
| `generated-gitignore` | The generated project has no `.gitignore` | medium | open |
| `module-and-name-flags` | No `--module` and `--name`: the inferred path cannot be corrected | medium | open |
| `app-env-default-mismatch` | `APP_ENV` has two different defaults in the generated project | low | open |
| `unused-config-fields` | Configuration declared but unused (`LOG_LEVEL`, `RUNNING_MODE`, `API_SECRET`) | low | open |

## v1.0 — Real DDD in the generated project

_5 open · 0 done_

| id | Titolo | Priorità | Status |
|----|--------|----------|--------|
| `ddd-layout-generated` | The generated project contains no DDD layer | high | open |
| `scaffold-add-module` | `scaffold add module <name>` command to generate a new bounded context | high | open |
| `architecture-test` | Architecture test in the generated project: the domain does not import infrastructure | medium | open |
| `ddd-guide-docs` | Guide to the generated DDD layout: where the rules go, anemic vs rich model | medium | open |
| `di-wiring` | Dependency injection: explicit composition root, Wire as an option | low | open |

## v1.1 — Production ready

_6 open · 0 done_

| id | Titolo | Priorità | Status |
|----|--------|----------|--------|
| `db-migrations` | Database migrations in the generated project | medium | open |
| `generated-ci-and-lint` | CI and `golangci-lint` for the generated project | medium | open |
| `health-readiness` | `/livez` and `/readyz`: today `/health` does not check the database | medium | open |
| `observability` | Observability in the generated project: metrics, request id, tracing | medium | open |
| `toolchain-current` | Go 1.22 and `alpine:3.20` are end of life: move to supported versions | medium | open |
| `integration-tests` | Integration tests against a real database in the generated project | low | open |

## v1.2 — Stack choices

_3 open · 0 done_

| id | Titolo | Priorità | Status |
|----|--------|----------|--------|
| `db-option` | Database choice: `--db mysql\|postgres\|none` | medium | open |
| `http-router-option` | Router choice: `--http fiber\|std` | medium | open |
| `dry-run` | `--dry-run`: show what will be generated without writing | low | open |

## Non pianificati (senza milestone)

_0 item open senza milestone. Assegnane una con `- **milestone**: <titolo>` in [`backlog.md`](backlog.md): se il titolo non esiste ancora su GitHub, il sync la crea._
