package scaffold

import (
	"bufio"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"

	"golang.org/x/mod/module"
)

// knownHosts are the code hosts recognised inside a directory path. A generic
// "looks like name.tld" rule is not enough: on macOS and Windows the username is
// often "first.last", and /Users/john.doe/code/svc must not become the module
// john.doe/code/svc. Anything else is reachable through the git remote or the
// --module flag.
var knownHosts = map[string]bool{
	"github.com":          true,
	"gitlab.com":          true,
	"bitbucket.org":       true,
	"codeberg.org":        true,
	"gitea.com":           true,
	"git.sr.ht":           true,
	"dev.azure.com":       true,
	"go.googlesource.com": true,
}

// ModulePath guesses the Go module path of a project directory, in order:
//
//  1. the "origin" remote of the git repository that contains the directory,
//     plus the directory's path inside the repository (a monorepo subfolder);
//  2. a known code host in the directory path (…/github.com/acme/svc);
//  3. the legacy GOPATH layout (…/src/<path>);
//  4. the bare directory name.
//
// It is a default: Options.Module overrides it.
func ModulePath(absDir string) string {
	if p, ok := moduleFromGitRemote(absDir); ok {
		return p
	}
	return moduleFromDir(absDir)
}

// moduleFromDir infers the module path from the directory path alone.
func moduleFromDir(absDir string) string {
	segs := strings.Split(filepath.ToSlash(absDir), "/")

	for i, seg := range segs {
		if i < len(segs)-1 && knownHosts[strings.ToLower(seg)] {
			return strings.Join(segs[i:], "/")
		}
	}
	for i := len(segs) - 2; i > 0; i-- {
		if segs[i] == "src" {
			return strings.Join(segs[i+1:], "/")
		}
	}
	return segs[len(segs)-1]
}

// moduleFromGitRemote looks for the git repository that contains absDir (the
// directory itself may not exist yet) and turns its "origin" URL into a module
// path. It reads .git/config directly: running git would make the CLI depend on
// a binary being installed.
func moduleFromGitRemote(absDir string) (string, bool) {
	for dir := absDir; ; dir = filepath.Dir(dir) {
		info, err := os.Stat(filepath.Join(dir, ".git"))
		if err == nil && info.IsDir() {
			url, ok := originURL(filepath.Join(dir, ".git", "config"))
			if !ok {
				return "", false
			}
			base, ok := moduleFromRemoteURL(url)
			if !ok {
				return "", false
			}
			rel, err := filepath.Rel(dir, absDir)
			if err != nil {
				return "", false
			}
			if rel == "." {
				return base, true
			}
			return path.Join(base, filepath.ToSlash(rel)), true
		}
		if parent := filepath.Dir(dir); parent == dir {
			return "", false
		}
	}
}

// originURL returns the url of [remote "origin"] in a git config file.
func originURL(configFile string) (string, bool) {
	f, err := os.Open(configFile)
	if err != nil {
		return "", false
	}
	defer func() { _ = f.Close() }()

	inOrigin := false
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if strings.HasPrefix(line, "[") {
			inOrigin = line == `[remote "origin"]`
			continue
		}
		if !inOrigin {
			continue
		}
		key, value, ok := strings.Cut(line, "=")
		if ok && strings.TrimSpace(key) == "url" {
			return strings.TrimSpace(value), true
		}
	}
	return "", false
}

// scpLike matches the scp-style ssh syntax: git@github.com:acme/svc.git
var scpLike = regexp.MustCompile(`^(?:[\w.-]+@)?([\w.-]+):([^/].*)$`)

// moduleFromRemoteURL maps a git remote URL to host/owner/repo.
func moduleFromRemoteURL(url string) (string, bool) {
	var host, p string
	if i := strings.Index(url, "://"); i >= 0 {
		rest := url[i+3:]
		if at := strings.LastIndex(rest, "@"); at >= 0 {
			rest = rest[at+1:]
		}
		host, p, _ = strings.Cut(rest, "/")
		if h, _, ok := strings.Cut(host, ":"); ok {
			host = h // drop the port
		}
	} else if m := scpLike.FindStringSubmatch(url); m != nil {
		host, p = m[1], m[2]
	} else {
		return "", false
	}

	p = strings.TrimSuffix(strings.Trim(p, "/"), ".git")
	if host == "" || p == "" || !strings.Contains(host, ".") {
		return "", false
	}
	mod := strings.ToLower(host) + "/" + p
	if module.CheckImportPath(mod) != nil {
		return "", false
	}
	return mod, true
}

// validName is what docker compose accepts as a service name, and what is safe
// as a binary name in the Dockerfile.
var validName = regexp.MustCompile(`^[a-z0-9][a-z0-9_.-]*$`)

// NormalizeName turns a directory name into a project name: lowercase, with
// spaces and other unsupported characters replaced by "-". "Service Name"
// becomes "service-name".
func NormalizeName(dir string) string {
	var b strings.Builder
	dash := false
	for _, r := range strings.ToLower(dir) {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9', r == '_', r == '.':
			b.WriteRune(r)
			dash = false
		case !dash && b.Len() > 0:
			b.WriteByte('-')
			dash = true
		}
	}
	return strings.Trim(b.String(), "-._")
}

// checkName reports whether name can be used as the project name.
func checkName(name string) error {
	if !validName.MatchString(name) {
		return fmt.Errorf("invalid project name %q: use lowercase letters, digits, '-', '_' or '.', starting with a letter or a digit", name)
	}
	return nil
}

// checkModule reports whether mod can be used as the module path in go.mod.
// inferred adds the hint to pass one explicitly.
func checkModule(mod string, inferred bool) error {
	if err := module.CheckImportPath(mod); err != nil {
		if inferred {
			return fmt.Errorf("inferred module path %q is invalid: %v; pass one with --module", mod, err)
		}
		return fmt.Errorf("invalid module path %q: %v", mod, err)
	}
	return nil
}

// ModuleSpec names a bounded context to generate.
type ModuleSpec struct {
	// Name is the folder under internal/, the route under /api and the table
	// prefix: lowercase letters and digits, starting with a letter ("billing").
	Name string
	// Entity is the aggregate's type name ("Invoice"). Empty means Name with an
	// upper-case first letter.
	Entity string
}

// DefaultExample is the bounded context `scaffold init` generates.
var DefaultExample = ModuleSpec{Name: "catalog", Entity: "Product"}

var (
	validModule = regexp.MustCompile(`^[a-z][a-z0-9]*$`)
	validEntity = regexp.MustCompile(`^[A-Z][A-Za-z0-9]*$`)
)

// reservedModules would collide with what the project already has under
// internal/ or with the Go toolchain's special directory names.
var reservedModules = map[string]bool{"archtest": true, "internal": true, "vendor": true, "testdata": true}

// moduleData validates spec and returns base completed with the module fields.
func moduleData(base data, spec ModuleSpec) (data, error) {
	name := spec.Name
	if !validModule.MatchString(name) || reservedModules[name] {
		return data{}, fmt.Errorf("invalid module name %q: lowercase letters and digits, starting with a letter (e.g. billing), and not archtest/internal/vendor/testdata", name)
	}
	title := strings.ToUpper(name[:1]) + name[1:]

	entity := spec.Entity
	if entity == "" {
		entity = title
	}
	if !validEntity.MatchString(entity) {
		return data{}, fmt.Errorf("invalid entity name %q: an exported Go identifier in CamelCase (e.g. Invoice)", entity)
	}

	base.Module = name
	base.ModuleTitle = title
	base.Entity = entity
	base.EntityVar = strings.ToLower(snake(entity, ' '))
	base.Table = name + "_" + strings.ToLower(snake(entity, '_'))
	return base, nil
}

// snake splits a CamelCase identifier with sep: "InvoiceLine" -> "Invoice_Line".
func snake(s string, sep rune) string {
	var b strings.Builder
	for i, r := range s {
		if i > 0 && r >= 'A' && r <= 'Z' {
			b.WriteRune(sep)
		}
		b.WriteRune(r)
	}
	return b.String()
}
