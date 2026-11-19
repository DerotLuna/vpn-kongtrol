// Package profileimport stores a WireGuard .conf or OpenVPN .ovpn file the
// user picked, after the same checks the privileged helper applies before it
// runs one (privops.ValidateWireGuardConfig / SanitizeOpenVPNConfig). It is
// the one implementation behind POST /api/v1/vpns/import (dashboard) and
// `kongtrol profile import` (CLI), so both accept and refuse the same files
// with the same translated errors (ADR-0042).
//
// Rules: the name is a plain file name (no separators, no hidden or Windows
// device names); the file lands in ~/.kongtrol/configs/<name>.conf|.ovpn
// with 0600; it is never replaced unless Overwrite is set (O_EXCL), never
// through a link or onto a folder; an imported .ovpn must be self-contained
// (no references to other files), so an import never makes Kongtrol read
// some other path.
package profileimport

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/vpn-kongtrol/kongtrol/internal/config"
	"github.com/vpn-kongtrol/kongtrol/internal/privops"
)

// MaxSize is the largest file accepted (the helper's own limit).
const MaxSize = privops.MaxConfigSize

// nameRE: letters, digits, dot, dash, underscore; starts with a letter or
// digit; ≤ 64. No separators, so no path traversal; no leading dot.
var nameRE = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]{0,63}$`)

// windowsReserved are device names Windows maps to devices in any folder.
var windowsReserved = map[string]bool{"con": true, "prn": true, "aux": true, "nul": true,
	"com1": true, "com2": true, "com3": true, "com4": true, "com5": true, "com6": true, "com7": true, "com8": true, "com9": true,
	"lpt1": true, "lpt2": true, "lpt3": true, "lpt4": true, "lpt5": true, "lpt6": true, "lpt7": true, "lpt8": true, "lpt9": true}

// Request is one import.
type Request struct {
	Name      string // file name without extension
	Type      string // wireguard | openvpn
	Content   []byte // the .conf / .ovpn file
	Overwrite bool   // replace an existing file of that name
}

// Result says where the file was stored.
type Result struct {
	Path        string `json:"path"`
	Type        string `json:"type"`
	Overwritten bool   `json:"overwritten"`
}

// ErrStore marks a failure writing the file (not the user's input).
var ErrStore = errors.New("store imported file")

// DefaultDir is ~/.kongtrol/configs.
func DefaultDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".kongtrol", "configs"), nil
}

// TypeForFile is the profile type a file name implies (.conf → wireguard,
// .ovpn → openvpn), or "" for any other extension.
func TypeForFile(path string) string {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".conf":
		return "wireguard"
	case ".ovpn":
		return "openvpn"
	}
	return ""
}

// CleanName turns a file path into a name Import accepts (and a valid
// profile name): the file name without extension, every character outside
// A-Z a-z 0-9 . _ - replaced by "-", leading characters that are not a
// letter or digit dropped, at most 64 characters. "" when nothing is left.
func CleanName(path string) string {
	base := filepath.Base(strings.ReplaceAll(path, `\`, "/"))
	base = strings.TrimSuffix(base, filepath.Ext(base))
	var b strings.Builder
	for _, r := range base {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '.', r == '_', r == '-':
			b.WriteRune(r)
		default:
			b.WriteByte('-')
		}
	}
	s := strings.TrimLeftFunc(b.String(), func(r rune) bool {
		return !(r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9')
	})
	for strings.Contains(s, "--") {
		s = strings.ReplaceAll(s, "--", "-")
	}
	s = strings.ReplaceAll(s, "..", ".")
	if len(s) > 64 {
		s = s[:64]
	}
	return strings.TrimRight(s, "-.")
}

// Import validates req and stores it in dir. A refusal is a
// *config.ValidationError (api.err.import_*: translated message + what to
// do); a write failure wraps ErrStore.
func Import(dir string, req Request) (Result, error) {
	name := strings.TrimSpace(req.Name)
	if !nameRE.MatchString(name) || strings.Contains(name, "..") || windowsReserved[strings.ToLower(strings.SplitN(name, ".", 2)[0])] {
		return Result{}, &config.ValidationError{Key: "api.err.import_name", Args: []any{name}}
	}
	typ := strings.ToLower(strings.TrimSpace(req.Type))
	var ext string
	switch typ {
	case "wireguard":
		ext = ".conf"
	case "openvpn":
		ext = ".ovpn"
	default:
		return Result{}, &config.ValidationError{Key: "api.err.import_type", Args: []any{req.Type}}
	}
	if len(req.Content) == 0 {
		return Result{}, &config.ValidationError{Key: "api.err.import_content"}
	}
	if len(req.Content) > MaxSize {
		return Result{}, TooBig()
	}
	if err := Validate(typ, req.Content); err != nil {
		return Result{}, err
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return Result{}, fmt.Errorf("%w: %w", ErrStore, err)
	}
	path := filepath.Join(dir, name+ext)
	overwritten, err := store(path, req.Content, req.Overwrite)
	switch {
	case errors.Is(err, fs.ErrExist):
		return Result{}, &config.ValidationError{Key: "api.err.import_exists", Args: []any{name + ext}}
	case errors.Is(err, errNotRegular):
		return Result{}, &config.ValidationError{Key: "api.err.import_not_file", Args: []any{name + ext}}
	case err != nil:
		return Result{}, fmt.Errorf("%w: %w", ErrStore, err)
	}
	return Result{Path: path, Type: typ, Overwritten: overwritten}, nil
}

// TooBig is the refusal for a file over MaxSize (callers that see the size
// first use it without reading the file).
func TooBig() error {
	return &config.ValidationError{Key: "api.err.import_too_big", Args: []any{MaxSize >> 10}}
}

// fileRefError is an .ovpn line that names another file.
type fileRefError struct{ path string }

func (e *fileRefError) Error() string {
	return fmt.Sprintf("it references the file %q: put its content inline (<ca>…</ca>, <cert>, <key>…)", e.path)
}

// Validate runs the helper's own checks for typ (wireguard | openvpn) and
// turns a refusal into a translated reason. An .ovpn must be
// self-contained: every file reference is refused (no reader).
func Validate(typ string, content []byte) error {
	var err error
	if typ == "wireguard" {
		err = privops.ValidateWireGuardConfig(content)
	} else {
		_, err = privops.SanitizeOpenVPNConfig(content, func(p string) ([]byte, error) {
			return nil, &fileRefError{path: p}
		})
	}
	if err == nil {
		return nil
	}
	return reason(err)
}

// The helper's refusals are plain errors with these shapes (privops
// wgconf.go / ovpnconf.go; profileimport_test.go runs the real validators,
// so a reworded message fails a test instead of losing the translation).
var (
	hookRE      = regexp.MustCompile(`line (\d+): (\w+) is not allowed \(runs commands`)
	directiveRE = regexp.MustCompile(`line (\d+): directive "([^"]+)" is not allowed`)
	scriptSecRE = regexp.MustCompile(`line (\d+): script-security above 1`)
	wgShapeRE   = regexp.MustCompile(`needs exactly one \[Interface\] with a PrivateKey`)
)

var wgHookNames = map[string]string{"preup": "PreUp", "postup": "PostUp", "predown": "PreDown", "postdown": "PostDown"}

// reason maps a validator error to the most specific api.err.import_* key;
// anything else stays api.err.import_invalid with the raw reason.
func reason(err error) error {
	var ref *fileRefError
	if errors.As(err, &ref) {
		return &config.ValidationError{Key: "api.err.import_file_ref", Args: []any{ref.path}}
	}
	msg := err.Error()
	line := func(s string) int { n, _ := strconv.Atoi(s); return n }
	if m := hookRE.FindStringSubmatch(msg); m != nil {
		name := wgHookNames[strings.ToLower(m[2])]
		if name == "" {
			name = m[2]
		}
		return &config.ValidationError{Key: "api.err.import_hook", Args: []any{name, line(m[1])}}
	}
	if m := directiveRE.FindStringSubmatch(msg); m != nil {
		return &config.ValidationError{Key: "api.err.import_directive", Args: []any{m[2], line(m[1])}}
	}
	if m := scriptSecRE.FindStringSubmatch(msg); m != nil {
		return &config.ValidationError{Key: "api.err.import_script_security", Args: []any{line(m[1])}}
	}
	if wgShapeRE.MatchString(msg) {
		return &config.ValidationError{Key: "api.err.import_not_wireguard"}
	}
	return &config.ValidationError{Key: "api.err.import_invalid", Args: []any{strings.TrimPrefix(msg, privops.ErrInvalidRequest.Error()+": ")}}
}

var errNotRegular = errors.New("not a regular file")

// store writes content to path with 0600. Without overwrite it creates the
// file exclusively (O_EXCL: never replaces, even in a race); with overwrite
// it replaces a regular file atomically, never a link or a folder.
func store(path string, content []byte, overwrite bool) (bool, error) {
	fi, err := os.Lstat(path)
	exists := err == nil
	if exists && !fi.Mode().IsRegular() {
		return false, errNotRegular
	}
	if !overwrite {
		f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
		if err != nil {
			return false, err // fs.ErrExist when it is there
		}
		_, werr := f.Write(content)
		cerr := f.Close()
		if err := errors.Join(werr, cerr); err != nil {
			_ = os.Remove(path)
			return false, err
		}
		return false, nil
	}
	return exists, config.WriteFileAtomic(path, content, 0o600)
}
