package scaffold

import (
	"io"
	"os"
	"path/filepath"
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fixture is a two-tree asset set covering every mapping rule the generator
// implements, plus the characters html/template used to mangle.
func fixture() (templates, static fstest.MapFS) {
	templates = fstest.MapFS{
		"cmd/main.tmpl":  {Data: []byte("package main // {{.ProjectName}}\nvar quit = <-c && true\n")},
		"go.mod.tpl":     {Data: []byte("module {{.ProjectPath}}\n")},
		"Dockerfile":     {Data: []byte("CMD [\"./{{.ProjectName}}\"]\n")},
		"env/.env.local": {Data: []byte("APP_ENV=local\n")},
	}
	static = fstest.MapFS{
		"Makefile":          {Data: []byte("run:\n\tgo run cmd/main.go\n")},
		"config/config.yml": {Data: []byte("name: {{.ProjectName}}\n")},
	}
	return templates, static
}

func generate(t *testing.T, dest string, force bool) Result {
	t.Helper()
	templates, static := fixture()
	res, err := New(Options{
		Templates: templates,
		Static:    static,
		Force:     force,
		Log:       io.Discard,
	}).Generate(dest)
	require.NoError(t, err)
	return res
}

func read(t *testing.T, parts ...string) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(parts...))
	require.NoError(t, err)
	return string(b)
}

func TestGenerateWritesTheExpectedLayout(t *testing.T) {
	dest := filepath.Join(t.TempDir(), "myservice")

	res := generate(t, dest, false)

	assert.ElementsMatch(t, []string{
		"Dockerfile",
		"cmd/main.go", // .tmpl becomes .go
		"env/.env.local",
		"go.mod", // .tpl just loses its suffix
		"Makefile",
		"config/config.yml",
	}, res.Created)
	assert.Empty(t, res.Skipped)
}

func TestGenerateSubstitutesTheProjectName(t *testing.T) {
	dest := filepath.Join(t.TempDir(), "myservice")

	generate(t, dest, false)

	assert.Contains(t, read(t, dest, "cmd/main.go"), "// myservice")
	assert.Contains(t, read(t, dest, "Dockerfile"), `CMD ["./myservice"]`)
}

// The generator must not HTML-escape its payload: it emits Go source, where
// "<-" and "&&" are operators. This is what html/template used to break.
func TestGenerateDoesNotEscapeGoOperators(t *testing.T) {
	dest := t.TempDir()

	generate(t, dest, true)

	main := read(t, dest, "cmd/main.go")
	assert.Contains(t, main, "<-c && true")
	assert.NotContains(t, main, "&lt;")
	assert.NotContains(t, main, "&amp;")
}

// Static files are copied byte-for-byte, so a literal "{{" survives.
func TestGenerateCopiesStaticVerbatim(t *testing.T) {
	dest := filepath.Join(t.TempDir(), "myservice")

	generate(t, dest, false)

	assert.Equal(t, "name: {{.ProjectName}}\n", read(t, dest, "config/config.yml"))
}

func TestGenerateSkipsExistingFilesByDefault(t *testing.T) {
	dest := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(dest, "cmd"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dest, "cmd", "main.go"), []byte("MINE"), 0o644))

	res := generate(t, dest, false)

	assert.Equal(t, "MINE", read(t, dest, "cmd/main.go"), "existing file must not be clobbered")
	assert.Contains(t, res.Skipped, "cmd/main.go")
	assert.NotContains(t, res.Created, "cmd/main.go")
	assert.Contains(t, res.Created, "go.mod", "the other files are still generated")
}

func TestGenerateForceOverwritesExistingFiles(t *testing.T) {
	dest := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(dest, "cmd"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dest, "cmd", "main.go"), []byte("MINE"), 0o644))

	res := generate(t, dest, true)

	assert.Contains(t, read(t, dest, "cmd/main.go"), "package main")
	assert.Empty(t, res.Skipped)
}

func TestGenerateCreatesFilesWithSaneModes(t *testing.T) {
	dest := t.TempDir()

	generate(t, dest, false)

	file, err := os.Stat(filepath.Join(dest, "cmd", "main.go"))
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0o644), file.Mode().Perm())

	dir, err := os.Stat(filepath.Join(dest, "cmd"))
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0o755), dir.Mode().Perm())
}

// A broken template must not leave a partially written file on disk.
func TestGenerateRemovesPartialFileOnTemplateError(t *testing.T) {
	dest := t.TempDir()

	_, err := New(Options{
		Templates: fstest.MapFS{"cmd/main.tmpl": {Data: []byte("package main // {{.Nope}}\n")}},
		Log:       io.Discard,
	}).Generate(dest)

	require.Error(t, err)
	_, statErr := os.Stat(filepath.Join(dest, "cmd", "main.go"))
	assert.True(t, os.IsNotExist(statErr), "partial file should have been removed, got %v", statErr)
}

func TestGenerateWorksWithoutGOPATH(t *testing.T) {
	t.Setenv("GOPATH", "")
	dest := t.TempDir()

	res := generate(t, dest, false)

	assert.NotEmpty(t, res.Created)
}

func TestModulePath(t *testing.T) {
	for _, tc := range []struct {
		name string
		dir  string
		want string
	}{
		{"host segment starts the module path", "/home/me/code/github.com/fooOrg/foo", "github.com/fooOrg/foo"},
		{"nested group path is kept whole", "/w/gitlab.com/org/team/svc", "gitlab.com/org/team/svc"},
		{"legacy GOPATH layout", "/home/me/go/src/example/foo", "example/foo"},
		{"dot-directories are not hosts", "/home/me/.local/share/foo", "foo"},
		{"a dotted leaf is not a host", "/home/me/projects/my.app", "my.app"},
		{"plain directory falls back to its name", "/tmp/scratch/svc", "svc"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, ModulePath(filepath.FromSlash(tc.dir)))
		})
	}
}
