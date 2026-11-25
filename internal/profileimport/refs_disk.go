package profileimport

import (
	"errors"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"

	"github.com/vpn-kongtrol/kongtrol/internal/config"
)

// Reading the files an .ovpn names from disk (ADR-0105, ADR-0112).

var errRefMissing = errors.New("referenced file not found")

// refPath turns reference p (as written in the config) into a full path,
// and says whether it lies inside dir. ok is false for what is never
// opened: empty, NUL, a drive-relative C:ca.crt or an alternate data
// stream, or a relative reference with no folder to resolve it in.
func refPath(dir, p string) (full string, inside, ok bool) {
	rel := strings.ReplaceAll(strings.TrimSpace(p), `\`, "/")
	if rel == "" || strings.ContainsRune(rel, 0) {
		return "", false, false
	}
	native := filepath.FromSlash(rel)
	switch {
	case filepath.IsAbs(native):
		full = filepath.Clean(native)
	case filepath.VolumeName(native) != "", runtime.GOOS == "windows" && strings.Contains(rel, ":"):
		return "", false, false
	case dir == "":
		return "", false, false
	default:
		full = filepath.Join(dir, native)
	}
	if dir == "" {
		return full, false, true
	}
	dir = filepath.Clean(dir)
	in, err := filepath.Rel(dir, full)
	inside = err == nil && in != "." && in != ".." && !filepath.IsAbs(in) &&
		!strings.HasPrefix(in, ".."+string(filepath.Separator))
	return full, inside, true
}

// readDisk reads reference p from disk: inside Dir with readInside; an
// absolute path or a "../" that leaves Dir only when Disk is set, with
// readOutside. Anything else → errRefMissing (the user can add it).
func (r *Refs) readDisk(p, base string) ([]byte, error) {
	full, inside, ok := refPath(r.Dir, p)
	switch {
	case !ok:
		return nil, errRefMissing
	case inside:
		return readInside(r.Dir, full, base)
	case r.Disk:
		return readOutside(full, base)
	}
	return nil, errRefMissing
}

// readInside reads full, a path inside dir: every folder on the way from
// dir must be a real folder (no link, no junction).
func readInside(dir, full, base string) ([]byte, error) {
	unreadable := &config.ValidationError{Key: ErrKeyRefUnreadable, Args: []any{base}}
	dir = filepath.Clean(dir)
	in, err := filepath.Rel(dir, full)
	if err != nil {
		return nil, errRefMissing
	}
	parts := strings.Split(in, string(filepath.Separator))
	cur := dir
	for _, part := range parts[:len(parts)-1] {
		cur = filepath.Join(cur, part)
		fi, err := os.Lstat(cur)
		if errors.Is(err, fs.ErrNotExist) {
			return nil, errRefMissing
		}
		if err != nil || !fi.IsDir() || !plain(cur, fi) {
			return nil, unreadable
		}
	}
	return readRefFile(full, base)
}

// readOutside reads a reference outside the .ovpn's folder: an absolute
// path, or a "../" that leaves it. OpenVPN GUI would read it with the same
// rights (the daemon and the CLI run as the user), so it is read too, with
// stricter checks: a plain local path (CleanLocalPath: no UNC share, no
// device or /proc tree), not on a network drive, every folder from the
// root down a real folder, the file a regular file of at most
// MaxReferenceSize, and its content PEM certificates or keys (or an
// OpenVPN static key) only: any other file (an SSH key in OpenSSH format, a
// token, a document) is refused with api.err.import_ref_not_key. A path
// that is not local or not there counts as missing and is never opened.
func readOutside(full, base string) ([]byte, error) {
	p, ok := CleanLocalPath(full)
	if !ok || onRemoteDrive(p) {
		return nil, errRefMissing
	}
	unreadable := &config.ValidationError{Key: ErrKeyRefUnreadable, Args: []any{base}}
	var chain []string
	for d := filepath.Dir(p); ; d = filepath.Dir(d) {
		chain = append(chain, d)
		if filepath.Dir(d) == d {
			break
		}
	}
	for i := len(chain) - 1; i >= 0; i-- {
		fi, err := os.Lstat(chain[i])
		if errors.Is(err, fs.ErrNotExist) {
			return nil, errRefMissing
		}
		if err != nil || !fi.IsDir() || !plain(chain[i], fi) {
			return nil, unreadable
		}
	}
	b, err := readRefFile(p, base)
	if err != nil {
		return nil, err
	}
	if !LooksLikeKeyFile(b) {
		return nil, &config.ValidationError{Key: ErrKeyRefNotKey, Args: []any{base}}
	}
	return b, nil
}

// readRefFile reads full: there but a link, a folder, a device, unreadable
// or swapped → api.err.import_ref_unreadable; not there → errRefMissing;
// too big → api.err.import_ref_too_big. Opened without following a link
// (O_NOFOLLOW on Unix) and checked again after opening (os.SameFile).
func readRefFile(full, base string) ([]byte, error) {
	unreadable := &config.ValidationError{Key: ErrKeyRefUnreadable, Args: []any{base}}
	before, err := os.Lstat(full)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, errRefMissing
	}
	if err != nil || before.IsDir() || !plain(full, before) {
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
	if err != nil || after.IsDir() || after.Mode()&notFile != 0 || !os.SameFile(before, after) {
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

const notFile = fs.ModeSymlink | fs.ModeNamedPipe | fs.ModeSocket | fs.ModeDevice | fs.ModeCharDevice

// pemBegin finds PEM headers ("-----BEGIN CERTIFICATE-----", RFC 7468).
var pemBegin = regexp.MustCompile(`(?m)^[ \t]*-----BEGIN ([A-Za-z0-9 .-]+)-----[ \t\r]*$`)

// keyLabels are the PEM blocks an OpenVPN file directive takes:
// certificates (ca, cert, extra-certs), CRLs (crl-verify), private keys
// (key), DH parameters (dh) and OpenVPN's own static and tls-crypt-v2
// client keys (tls-auth, tls-crypt, secret, tls-crypt-v2).
var keyLabels = map[string]bool{
	"CERTIFICATE": true, "TRUSTED CERTIFICATE": true, "X509 CRL": true,
	"PRIVATE KEY": true, "RSA PRIVATE KEY": true, "EC PRIVATE KEY": true, "ENCRYPTED PRIVATE KEY": true,
	"DH PARAMETERS": true, "X9.42 DH PARAMETERS": true,
	"OPENVPN STATIC KEY V1": true, "OPENVPN TLS-CRYPT-V2 CLIENT KEY": true,
}

// LooksLikeKeyFile: b holds at least one PEM block and every PEM block in
// it is one an OpenVPN directive takes (keyLabels). Text around the blocks
// (openssl's "Bag Attributes", comments) is allowed.
func LooksLikeKeyFile(b []byte) bool {
	ms := pemBegin.FindAllSubmatch(b, -1)
	if len(ms) == 0 {
		return false
	}
	for _, m := range ms {
		if !keyLabels[strings.ToUpper(strings.TrimSpace(string(m[1])))] {
			return false
		}
	}
	return true
}
