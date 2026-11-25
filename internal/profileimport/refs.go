package profileimport

import (
	"errors"
	"path"
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
//     resolved relative to it, like OpenVPN GUI's --cd. Every folder on the
//     way must be a real folder (no link, no junction) and the file a
//     regular file (no link, no device), opened without following a link
//     and checked again after opening (no swap), at most
//     privops.MaxReferenceSize bytes, with the user's own rights (the daemon
//     and the CLI run as the user). For a dropped file (no path) Dir is the
//     folder of its identical original when Locate finds exactly one, or
//     the folder the user picked for its files (ADR-0112).
//   - Disk: an absolute path, or a ".." that leaves Dir, is read too, with
//     stricter checks (readOutside: local path only, real folders from the
//     root, PEM content only, ADR-0112). Off: they count as missing.
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
	// Disk allows references outside Dir (absolute, or "../" leaving it),
	// read with readOutside's checks (ADR-0112). Off for the elevated
	// root-only read, which releases only a self-contained profile.
	Disk bool
	// Locate finds the original of a dropped file (no Dir) when references
	// are missing (ADR-0112); nil = no search.
	Locate *Locator
}

// Translated refusals about referenced files (ADR-0105).
const (
	ErrKeyMissingFiles  = "api.err.import_missing_files"
	ErrKeyRefUnreadable = "api.err.import_ref_unreadable"
	ErrKeyRefTooBig     = "api.err.import_ref_too_big"
	ErrKeyRefNotText    = "api.err.import_ref_not_text"
	ErrKeyRefNotKey     = "api.err.import_ref_not_key" // outside the folder, not a PEM key (ADR-0112)
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
		b, err := refs.readDisk(p, base)
		if err == nil {
			included = append(included, base)
			return b, nil
		}
		if !errors.Is(err, errRefMissing) {
			return nil, err
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

// ReferencedPaths lists the file references of an .ovpn exactly as written
// (ca ca.crt → "ca.crt"), in order, without reading any of them. Nil when
// the config is refused for another reason (Resolve then says why).
func ReferencedPaths(content []byte) []string {
	var paths []string
	_, err := privops.SanitizeOpenVPNConfig(content, func(p string) ([]byte, error) {
		paths = append(paths, p)
		return []byte{}, nil
	})
	if err != nil {
		return nil
	}
	return paths
}
