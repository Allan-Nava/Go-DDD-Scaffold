# AGENTS.md — Go-DDD-Scaffold

Generatore CLI (`github.com/Allan-Nava/Go-DDD-Scaffold`) che crea lo scheletro di un progetto Go a layout Domain Driven Design: CLI `scaffold init` (`main.go` + `scaffold/`), template del progetto generato (`template/`, `static/`), estensione VSCode gemella (`extensions/vscode/`), docs Jekyll su GitHub Pages (`docs/`), CI GitHub Actions (`.github/workflows/`).

Questo file definisce le regole operative per gli agent (Copilot, Claude, altri tool AI) quando lavorano in questo repository.

## Regole di lavoro (SEMPRE)

- **Il generatore e l'estensione VSCode sono gemelli**: ogni modifica ai template va propagata su **entrambi** i set: `template/` + `static/` (CLI) e `extensions/vscode/src/templates/*.tpl` (estensione). Sono gia in drift (l'estensione ha `config/`, `utils/api_messages` che la CLI non ha): quando tocchi uno dei due, allinea o annota esplicitamente il drift. Mai fixare solo un lato in silenzio.
- **Un template che genera codice non compilabile e un bug bloccante.** Il gate e: `scaffold init` in una dir vuota -> `go mod tidy && go build ./... && go vet ./...` del progetto generato deve passare. Verificare **sempre** cosi, non a lettura: l'output di `scaffold init` e codice Go, non testo.
- **Ogni release = tag `vX.Y.Z`**: `minor` per novita sostanziali (nuovi template/comandi/feature, rimozioni), `patch` per fix/bump dipendenze. La versione vive in **3 posti** da tenere allineati: tag git, `Version` in `main.go`, `version` in `extensions/vscode/package.json`. Il tag scatena `release.yml` (binari multi-arch) e `vscode-publish.yml` (Open VSX + Marketplace): **un tag pubblica sui marketplace pubblici**, quindi taggare solo su richiesta esplicita.
- **MAI `git push`**: lo fa sempre l'utente. MAI `Co-Authored-By` nei commit.
- **Niente binari nel repo**: gli artefatti li produce la CI (`release.yml`). Se trovi un eseguibile committato, va rimosso e aggiunto a `.gitignore`, non aggiornato.
- **Zero dipendenze runtime dall'ambiente**: template e static sono in `embed.FS`, compilati nel binario. Non reintrodurre lookup su `$GOPATH`, path assoluti o file letti da disco a runtime: un `go install` del binario non ha il repo accanto.
- **Documentare le modifiche fattuali**: `README.md` (albero del progetto generato, installazione), `docs/` per le pagine Pages. Ogni cambio del layout generato va riflesso nell'albero ASCII del README.

## Pattern per modifiche al generatore (validato)

1. **Baseline**: `go build ./... && go vet ./... && go test ./...` + `scaffold init` in una tempdir e `go build ./...` sul generato. Salvare l'output com'e *prima*: e il riferimento.
2. **Modifica** al generatore o ai template, un layer per volta (mai generatore + template + CI nello stesso passo).
3. **Test di generazione end-to-end**: dir vuota -> `scaffold init` -> `go mod tidy && go build ./... && go vet ./...`. Un `TestGenerate` che verifica solo "nessun errore" non basta: deve asserire i **file attesi** e il **contenuto sostituito** (`{{.ProjectName}}` risolto, niente `TODO/`).
4. **Idempotenza**: rilanciare `scaffold init` sulla stessa dir non deve distruggere file esistenti (default: skip + warning; sovrascrittura solo con `--force`).
5. **Chiusura**: allineare estensione VSCode, README, versione nei 3 posti, `.gitignore`; poi commit.

## Trappole note / regole tecniche

- **`text/template`, MAI `html/template`**: `html/template` fa escaping HTML sul **codice Go** (`&&` -> `&amp;&amp;`, `<-` -> `&lt;-`) e produce sorgenti rotti. Era il motivo dell'helper `unescaped` e del campo `data.Quit: "<-quit"`: workaround di un bug, non feature. Se serve un `<-` in un template, il fix e `text/template`, non un altro escape hatch.
- **`$GOPATH` non esiste piu come convenzione**: la variabile e vuota su qualsiasi setup moderno (`go env GOPATH` risponde comunque `~/go`). Il vecchio `panic("cannot find $GOPATH")` nell'`init()` del package rendeva la CLI inutilizzabile fuori dal GOPATH, e i test passavano solo perche il repo era clonato dentro `~/go/src/`. Non dedurre path dal GOPATH: il target di generazione e **`os.Getwd()`**, non `filepath.Dir(os.Args[0])` (quello e la dir del binario: bug storico, i file finivano accanto all'eseguibile).
- **`defer f.Close()` dentro un callback di walk e un leak**: i defer si accumulano fino alla fine dell'intera camminata, non per iterazione. Chiudere esplicitamente (o isolare in una funzione dedicata) in ogni loop su file.
- **Permessi**: `0o755` per le dir, `0o644` per i file. Mai `os.ModePerm` (0777).
- **`filepath.WalkDir` batte `filepath.Walk`** (niente `lstat` per entry); su `embed.FS` usare `fs.WalkDir`. E il valore di ritorno del walk **va controllato**: scartarlo nasconde errori di I/O.
- **File senza estensione `.tmpl` in `template/`** (`go.mod`, `Dockerfile`, `README.md`, `env/.env.local`) sono comunque passati dal motore template: possono usare `{{.ProjectName}}`, ma un `{{` letterale li rompe. `static/` invece e copia byte-per-byte, senza rendering.
- **Il `go.mod` generato ha `module TODO/{{.ProjectName}}`**: il prefisso `TODO/` e un placeholder che l'utente deve sostituire, non un module path valido per `go get`. Segnalarlo nel README/output, non silenziarlo.
- **Dockerfile: `go mod tidy` in build e un antipattern**: richiede rete, muta `go.mod`/`go.sum` e invalida la cache dei layer. Usare `COPY go.mod go.sum ./` + `go mod download` come layer separato **prima** di `COPY . .`, e uno stage finale `scratch`/`alpine` (immagine finale che parte da `golang:` = ~1 GB inutili).
- **La versione Go va tenuta in sync in 4 posti**: `go.mod`, `Dockerfile`, `template/Dockerfile`, matrice `go-version` dei workflow. Il drift storico (`go.mod` 1.19 vs `Dockerfile` 1.18) rompeva la build Docker senza rompere la CI.
- **`dependabot.yml` punta a `/tests`** (directory inesistente; quella vera e `test/` e non ha un `go.mod` proprio): la voce e morta, non aggiungerne di simili.
- `test/a_main_test.go` e solo lo scheletro `TestMain` + helper: i test veri stanno accanto al codice (`scaffold/scaffold_test.go`). Nei test usare `t.TempDir()` (auto-cleanup), non `ioutil.TempDir` + `defer os.RemoveAll`.
- **`echo ::set-output`** nei workflow e deprecato da GitHub (`docker-publish.yml`): usare `$GITHUB_OUTPUT`.

## Puntatori

- CLI: `main.go` (comandi urfave/cli) - generatore: `scaffold/scaffold.go` - test: `scaffold/scaffold_test.go`
- Template del progetto generato: `template/` (rendering) + `static/` (copia raw) - gemelli VSCode: `extensions/vscode/src/templates/`
- CI: `.github/workflows/` - `go.yml`/`go-test.yml` (build+test), `release.yml` (binari multi-arch su release), `go-release.yml` (tarball via `build.sh`), `docker-publish.yml` (ghcr.io), `vscode-publish.yml` (marketplace su tag), `jekyll-gh-pages.yml` (docs)
- Audit tecnico e stato dei fix: `docs/audit-2026-08-11.md`
- Stack del progetto generato: Fiber v2, GORM (MySQL), zap, `caarlos0/env`, godotenv, `go-playground/validator`
