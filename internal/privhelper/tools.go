package privhelper

// Copies of VPN tools that live in user-owned prefixes (Homebrew,
// linuxbrew). The helper runs tools as root and only from root-owned
// locations, so `service install` copies them into privops.ToolsDir and
// records where each copy came from in a manifest. This file holds the
// OS-independent parts (selection, content checks, manifest compare) so
// they are tested everywhere; tools_unix.go does the copying.

import (
	"bytes"
	"crypto/sha256"
	"debug/elf"
	"debug/macho"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"sort"
	"strings"

	"github.com/vpn-kongtrol/kongtrol/internal/privops"
)

const (
	toolsFormat      = 1
	toolsManifest    = "manifest.json"
	maxToolSize      = 64 << 20
	maxManifestBytes = 1 << 20
)

// toolSpec says where one tool may live.
type toolSpec struct {
	Name string
	// System are root-owned locations the helper uses in place; when one
	// is trusted nothing is copied.
	System []string
	// Sources are fixed user-owned locations to copy from, in order.
	Sources []string
	// Requires names a tool that must be usable for this one to matter
	// (bash and wireguard-go are only needed by wg-quick).
	Requires string
}

// toolPick is one planned tool: used in place (System) or copied (Source).
type toolPick struct {
	Name     string
	Source   string
	System   bool
	Requires string
}

// planTools picks, for each spec in order, the trusted system copy or else
// the first source that exists. Specs whose required tool was not picked
// are skipped. trusted reports whether any of the given system paths is
// usable as is; exists whether a path resolves to a regular file.
func planTools(specs []toolSpec, trusted func(paths ...string) bool, exists func(path string) bool) []toolPick {
	picked := map[string]bool{}
	var out []toolPick
	for _, s := range specs {
		if s.Requires != "" && !picked[s.Requires] {
			continue
		}
		if len(s.System) > 0 && trusted(s.System...) {
			out = append(out, toolPick{Name: s.Name, System: true, Requires: s.Requires})
			picked[s.Name] = true
			continue
		}
		for _, src := range s.Sources {
			if exists(src) {
				out = append(out, toolPick{Name: s.Name, Source: src, Requires: s.Requires})
				picked[s.Name] = true
				break
			}
		}
	}
	return out
}

// ToolRecord is one copied tool in the manifest.
type ToolRecord struct {
	Source   string `json:"source"`   // fixed candidate path (often a symlink)
	Resolved string `json:"resolved"` // real file copied
	SHA256   string `json:"sha256"`   // of the source bytes
	Size     int64  `json:"size"`
}

// ToolSkip is a tool that was found but cannot be run by the helper.
type ToolSkip struct {
	Source string `json:"source"`
	Reason string `json:"reason"`
}

// ToolManifest is privops.ToolsDir/manifest.json (root-owned, 0644).
type ToolManifest struct {
	Format   int                   `json:"format"`
	Tools    map[string]ToolRecord `json:"tools"`
	Unusable map[string]ToolSkip   `json:"unusable,omitempty"`
}

// ToolChange is a copied tool whose source changed or vanished since install.
type ToolChange struct {
	Tool   string
	Source string
}

// ToolUnusable is a tool install found but refused to copy.
type ToolUnusable struct {
	Tool   string
	Source string
	Reason string
}

// changedTools re-hashes every recorded source with hash and lists those
// that differ or can no longer be read, sorted by tool name.
func changedTools(m ToolManifest, hash func(path string) (sum string, size int64, err error)) []ToolChange {
	var out []ToolChange
	for name, rec := range m.Tools {
		sum, size, err := hash(rec.Source)
		if err != nil || sum != rec.SHA256 || size != rec.Size {
			out = append(out, ToolChange{Tool: name, Source: rec.Source})
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Tool < out[j].Tool })
	return out
}

func unusableTools(m ToolManifest) []ToolUnusable {
	var out []ToolUnusable
	for name, s := range m.Unusable {
		out = append(out, ToolUnusable{Tool: name, Source: s.Source, Reason: s.Reason})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Tool < out[j].Tool })
	return out
}

// hashFile returns the SHA-256 and size of the file at p (symlinks
// followed), refusing anything that is not a regular file of at most
// maxToolSize bytes.
func hashFile(p string) (string, int64, error) {
	f, err := os.Open(p)
	if err != nil {
		return "", 0, err
	}
	defer f.Close()
	fi, err := f.Stat()
	if err != nil {
		return "", 0, err
	}
	if !fi.Mode().IsRegular() || fi.Size() > maxToolSize {
		return "", 0, fmt.Errorf("%s is not a regular file of at most %d bytes", p, maxToolSize)
	}
	h := sha256.New()
	n, err := io.Copy(h, io.LimitReader(f, maxToolSize+1))
	if err != nil {
		return "", 0, err
	}
	if n > maxToolSize {
		return "", 0, fmt.Errorf("%s is larger than %d bytes", p, maxToolSize)
	}
	return hex.EncodeToString(h.Sum(nil)), n, nil
}

// CheckTools reads the tools manifest (unprivileged) and reports copied
// tools whose source changed and tools install could not use. Without a
// manifest (nothing copied, or Windows) both lists are empty.
func CheckTools() ([]ToolChange, []ToolUnusable, error) {
	p := toolsManifestPath()
	if p == "" {
		return nil, nil, nil
	}
	f, err := os.Open(p)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil, nil
	}
	if err != nil {
		return nil, nil, err
	}
	defer f.Close()
	b, err := io.ReadAll(io.LimitReader(f, maxManifestBytes+1))
	if err != nil {
		return nil, nil, err
	}
	if len(b) > maxManifestBytes {
		return nil, nil, fmt.Errorf("privhelper: %s is too large", p)
	}
	var m ToolManifest
	if err := privops.DecodeStrict(b, &m); err != nil {
		return nil, nil, fmt.Errorf("privhelper: %s: %w", p, err)
	}
	if m.Format != toolsFormat {
		return nil, nil, fmt.Errorf("privhelper: %s format %d unsupported", p, m.Format)
	}
	return changedTools(m, hashFile), unusableTools(m), nil
}

// ── Content checks ──────────────────────────────────────────────────────
//
// A copy is frozen, but a binary that loads libraries from a user-owned
// prefix (Homebrew's openssl, linuxbrew's ld.so) could still be hijacked
// by swapping those libraries, so such binaries are refused. Scripts keep
// only trusted interpreters.

// prepareToolContent checks data (a tool's bytes) and returns what to
// install: binaries unchanged, scripts with a bash shebang rewritten to
// "#!/usr/bin/env bash" so the helper's PATH picks a trusted bash >= 4.
func prepareToolContent(data []byte) ([]byte, error) {
	switch {
	case bytes.HasPrefix(data, []byte("#!")):
		return prepareScript(data)
	case bytes.HasPrefix(data, []byte("\x7fELF")):
		return data, checkELF(data)
	case isMachO(data):
		return data, checkMachO(data)
	}
	return nil, errors.New("unknown executable format")
}

func prepareScript(data []byte) ([]byte, error) {
	line, rest, _ := bytes.Cut(data, []byte("\n"))
	fields := strings.Fields(strings.TrimSpace(strings.TrimPrefix(string(line), "#!")))
	if len(fields) == 0 {
		return nil, errors.New("script has an empty #! line")
	}
	interp := fields[0]
	switch {
	case interp == "/usr/bin/env" && len(fields) == 2 && fields[1] == "bash":
		return data, nil
	case path.Base(interp) == "bash" && len(fields) == 1:
		return append([]byte("#!/usr/bin/env bash\n"), rest...), nil
	case interp == "/bin/sh" && len(fields) == 1:
		return data, nil
	}
	return nil, fmt.Errorf("script interpreter %q is not trusted", strings.Join(fields, " "))
}

// systemLib reports whether p is a clean absolute path under one of prefixes.
func systemLib(p string, prefixes ...string) bool {
	return path.Clean(p) == p && underPrefix(p, prefixes)
}

func underPrefix(p string, prefixes []string) bool {
	for _, pre := range prefixes {
		if strings.HasPrefix(p, pre) {
			return true
		}
	}
	return false
}

var elfLibDirs = []string{"/lib/", "/lib64/", "/lib32/", "/usr/lib/", "/usr/lib64/", "/usr/lib32/"}

func checkELF(data []byte) error {
	f, err := elf.NewFile(bytes.NewReader(data))
	if err != nil {
		return err
	}
	defer f.Close()
	for _, p := range f.Progs {
		if p.Type != elf.PT_INTERP {
			continue
		}
		b, err := io.ReadAll(io.LimitReader(p.Open(), 4096))
		if err != nil {
			return err
		}
		if interp := strings.TrimRight(string(b), "\x00"); !systemLib(interp, elfLibDirs...) {
			return fmt.Errorf("uses the dynamic loader %s outside the system", interp)
		}
	}
	for _, tag := range []elf.DynTag{elf.DT_RPATH, elf.DT_RUNPATH} {
		vals, err := f.DynString(tag)
		if err != nil {
			return err
		}
		for _, v := range vals {
			for _, dir := range strings.Split(v, ":") {
				d := strings.TrimSuffix(dir, "/")
				if path.Clean(d) != d || !underPrefix(d+"/", elfLibDirs) {
					return fmt.Errorf("loads libraries from %q", dir)
				}
			}
		}
	}
	return nil
}

func isMachO(data []byte) bool {
	if len(data) < 4 {
		return false
	}
	switch binary.BigEndian.Uint32(data) {
	case 0xfeedface, 0xfeedfacf, 0xcefaedfe, 0xcffaedfe, 0xcafebabe:
		return true
	}
	return false
}

// Mach-O load commands that name a library, the loader or a search path.
// Each stores the path's offset at byte 8 of the command.
const (
	lcLoadDylib     = 0xc
	lcLoadDylinker  = 0xe
	lcLoadWeakDylib = 0x80000018
	lcRpath         = 0x8000001c
	lcReexportDylib = 0x8000001f
	lcLazyLoadDylib = 0x20
	lcLoadUpward    = 0x80000023
	lcDyldEnv       = 0x27
)

var machoLibDirs = []string{"/usr/lib/", "/System/Library/"}

func checkMachO(data []byte) error {
	r := bytes.NewReader(data)
	var files []*macho.File
	if binary.BigEndian.Uint32(data) == 0xcafebabe {
		fat, err := macho.NewFatFile(r)
		if err != nil {
			return err
		}
		defer fat.Close()
		for _, a := range fat.Arches {
			files = append(files, a.File)
		}
	} else {
		f, err := macho.NewFile(r)
		if err != nil {
			return err
		}
		defer f.Close()
		files = append(files, f)
	}
	for _, f := range files {
		for _, l := range f.Loads {
			raw := l.Raw()
			if len(raw) < 12 {
				continue
			}
			switch cmd := f.ByteOrder.Uint32(raw); cmd {
			case lcDyldEnv:
				return errors.New("sets dyld environment variables")
			case lcLoadDylib, lcLoadDylinker, lcLoadWeakDylib, lcRpath, lcReexportDylib, lcLazyLoadDylib, lcLoadUpward:
				off := f.ByteOrder.Uint32(raw[8:12])
				if int(off) >= len(raw) {
					return errors.New("malformed load command")
				}
				name, _, _ := bytes.Cut(raw[off:], []byte{0})
				if !systemLib(string(name), machoLibDirs...) {
					return fmt.Errorf("loads %s outside the system", name)
				}
			}
		}
	}
	return nil
}
