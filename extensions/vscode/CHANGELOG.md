# Change Log

All notable changes to the "go-ddd-scaffold" extension will be documented in this file.

Check [Keep a Changelog](http://keepachangelog.com/) for recommendations on how to structure this file.

## [0.3.0]

### Added

- **Go DDD: Add Bounded Context** (command palette and explorer context menu):
  asks for the module name and, optionally, the aggregate type, then runs
  `scaffold add module`. It finds the project root by walking up to
  `cmd/modules.go`. Needs CLI v1.0.0 or later.

## [0.2.0]

### Added

- The command asks for the **Go module path** before generating. Leave it empty to
  keep the CLI's inference (the git `origin` remote, else a known host in the
  folder path, else the folder name); a value is passed as `scaffold init --module`.
  Needs CLI v0.9.0 or later.

## [0.1.0]

### Changed

- **Rewritten as a wrapper around the `scaffold` CLI.** The previous version could
  not generate anything: the command computed a `…_bloc.dart` target path (Dart,
  from the Flutter/BLoC generator it was copied from) and its `writeFile` call was
  commented out. The extension now shells out to the CLI, so the templates have a
  single source of truth.
- Command id `extension.new-scaffold` → `go-ddd-scaffold.init`, title
  `Go DDD: New Scaffold Project`. It works from the command palette and from the
  explorer context menu on a folder.
- The destination is the folder you pick — the right-clicked one, the workspace
  folder, or one chosen from a dialog.
- A non-empty folder asks before writing: existing files are kept unless you
  explicitly choose to overwrite them (`--force`).

### Added

- A guard against a **pre-rewrite CLI**. Verified in a real editor host: a
  `go install` from before the rewrite leaves a binary that ignores the folder you
  pick and writes next to its own executable, so the command appeared to do
  nothing. Its version cannot be used to detect this — that build hardcoded
  `v1.0.1`, *higher* than the versions that fixed it — so the extension probes for
  the `init --force` flag instead and, when it is missing, says so and offers to
  update the CLI.
- `goDddScaffold.binaryPath` setting, and automatic discovery of the CLI on the
  `PATH` and in `$GOBIN` / `$GOPATH/bin` / `~/go/bin` (a GUI editor does not read
  the shell profile). Both `scaffold` and the `go install` name
  `Go-DDD-Scaffold` are recognised, and a same-named program that is not this CLI
  is skipped rather than run.
- An offer to run `go install github.com/Allan-Nava/Go-DDD-Scaffold@latest` in a
  terminal when the CLI is missing.
- A "Go DDD Scaffold" output channel with the CLI's per-file output.
- Unit tests for the wrapper (binary discovery, argv, output parsing) that run
  without a VS Code host: `npm test`.

### Removed

- `src/templates/` — a second set of templates, never read by any code path,
  which had drifted from the CLI's (`database/db.tpl` was a verbatim duplicate of
  `config/config.tpl`). The CLI's embedded templates are now the only ones.
- `src/utils/get-selected-text.ts`, dead Dart-specific code, and the
  `go-ddd-scaffold.helloWorld` placeholder command.
- Runtime dependencies `change-case`, `lodash`, `mkdirp`, `node-fetch` and
  `semver`: the wrapper needs none of them. The extension now ships with zero
  runtime dependencies.
- The broken `assets/logo.png` icon reference (the directory does not exist).

### Fixed

- `strict` is on in `tsconfig.json`, and the `uri` the context menu may or may not
  pass is now typed as optional.
