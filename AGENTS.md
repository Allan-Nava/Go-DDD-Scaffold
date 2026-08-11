# AGENTS.md - Go-DDD-Scaffold

Generatore CLI (`github.com/Allan-Nava/Go-DDD-Scaffold`) che crea lo scheletro di un progetto Go a layout Domain Driven Design: CLI `scaffold init` (`main.go` + `scaffold/`), template del progetto generato (`template/`, `static/`, **embeddati nel binario**), estensione VSCode gemella (`extensions/vscode/`), docs Jekyll su GitHub Pages (`docs/`), CI GitHub Actions (`.github/workflows/`).


Questo file definisce le regole operative per gli agent (Copilot, Claude, altri tool AI) quando lavorano in questo repository.

## Regole di lavoro (SEMPRE)

- **Un template che genera codice non compilabile e un bug bloccante.** Il gate e **`make e2e`**: genera in una tempdir e lancia `go mod tidy && go build ./... && go vet ./... && gofmt -l`. Verificare **sempre** cosi, non a lettura: l'output di `scaffold init` e codice Go, non testo. `make check && make test` non basta - i test unitari controllano sintassi e formattazione, non la compilazione con le dipendenze reali.
- **Un solo set di template, ed e quello della CLI.** L'estensione VSCode e un **wrapper** che invoca il binario `scaffold`: non ha template propri e non deve tornare ad averne. Ne aveva un secondo set (`extensions/vscode/src/templates/*.tpl` + `scaffold.template.ts`), mai letto da alcun codice e gia in drift dal primo - `database/db.tpl` era un duplicato identico di `config/config.tpl`. Rimosso nella riscrittura: se qualcuno propone di "bundlare i template nell'estensione", quella e la regressione da rifiutare. Il contratto fra i due e solo l'output della CLI (righe `Create`/`Skip`, `--version`, exit code): se lo cambi, aggiorna `extensions/vscode/src/scaffold/cli.ts` e i suoi test.
- **Ogni release = tag `vX.Y.Z`**: `minor` per novita sostanziali (nuovi template/comandi/feature, rimozioni), `patch` per fix/bump dipendenze. La versione del binario arriva da `-ldflags "-X main.version=..."` (il workflow la prende da `github.ref_name`): **non** hardcodarla in `main.go`. Resta da allineare a mano `version` in `extensions/vscode/package.json`. Il tag scatena `release.yml` (binari multi-arch) e `vscode-publish.yml` (Open VSX + Marketplace) - **un tag pubblica sui marketplace pubblici**, quindi taggare solo su richiesta esplicita.
- **Todo -> `docs/backlog.md`** (sorgente unica, item con `id` stabile, sync idempotente a issue GitHub). Non sparpagliare TODO nel codice o nei doc. Le **milestone sono dinamiche**: non esiste un elenco, esistono solo i titoli scritti negli item - il sync crea su GitHub quelle mancanti. ATTENZIONE: Se cambi `docs/backlog.md` **rigenera `docs/roadmap.md`** (`make roadmap`) e committala, altrimenti il gate `generated-pages-check` fallisce. Item di hardening/cleanup = voce nel backlog, non proporli come "next".
- **MAI `git push`** - lo fa sempre l'utente. MAI `Co-Authored-By` nei commit.
- **Niente binari nel repo**: gli artefatti li produce la CI. Se trovi un eseguibile committato, va rimosso e aggiunto a `.gitignore`, non aggiornato (c'era `scaffold-cli-macos`, 4,8 MB).
- **Zero dipendenze runtime dall'ambiente**: `template/` e `static/` sono in `embed.FS` (dichiarati in `main.go`, iniettati come `fs.FS` in `scaffold.Options`). Non reintrodurre lookup su `$GOPATH`, path assoluti o file letti da disco a runtime - un `go install` del binario non ha il repo accanto.
- **Documentare le modifiche fattuali**: `README.md` e `docs/index.md` (albero del progetto generato, installazione, flag) vanno riflessi a ogni cambio del layout. Interventi non banali -> doc `.md` in `docs/` con schema ASCII, come `docs/audit-2026-08-11.md`.

## Pattern per modifiche al generatore (validato)

1. **Baseline**: `make check && make test && make e2e`. Salvare l'output com'e *prima*: e il riferimento.
2. **Modifica** al generatore o ai template, un layer per volta (mai generatore + template + CI nello stesso passo).
3. **Gate**: `make e2e`. Se cambia il set di file generati, aggiornare l'asserzione in `main_test.go` (`TestEmbeddedAssetsProduceTheDocumentedLayout`) - e volutamente esatta, non "contiene".
4. **Idempotenza**: rilanciare `scaffold init` non deve distruggere file esistenti (default: skip via `O_EXCL`; sovrascrittura solo con `--force`).
5. **Smoke funzionale** se hai toccato `template/cmd/` o `template/database/`: avviare il servizio generato e verificare `GET /health` + shutdown su `SIGTERM`, non solo la compilazione.
6. **Chiusura**: allineare estensione VSCode, README, `docs/`, versione dell'estensione, `.gitignore`; poi commit.

## Trappole note / regole tecniche

- **`text/template`, MAI `html/template`**: `html/template` fa escaping HTML sul **codice Go** (`&&` -> `&amp;&amp;`, `<-` -> `&lt;-`) e produce sorgenti rotti. Era il motivo dell'helper `unescaped` e del campo `data.Quit = "<-quit"`: workaround di un bug, non feature. Entrambi rimossi; `TestGenerateDoesNotEscapeGoOperators` e la regressione.
- **`template/` non puo contenere un `go.mod`**: renderebbe la directory un modulo separato e `//go:embed all:template` fallirebbe con *"cannot embed directory template: in different module"*. Per questo il template si chiama `go.mod.tpl`.
- **Due suffissi, due regole**: `.tmpl` -> `.go` (`cmd/main.tmpl` -> `cmd/main.go`); `.tpl` -> perde il suffisso (`go.mod.tpl` -> `go.mod`). File senza suffisso in `template/` (`Dockerfile`, `README.md`, `env/.env.local`) sono comunque passati dal motore template: possono usare `{{.ProjectName}}`, ma un `{{` letterale li rompe. `static/` invece e copia byte-per-byte, senza rendering - e li che va un file con `{{` letterali.
- **`embed` e i dot-file**: serve il prefisso `all:` (`//go:embed all:template`), altrimenti `env/.env.local` e `.dockerignore` non entrano nel binario.
- **`$GOPATH` non esiste piu come convenzione**: la variabile e vuota su qualsiasi setup moderno (`go env GOPATH` risponde comunque `~/go`). Il vecchio `panic("cannot find $GOPATH")` nell'`init()` del package rendeva la CLI inutilizzabile fuori dal GOPATH - e i test passavano solo perche il repo era clonato dentro `~/go/src/`. Il target di generazione e **`os.Getwd()`**, non `filepath.Dir(os.Args[0])` (quello e la dir del binario - bug storico: i file finivano accanto all'eseguibile). Il module path si deduce dalla dir con `scaffold.ModulePath` (primo segmento host-like), non dal GOPATH.
- **`go build -o scaffold` scrive DENTRO `scaffold/`**: Go interpreta `-o <directory-esistente>` come "metti il binario li", producendo `scaffold/Go-DDD-Scaffold`. Buildare sempre in `bin/`. Per lo stesso motivo **mai** una riga `scaffold` in `.gitignore`/`.dockerignore`: escluderebbe il package.
- **`defer f.Close()` dentro un callback di walk e un leak**: i defer si accumulano fino alla fine dell'intera camminata, non per iterazione. Chiudere per file (o in una closure dedicata, come fa `copyStatic`).
- **Permessi**: `0o755` per le dir, `0o644` per i file. Mai `os.ModePerm` (0777).
- **`fs.WalkDir` / `filepath.WalkDir`**, non `Walk` (niente `lstat` per entry). E il valore di ritorno del walk **va controllato**: scartarlo nasconde errori di I/O (era il caso di `getTemplateSets`).
- **Mai il DSN nei log o negli errori**: contiene la password. Logare host/porta/nome database.
- **La versione Go va tenuta in sync in 4 posti**: `go.mod`, `Dockerfile`, `template/Dockerfile`, matrice `go-version` in `go.yml`. Il drift storico (`go.mod` 1.19 vs `Dockerfile` 1.18 vs `template/Dockerfile` 1.20) rompeva la build Docker senza rompere la CI.
- **Dockerfile: `go mod tidy` in build e un antipattern** - richiede rete, muta `go.mod`/`go.sum` e invalida la cache dei layer. `COPY go.mod go.sum ./` + `go mod download` come layer separato **prima** di `COPY . .`, e stage finale `alpine` (l'immagine single-stage `golang:*` pesava ~1 GB; ora 12,2 MB).
- **La lista issue di GitHub e eventualmente consistente**: subito dopo una `POST /issues`, un `GET /issues?labels=...` puo restituire uno stato di qualche secondo prima. Due run di `sync-backlog-to-issues.py` a distanza di secondi hanno creato 2 doppioni proprio cosi (#102, #103 - poi chiusi). Non "risolvere" con una sleep: la rete di sicurezza e il **dedup per `backlog-id`** nel piano (tiene la issue col numero piu basso, chiude le altre), che rende il sync autoriparante. Vale per qualsiasi cosa legga le issue subito dopo averle scritte.
- **In un workflow `run: cd <dir>` e un no-op**: ogni step apre una shell nuova. Usare `working-directory` (o `defaults.run.working-directory`).
- **`echo ::set-output` e disattivato** da GitHub: per i tag/label Docker usare `docker/metadata-action`.
- Nei test usare `t.TempDir()` (auto-cleanup) e `fstest.MapFS` per gli asset: i test del package `scaffold` non devono dipendere da `template/` reale - quelli end-to-end stanno in `main_test.go`, dove vive l'`embed`.

## Puntatori

- CLI: `main.go` (comandi urfave/cli + direttive `embed`) - generatore: `scaffold/scaffold.go` - test unitari: `scaffold/scaffold_test.go` - test end-to-end sugli asset reali: `main_test.go`
- Template del progetto generato: `template/` (rendering) + `static/` (copia raw) - gemelli VSCode: `extensions/vscode/src/templates/`
- Dev loop: `Makefile` - `check`, `test`, `e2e`, `build`, `docker`, `backlog-lint`, `roadmap`, `backlog-sync[-apply]`
- **Backlog operativo**: `docs/backlog.md` (sorgente unica) - **Roadmap per milestone**: `docs/roadmap.md` (**generata**, non editarla) - tooling: `docs/scripts/` - `lib/backlog.py` (parsing + regole, fonte unica), `backlog-lint.py`, `generate-roadmap.py`, `sync-backlog-to-issues.py` (issue GitHub idempotenti, milestone create al volo). Il sync tocca **solo** le issue con label `backlog-sync`: Renovate/Dependabot e le segnalazioni di terzi non vengono mai lette ne chiuse.
- CI: `.github/workflows/` - `go.yml` (gofmt+vet+test matrice 1.22-1.24, **+ job che compila il progetto generato e ne fa lo smoke test**), `release.yml` (binari multi-arch su release), `go-release.yml` (tarball via `build.sh`), `docker-publish.yml` (ghcr.io, multi-arch, cache GHA), `vscode-publish.yml` (marketplace su tag), `jekyll-gh-pages.yml` (docs)
- **Audit tecnico, correzioni e debito residuo**: `docs/audit-2026-08-11.md`
- **Estensione VSCode** (`extensions/vscode/`): riscritta come wrapper della CLI. `src/scaffold/cli.ts` e **puro** - non importa `vscode` - ed e li che vive la logica (scoperta del binario, argv, parsing dell'output); i suoi test girano con `node --test`, senza host editor (`npm test`). L'API dell'editor sta solo in `extension.ts` e `commands/init.ts`. Zero dipendenze runtime: non aggiungerne per comodita. `go install` produce un binario chiamato **`Go-DDD-Scaffold`** (dal nome del modulo), non `scaffold`: la scoperta cerca entrambi, e cerca anche in `$GOBIN`/`$GOPATH/bin`/`~/go/bin` perche un editor GUI non legge il profilo della shell. Un programma omonimo che non risponde `scaffold version` va scartato, non eseguito. Resta aperto `publisher: "None"` (item `vscode-extension-publish-metadata`).
- **`node --test <directory>` non funziona** su tutte le versioni di Node (su 23.3 da `MODULE_NOT_FOUND` sulla dir): usare il glob `node --test "out/**/*.test.js"`.
- Stack del progetto generato: Fiber v2, GORM (MySQL), zap, `caarlos0/env/v11`, godotenv

## graphify

Questo repo **non ha ancora** un knowledge graph. Se serve navigazione strutturale (e un repo piccolo: ~700 righe Go + estensione TS), generarlo con `graphify build .` e poi:

- `graphify query "<domanda>"` per domande sul codice, `graphify path "<A>" "<B>"` per relazioni, `graphify explain "<concetto>"` per concetti puntuali - ritornano un sottografo scoped, molto piu piccolo di `GRAPH_REPORT.md` o del grep grezzo.
- `graphify-out/wiki/index.md` per la navigazione ampia, `GRAPH_REPORT.md` solo per review architetturale.
- Dopo modifiche al codice: `graphify update .` (solo AST, nessun costo API).
