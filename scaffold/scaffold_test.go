package scaffold

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
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
		// Regression: a "first.last" username used to become the module host.
		{"dotted macOS username is not a host", "/Users/john.doe/code/svc", "svc"},
		{"dotted Windows username is not a host", "C:/Users/jane.smith/src2/api", "api"},
		{"known host after a dotted username", "/Users/john.doe/github.com/acme/svc", "github.com/acme/svc"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, ModulePath(filepath.FromSlash(tc.dir)))
		})
	}
}

func TestModuleFromRemoteURL(t *testing.T) {
	for _, tc := range []struct {
		url  string
		want string
		ok   bool
	}{
		{"git@github.com:acme/svc.git", "github.com/acme/svc", true},
		{"https://github.com/acme/svc.git", "github.com/acme/svc", true},
		{"https://user:token@gitlab.com/org/team/svc", "gitlab.com/org/team/svc", true},
		{"ssh://git@git.example.com:2222/acme/svc.git", "git.example.com/acme/svc", true},
		{"https://GitHub.com/acme/svc/", "github.com/acme/svc", true},
		{"/srv/git/svc.git", "", false},
		{"file:///srv/git/svc.git", "", false},
	} {
		t.Run(tc.url, func(t *testing.T) {
			got, ok := moduleFromRemoteURL(tc.url)
			assert.Equal(t, tc.ok, ok)
			assert.Equal(t, tc.want, got)
		})
	}
}

// writeGitConfig makes root look like a git checkout with the given origin.
func writeGitConfig(t *testing.T, root, origin string) {
	t.Helper()
	require.NoError(t, os.MkdirAll(filepath.Join(root, ".git"), 0o755))
	cfg := "[core]\n\tbare = false\n[remote \"upstream\"]\n\turl = git@github.com:other/fork.git\n" +
		"[remote \"origin\"]\n\turl = " + origin + "\n\tfetch = +refs/heads/*:refs/remotes/origin/*\n"
	require.NoError(t, os.WriteFile(filepath.Join(root, ".git", "config"), []byte(cfg), 0o644))
}

func TestModulePathPrefersTheGitRemote(t *testing.T) {
	root := filepath.Join(t.TempDir(), "john.doe", "code", "svc")
	writeGitConfig(t, root, "git@github.com:acme/svc.git")

	assert.Equal(t, "github.com/acme/svc", ModulePath(root))
	// A folder inside the repository (a monorepo) keeps its relative path, even
	// before it exists.
	assert.Equal(t, "github.com/acme/svc/services/billing", ModulePath(filepath.Join(root, "services", "billing")))
}

func TestModulePathIgnoresARepoWithoutOrigin(t *testing.T) {
	root := filepath.Join(t.TempDir(), "svc")
	require.NoError(t, os.MkdirAll(filepath.Join(root, ".git"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(root, ".git", "config"), []byte("[core]\n"), 0o644))

	assert.Equal(t, "svc", ModulePath(root))
}

func TestNormalizeName(t *testing.T) {
	for in, want := range map[string]string{
		"myservice":        "myservice",
		"Service Name":     "service-name",
		"  My  Cool--App ": "my-cool-app",
		"api_v2.1":         "api_v2.1",
		"Ünïcode svc":      "n-code-svc",
		"---":              "",
	} {
		assert.Equal(t, want, NormalizeName(in), "NormalizeName(%q)", in)
	}
}

func generateWith(t *testing.T, dest string, opts Options) (Result, error) {
	t.Helper()
	templates, static := fixture()
	opts.Templates, opts.Static, opts.Log = templates, static, io.Discard
	return New(opts).Generate(dest)
}

// Regression: "Service Name" ended up verbatim in go.mod and docker-compose.yml.
func TestGenerateNormalisesADirectoryNameWithSpaces(t *testing.T) {
	dest := filepath.Join(t.TempDir(), "Service Name")

	_, err := generateWith(t, dest, Options{})
	require.NoError(t, err)

	assert.Equal(t, "module service-name\n", read(t, dest, "go.mod"))
	assert.Contains(t, read(t, dest, "Dockerfile"), `CMD ["./service-name"]`)
}

func TestGenerateUsesTheModuleAndNameFlags(t *testing.T) {
	dest := filepath.Join(t.TempDir(), "john.doe", "whatever")

	_, err := generateWith(t, dest, Options{Module: "github.com/acme/billing", Name: "billing"})
	require.NoError(t, err)

	assert.Equal(t, "module github.com/acme/billing\n", read(t, dest, "go.mod"))
	assert.Contains(t, read(t, dest, "Dockerfile"), `CMD ["./billing"]`)
}

func TestGenerateRejectsInvalidInputBeforeWritingAnything(t *testing.T) {
	for _, tc := range []struct {
		name string
		dest string
		opts Options
		want string
	}{
		{"invalid module flag", "svc", Options{Module: "github.com/acme/my svc"}, `invalid module path "github.com/acme/my svc"`},
		{"invalid name flag", "svc", Options{Name: "My Svc"}, "invalid project name"},
		{"inferred module with a space", "github.com/acme/My Svc", Options{}, "pass one with --module"},
		{"name with nothing usable", "---", Options{}, "--name"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dest := filepath.Join(t.TempDir(), filepath.FromSlash(tc.dest))

			_, err := generateWith(t, dest, tc.opts)

			require.Error(t, err)
			assert.Contains(t, err.Error(), tc.want)
			_, statErr := os.Stat(dest)
			assert.True(t, os.IsNotExist(statErr), "nothing must be written, got %v", statErr)
		})
	}
}

// failingFS serves files whose Read fails after the first chunk: a static copy
// that breaks halfway, like a flaky disk or a truncated embed.
type failingFS struct{ fstest.MapFS }

type failingFile struct {
	fs.File
	served bool
}

var errHalfway = errors.New("read failed halfway")

func (f *failingFile) Read(p []byte) (int, error) {
	if f.served {
		return 0, errHalfway
	}
	f.served = true
	return copy(p, "PARTIAL"), nil
}

func (ffs failingFS) Open(name string) (fs.File, error) {
	f, err := ffs.MapFS.Open(name)
	if err != nil {
		return nil, err
	}
	if info, err := f.Stat(); err == nil && info.IsDir() {
		return f, nil
	}
	return &failingFile{File: f}, nil
}

// Regression: with --force the file was truncated, then removed on error.
func TestGenerateNeverDamagesExistingFilesOnError(t *testing.T) {
	for _, tc := range []struct {
		name string
		opts Options
		file string
	}{
		{"template fails halfway", Options{Templates: fstest.MapFS{"main.tmpl": {Data: []byte("package main // {{.ProjectName}} {{.Nope}}\n")}}}, "main.go"},
		{"static copy fails halfway", Options{Static: failingFS{fstest.MapFS{"Makefile": {Data: []byte("run:\n")}}}}, "Makefile"},
	} {
		for _, force := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/force=%v", tc.name, force), func(t *testing.T) {
				dest := t.TempDir()
				opts := tc.opts
				opts.Force, opts.Log = force, io.Discard

				// A fresh destination: the failed file must not appear at all.
				_, err := New(opts).Generate(filepath.Join(dest, "fresh"))
				require.Error(t, err)
				_, statErr := os.Stat(filepath.Join(dest, "fresh", tc.file))
				assert.True(t, os.IsNotExist(statErr), "partial file left behind: %v", statErr)

				// An existing file: it must survive untouched.
				existing := filepath.Join(dest, "existing")
				require.NoError(t, os.MkdirAll(existing, 0o755))
				require.NoError(t, os.WriteFile(filepath.Join(existing, tc.file), []byte("MINE"), 0o644))
				_, err = New(opts).Generate(existing)
				if force {
					require.Error(t, err)
				}
				assert.Equal(t, "MINE", read(t, existing, tc.file))

				for _, d := range []string{"fresh", "existing"} {
					entries, _ := os.ReadDir(filepath.Join(dest, d))
					for _, e := range entries {
						assert.NotContains(t, e.Name(), ".scaffold-", "temporary file left in %s", d)
					}
				}
			})
		}
	}
}
