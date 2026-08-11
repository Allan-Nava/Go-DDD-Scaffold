#!/bin/sh
# Builds the scaffold CLI and packages it for a release.
set -eu

VERSION="${VERSION:-dev}"
GOOS="${GOOS:-linux}"
GOARCH="${GOARCH:-amd64}"
OUT_DIR="${OUT_DIR:-/tmp/bin}"

echo "building scaffold ${VERSION} for ${GOOS}/${GOARCH}"

# The output goes under bin/: "-o scaffold" would resolve to the existing
# scaffold/ package directory and drop the binary inside it.
# -trimpath keeps absolute build paths out of the binary; -s -w drops the
# symbol table and DWARF data.
CGO_ENABLED=0 GOOS="${GOOS}" GOARCH="${GOARCH}" go build \
	-trimpath \
	-ldflags="-s -w -X main.version=${VERSION}" \
	-o bin/scaffold .

mkdir -p "${OUT_DIR}"
tar -czf "${OUT_DIR}/go-scaffold.tar.gz" -C bin scaffold

ls -l "${OUT_DIR}/go-scaffold.tar.gz"
echo "built"
