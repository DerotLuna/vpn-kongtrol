package profileimport

import (
	"bytes"
	"crypto/sha256"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

// A browser drop gives the page a file's name and bytes, never its path
// (ADR-0105). Before telling the user that the certificates an .ovpn names
// are missing, the daemon looks for the file's original in the folders
// where VPN files usually live (SearchRoots: OpenVPN GUI's config folders,
// Downloads, Desktop, Documents — wherever the OS keeps them, OneDrive
// included — and the other clients' folders on Linux and macOS). A match
// has the same base name and the same bytes: only files with that name and
// that size are read, and only to compare their SHA-256. Exactly one
// folder with an identical copy is used like a picked file's folder;
// several are narrowed to the one holding every referenced file; still
// more than one is never guessed (ADR-0112).
//
// Bounds: folders at most MaxDepth below a root, MaxEntries directory
// entries looked at in all, Budget of wall time, at most maxHashed files
// read; links, junctions and other reparse points (except OneDrive's own
// files) are never followed, network drives never searched.

// Locator searches for the original of a dropped file.
type Locator struct {
	Names      []string      // names the original may have (the dropped file's name)
	Roots      []string      // folders searched; nil = SearchRoots()
	MaxDepth   int           // folders below a root; 0 = 3
	MaxEntries int           // directory entries looked at in all; 0 = 20000
	Budget     time.Duration // wall time; 0 = 1 s

	now func() time.Time // clock; nil = time.Now (tests step it: Windows clock ticks are coarse)
}

// maxHashed bounds how many same-name, same-size files are read.
const maxHashed = 32

// skipDirs are never entered (large, and never where a provider's files land).
var skipDirs = map[string]bool{"node_modules": true, ".git": true, "$recycle.bin": true, ".cache": true, ".trash": true}

// Find returns the folder of the one identical original of content, or ""
// when there is none or more than one (see the package note above).
func (l *Locator) Find(content []byte) string {
	if l == nil || len(content) == 0 {
		return ""
	}
	names := map[string]bool{}
	for _, n := range l.Names {
		if b := RefBase(n); b != "" {
			names[foldName(b)] = true
		}
	}
	if len(names) == 0 {
		return ""
	}
	roots := l.Roots
	if roots == nil {
		roots = SearchRoots()
	}
	now := l.now
	if now == nil {
		now = time.Now
	}
	s := &search{
		now:   now,
		names: names, size: int64(len(content)), sum: sha256.Sum256(content),
		depth: or(l.MaxDepth, 3), left: or(l.MaxEntries, 20000),
		deadline: now().Add(orDur(l.Budget, time.Second)),
		visited:  map[string]bool{},
	}
	for _, root := range roots {
		if s.stopped() {
			break
		}
		r, ok := CleanLocalPath(root)
		if !ok || onRemoteDrive(r) {
			continue
		}
		s.walk(r, 0)
	}
	switch len(s.found) {
	case 0:
		return ""
	case 1:
		return s.found[0]
	}
	// Several folders hold the same file: the one that also holds every
	// file it names (checked by name only, nothing more is read).
	refs := ReferencedPaths(content)
	var complete []string
	for _, dir := range s.found {
		if allPresent(dir, refs) {
			complete = append(complete, dir)
		}
	}
	if len(complete) == 1 {
		return complete[0]
	}
	return ""
}

type search struct {
	names    map[string]bool
	size     int64
	sum      [sha256.Size]byte
	depth    int
	left     int
	hashed   int
	deadline time.Time
	now      func() time.Time
	visited  map[string]bool
	found    []string
	foundFI  []os.FileInfo
}

func (s *search) stopped() bool { return s.left <= 0 || s.now().After(s.deadline) }

func (s *search) walk(dir string, depth int) {
	key := foldName(dir)
	if s.visited[key] {
		return
	}
	s.visited[key] = true
	fi, err := os.Lstat(dir)
	if err != nil || !fi.IsDir() || !plain(dir, fi) {
		return
	}
	f, err := os.Open(dir)
	if err != nil {
		return
	}
	var subdirs []string
	for !s.stopped() {
		entries, err := f.ReadDir(256)
		for _, e := range entries {
			if s.left--; s.stopped() {
				break
			}
			p := filepath.Join(dir, e.Name())
			switch {
			case e.IsDir():
				if depth < s.depth && !skipDirs[strings.ToLower(e.Name())] {
					subdirs = append(subdirs, p)
				}
			case s.names[foldName(e.Name())]:
				s.check(dir, p)
			}
		}
		if err != nil { // io.EOF or a read error: this folder is done
			break
		}
	}
	_ = f.Close()
	for _, sub := range subdirs {
		if s.stopped() {
			return
		}
		s.walk(sub, depth+1)
	}
}

// check reads p only when it is a plain file of the dropped file's size,
// and records its folder when the bytes are the same.
func (s *search) check(dir, p string) {
	fi, err := os.Lstat(p)
	if err != nil || fi.IsDir() || !plain(p, fi) || fi.Size() != s.size || s.hashed >= maxHashed {
		return
	}
	s.hashed++
	f, err := openNoFollow(p)
	if err != nil {
		return
	}
	defer f.Close()
	after, err := f.Stat()
	if err != nil || after.IsDir() || after.Mode()&notFile != 0 || !os.SameFile(fi, after) {
		return
	}
	h := sha256.New()
	if n, err := io.Copy(h, io.LimitReader(f, s.size+1)); err != nil || n != s.size {
		return
	}
	if !bytes.Equal(h.Sum(nil), s.sum[:]) {
		return
	}
	dfi, err := os.Lstat(dir)
	if err != nil {
		return
	}
	for _, seen := range s.foundFI {
		if os.SameFile(seen, dfi) {
			return
		}
	}
	s.found = append(s.found, dir)
	s.foundFI = append(s.foundFI, dfi)
}

// allPresent: every reference inside dir is a plain file there (by name,
// never read). References outside dir do not depend on the folder.
func allPresent(dir string, refs []string) bool {
	for _, p := range refs {
		full, inside, ok := refPath(dir, p)
		if !ok {
			return false
		}
		if !inside {
			continue
		}
		fi, err := os.Lstat(full)
		if err != nil || fi.IsDir() || !plain(full, fi) {
			return false
		}
	}
	return true
}

// foldName compares names like the OS does: case-insensitive on Windows
// and macOS (their default file systems), exact elsewhere.
func foldName(s string) string {
	if runtime.GOOS == "windows" || runtime.GOOS == "darwin" {
		return strings.ToLower(s)
	}
	return s
}

func or(v, d int) int {
	if v > 0 {
		return v
	}
	return d
}

func orDur(v, d time.Duration) time.Duration {
	if v > 0 {
		return v
	}
	return d
}

// home is os.UserHomeDir, "" when unknown.
func home() string {
	h, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return h
}

// existing keeps the roots that are set (later duplicates dropped).
func existing(roots ...string) []string {
	var out []string
	seen := map[string]bool{}
	for _, r := range roots {
		if r == "" || seen[foldName(filepath.Clean(r))] {
			continue
		}
		seen[foldName(filepath.Clean(r))] = true
		out = append(out, r)
	}
	return out
}
