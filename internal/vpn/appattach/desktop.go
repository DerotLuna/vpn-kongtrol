package appattach

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/vpn-kongtrol/kongtrol/internal/vpn"
)

// Desktop describes a consumer VPN app controlled in the user session.
// Location commands run in order before connect and never receive credentials.
type Desktop struct {
	Type          string
	App           App
	Matcher       Matcher
	MacExecutable string
	DetectPaths   []string
	CLI           []string
	ConnectArgs   func(vpn.AdapterConfig) ([][]string, error)
}

var desktops = map[string]Desktop{} // registered by provider package init functions

func RegisterDesktop(d Desktop) {
	desktops[d.Type] = d
	RegisterMatcher(d.Type, d.Matcher)
	vpn.Register(d.Type, func() vpn.VPNAdapter { return &DesktopAdapter{desktop: d} })
}

func DesktopTypes() []string {
	types := make([]string, 0, len(desktops))
	for typ := range desktops {
		types = append(types, typ)
	}
	slices.Sort(types)
	return types
}

func DesktopDefinition(typ string) (Desktop, bool) {
	d, ok := desktops[typ]
	return d, ok
}

func (d Desktop) Locations() []string {
	var locations []string
	switch runtime.GOOS {
	case "windows":
		locations = d.App.WindowsPaths
	case "darwin":
		if d.MacExecutable != "" {
			for _, dir := range applicationDirectories() {
				locations = append(locations, filepath.Join(dir, d.App.MacApp+".app", "Contents", "MacOS", d.MacExecutable))
			}
		}
	case "linux":
		locations = d.App.LinuxCmds
	}
	return slices.Concat(locations, d.DetectPaths, d.CLI)
}

func applicationDirectories() []string {
	dirs := []string{"/Applications"}
	if home, err := os.UserHomeDir(); err == nil && filepath.IsAbs(home) {
		dirs = append(dirs, filepath.Join(home, "Applications"))
	}
	return dirs
}

func ProgramFolders(parts ...string) []string {
	var paths []string
	for _, name := range []string{"ProgramW6432", "ProgramFiles", "ProgramFiles(x86)"} {
		if base := os.Getenv(name); filepath.IsAbs(base) {
			paths = append(paths, filepath.Join(append([]string{base}, parts...)...))
		}
	}
	return paths
}

func CommandPaths(name string, windows, mac, linux []string) []string {
	switch runtime.GOOS {
	case "windows":
		return append(windows, name+".exe")
	case "darwin":
		return append(mac, name)
	case "linux":
		return append(linux, name)
	}
	return nil
}

// ExactLocation accepts one literal CLI argument, including spaces in a
// provider's region name. It cannot be interpreted as a CLI option.
func ExactLocation(value string) (string, error) {
	value = strings.TrimSpace(value)
	if strings.HasPrefix(value, "-") || strings.ContainsAny(value, "\x00\n\r") {
		return "", errors.New("invalid provider location")
	}
	return value, nil
}

type DesktopAdapter struct {
	desktop Desktop
	mu      sync.RWMutex
	cfg     vpn.AdapterConfig
	session Session
}

func (a *DesktopAdapter) Name() string    { return a.desktop.Type }
func (a *DesktopAdapter) Version() string { return "app" }
func (a *DesktopAdapter) Configure(cfg vpn.AdapterConfig) {
	cfg.Password = ""
	a.mu.Lock()
	a.cfg = cfg
	a.mu.Unlock()
}
func (a *DesktopAdapter) config() vpn.AdapterConfig {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.cfg
}

func (a *DesktopAdapter) provider(cfg vpn.AdapterConfig, withCLI bool) (Provider, Options) {
	options := ParseOptions(cfg.Extra)
	p := Provider{App: a.desktop.App, Matcher: a.desktop.Matcher.WithOverride(options.Interface)}
	if !withCLI || a.desktop.ConnectArgs == nil {
		return p, options
	}
	binary := ""
	for _, path := range a.desktop.CLI {
		if found, err := exec.LookPath(path); err == nil {
			binary = found
			break
		}
	}
	if binary == "" {
		return p, options
	}
	p.Location = strings.TrimSpace(cfg.Extra["server"])
	if p.Location == "" {
		p.Location = strings.Trim(strings.Join([]string{cfg.Extra["country"], cfg.Extra["city"]}, " "), " ")
	}
	p.AutoConnect = func(ctx context.Context) error {
		commands, err := a.desktop.ConnectArgs(cfg)
		if err != nil {
			return err
		}
		for _, args := range commands {
			if err := runDesktopCLI(ctx, binary, args...); err != nil {
				return err
			}
		}
		return nil
	}
	p.Disconnect = func(ctx context.Context) error { return runDesktopCLI(ctx, binary, "disconnect") }
	return p, options
}

func runDesktopCLI(ctx context.Context, binary string, args ...string) error {
	ctx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, binary, args...)
	vpn.HideChildWindow(cmd)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("%s %s: %w", filepath.Base(binary), args[0], err)
	}
	return nil
}

func (a *DesktopAdapter) Capabilities() vpn.Capabilities {
	p, o := a.provider(a.config(), true)
	return vpn.Capabilities{ManualReconnect: p.ManualReconnect(o)}
}
func (a *DesktopAdapter) Connect(ctx context.Context, cfg vpn.AdapterConfig) error {
	cfg.Password = ""
	a.Configure(cfg)
	p, o := a.provider(cfg, true)
	return a.session.Connect(ctx, p, o, cfg.OnAttach)
}
func (a *DesktopAdapter) Disconnect(ctx context.Context) error {
	p, o := a.provider(a.config(), true)
	return a.session.Disconnect(ctx, p, o)
}
func (a *DesktopAdapter) Reconnect(ctx context.Context) error {
	if err := a.Disconnect(ctx); err != nil {
		return err
	}
	return a.Connect(ctx, a.config())
}
func (a *DesktopAdapter) Status() vpn.Status {
	p, o := a.provider(a.config(), false)
	return a.session.Status(p, !o.AutoConnect || len(a.desktop.CLI) == 0).Normalize()
}
func (a *DesktopAdapter) TunnelInfo() (*vpn.TunnelInfo, error) {
	if a.Status() != vpn.StatusConnected {
		return nil, nil
	}
	return a.session.Info(), nil
}
func (a *DesktopAdapter) AttachState() *vpn.AttachState { return a.session.AttachState() }
