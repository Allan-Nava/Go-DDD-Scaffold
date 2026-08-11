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
	"unicode"
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

	d := data{
		AbsGenProjectPath: absDest,
		ProjectName:       filepath.Base(absDest),
		ProjectPath:       ModulePath(absDest),
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
func (w *writer) write(rel string, fn func(io.Writer) error) error {
	abs := filepath.Join(w.root, filepath.FromSlash(rel))

	dir := filepath.Dir(abs)
	if _, done := w.dirs[dir]; !done {
		if err := os.MkdirAll(dir, dirPerm); err != nil {
			return err
		}
		w.dirs[dir] = struct{}{}
	}

	// O_EXCL makes "skip if it exists" atomic, with no extra stat call.
	flag := os.O_WRONLY | os.O_CREATE | os.O_EXCL
	if w.force {
		flag = os.O_WRONLY | os.O_CREATE | os.O_TRUNC
	}
	f, err := os.OpenFile(abs, flag, filePerm)
	if err != nil {
		if !w.force && errors.Is(err, fs.ErrExist) {
			w.result.Skipped = append(w.result.Skipped, rel)
			_, _ = fmt.Fprintf(w.log, "Skip   %s (already exists, use --force to overwrite)\n", rel)
			return nil
		}
		return err
	}

	buf := bufio.NewWriter(f)
	if err := fn(buf); err != nil {
		// The write error is what we report; closing and removing are best-effort
		// cleanup so a half-written source file never survives.
		_ = f.Close()
		_ = os.Remove(abs)
		return fmt.Errorf("write %s: %w", rel, err)
	}
	if err := buf.Flush(); err != nil {
		_ = f.Close()
		return fmt.Errorf("flush %s: %w", rel, err)
	}
	if err := f.Close(); err != nil {
		return fmt.Errorf("close %s: %w", rel, err)
	}

	w.result.Created = append(w.result.Created, rel)
	_, _ = fmt.Fprintf(w.log, "Create %s\n", rel)
	return nil
}

// ModulePath guesses a Go module path for a project directory: the first path
// segment that looks like a host (github.com, gitlab.com, …) starts the path.
// It falls back to the legacy "…/src/<path>" layout, then to the bare
// directory name.
func ModulePath(absDir string) string {
	segs := strings.Split(filepath.ToSlash(absDir), "/")

	for i, seg := range segs {
		if i < len(segs)-1 && looksLikeHost(seg) {
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

// looksLikeHost reports whether seg could be a module host, ie. "name.tld"
// with an alphabetic tld. Dot-prefixed directories such as ".config" are not.
func looksLikeHost(seg string) bool {
	dot := strings.LastIndex(seg, ".")
	if dot < 1 || dot == len(seg)-1 {
		return false
	}
	for _, r := range seg[dot+1:] {
		if !unicode.IsLetter(r) {
			return false
		}
	}
	return len(seg)-dot-1 >= 2
}
