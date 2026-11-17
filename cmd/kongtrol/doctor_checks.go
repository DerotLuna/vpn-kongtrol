package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"sort"
	"strings"
	"time"

	"github.com/vpn-kongtrol/kongtrol/internal/config"
	"github.com/vpn-kongtrol/kongtrol/internal/daemonctl"
	"github.com/vpn-kongtrol/kongtrol/internal/monitor"
	"github.com/vpn-kongtrol/kongtrol/internal/privhelper"
	"github.com/vpn-kongtrol/kongtrol/internal/privops"
	"github.com/vpn-kongtrol/kongtrol/internal/routing"
	"github.com/vpn-kongtrol/kongtrol/internal/security"
	"github.com/vpn-kongtrol/kongtrol/internal/vpn"
	"github.com/vpn-kongtrol/kongtrol/internal/vpn/nordvpn"
	"github.com/vpn-kongtrol/kongtrol/internal/vpn/openvpn"
	"github.com/vpn-kongtrol/kongtrol/internal/vpn/protonvpn"
	"github.com/vpn-kongtrol/kongtrol/internal/vpndetect"
)

func (d *doctor) checkConfig() (string, *config.Config) {
	path := cfgPath
	if path == "" {
		for _, candidate := range config.DefaultPaths() {
			if _, err := os.Stat(candidate); err == nil {
				path = candidate
				break
			}
		}
	}

	if path == "" {
		d.fail(ct("cli.doctor.label.config_file"), ct("cli.doctor.config.not_found"))
		return "", nil
	}
	d.ok(ct("cli.doctor.label.config_file"), path)

	cfg, err := config.Load(path)
	if err != nil {
		d.fail(ct("cli.doctor.label.config_valid"), localizedErrText(cliLang(), err))
		return path, nil
	}
	d.ok(ct("cli.doctor.label.config_valid"), cf("cli.doctor.config.profiles_defined", len(cfg.VPNs)))
	return path, cfg
}

// ── binaries ──────────────────────────────────────────────────────────────────

// checkBinaries checks, per adapter type in use, the binary the adapter will
// actually execute on this OS (shared knowledge in internal/vpndetect), or the
// profile's binary_path override when set.
func (d *doctor) checkBinaries(cfg *config.Config) {
	if cfg == nil {
		d.warn(ct("cli.doctor.label.skipped"), ct("cli.doctor.skipped.no_valid_config"))
		return
	}

	overrides := make(map[string]string) // adapter type → first binary_path seen
	types := make([]string, 0, len(cfg.VPNs))
	for _, name := range sortedProfileNames(cfg) {
		v := cfg.VPNs[name]
		if _, seen := overrides[v.Type]; !seen {
			types = append(types, v.Type)
			overrides[v.Type] = ""
		}
		if overrides[v.Type] == "" && v.BinaryPath != "" {
			overrides[v.Type] = v.BinaryPath
		}
	}
	sort.Strings(types)

	for _, adapterType := range types {
		label := adapterType + " " + ct("cli.doctor.label.binary")
		// No CLI on this OS is not a failure: the provider adapter opens the
		// app and waits for the user to connect (attach mode).
		if adapterType == "protonvpn" && errors.Is(protonvpn.CheckPlatform(), protonvpn.ErrUnsupportedOS) {
			d.warn(label, ct("cli.doctor.binary.proton_os"))
			continue
		}
		if adapterType == "nordvpn" && errors.Is(nordvpn.CheckPlatform(), nordvpn.ErrUnsupportedOS) {
			d.warn(label, ct("cli.doctor.binary.nord_os"))
			continue
		}
		found, ok := doctorFindBinary(adapterType, overrides[adapterType])
		if !ok {
			if override := overrides[adapterType]; override != "" {
				d.fail(label, fmt.Sprintf(ct("cli.doctor.file.not_found"), override))
			} else if vpndetect.Known(adapterType) {
				d.fail(label, ct("cli.doctor.binary.not_found"))
			}
			continue
		}
		ver := doctorBinaryVersion(adapterType, found)
		if len(ver) > 48 {
			ver = ver[:48] + "…"
		}
		if ver == "" {
			d.ok(label, found)
		} else {
			d.ok(label, ver)
		}
		if adapterType == "forticlient" {
			d.checkFortiSupport(cfg)
		}
	}
}

func doctorFindBinary(adapterType, override string) (string, bool) {
	if adapterType == "openvpn" {
		p, err := openvpn.FindBinary(override)
		return p, err == nil
	}
	if override != "" {
		if fi, err := os.Stat(override); err == nil && fi.Mode().IsRegular() {
			return override, true
		}
		return "", false
	}
	return vpndetect.FindStandard(adapterType)
}

// doctorBinaryVersion runs the binary's version command with a timeout, or
// reads the file version for GUI apps that must not be launched.
func doctorBinaryVersion(adapterType, path string) string {
	ctx, cancel := context.WithTimeout(context.Background(), doctorVersionCheckTimeout)
	defer cancel()
	args := vpndetect.VersionArgs(adapterType)
	if len(args) == 0 {
		return fileVersion(ctx, path)
	}
	out, err := exec.CommandContext(ctx, path, args...).Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(strings.Split(string(out), "\n")[0])
}

func sortedProfileNames(cfg *config.Config) []string {
	names := make([]string, 0, len(cfg.VPNs))
	for name := range cfg.VPNs {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// ── certs & keys ──────────────────────────────────────────────────────────────

func (d *doctor) checkCerts(cfg *config.Config) {
	if cfg == nil {
		d.warn(ct("cli.doctor.label.skipped"), ct("cli.doctor.skipped.no_valid_config"))
		return
	}
	any := false
	for name, v := range cfg.VPNs {
		if v.Auth.Cert != "" {
			any = true
			d.checkFile(name+": "+ct("cli.doctor.label.cert"), v.Auth.Cert)
		}
		if v.Auth.Key != "" {
			any = true
			d.checkFile(name+": "+ct("cli.doctor.label.key"), v.Auth.Key)
		}
		if v.ConfigFile != "" {
			any = true
			d.checkFile(name+": "+ct("cli.doctor.label.config_path"), v.ConfigFile)
		}
	}
	if !any {
		d.ok(ct("cli.doctor.label.no_file_paths"), ct("cli.doctor.value.skipped"))
	}
}

func (d *doctor) checkFile(label, path string) {
	info, err := os.Stat(path)
	if err != nil {
		d.fail(label, fmt.Sprintf(ct("cli.doctor.file.not_found"), path))
		return
	}
	if info.IsDir() {
		d.fail(label, fmt.Sprintf(ct("cli.doctor.file.is_dir"), path))
		return
	}
	d.ok(label, path)
}

// ── keychain ──────────────────────────────────────────────────────────────────

func (d *doctor) checkKeychain(cfg *config.Config) {
	if cfg == nil {
		d.warn(ct("cli.doctor.label.skipped"), ct("cli.doctor.skipped.no_valid_config"))
		return
	}
	any := false
	for name, v := range cfg.VPNs {
		if v.Auth.PasswordKeychain != "" {
			any = true
			_, err := config.GetCredential(name, "password")
			if err != nil {
				d.fail(name+": password", ct("cli.doctor.keychain.not_found"))
			} else {
				d.ok(name+": password", ct("cli.doctor.keychain.found"))
			}
		}
		if v.Auth.UsernameKeychain != "" {
			any = true
			_, err := config.GetCredential(name, "username")
			if err != nil {
				d.fail(name+": username", ct("cli.doctor.keychain.not_found"))
			} else {
				d.ok(name+": username", ct("cli.doctor.keychain.found"))
			}
		}
	}
	if !any {
		d.ok(ct("cli.doctor.label.no_keychain"), ct("cli.doctor.value.skipped"))
	}
}

// ── permissions ───────────────────────────────────────────────────────────────

// checkPermissions reports whether privileged network changes (kill switch,
// DNS guard, routes, WireGuard/OpenVPN tunnels) can run: this process is
// elevated, or the Kongtrol helper is installed, answering and the same
// version. The kill switch / DNS guard lines only warn when neither holds.
func (d *doctor) checkPermissions() {
	ready := d.checkHelper()
	ksDetail, dnsDetail := ct("cli.doctor.permission.available"), ct("cli.doctor.permission.available")
	if !ready {
		missing := fmt.Sprintf(ct("cli.doctor.permission.maybe_elevated"), ct("cli.doctor.permission.required"))
		d.warn(ct("cli.doctor.label.kill_switch"), missing)
		d.warn(ct("cli.doctor.label.dns_guard"), missing)
	} else {
		d.ok(ct("cli.doctor.label.kill_switch"), ksDetail)
		if d.dnsGuardActive() {
			dnsDetail = ct("cli.doctor.dns.already_active")
		}
		d.ok(ct("cli.doctor.label.dns_guard"), dnsDetail)
	}
	if runtime.GOOS == "darwin" { // display note only
		d.warn(ct("cli.doctor.label.kill_switch"), ct("doctor.killswitch.macos_beta"))
	}
}

// checkHelper reports the helper line and whether privileged operations
// are available to the daemon.
func (d *doctor) checkHelper() bool {
	label := ct("cli.doctor.label.helper")
	if privops.IsElevated() {
		d.ok(label, ct("cli.doctor.helper.elevated"))
		return true
	}
	st := privhelper.QueryStatus()
	if !st.Installed {
		d.warn(label, ct("service.error.not_installed"))
		return false
	}
	if !st.Running {
		d.warn(label, ct("service.status.stopped")+" "+ct("service.status.stopped_hint"))
		return false
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	res, err := privops.NewClient().Ping(ctx)
	switch {
	case errors.Is(err, privops.ErrDenied):
		d.warn(label, ct("service.error.denied"))
		return false
	case err != nil:
		d.warn(label, ct("service.install.no_response")+" "+ct("service.install.no_response_hint"))
		return false
	case res.Protocol != privops.ProtocolVersion:
		d.warn(label, cf("service.status.version_mismatch", res.Version, version)+" "+ct("service.error.outdated"))
		return false
	case res.Version != version:
		// Same protocol: everything works, but suggest updating.
		d.warn(label, cf("service.status.version_mismatch", res.Version, version)+" "+ct("service.status.version_mismatch_hint"))
		return true
	}
	d.ok(label, cf("cli.doctor.helper.ok", res.Version))
	return true
}

// doctorRuntime holds what the runtime checks need, built without loadConfig:
// loadConfig also starts the metrics collector, creates the API token and opens
// the audit log, none of which a read-only diagnostic should do. Nothing here
// starts goroutines or connects anything.
type doctorRuntime struct {
	adapters map[string]vpn.VPNAdapter
	initErrs map[string]error
	routes   routing.RouteManager
	// daemon* hold the running daemon's own view (nil when none answers).
	// The daemon owns the tunnels, kill switch and DNS guard, so its state
	// wins over this process's fresh, never-connected adapters.
	daemonTunnels map[string]monitor.TunnelMetrics
	daemonSec     *apiSecurityStatus
}

func newDoctorRuntime(cfg *config.Config) *doctorRuntime {
	rt := &doctorRuntime{
		adapters: make(map[string]vpn.VPNAdapter),
		initErrs: make(map[string]error),
		routes:   routing.NewRouteManager(),
	}
	if cfg == nil {
		return rt
	}
	rt.loadDaemonState(cfg)
	for name, vpnCfg := range cfg.VPNs {
		a, err := vpn.New(vpnCfg.Type)
		if err != nil {
			rt.initErrs[name] = err
			continue
		}
		// Same pre-seed as loadConfig so Status() can probe tunnels started
		// outside this process (no credentials are read).
		if c, ok := a.(vpn.Configurable); ok {
			c.Configure(vpn.AdapterConfig{
				Host:       vpnCfg.Host,
				Port:       vpnCfg.Port,
				TunnelName: vpnCfg.TunnelName,
				CertPath:   vpnCfg.Auth.Cert,
				KeyPath:    vpnCfg.Auth.Key,
				ConfigPath: vpnCfg.ConfigFile,
				Username:   vpnCfg.Auth.Username,
				Extra:      vpnCfg.AdapterExtra(),
			})
		}
		rt.adapters[name] = a
	}
	return rt
}

func (d *doctor) checkRuntimeNetworking(cfg *config.Config) {
	if d.rt == nil || d.rt.routes == nil {
		d.warn(ct("cli.doctor.label.routes_probe"), ct("cli.doctor.routes.unavailable"))
	} else if routes, err := d.rt.routes.List(); err != nil {
		d.warn(ct("cli.doctor.label.routes_probe"), fmt.Sprintf(ct("cli.doctor.routes.read_failed"), err))
	} else {
		d.ok(ct("cli.doctor.label.routes_probe"), fmt.Sprintf(ct("cli.doctor.routes.visible"), len(routes)))
	}

	if cfg == nil {
		d.warn(ct("cli.doctor.label.dns_guard_runtime"), ct("cli.doctor.skipped.no_valid_config"))
		return
	}
	if d.rt != nil && d.rt.daemonSec != nil {
		s := d.rt.daemonSec
		d.ok(ct("cli.doctor.label.kill_switch_runtime"), killSwitchLabel(s.KillSwitchEnabled, s.KillSwitch, s.KillSwitchStatus))
	}
	if !cfg.Security.DNSGuard.Enabled {
		d.ok(ct("cli.doctor.label.dns_guard_runtime"), ct("cli.doctor.dns.disabled"))
		return
	}
	if d.dnsGuardActive() {
		d.ok(ct("cli.doctor.label.dns_guard_runtime"), ct("cli.doctor.dns.active"))
		return
	}
	d.warn(ct("cli.doctor.label.dns_guard_runtime"), ct("cli.doctor.dns.idle"))
}

func (d *doctor) checkAdapterStatus(cfg *config.Config) {
	if cfg == nil {
		d.warn(ct("cli.doctor.label.adapter_status"), ct("cli.doctor.skipped.no_valid_config"))
		return
	}
	for _, name := range sortedProfileNames(cfg) {
		label := name + ": " + ct("cli.doctor.label.adapter_status")
		adapter, ok := d.rt.adapters[name]
		if !ok {
			detail := ct("cli.doctor.adapter.not_initialized")
			if err := d.rt.initErrs[name]; err != nil {
				detail += " (" + err.Error() + ")"
			}
			d.fail(label, detail)
			continue
		}
		st := adapter.Status().Normalize()
		if t, ok := d.rt.daemonTunnels[name]; ok {
			st = t.Status.Normalize()
		}
		if st == "" {
			d.fail(label, ct("cli.doctor.adapter.empty_status"))
			continue
		}
		if st == vpn.StatusError {
			d.warn(label, cf("cli.doctor.adapter.error_state", name))
			continue
		}
		d.ok(label, string(st))
	}
}

// ── adapters ──────────────────────────────────────────────────────────────────

func (d *doctor) checkAdapters(cfg *config.Config) {
	registered := vpn.Registered()
	d.ok(ct("cli.doctor.label.registered_adapters"), strings.Join(registered, ", "))

	if cfg == nil {
		return
	}
	for name, v := range cfg.VPNs {
		found := false
		for _, r := range registered {
			if r == v.Type {
				found = true
				break
			}
		}
		if !found {
			d.fail(name+": "+ct("cli.doctor.label.adapter"), fmt.Sprintf(ct("cli.doctor.adapter.not_registered"), v.Type))
		}
	}
}

// ── output helpers ────────────────────────────────────────────────────────────

const checkWidth = 36

// loadDaemonState fills daemonTunnels/daemonSec when a daemon answers on the
// configured dashboard address. Read-only: the API token is read, never created.
func (rt *doctorRuntime) loadDaemonState(cfg *config.Config) {
	if apiToken == "" {
		tok, err := config.ReadAPIToken()
		if err != nil {
			return
		}
		apiToken = tok
	}
	base := daemonctl.APIBase(cfg.Monitor.Dashboard.Bind, cfg.Monitor.Dashboard.Port)
	tunnels, sec, err := fetchDaemonSnapshotAt(base)
	if err != nil {
		return
	}
	rt.setDaemonState(tunnels, sec)
}

func (rt *doctorRuntime) setDaemonState(tunnels []monitor.TunnelMetrics, sec *apiSecurityStatus) {
	rt.daemonTunnels = make(map[string]monitor.TunnelMetrics, len(tunnels))
	for _, t := range tunnels {
		rt.daemonTunnels[t.Name] = t
	}
	rt.daemonSec = sec
}

// dnsGuardActive prefers the daemon's answer; with no daemon it asks the OS
// whether a guard is in place (doctor has no in-memory manager of its own).
func (d *doctor) dnsGuardActive() bool {
	if d.rt != nil && d.rt.daemonSec != nil {
		return d.rt.daemonSec.DNSGuard
	}
	return security.NewDNSGuard().IsActive()
}
