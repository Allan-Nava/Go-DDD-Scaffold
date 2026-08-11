#!/bin/sh
# Builds the scaffold CLI and packages it for a release.
#
# CONTRACT: stdout carries ONLY the path of the built binary; every progress
# message goes to stderr. release.yml runs this through go-release.action, whose
# build.sh does `OUTPUT=$(./build.sh "$CMD_PATH")` and archives whatever it
# printed — a stray log line there ends up in the release archive as a filename.
set -eu

VERSION="${VERSION:-dev}"
GOOS="${GOOS:-linux}"
GOARCH="${GOARCH:-amd64}"
OUT_DIR="${OUT_DIR:-/tmp/bin}"

# Windows needs the .exe suffix or the artifact is not executable.
EXT=""
[ "${GOOS}" = "windows" ] && EXT=".exe"
BIN="bin/scaffold${EXT}"

echo "building scaffold ${VERSION} for ${GOOS}/${GOARCH}" >&2

# -trimpath keeps absolute build paths out of the binary; -s -w drops the
# symbol table and DWARF data. The output goes under bin/: "-o scaffold" would
# resolve to the existing scaffold/ package directory and land inside it.
CGO_ENABLED=0 GOOS="${GOOS}" GOARCH="${GOARCH}" go build \
	-trimpath \
	-ldflags="-s -w -X main.version=${VERSION}" \
	-o "${BIN}" . >&2

mkdir -p "${OUT_DIR}"
tar -czf "${OUT_DIR}/go-scaffold.tar.gz" -C bin "scaffold${EXT}" >&2
ls -l "${OUT_DIR}/go-scaffold.tar.gz" >&2

# The one line on stdout: what the caller should archive.
echo "${BIN}"
