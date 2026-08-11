package main

import (
	"bytes"
	"go/format"
	"go/parser"
	"go/token"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Allan-Nava/Go-DDD-Scaffold/scaffold"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// generateFromEmbeddedAssets runs the real assets shipped in the binary into a
// throwaway directory whose path yields a predictable module path.
func generateFromEmbeddedAssets(t *testing.T) (dest string, res scaffold.Result) {
	t.Helper()

	dest = filepath.Join(t.TempDir(), "github.com", "acme", "myservice")
	require.NoError(t, os.MkdirAll(dest, 0o755))

	templates, err := fs.Sub(templateAssets, "template")
	require.NoError(t, err)
	static, err := fs.Sub(staticAssets, "static")
	require.NoError(t, err)

	res, err = scaffold.New(scaffold.Options{
		Templates: templates,
		Static:    static,
		Log:       io.Discard,
	}).Generate(dest)
	require.NoError(t, err)

	return dest, res
}

func TestEmbeddedAssetsProduceTheDocumentedLayout(t *testing.T) {
	_, res := generateFromEmbeddedAssets(t)

	assert.ElementsMatch(t, []string{
		".dockerignore",
		"Dockerfile",
		"Makefile",
		"README.md",
		"cmd/main.go",
		"config/config.yml",
		"database/db.go",
		"docker-compose.yml",
		"env/.env.local",
		"env/env.go",
		"go.mod",
	}, res.Created)
}

// Every generated Go file must be valid, gofmt-clean source. This is the cheap
// half of the gate; the CI job additionally runs `go build` on the result.
func TestGeneratedGoFilesAreValidAndFormatted(t *testing.T) {
	dest, res := generateFromEmbeddedAssets(t)

	var checked int
	for _, rel := range res.Created {
		if !strings.HasSuffix(rel, ".go") {
			continue
		}
		checked++

		src, err := os.ReadFile(filepath.Join(dest, filepath.FromSlash(rel)))
		require.NoError(t, err)

		if _, err := parser.ParseFile(token.NewFileSet(), rel, src, parser.AllErrors); err != nil {
			t.Errorf("%s does not parse as Go: %v", rel, err)
			continue
		}

		formatted, err := format.Source(src)
		require.NoError(t, err)
		assert.Equal(t, string(formatted), string(src), "%s is not gofmt-clean", rel)
	}
	assert.NotZero(t, checked, "no Go file was generated")
}

// A leftover "{{" means a placeholder was never substituted.
func TestNoUnrenderedPlaceholdersInRenderedFiles(t *testing.T) {
	dest, res := generateFromEmbeddedAssets(t)

	// static/ is copied verbatim, so a literal "{{" there is legitimate.
	raw := map[string]bool{".dockerignore": true, "Makefile": true, "config/config.yml": true}

	for _, rel := range res.Created {
		if raw[rel] {
			continue
		}
		src, err := os.ReadFile(filepath.Join(dest, filepath.FromSlash(rel)))
		require.NoError(t, err)
		assert.False(t, bytes.Contains(src, []byte("{{")), "%s still contains a template placeholder", rel)
	}
}

func TestGeneratedModulePathMatchesTheDestination(t *testing.T) {
	dest, _ := generateFromEmbeddedAssets(t)

	gomod, err := os.ReadFile(filepath.Join(dest, "go.mod"))
	require.NoError(t, err)

	assert.Contains(t, string(gomod), "module github.com/acme/myservice")
	assert.NotContains(t, string(gomod), "TODO/", "the module path placeholder is gone")
}
