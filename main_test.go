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
		Templates:       templates,
		Static:          static,
		ModuleTemplates: embeddedModules(t),
		Log:             io.Discard,
	}).Generate(dest)
	require.NoError(t, err)

	return dest, res
}

func embeddedModules(t *testing.T) fs.FS {
	t.Helper()
	modules, err := fs.Sub(moduleAssets, "module")
	require.NoError(t, err)
	return modules
}

// moduleFiles is what one bounded context adds: four layers, their tests and
// its wiring file.
func moduleFiles(name string) []string {
	return []string{
		"cmd/module_" + name + ".go",
		"internal/" + name + "/domain/model.go",
		"internal/" + name + "/domain/model_test.go",
		"internal/" + name + "/domain/repository.go",
		"internal/" + name + "/application/service.go",
		"internal/" + name + "/application/service_test.go",
		"internal/" + name + "/infrastructure/gorm.go",
		"internal/" + name + "/infrastructure/memory.go",
		"internal/" + name + "/infrastructure/record.go",
		"internal/" + name + "/infrastructure/store_test.go",
		"internal/" + name + "/transport/http/handler.go",
		"internal/" + name + "/transport/http/handler_test.go",
	}
}

func TestEmbeddedAssetsProduceTheDocumentedLayout(t *testing.T) {
	_, res := generateFromEmbeddedAssets(t)

	assert.ElementsMatch(t, append([]string{
		".dockerignore",
		".gitignore",
		"Dockerfile",
		"Makefile",
		"README.md",
		"cmd/main.go",
		"cmd/main_test.go",
		"config/config.yml",
		"database/db.go",
		"docker-compose.yml",
		"env/.env.local",
		"env/env.go",
		"env/env_test.go",
		"go.mod",
		"cmd/modules.go",
		"internal/archtest/arch_test.go",
	}, moduleFiles("catalog")...), res.Created)
}

func TestAddModuleWithTheEmbeddedAssets(t *testing.T) {
	dest, _ := generateFromEmbeddedAssets(t)
	add := scaffold.New(scaffold.Options{ModuleTemplates: embeddedModules(t), Log: io.Discard})

	res, err := add.AddModule(dest, scaffold.ModuleSpec{Name: "billing", Entity: "InvoiceLine"})
	require.NoError(t, err)
	assert.ElementsMatch(t, moduleFiles("billing"), res.Created)

	model, err := os.ReadFile(filepath.Join(dest, "internal", "billing", "domain", "model.go"))
	require.NoError(t, err)
	assert.Contains(t, string(model), "type InvoiceLine struct")
	assert.Contains(t, string(model), `errors.New("invoice line not found")`)
	gorm, err := os.ReadFile(filepath.Join(dest, "internal", "billing", "infrastructure", "record.go"))
	require.NoError(t, err)
	assert.Contains(t, string(gorm), `return "billing_invoice_line"`)

	// Idempotent like init: a second run skips everything.
	again, err := add.AddModule(dest, scaffold.ModuleSpec{Name: "billing", Entity: "InvoiceLine"})
	require.NoError(t, err)
	assert.Empty(t, again.Created)
	assert.Len(t, again.Skipped, len(res.Created))
}

// Every generated Go file must be valid, gofmt-clean source. This is the cheap
// half of the gate; the CI job additionally runs `go build` on the result.
func TestGeneratedGoFilesAreValidAndFormatted(t *testing.T) {
	dest, res := generateFromEmbeddedAssets(t)
	// A second module with a multi-word entity: names of different lengths are
	// what breaks gofmt alignment in a template.
	added, err := scaffold.New(scaffold.Options{ModuleTemplates: embeddedModules(t), Log: io.Discard}).
		AddModule(dest, scaffold.ModuleSpec{Name: "billing", Entity: "InvoiceLine"})
	require.NoError(t, err)
	res.Created = append(res.Created, added.Created...)

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
	raw := map[string]bool{".dockerignore": true, ".gitignore": true, "Makefile": true, "config/config.yml": true}

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

// The first `make run` loads env/.env.local: it must not point at a database
// that a fresh machine does not have.
func TestLocalDotenvStartsWithoutADatabase(t *testing.T) {
	dest, _ := generateFromEmbeddedAssets(t)

	dotenv, err := os.ReadFile(filepath.Join(dest, "env", ".env.local"))
	require.NoError(t, err)

	assert.Regexp(t, `(?m)^DB_HOST=$`, string(dotenv))
	assert.Regexp(t, `(?m)^CORS_ALLOWED_ORIGINS=$`, string(dotenv))
	assert.NotContains(t, string(dotenv), "API_SECRET", "unused settings are gone")
	assert.NotContains(t, string(dotenv), "RUNNING_MODE", "unused settings are gone")
}

// The generated .gitignore keeps the build output and the dotenv credentials out
// of the first `git add .`.
func TestGeneratedGitignoreCoversSecretsAndBuildOutput(t *testing.T) {
	dest, _ := generateFromEmbeddedAssets(t)

	gitignore, err := os.ReadFile(filepath.Join(dest, ".gitignore"))
	require.NoError(t, err)

	for _, pattern := range []string{"bin/", "env/.env.*", "!env/.env.example"} {
		assert.Contains(t, string(gitignore), pattern+"\n")
	}
}
