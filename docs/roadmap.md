---
layout: default
title: Roadmap
nav_order: 8
description: "Milestone del backlog — pagina generata da docs/backlog.md"
---

# Roadmap — milestone del backlog

<!-- GENERATO da docs/scripts/generate-roadmap.py — NON editare a mano. -->
> Pagina **generata** da [`docs/scripts/generate-roadmap.py`](scripts/generate-roadmap.py) leggendo [`docs/backlog.md`](backlog.md) (unica sorgente). Le milestone sono quelle dichiarate dagli item: non c'è un elenco fisso. Affianca la [board issue di GitHub](https://github.com/Allan-Nava/Go-DDD-Scaffold/milestones).

_4 milestone · 8 item pianificati (3 open · 5 done)._

## Audit 2026-08-11

_0 open · 1 done_

| id | Titolo | Priorità | Status |
|----|--------|----------|--------|
| `audit-2026-08-11` | Audit 2026-08-11: risanamento del generatore (32 finding) | high | done |

## Estensione VSCode

_1 open · 1 done_

| id | Titolo | Priorità | Status |
|----|--------|----------|--------|
| `vscode-extension-publish-metadata` | Estensione VSCode: `publisher: "None"` e versione 0.0.1 → il publish fallisce | medium | open |
| `vscode-extension-rewrite` | Estensione VSCode: il comando non genera nulla, serve una riscrittura | high | done |

## Hardening CI/CD

_1 open · 2 done_

| id | Titolo | Priorità | Status |
|----|--------|----------|--------|
| `release-drop-thirdparty-action` | release.yml: valutare se togliere del tutto la action di terze parti | low | open |
| `golangci-lint` | Nessun linter oltre gofmt + go vet | medium | done |
| `release-yml-pin-and-labels` | `release.yml`: action di terze parti non pinnata a SHA, job etichettati male | medium | done |

## Progetto generato

_1 open · 1 done_

| id | Titolo | Priorità | Status |
|----|--------|----------|--------|
| `config-yml-double-source` | `config/config.yml` non è letto: doppia fonte di configurazione | low | open |
| `generated-project-has-no-tests` | Il progetto generato non contiene nessun test | medium | done |

## Non pianificati (senza milestone)

_0 item open senza milestone. Assegnane una con `- **milestone**: <titolo>` in [`backlog.md`](backlog.md): se il titolo non esiste ancora su GitHub, il sync la crea._
