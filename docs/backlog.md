---
layout: default
title: Backlog
nav_order: 7
description: "Backlog operativo — sorgente di verità per le issue GitHub"
---

# Backlog operativo — sorgente di verità per le issue GitHub

Questo file è l'**unica sorgente di verità** dei todo del repo. Uno script idempotente
([`scripts/sync-backlog-to-issues.py`](scripts/sync-backlog-to-issues.py)) lo legge e
**apre/aggiorna/chiude** una issue GitHub per ogni item, creando al volo le milestone mancanti.

## Come funziona (flusso)

```
  docs/backlog.md  ──parse(id)──▶  sync-backlog-to-issues.py  ──API GitHub──▶  Issues
   ### `id` — Titolo                  (idempotente)                           (label backlog-sync
   - status / labels / milestone      fingerprint nel corpo:                    + fingerprint + milestone)
   - priority / ref                   <!-- backlog-id: id | hash: sha -->
                                                │
   item open  + nessuna issue   → CREATE        │  milestone dichiarata dall'item
   item open  + issue aperta    → SKIP          └─ cercata per titolo esatto sul repo
                  hash diverso  → UPDATE           └─ se manca → CREATE (dinamica)
   item open  + issue chiusa    → REOPEN
   item done / rimosso          → CLOSE (con commento + estratto dal backlog)
          │
          └──▶  generate-roadmap.py  ──▶  docs/roadmap.md  (pagina generata, una sezione
                                                            per milestone, conteggi open/done)
```

Re-run sicuro: il match è per `id` stabile (fingerprint nel corpo issue), **non** per testo → editare
titolo/descrizione non crea doppioni. Lo script tocca **solo** le issue con label `backlog-sync`:
quelle di Renovate/Dependabot o aperte da terzi non vengono mai lette né chiuse.

## Milestone dinamiche

**Non esiste un elenco di milestone da nessuna parte.** Esistono solo i titoli scritti negli item:

- aggiungere una milestone = scrivere `- **milestone**: <titolo>` su un item. Al primo sync lo
  script la cerca per titolo esatto sul repo e, se manca, **la crea**;
- `docs/roadmap.md` è **generata** da `generate-roadmap.py`: scopre le milestone dagli item, le
  ordina e conta open/done. Non editarla a mano;
- il titolo è incluso nell'hash del fingerprint → cambiarlo o toglierlo riallinea la issue al sync
  successivo. Una milestone assegnata a mano non viene mai rimossa se l'item non ne dichiara una;
- item della stessa milestone → titolo **identico carattere per carattere**. `backlog-lint` avvisa
  se trova varianti di case/spazi dello stesso titolo.

## Convenzione di scrittura di un item

- Un item inizia con `### \`<id-stabile>\` — <Titolo>` (l'`id` è in backtick, kebab-case, **non cambiarlo mai**).
- Metadati come bullet `- **chiave**: valore`:
  - **status**: `open` (default) | `done` → `done` chiude la issue.
  - **labels**: lista separata da virgola (la label `backlog-sync` è aggiunta in automatico; le
    label mancanti sul repo vengono create).
  - **priority**: `low` | `medium` | `high` (opzionale).
  - **milestone**: titolo della milestone (opzionale, creata se manca — vedi sopra).
  - **owner**: login GitHub a cui assegnare la issue (opzionale).
  - **ref**: link/percorso al doc di riferimento (opzionale).
- Tutto il resto del blocco (prosa) diventa il **corpo** della issue.

> Per "chiudere" un todo: metti `status: done` (storico) **oppure** rimuovi l'item. In entrambi i casi
> la issue viene chiusa al prossimo sync. Preferire `done` per gli item **già pushati** (la issue si
> chiude con la traccia del perché); la rimozione va bene solo per item mai sincronizzati.

## Uso

```sh
make backlog-lint             # valida il backlog, non tocca GitHub
make roadmap                  # rigenera docs/roadmap.md
make backlog-sync             # dry-run: mostra il piano
make backlog-sync-apply       # applica: crea/aggiorna/chiude le issue e le milestone
```

⚠️ Se cambi questo file **rigenera `roadmap.md`** e committala, altrimenti il gate
`generated-pages-check` in CI fallisce.

---

## Item attivi

### `vscode-extension-rewrite` — Estensione VSCode: il comando non genera nulla, serve una riscrittura

- **status**: open
- **priority**: high
- **labels**: bug, vscode, javascript
- **milestone**: Estensione VSCode
- **ref**: [audit-2026-08-11.md](audit-2026-08-11.md)

Il comando `extension.new-scaffold` non scrive **nessun** file: è il residuo di un generatore
Flutter/BLoC riadattato male. Verificato leggendo `extensions/vscode/src/`:

- `createScaffoldTemplate()` calcola un target `…_bloc.dart` (**Dart**, non Go) e la `writeFile` è
  **commentata**: la funzione fa solo `console.log` e un `throw` se il file esiste;
- `getScaffoldTemplate()` (in `src/templates/scaffold.template.ts`) non è importato da nessuna parte;
- i file `src/templates/**/*.tpl` **non sono letti da alcun codice**: sono documentazione morta;
- `workspace.getConfiguration("bloc").get("newBlocTemplate.createDirectory")` legge il namespace di
  configurazione di un'**altra** estensione;
- `createDirectoryV2()` non attende `mkdirp` → race sulla creazione della directory;
- la logica è duplicata fra `extension.ts` e `commands/new-scaffold.command.ts`; quest'ultimo non è
  mai registrato e usa `mkdirp` con callback, API rimossa in mkdirp v1+;
- resta un comando placeholder `go-ddd-scaffold.helloWorld`.

Serve una **riscrittura**, non una patch. Decisione preliminare da prendere: riscrivere l'estensione
come wrapper del binario `scaffold` (una sola implementazione di verità, l'estensione invoca la CLI)
oppure reimplementare la generazione in TypeScript (due implementazioni da tenere gemelle).
La prima elimina per costruzione il drift fra i due set di template.

In questo passaggio sono stati corretti solo i `.tpl` gemelli (errori di compilazione;
`database/db.tpl` era un duplicato identico di `config/config.tpl`), per rispettare la regola dei
gemelli di `CLAUDE.md`.

### `vscode-extension-publish-metadata` — Estensione VSCode: `publisher: "None"` e versione 0.0.1 → il publish fallisce

- **status**: open
- **priority**: medium
- **labels**: bug, vscode, github_actions
- **milestone**: Estensione VSCode
- **ref**: [audit-2026-08-11.md](audit-2026-08-11.md)

`extensions/vscode/package.json` ha `"publisher": "None"` e `"version": "0.0.1"` mentre il repo è a
`v0.7.7`. `vscode-publish.yml` scatta su ogni tag `v*` e pubblica su Open VSX + Visual Studio
Marketplace: con questi metadati il publish fallisce (publisher inesistente) e, se anche riuscisse,
pubblicherebbe una versione che non corrisponde al tag.

Da fare:

- [ ] impostare il `publisher` reale (l'ID del publisher su marketplace.visualstudio.com);
- [ ] allineare `version` al tag, o derivarla dal tag nel workflow (`npm version --no-git-tag-version`);
- [ ] `description` è vuota;
- [ ] valutare se il publish debba essere gated (solo tag che toccano `extensions/`), per non
      pubblicare l'estensione a ogni release della CLI.

Dipende da `vscode-extension-rewrite`: pubblicare un'estensione che non genera nulla non ha senso.

### `release-yml-pin-and-labels` — `release.yml`: action di terze parti non pinnata a SHA, job etichettati male

- **status**: open
- **priority**: medium
- **labels**: github_actions, documentation
- **milestone**: Hardening CI/CD
- **ref**: [audit-2026-08-11.md](audit-2026-08-11.md)

`.github/workflows/release.yml` usa `Allan-Nava/go-release.action@v1.5.01` su **tutti** i job di
release, con `permissions: write-all`. Una action di terze parti riferita per tag mutabile e con
permessi totali sul token è la superficie di attacco tipica della supply chain CI: chi controlla quel
tag può riscrivere gli asset di release.

Da fare:

- [ ] pinnare la action al **SHA** del commit (`@<sha40>`), non al tag;
- [ ] restringere `permissions` al minimo per job (`contents: write` sui job che pubblicano);
- [ ] correggere le label dei job: `release-macos-amd64` si chiama `release macos/32` ma builda amd64;
- [ ] rimuovere i blocchi commentati (macos/386) o riattivarli, non lasciarli a metà.

Non toccato nell'audit del 2026-08-11: il workflow non è eseguibile in locale, e modificarlo alla
cieca rischia di rompere la pubblicazione dei binari.

### `golangci-lint` — Nessun linter oltre gofmt + go vet

- **status**: open
- **priority**: medium
- **labels**: enhancement, go, github_actions
- **milestone**: Hardening CI/CD
- **ref**: [audit-2026-08-11.md](audit-2026-08-11.md)

La CI esegue `gofmt -l` e `go vet ./...`, che non coprono la classe di errori più frequente in questo
repo: valori di ritorno ignorati. Il bug storico più costoso dell'audit era esattamente questo — il
valore di ritorno di `filepath.Walk` scartato in `getTemplateSets()`, che nascondeva ogni errore di
I/O sui template. `errcheck` lo avrebbe segnalato.

Da fare:

- [ ] aggiungere `.golangci.yml` con almeno `errcheck`, `ineffassign`, `misspell`, `revive`;
- [ ] job `golangci-lint` in `go.yml` (action `golangci/golangci-lint-action`, pinnata);
- [ ] target `make lint` per avere lo stesso gate in locale;
- [ ] valutare se estenderlo anche al **progetto generato** (un linter sui template renderebbe
      esplicite le convenzioni che il template insegna a chi lo usa).

### `generated-project-has-no-tests` — Il progetto generato non contiene nessun test

- **status**: open
- **priority**: medium
- **labels**: enhancement, go, template
- **milestone**: Progetto generato
- **ref**: [audit-2026-08-11.md](audit-2026-08-11.md)

`scaffold init` produce 11 file e **zero** `_test.go`. Il `Makefile` generato ha un target `test` che
esegue `go test ./...` su un progetto senza test: passa sempre, quindi non dice niente. Uno scaffold
insegna le convenzioni del progetto che genera: senza un test di esempio insegna che i test sono
opzionali.

Da fare:

- [ ] `template/cmd/main_test.tmpl` con uno smoke test dell'endpoint `/health` via
      `app.Test(httptest.NewRequest(...))` (Fiber lo supporta nativamente, non serve una porta);
- [ ] `template/env/env_test.tmpl` sui default della `Configuration` (`envDefault` è facile da
      rompere e nessuno se ne accorge);
- [ ] aggiornare l'asserzione esatta di `TestEmbeddedAssetsProduceTheDocumentedLayout` in
      `main_test.go` e l'albero in `README.md` + `docs/index.md`;
- [ ] estendere il job `generated-project-builds` con un `go test ./...` sul generato.

### `config-yml-double-source` — `config/config.yml` non è letto: doppia fonte di configurazione

- **status**: open
- **priority**: low
- **labels**: enhancement, template, documentation
- **milestone**: Progetto generato
- **ref**: [audit-2026-08-11.md](audit-2026-08-11.md)

Il progetto generato legge la configurazione **solo** dall'ambiente (`env/env.go`, `caarlos0/env` +
`godotenv`), ma porta anche un `config/config.yml` che nessuna riga di codice apre. Due fonti di
configurazione di cui una finta: chi genera il progetto modifica `config.yml` e non capisce perché
non cambia nulla. Il `Dockerfile` generato ci dipende (`COPY --from=builder /src/config`), quindi non
si può togliere senza toccarlo.

Opzioni, da decidere:

1. **rimuoverlo** e togliere la `COPY` dal Dockerfile — l'ambiente resta l'unica fonte (più coerente
   con la 12-factor e con quello che il codice fa già);
2. **usarlo davvero** per i parametri non segreti (timeout, `logMode`), leggendolo in `env.go` come
   layer di default sotto le variabili d'ambiente.

Oggi il file esiste con commenti che dichiarano che i valori veri stanno nell'ambiente: è una
soluzione tampone, non una decisione.
