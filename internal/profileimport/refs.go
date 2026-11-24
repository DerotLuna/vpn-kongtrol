package profileimport

import (
	"errors"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"sort"
	"strings"

	"github.com/vpn-kongtrol/kongtrol/internal/config"
	"github.com/vpn-kongtrol/kongtrol/internal/privops"
)

// Files an .ovpn names (ca ca.crt, tls-auth ta.key 1, …) are read once at
// import time and put inside the stored copy (<ca>…</ca>), so the stored
// profile stays self-contained and the helper never opens another user path
// (ADR-0105). They come from two places:
//
//   - Dir: the folder of the config the user picked with a real path (the
//     native file dialog, `kongtrol profile import <file>`). A reference is
//     resolved relative to it, like OpenVPN GUI's --cd. Only files inside
//     that folder (or its subfolders) are read: an absolute path or a ".."
//     that leaves it is never opened, it counts as missing. Every folder on
//     the way must be a real folder (no link, no junction) and the file a
//     regular file (no link, no device), opened without following a link
//     and checked again after opening (no swap), at most
//     privops.MaxReferenceSize bytes, with the user's own rights (the daemon
//     and the CLI run as the user).
//   - Files: files the user added by hand (the browser cannot give a path:
//     several files dropped at once, the "Agregar <name>" button, extra CLI
//     arguments), matched by base name.
//
// What is still missing is listed (MissingFilesError), so the panel can
// offer "Agregar <name>" for each one. Which directives may name a file is
// the helper's own list (privops ovpnFileRefs: ca, cert, key, tls-auth,
// tls-crypt, tls-crypt-v2, secret, dh, extra-certs, crl-verify, pkcs12);
// auth-user-pass with a file and the rest of the denylist stay refused.

// MaxExtraFiles is how many files may be added by hand to one import.
const MaxExtraFiles = 8

// MaxReferenceSize is the largest referenced file (the helper's own limit).
const MaxReferenceSize = privops.MaxReferenceSize

// Refs says where the files an .ovpn names come from.
type Refs struct {
	Dir   string            // folder of the picked config; "" = none
	Files map[string][]byte // added by hand, by base name
}

// Translated refusals about referenced files (ADR-0105).
const (
	ErrKeyMissingFiles  = "api.err.import_missing_files"
	ErrKeyRefUnreadable = "api.err.import_ref_unreadable"
	ErrKeyRefTooBig     = "api.err.import_ref_too_big"
	ErrKeyRefNotText    = "api.err.import_ref_not_text"
	ErrKeyTooManyFiles  = "api.err.import_too_many_files"
)

// MissingFilesError lists the referenced files Kongtrol could not find
// (base names, in the order the config names them). It unwraps to the
// translated api.err.import_missing_files.
type MissingFilesError struct{ Names []string }

func (e *MissingFilesError) Error() string {
	return "missing referenced files: " + strings.Join(e.Names, ", ")
}

func (e *MissingFilesError) Unwrap() error {
	return &config.ValidationError{Key: ErrKeyMissingFiles, Args: []any{strings.Join(e.Names, ", ")}}
}

// NewRefs checks files added by hand: at most MaxExtraFiles, each a
// non-empty base name of at most MaxReferenceSize bytes, all together at
// most MaxSize (the stored config's own limit). Only the base name of each
// is kept; Dir stays as given.
func NewRefs(dir string, files map[string][]byte) (*Refs, error) {
	if len(files) > MaxExtraFiles {
		return nil, &config.ValidationError{Key: ErrKeyTooManyFiles, Args: []any{MaxExtraFiles}}
	}
	r := &Refs{Dir: dir, Files: map[string][]byte{}}
	total := 0
	for name, b := range files {
		base := RefBase(name)
		if base == "" {
			return nil, &config.ValidationError{Key: "api.err.import_content"}
		}
		if len(b) > MaxReferenceSize {
			return nil, &config.ValidationError{Key: ErrKeyRefTooBig, Args: []any{base, MaxReferenceSize >> 10}}
		}
		total += len(b)
		r.Files[base] = b
	}
	if total > MaxSize {
		return nil, TooBig()
	}
	return r, nil
}

// RefBase is the file name a reference or an added file is matched by:
// the last element, with \ and / both treated as separators (a config made
// on Windows names keys\ca.crt). "" when there is none.
func RefBase(p string) string {
	p = strings.TrimSpace(strings.ReplaceAll(p, `\`, "/"))
	if p == "" || strings.ContainsRune(p, 0) {
		return ""
	}
	b := path.Base(p)
	if b == "." || b == "/" || b == ".." {
		return ""
	}
	return b
}

func (r *Refs) file(base string) ([]byte, bool) {
	if b, ok := r.Files[base]; ok {
		return b, true
	}
	// Case-insensitive next (Windows and macOS file names), in a fixed order.
	names := make([]string, 0, len(r.Files))
	for n := range r.Files {
		names = append(names, n)
	}
	sort.Strings(names)
	for _, n := range names {
		if strings.EqualFold(n, base) {
			return r.Files[n], true
		}
	}
	return nil, false
}

// Resolve runs the helper's OpenVPN checks with refs as the source of
// every referenced file and returns the self-contained copy, the base names
// it put inside (in config order), or a translated refusal. Missing files
// are all collected first: the error lists every one of them.
func Resolve(content []byte, refs *Refs) ([]byte, []string, error) {
	var missing, included []string
	seen := map[string]bool{}
	read := func(p string) ([]byte, error) {
		base := RefBase(p)
		if base == "" {
			base = p
		}
		if b, ok := refs.file(base); ok {
			included = append(included, base)
			return b, nil
		}
		if refs.Dir != "" {
			b, err := readRef(refs.Dir, p, base)
			if err == nil {
				included = append(included, base)
				return b, nil
			}
			if !errors.Is(err, errRefMissing) {
				return nil, err
			}
		}
		if !seen[base] {
			seen[base] = true
			missing = append(missing, base)
		}
		return []byte{}, nil // placeholder: keep checking the rest
	}
	out, err := privops.SanitizeOpenVPNConfig(content, read)
	if err != nil {
		return nil, nil, reason(err)
	}
	if len(missing) > 0 {
		return nil, nil, &MissingFilesError{Names: missing}
	}
	return out, included, nil
}

var errRefMissing = errors.New("referenced file not found")

// readRef reads reference p (as written in the config) from inside dir.
// Outside dir, or not there → errRefMissing (the user can add it by hand);
// there but a link, a folder, a device, unreadable or swapped →
// api.err.import_ref_unreadable; too big → api.err.import_ref_too_big.
func readRef(dir, p, base string) ([]byte, error) {
	unreadable := &config.ValidationError{Key: ErrKeyRefUnreadable, Args: []any{base}}
	rel := strings.ReplaceAll(strings.TrimSpace(p), `\`, "/")
	if rel == "" || strings.ContainsRune(rel, 0) {
		return nil, errRefMissing
	}
	native := filepath.FromSlash(rel)
	var full string
	switch {
	case filepath.IsAbs(native):
		full = filepath.Clean(native)
	case filepath.VolumeName(native) != "", runtime.GOOS == "windows" && strings.Contains(rel, ":"):
		// C:ca.crt (drive-relative) or an alternate data stream: never.
		return nil, errRefMissing
	default:
		full = filepath.Join(dir, native)
	}
	dir = filepath.Clean(dir)
	inside, err := filepath.Rel(dir, full)
	if err != nil || inside == "." || inside == ".." || filepath.IsAbs(inside) ||
		strings.HasPrefix(inside, ".."+string(filepath.Separator)) {
		return nil, errRefMissing
	}
	// Every folder on the way must be a real folder (no link, no junction).
	parts := strings.Split(inside, string(filepath.Separator))
	cur := dir
	for _, part := range parts[:len(parts)-1] {
		cur = filepath.Join(cur, part)
		fi, err := os.Lstat(cur)
		if errors.Is(err, fs.ErrNotExist) {
			return nil, errRefMissing
		}
		if err != nil || !fi.IsDir() || fi.Mode()&(fs.ModeSymlink|fs.ModeIrregular) != 0 {
			return nil, unreadable
		}
	}
	before, err := os.Lstat(full)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, errRefMissing
	}
	if err != nil || !before.Mode().IsRegular() {
		return nil, unreadable
	}
	if before.Size() > MaxReferenceSize {
		return nil, &config.ValidationError{Key: ErrKeyRefTooBig, Args: []any{base, MaxReferenceSize >> 10}}
	}
	f, err := openNoFollow(full)
	if err != nil {
		return nil, unreadable
	}
	defer f.Close()
	after, err := f.Stat()
	if err != nil || !after.Mode().IsRegular() || !os.SameFile(before, after) {
		return nil, unreadable
	}
	b, err := io.ReadAll(io.LimitReader(f, MaxReferenceSize+1))
	if err != nil {
		return nil, unreadable
	}
	if len(b) > MaxReferenceSize {
		return nil, &config.ValidationError{Key: ErrKeyRefTooBig, Args: []any{base, MaxReferenceSize >> 10}}
	}
	return b, nil
}
