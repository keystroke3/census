// Package index takes finds all the files in the provided directories
package index

import (
	"census/types"
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"os/signal"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"syscall"
)

type File struct {
	Name      string
	Directory string
}

type FormatArgs struct {
	EscapeChars   map[rune]bool
	TrimPrefixes  []string
	ApplyQuote    bool
	Relative      bool
}

func NewMemIndex(args *types.Command) *MemIndex {
	index := &MemIndex{
		Files:       make(map[string]*File),
		Dirs:        make(map[string]bool),
		paths:       args.Paths,
		ignore:      args.IgnorePaths,
		showHidden:  args.ShowHidden,
		depth:       args.Depth,
		escapeChars: args.EscapeChars,
		quote:       args.Quote,
	}

	return index
}

func NewFormatArgs(args *types.Command) *FormatArgs {
	fArgs := &FormatArgs{
		EscapeChars:  map[rune]bool{},
		TrimPrefixes: []string{},
	}

	home, err := os.UserHomeDir()
	if err != nil {
		fmt.Println("unable to determine user's home directory")
		os.Exit(1)
	}
	sep := string(os.PathSeparator)

	if args.Relative {
		fArgs.Relative = true
		for _, p := range args.Paths {
			var resolved string
			if strings.HasPrefix(p, "~"+sep) {
				resolved = filepath.Join(home, p[2:])
			} else if p == "~" {
				resolved = home
			} else {
				resolved = p
			}
			if strings.HasSuffix(p, sep) {
				resolved += sep
			}
			fArgs.TrimPrefixes = append(fArgs.TrimPrefixes, resolved)
		}
	}
	for _, pfx := range args.Trim {
		var resolved string
		if strings.HasPrefix(pfx, "~"+sep) {
			resolved = filepath.Join(home, pfx[2:])
		} else if pfx == "~" {
			resolved = home
		} else {
			resolved = pfx
		}
		if strings.HasSuffix(pfx, sep) {
			resolved += sep
		}
		fArgs.TrimPrefixes = append(fArgs.TrimPrefixes, resolved)
	}
	if len(args.EscapeChars) > 0 {
		for _, c := range args.EscapeChars {
			fArgs.EscapeChars[c] = true
		}
	}
	fArgs.ApplyQuote = args.Quote
	return fArgs
}

func Query(args *types.Command, memIndex *MemIndex) (string, error) {
	for _, path := range args.Paths {
		_, err := os.Stat(path)
		if err != nil {
			if os.IsNotExist(err) {
				return "", fmt.Errorf("path not found '%v'", path)
			}
			return "", err
		}
	}

	if memIndex == nil {
		memIndex = NewMemIndex(args)
	}

	Walk(args.Paths, &memIndex.Root, &memIndex.Current, memIndex.Add)

	var allPaths []string
	if args.DirMode {
		allPaths = memIndex.GetDirs()
	} else {
		allPaths = memIndex.GetFiles()
	}
	formatArgs := NewFormatArgs(args)
	var filters []Filter
	if args.Vgrep != "" {
		re, err := regexp.Compile(strings.ToLower(args.Vgrep))
		if err != nil {
			fmt.Printf("could not compile regular expression '%s': %s", args.Vgrep, err.Error())
		}
		filters = append(filters, Filter{
			Exp:       re,
			Sensitive: false,
			Inclusive: false,
		})
	}
	if args.Vsensitive != "" {
		re, err := regexp.Compile(args.Vsensitive)
		if err != nil {
			fmt.Printf("could not compile regular expression '%s': %s", args.Vsensitive, err.Error())
		}
		filters = append(filters, Filter{
			Exp:       re,
			Sensitive: true,
			Inclusive: false,
		})
	}
	if args.Grep != "" {
		re, err := regexp.Compile(strings.ToLower(args.Grep))
		if err != nil {
			fmt.Printf("could not compile regular expression '%s': %s", args.Grep, err.Error())
		}
		filters = append(filters, Filter{
			Exp:       re,
			Sensitive: false,
			Inclusive: true,
		})
	}
	if args.Gsensitive != "" {
		re, err := regexp.Compile(args.Gsensitive)
		if err != nil {
			fmt.Printf("could not compile regular expression '%s': %s", args.Gsensitive, err.Error())
		}
		filters = append(filters, Filter{
			Exp:       re,
			Sensitive: true,
			Inclusive: true,
		})
	}
	allPaths = FilterPaths(allPaths, filters, formatArgs)
	return fmt.Sprint(strings.Join(allPaths, "\n")), nil
}

type MemIndex struct {
	Files       map[string]*File
	Dirs        map[string]bool
	Current     string
	Root        string
	paths       []string
	ignore      []string
	showHidden  bool
	depth       int
	quote       bool
	escapeChars string
}

func (i *MemIndex) Poll(interval int) {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	quit := false
	go func() {
		sig := <-c
		if sig != nil {
			slog.Info("stop polling")
			quit = true
			return
		}
	}()

	for {
		if quit {
			return
		}

	}
}

// Add puts a new `File` entry to the index
func (i *MemIndex) Add(path string, f fs.DirEntry, err error) error {
	if err != nil {
		return err
	}
	if path == "." {
		return nil
	}
	stat, err := f.Info()
	if err != nil {
		return err
	}
	fullPath := filepath.Join(i.Current, path)
	relPath, err := filepath.Rel(i.Root, fullPath)
	if err != nil {
		fmt.Println("could not determine relative path,", err)
		syscall.Exit(1)
	}
	depth := strings.Count(relPath, string(os.PathSeparator))
	if stat.IsDir() {
		if depth == i.depth {
			return fs.SkipDir
		}
		_, leaf := filepath.Split(path)
		if slices.Contains(i.ignore, leaf) {
			return fs.SkipDir
		}
		if !i.showHidden && strings.HasPrefix(leaf, ".") {
			return fs.SkipDir
		}
		i.Dirs[fullPath] = true
		return nil
	}
	if !i.showHidden && strings.HasPrefix(stat.Name(), ".") {
		return nil
	}
	file := File{
		Name:      fullPath,
		Directory: filepath.Dir(path),
	}

	i.Files[fullPath] = &file
	return nil
}

// FormatPath applies transformations on a that were passed as arguments.
// It removes leading separators from a relative path so it is not
// confused as a root path.
// Leading separators are removed when the trim argument has a trailing separator.
//
// E.g. -p '~/foo/bar/baz' -t '~/foo/'
//
// becomes
//
// bar/baz
//
// and -t '~/foo'
//
// becomes
//
// '/bar/baz'
func FormatPath(args *FormatArgs, p string) string {
	path := p
	lastSep := false
	for _, prefix := range args.TrimPrefixes {
		if after, ok := strings.CutPrefix(path, prefix); ok {
			path = after
			lastSep = strings.HasSuffix(prefix, string(os.PathSeparator))
		}
	}
	if args.Relative || lastSep {
		for len(path) > 0 && path[0] == os.PathSeparator {
			path = path[1:]
		}
	}
	if args.ApplyQuote {
		// use this syntax instead of "%q" to prevent double escaping of spaces
		path = fmt.Sprintf("\"%s\"", path)
	}
	if len(args.EscapeChars) > 0 {
		for _, c := range path {
			if args.EscapeChars[c] {
				path = fmt.Sprintf("%s\\%c", path, c)
			} else {
				path = fmt.Sprintf("%s%c", path, c)
			}
		}
	}
	return path
}

func (i *MemIndex) GetFiles() []string {
	paths := []string{}
	for _, p := range i.Files {
		paths = append(paths, p.Name)
	}
	return paths
}

func (i *MemIndex) GetDirs() []string {
	dirs := []string{}
	for p := range i.Dirs {
		dirs = append(dirs, p)
	}
	return dirs
}

type Filter struct {
	Exp       *regexp.Regexp
	Sensitive bool
	Inclusive bool
}

// FilterPaths applies filters sequentially to paths, formatting matched results.
func FilterPaths(paths []string, filters []Filter, format *FormatArgs) []string {
	filtered := make([]string, 0, len(paths))
	for _, p := range paths {
		survives := true
		for _, f := range filters {
			matched := f.Exp.MatchString(p)
			if !f.Sensitive {
				matched = f.Exp.MatchString(strings.ToLower(p))
			}
			if (f.Inclusive && !matched) || (!f.Inclusive && matched) {
				survives = false
				break
			}
		}
		if survives {
			filtered = append(filtered, FormatPath(format, p))
		}
	}
	return filtered
}

func Walk(paths []string, root *string, current *string, fn func(path string, d fs.DirEntry, err error) error) {
	for _, p := range paths {
		d := os.DirFS(p)
		*current = p
		*root = p
		fs.WalkDir(d, ".", fn)
	}
}
