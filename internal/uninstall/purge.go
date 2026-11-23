// Package uninstall plans and runs "remove Kongtrol" and "delete all my
// data" (ADR-0034). Keeping data is the default everywhere: nothing here runs
// unless the user asked for it (`kongtrol uninstall [--purge]`, the
// dashboard's "Borrar todos mis datos", or "Sí" in the Windows uninstaller).
package uninstall

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	keyring "github.com/zalando/go-keyring"
	"gopkg.in/yaml.v3"
)

// ManifestName is the file install.sh / install.ps1 write in the data dir to
// list what they installed, so `kongtrol uninstall` removes exactly that.
const ManifestName = "install.manifest"

// SystemCredential is the one keychain entry not tied to a profile: the
// audit log signing key (cmd/kongtrol/preferences.go).
var SystemCredential = Credential{Profile: "_system", Key: "audit_hmac_key"}

// Credential is one OS keychain entry Kongtrol stores (service
// "vpn-kongtrol", item "<profile>.<key>").
type Credential struct {
	Profile string
	Key     string
}

func (c Credential) String() string { return c.Profile + "." + c.Key }

// runtimeFiles stay when resetting from the running dashboard. The daemon
// and tray still use these files, including the flyout's WebView2 folder.
var runtimeFiles = map[string]bool{
	"api-token":  true,
	"run":        true,
	"daemon.log": true,
	"tray.log":   true,
	"webview2":   true,
	ManifestName: true,
}

// ConfigFiles returns the YAML files in dataDir that may name profiles:
// kongtrol.yaml, config.yaml and their backups (kongtrol.yaml.bak-…), plus
// any extra path (a config passed with --config elsewhere).
func ConfigFiles(dataDir string, extra ...string) []string {
	seen := map[string]bool{}
	var out []string
	add := func(p string) {
		if p == "" {
			return
		}
		p = filepath.Clean(p)
		if seen[p] {
			return
		}
		if st, err := os.Stat(p); err != nil || st.IsDir() {
			return
		}
		seen[p] = true
		out = append(out, p)
	}
	for _, pat := range []string{"*.yaml", "*.yml", "*.yaml.*", "*.yml.*"} {
		matches, _ := filepath.Glob(filepath.Join(dataDir, pat))
		for _, m := range matches {
			add(m)
		}
	}
	for _, p := range extra {
		add(p)
	}
	sort.Strings(out)
	return out
}

// ProfileNames reads the profile names (keys of `vpns:`) from every file.
// go-keyring cannot list entries, so this is how the purge knows which
// keychain items are Kongtrol's. Unreadable files are skipped.
func ProfileNames(files []string) []string {
	set := map[string]bool{}
	for _, f := range files {
		data, err := os.ReadFile(f)
		if err != nil {
			continue
		}
		var doc struct {
			VPNs map[string]yaml.Node `yaml:"vpns"`
		}
		if yaml.Unmarshal(data, &doc) != nil {
			continue
		}
		for name := range doc.VPNs {
			if name = strings.TrimSpace(name); name != "" {
				set[name] = true
			}
		}
	}
	out := make([]string, 0, len(set))
	for n := range set {
		out = append(out, n)
	}
	sort.Strings(out)
	return out
}

// Credentials lists exactly the keychain entries Kongtrol may have stored
// for these profiles: username and password of each, plus SystemCredential.
func Credentials(profiles []string) []Credential {
	out := make([]Credential, 0, 2*len(profiles)+1)
	for _, p := range profiles {
		out = append(out, Credential{p, "username"}, Credential{p, "password"})
	}
	return append(out, SystemCredential)
}

// KeychainService is the OS keychain service name (internal/config).
const KeychainService = "vpn-kongtrol"

// DeleteFromKeychain deletes one entry; a missing entry is not an error.
func DeleteFromKeychain(c Credential) error {
	err := keyring.Delete(KeychainService, c.String())
	if err == nil || errors.Is(err, keyring.ErrNotFound) {
		return nil
	}
	return err
}

// PurgeOptions says what to delete.
type PurgeOptions struct {
	DataDir string // ~/.kongtrol
	// ExtraFiles are files outside DataDir to delete too (a --config file,
	// the daily files of a custom audit log path); YAML ones are read for
	// profile names. Only regular files are removed (never RemoveAll).
	ExtraFiles []string
	// KeepRuntime keeps the files a running daemon and tray need
	// (api-token, run/, logs, WebView2) and the install manifest: the dashboard's "Borrar todos mis
	// datos", done by the running daemon itself.
	KeepRuntime bool
	// KeepManifest keeps only install.manifest: `--data-only` (the app
	// stays installed; the daemon is already stopped).
	KeepManifest bool
	DryRun       bool
	// Delete removes one keychain entry; nil = DeleteFromKeychain.
	Delete func(Credential) error
}

// Failure is one item that could not be deleted.
type Failure struct {
	Item string
	Err  error
}

// PurgeReport says what was (or, in a dry run, would be) deleted.
type PurgeReport struct {
	Credentials []Credential
	Paths       []string
	Failed      []Failure
}

// Purge deletes Kongtrol's keychain entries and its data directory. Profile
// names are read before any file is removed.
func Purge(opts PurgeOptions) PurgeReport {
	var rep PurgeReport
	del := opts.Delete
	if del == nil {
		del = DeleteFromKeychain
	}
	creds := Credentials(ProfileNames(ConfigFiles(opts.DataDir, opts.ExtraFiles...)))
	for _, c := range creds {
		if !opts.DryRun {
			if err := del(c); err != nil {
				rep.Failed = append(rep.Failed, Failure{Item: "keychain:" + c.String(), Err: err})
				continue
			}
		}
		rep.Credentials = append(rep.Credentials, c)
	}
	for _, p := range opts.ExtraFiles {
		if p == "" || isWithin(opts.DataDir, p) {
			continue
		}
		rep.removeFile(p, opts.DryRun)
	}
	entries, err := os.ReadDir(opts.DataDir)
	if err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			rep.Failed = append(rep.Failed, Failure{Item: opts.DataDir, Err: err})
		}
		return rep
	}
	if !opts.KeepRuntime && !opts.KeepManifest {
		rep.remove(opts.DataDir, opts.DryRun)
		return rep
	}
	for _, e := range entries {
		if (opts.KeepRuntime && runtimeFiles[e.Name()]) || e.Name() == ManifestName {
			continue
		}
		rep.remove(filepath.Join(opts.DataDir, e.Name()), opts.DryRun)
	}
	return rep
}

func (r *PurgeReport) remove(path string, dry bool) {
	if !dry {
		if err := os.RemoveAll(path); err != nil {
			r.Failed = append(r.Failed, Failure{Item: path, Err: err})
			return
		}
	}
	r.Paths = append(r.Paths, path)
}

// removeFile deletes one extra file outside the data directory. These
// paths come from settings (a --config file, security.audit_log.path), so
// only a regular file is removed — never a directory, symlink or device,
// and never recursively.
func (r *PurgeReport) removeFile(path string, dry bool) {
	fi, err := os.Lstat(path)
	if err != nil || !fi.Mode().IsRegular() {
		return
	}
	if !dry {
		if err := os.Remove(path); err != nil {
			r.Failed = append(r.Failed, Failure{Item: path, Err: err})
			return
		}
	}
	r.Paths = append(r.Paths, path)
}

func isWithin(dir, p string) bool {
	rel, err := filepath.Rel(dir, p)
	return err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}

// Err summarizes failures as one error (nil when everything went).
func (r PurgeReport) Err() error {
	if len(r.Failed) == 0 {
		return nil
	}
	parts := make([]string, 0, len(r.Failed))
	for _, f := range r.Failed {
		parts = append(parts, fmt.Sprintf("%s: %v", f.Item, f.Err))
	}
	return errors.New(strings.Join(parts, "; "))
}
