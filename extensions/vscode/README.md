# GO Domain Driven Design Scaffold

Generate a Go [Domain Driven Design project layout](https://github.com/Allan-Nava/Go-DDD-Scaffold)
from inside VS Code.

The extension is a **wrapper around the `scaffold` CLI**: it does not carry its own
copy of the templates. That is deliberate — the templates live in exactly one
place (embedded in the CLI binary), so the editor and the command line can never
generate different projects.

## Requirements

The `scaffold` CLI:

```sh
go install github.com/Allan-Nava/Go-DDD-Scaffold@latest
```

The extension looks for it on the `PATH` and then in the Go bin directories
(`$GOBIN`, `$GOPATH/bin`, `~/go/bin`) — that last part matters because a GUI
editor does not read your shell profile, so a CLI that works in the terminal can
still look missing from the editor. `go install` names the binary after the
module (`Go-DDD-Scaffold`), so both that name and `scaffold` are recognised.

If it is not found, the extension offers to run the `go install` above in a
terminal.

A CLI left over from before the extension was rewritten is **rejected** rather
than used: that build ignores the folder you pick and writes next to its own
executable. Its version number cannot be trusted for this (it was hardcoded to
`v1.0.1`), so the check probes for the `init --force` flag. If you see "the
scaffold CLI found on this machine is too old", run the `go install` again.

## Usage

- **Command palette**: `Go DDD: New Scaffold Project`
- **Explorer**: right-click a folder → `New Scaffold Project`

The project is generated in the folder you pick (the right-clicked one, the
workspace folder, or one chosen from a dialog).

If the folder is not empty you are asked what to do:

| Choice | Effect |
| --- | --- |
| Keep existing files | generates only what is missing |
| Overwrite existing files | passes `--force`, replacing `cmd/main.go`, `go.mod`, … |

Nothing is ever overwritten without that explicit answer.

Then, in the generated project:

```sh
go mod tidy
make run                        # listens on :8080
curl localhost:8080/health      # 200
```

## Extension Settings

| Setting | Default | Description |
| --- | --- | --- |
| `goDddScaffold.binaryPath` | `""` | Path to the `scaffold` CLI. Empty means "look it up on the `PATH`, then in the Go bin directories". |

## Output

Every run appends the CLI's per-file output to the **Go DDD Scaffold** output
channel, so a generation can be inspected after the notification is gone.

## Releasing

The extension has its own tag namespace, separate from the repository's `vX.Y.Z`
tags (which exist on every commit and only publish the Docker image):

```sh
# 1. bump "version" in package.json, commit
# 2. tag with the SAME version, prefixed
git tag -a vscode-v0.1.0 -m "..." && git push --tags
```

`vscode-publish.yml` verifies the tag matches `package.json` and fails loudly if
they diverge, then lints, tests, packages and publishes. The publish steps are
skipped — not failed — when the `VSCE_PAT` / `OVSX_PAT` secrets are absent.

## Known limitations

- Publishing needs the `VSCE_PAT` repository secret (and `OVSX_PAT` for Open VSX);
  without them the workflow packages the `.vsix` as a build artifact and skips the
  upload. See the repository backlog.
- There are no integration tests running in a real editor host: the tested part
  is the CLI wrapper (binary discovery, argv, output parsing), which is where the
  logic lives.

## Development

```sh
npm install
npm run compile   # tsc, strict
npm run lint      # eslint
npm test          # unit tests, no VS Code host needed
npm run webpack   # dev bundle into dist/
```

## Release Notes

See [CHANGELOG.md](CHANGELOG.md).
