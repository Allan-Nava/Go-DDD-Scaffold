// Package scaffold renders a project skeleton into a destination directory.
//
// The two asset trees are supplied by the caller as fs.FS, so the generator has
// no runtime dependency on $GOPATH or on files sitting next to the binary:
//   - Templates is rendered through text/template ("*.tmpl" becomes "*.go").
//   - Static is copied byte-for-byte, without any rendering.
package scaffold

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strings"
	"text/template"
)

const (
	// tmplExt marks a Go source template: "cmd/main.tmpl" -> "cmd/main.go".
	tmplExt = ".tmpl"
	// tplExt marks a template whose real name is the rest of the filename:
	// "go.mod.tpl" -> "go.mod". It also keeps a nested go.mod from turning
	// template/ into a separate module, which would break the embedding.
	tplExt = ".tpl"

	dirPerm  os.FileMode = 0o755
	filePerm os.FileMode = 0o644
)

// Options configures a Scaffold.
type Options struct {
	// Templates is rendered through text/template. Files ending in ".tmpl"
	// are written out with a ".go" extension; everything else keeps its name.
	Templates fs.FS
	// Static is copied verbatim, so it can hold files with literal "{{".
	Static fs.FS
	// Force overwrites destination files that already exist. Without it,
	// existing files are left untouched and reported as skipped.
	Force bool
	// Log receives the per-file progress lines. Defaults to os.Stdout.
	Log io.Writer
	// Debug turns on the internal trace output.
	Debug bool
	// Module is the Go module path written to go.mod. Empty means inferred
	// from the destination (see ModulePath).
	Module string
	// Name is the project name (compose service, binary, README title). Empty
	// means the destination directory name, normalised (see NormalizeName).
	Name string
}

// Scaffold generates a project layout from the assets in its Options.
type Scaffold struct {
	opts Options
}

// New returns a Scaffold that writes the given assets.
func New(opts Options) *Scaffold {
	if opts.Log == nil {
		opts.Log = os.Stdout
	}
	return &Scaffold{opts: opts}
}

// Result reports what Generate did, in destination-relative slash paths.
type Result struct {
	Created []string
	Skipped []string
}

// data is the value passed to every template.
type data struct {
	AbsGenProjectPath string // absolute path of the generated project
	ProjectPath       string // Go module path (eg. github.com/fooOrg/foo)
	ProjectName       string // project name, ie. the destination directory name
}

// Generate renders the assets into dest, creating it if needed.
func (s *Scaffold) Generate(dest string) (Result, error) {
	absDest, err := filepath.Abs(dest)
	if err != nil {
		return Result{}, err
	}

	d, err := s.projectData(absDest)
	if err != nil {
		// Nothing has been written yet: a bad name or module path is reported
		// before the first file, not discovered when go.mod fails to parse.
		return Result{}, err
	}
	s.debugf("generate dest=%s module=%s\n", absDest, d.ProjectPath)

	w := &writer{root: absDest, force: s.opts.Force, log: s.opts.Log, dirs: make(map[string]struct{})}

	if s.opts.Templates != nil {
		if err := s.render(w, d); err != nil {
			return w.result, err
		}
	}
	if s.opts.Static != nil {
		if err := s.copyStatic(w); err != nil {
			return w.result, err
		}
	}
	return w.result, nil
}

// projectData resolves and validates the project name and module path.
func (s *Scaffold) projectData(absDest string) (data, error) {
	base := filepath.Base(absDest)

	name := s.opts.Name
	if name == "" {
		name = NormalizeName(base)
		if name == "" {
			return data{}, fmt.Errorf("cannot derive a project name from %q: pass one with --name", base)
		}
	}
	if err := checkName(name); err != nil {
		return data{}, err
	}

	mod, inferred := s.opts.Module, s.opts.Module == ""
	if inferred {
		mod = ModulePath(absDest)
		if mod == base {
			// Nothing better than the directory name: use its normalised form,
			// so "Service Name" gives the module "service-name".
			mod = name
		}
	}
	if err := checkModule(mod, inferred); err != nil {
		return data{}, err
	}

	return data{AbsGenProjectPath: absDest, ProjectName: name, ProjectPath: mod}, nil
}

// render walks the template tree, parsing and executing each file in a single
// pass — no intermediate slice, and each template is read and parsed once.
func (s *Scaffold) render(w *writer, d data) error {
	return fs.WalkDir(s.opts.Templates, ".", func(p string, entry fs.DirEntry, err error) error {
		if err != nil || entry.IsDir() {
			return err
		}

		src, err := fs.ReadFile(s.opts.Templates, p)
		if err != nil {
			return err
		}

		// text/template, never html/template: the payload is Go source, and
		// HTML escaping would mangle "<-", "&&" and struct tags.
		tmpl, err := template.New(path.Base(p)).Parse(string(src))
		if err != nil {
			return fmt.Errorf("parse template %s: %w", p, err)
		}

		rel := p
		switch {
		case strings.HasSuffix(rel, tmplExt):
			rel = strings.TrimSuffix(rel, tmplExt) + ".go"
		case strings.HasSuffix(rel, tplExt):
			rel = strings.TrimSuffix(rel, tplExt)
		}
		s.debugf("render %s -> %s\n", p, rel)

		return w.write(rel, func(out io.Writer) error {
			return tmpl.Execute(out, d)
		})
	})
}

// copyStatic walks the static tree and copies every file unchanged.
func (s *Scaffold) copyStatic(w *writer) error {
	return fs.WalkDir(s.opts.Static, ".", func(p string, entry fs.DirEntry, err error) error {
		if err != nil || entry.IsDir() {
			return err
		}
		s.debugf("copy %s\n", p)
		return w.write(p, func(out io.Writer) error {
			src, err := s.opts.Static.Open(p)
			if err != nil {
				return err
			}
			// Per-file closure, so this runs per file. Closing a read-only handle
			// has nothing actionable to report.
			defer func() { _ = src.Close() }()

			_, err = io.Copy(out, src)
			return err
		})
	})
}

func (s *Scaffold) debugf(format string, a ...any) {
	if s.opts.Debug {
		_, _ = fmt.Fprintf(s.opts.Log, format, a...)
	}
}

// writer creates destination files, remembering which directories it has
// already made so a tree with many files in one directory costs one MkdirAll.
type writer struct {
	root   string
	force  bool
	log    io.Writer
	dirs   map[string]struct{}
	result Result
}

// write streams fn's output into the file at rel. When the file already exists
// and force is off, it records a skip and does not call fn.
//
// The content goes to a temporary file in the same directory first, and reaches
// rel only once it is complete: a failed render or flush never truncates, deletes
// or half-writes the destination, with or without force.
func (w *writer) write(rel string, fn func(io.Writer) error) error {
	abs := filepath.Join(w.root, filepath.FromSlash(rel))

	dir := filepath.Dir(abs)
	if _, done := w.dirs[dir]; !done {
		if err := os.MkdirAll(dir, dirPerm); err != nil {
			return err
		}
		w.dirs[dir] = struct{}{}
	}

	if !w.force {
		if _, err := os.Lstat(abs); err == nil {
			w.skip(rel)
			return nil
		}
	}

	tmp, err := w.writeTemp(dir, rel, fn)
	if err != nil {
		return err
	}
	// Removing an already-renamed temp file fails harmlessly.
	defer func() { _ = os.Remove(tmp) }()

	if w.force {
		if err := os.Rename(tmp, abs); err != nil {
			return fmt.Errorf("write %s: %w", rel, err)
		}
	} else if err := publishExclusive(tmp, abs); err != nil {
		if errors.Is(err, fs.ErrExist) {
			w.skip(rel) // created by someone else since the Lstat above
			return nil
		}
		return fmt.Errorf("write %s: %w", rel, err)
	}

	w.result.Created = append(w.result.Created, rel)
	_, _ = fmt.Fprintf(w.log, "Create %s\n", rel)
	return nil
}

// writeTemp writes fn's output to a new temporary file in dir and returns its
// path. On any error the temporary file is removed.
func (w *writer) writeTemp(dir, rel string, fn func(io.Writer) error) (string, error) {
	f, err := os.CreateTemp(dir, ".scaffold-*.tmp")
	if err != nil {
		return "", err
	}
	tmp := f.Name()
	fail := func(step string, err error) (string, error) {
		_ = f.Close()
		_ = os.Remove(tmp)
		return "", fmt.Errorf("%s %s: %w", step, rel, err)
	}

	buf := bufio.NewWriter(f)
	if err := fn(buf); err != nil {
		return fail("write", err)
	}
	if err := buf.Flush(); err != nil {
		return fail("flush", err)
	}
	// CreateTemp uses 0600: the generated files are ordinary sources.
	if err := f.Chmod(filePerm); err != nil {
		return fail("chmod", err)
	}
	if err := f.Close(); err != nil {
		_ = os.Remove(tmp)
		return "", fmt.Errorf("close %s: %w", rel, err)
	}
	return tmp, nil
}

// publishExclusive moves tmp to abs only if abs does not exist. A hard link is
// the atomic "create if absent"; on a filesystem without hard links it falls
// back to a rename after a fresh existence check.
func publishExclusive(tmp, abs string) error {
	err := os.Link(tmp, abs)
	if err == nil || errors.Is(err, fs.ErrExist) {
		return err
	}
	if _, statErr := os.Lstat(abs); statErr == nil {
		return fs.ErrExist
	}
	return os.Rename(tmp, abs)
}

func (w *writer) skip(rel string) {
	w.result.Skipped = append(w.result.Skipped, rel)
	_, _ = fmt.Fprintf(w.log, "Skip   %s (already exists, use --force to overwrite)\n", rel)
}
