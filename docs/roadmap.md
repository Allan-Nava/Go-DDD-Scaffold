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

## Distribuzione e adozione

_5 open · 0 done_

| id | Titolo | Priorità | Status |
|----|--------|----------|--------|
| `binary-name` | `go install` produce un binario `Go-DDD-Scaffold`, non `scaffold` | medium | open |
| `golden-tests` | Test golden dell'albero generato | medium | open |
| `goreleaser` | Release con GoReleaser: binari, checksum, SBOM, firma, Homebrew | medium | open |
| `product-readme` | README da prodotto: quickstart in 60 secondi, demo, confronto | medium | open |
| `community-files` | File per i contributori: CONTRIBUTING, SECURITY, template di issue, lingua unica | low | open |

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

## v0.9 — Primo avvio senza intoppi

_9 open · 0 done_

| id | Titolo | Priorità | Status |
|----|--------|----------|--------|
| `first-run-needs-mysql` | Il primo `make run` del progetto generato muore senza MySQL | high | open |
| `module-path-dotted-dirs` | Module path sbagliato se il percorso contiene un segmento con un punto | high | open |
| `project-name-validation` | Nomi di directory non validi producono un progetto rotto | high | open |
| `atomic-file-writes` | `--force` può cancellare il file dell'utente; un flush fallito lascia file parziali | medium | open |
| `cors-closed-by-default` | CORS aperto a tutti di default nel progetto generato | medium | open |
| `generated-gitignore` | Il progetto generato non ha un `.gitignore` | medium | open |
| `module-and-name-flags` | Mancano `--module` e `--name`: il path dedotto non si può correggere | medium | open |
| `app-env-default-mismatch` | `APP_ENV` ha due default diversi nel progetto generato | low | open |
| `unused-config-fields` | Configurazione dichiarata ma non usata (`LOG_LEVEL`, `RUNNING_MODE`, `API_SECRET`) | low | open |

## v1.0 — DDD vero nel progetto generato

_5 open · 0 done_

| id | Titolo | Priorità | Status |
|----|--------|----------|--------|
| `ddd-layout-generated` | Il progetto generato non contiene nessun livello DDD | high | open |
| `scaffold-add-module` | Comando `scaffold add module <nome>` per generare un nuovo bounded context | high | open |
| `architecture-test` | Test d'architettura nel progetto generato: il dominio non importa l'infrastruttura | medium | open |
| `ddd-guide-docs` | Guida al layout DDD generato: dove vanno le regole, modello anemico contro ricco | medium | open |
| `di-wiring` | Dependency injection: composition root esplicito, Wire come opzione | low | open |

## v1.1 — Pronto per la produzione

_6 open · 0 done_

| id | Titolo | Priorità | Status |
|----|--------|----------|--------|
| `db-migrations` | Migrazioni del database nel progetto generato | medium | open |
| `generated-ci-and-lint` | CI e `golangci-lint` per il progetto generato | medium | open |
| `health-readiness` | `/livez` e `/readyz`: oggi `/health` non controlla il database | medium | open |
| `observability` | Osservabilità nel progetto generato: metriche, request id, tracing | medium | open |
| `toolchain-current` | Go 1.22 e `alpine:3.20` sono a fine vita: allineare a versioni supportate | medium | open |
| `integration-tests` | Test d'integrazione con un database vero nel progetto generato | low | open |

## v1.2 — Scelte di stack

_3 open · 0 done_

| id | Titolo | Priorità | Status |
|----|--------|----------|--------|
| `db-option` | Scelta del database: `--db mysql\|postgres\|none` | medium | open |
| `http-router-option` | Scelta del router: `--http fiber\|std` | medium | open |
| `dry-run` | `--dry-run`: mostrare cosa verrà generato senza scrivere | low | open |

## Non pianificati (senza milestone)

_0 item open senza milestone. Assegnane una con `- **milestone**: <titolo>` in [`backlog.md`](backlog.md): se il titolo non esiste ancora su GitHub, il sync la crea._
