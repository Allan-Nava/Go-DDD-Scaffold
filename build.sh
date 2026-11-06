#!/bin/sh
# Builds the scaffold CLI for one GOOS/GOARCH and packages it for a release.
#
#   VERSION=v0.9.0 GOOS=darwin GOARCH=arm64 sh build.sh
#
# Output: dist/scaffold_<version>_<goos>_<goarch>.tar.gz (.zip on windows), with
# the binary, LICENSE and README.md, plus a .sha256 next to it. stdout carries
# only the archive path; progress goes to stderr. release.yml calls it once per
# matrix entry.
set -eu

VERSION="${VERSION:-dev}"
GOOS="${GOOS:-$(go env GOOS)}"
GOARCH="${GOARCH:-$(go env GOARCH)}"
OUT_DIR="${OUT_DIR:-dist}"

# Windows needs the .exe suffix or the binary is not executable.
EXT=""
[ "${GOOS}" = "windows" ] && EXT=".exe"

NAME="scaffold_${VERSION}_${GOOS}_${GOARCH}"
STAGE="${OUT_DIR}/${NAME}"

echo "building scaffold ${VERSION} for ${GOOS}/${GOARCH}" >&2
rm -rf "${STAGE}"
mkdir -p "${STAGE}"

# -trimpath keeps absolute build paths out of the binary; -s -w drops the
# symbol table and DWARF data. Never "-o scaffold": it would resolve to the
# existing scaffold/ package directory and land inside it.
CGO_ENABLED=0 GOOS="${GOOS}" GOARCH="${GOARCH}" go build \
	-trimpath \
	-ldflags="-s -w -X main.version=${VERSION}" \
	-o "${STAGE}/scaffold${EXT}" . >&2
cp LICENSE README.md "${STAGE}/"

if [ "${GOOS}" = "windows" ]; then
	ARCHIVE="${OUT_DIR}/${NAME}.zip"
	# zip is not everywhere (WSL, slim images); python3's zipfile is the fallback.
	if command -v zip >/dev/null 2>&1; then
		(cd "${OUT_DIR}" && rm -f "${NAME}.zip" && zip -qr "${NAME}.zip" "${NAME}") >&2
	else
		(cd "${OUT_DIR}" && rm -f "${NAME}.zip" && python3 -m zipfile -c "${NAME}.zip" "${NAME}") >&2
	fi
else
	ARCHIVE="${OUT_DIR}/${NAME}.tar.gz"
	tar -czf "${ARCHIVE}" -C "${OUT_DIR}" "${NAME}" >&2
fi
rm -rf "${STAGE}"

(cd "${OUT_DIR}" && sha256sum "$(basename "${ARCHIVE}")" > "$(basename "${ARCHIVE}").sha256") >&2
ls -l "${ARCHIVE}" >&2

echo "${ARCHIVE}"
