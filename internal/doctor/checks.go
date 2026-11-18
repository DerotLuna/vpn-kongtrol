package doctor

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"runtime"
	"slices"
	"sort"
	"strings"
	"time"

	"github.com/vpn-kongtrol/kongtrol/internal/config"
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

// versionCheckTimeout bounds each VPN client binary's --version-style
// probe: a hung client binary must not make the diagnostic hang.
const versionCheckTimeout = 5 * time.Second

// Probes that touch the OS; tests replace them.
var (
	helperQueryStatus = privhelper.QueryStatus
	helperPing        = func(ctx context.Context) (privops.PingResult, error) {
		return privops.NewClient().Ping(ctx)
	}
	isElevated     = privops.IsElevated
	dnsGuardActive = func() bool { return security.NewDNSGuard().IsActive() }
	newRouteLister = func() routeLister { return routing.NewRouteManager() }
)

type routeLister interface {
	List() ([]routing.Route, error)
}

// ── configuration ─────────────────────────────────────────────────────────────

func (d *run) checkConfig() *config.Config {
	label := d.t("cli.doctor.label.config_file")
	path := d.opt.ConfigPath
	if path == "" {
		for _, candidate := range config.DefaultPaths() {
			if _, err := os.Stat(candidate); err == nil {
				path = candidate
				break
			}
		}
	}
	if path == "" {
		d.fail(label, d.t("cli.doctor.config.not_found"), d.t("doctor.hint.config_missing"))
		return nil
	}
	if _, err := os.Stat(path); err != nil {
		d.fail(label, d.f("cli.doctor.file.not_found", path), d.t("doctor.hint.config_missing"))
		return nil
	}
	d.ok(label, path)

	cfg, err := config.Load(path)
	if err != nil {
		msg, hint, ok := config.LocalizeError(err, d.opt.Lang)
		if !ok {
			msg, hint = err.Error(), d.t("doctor.hint.config_invalid")
		}
		d.fail(d.t("cli.doctor.label.config_valid"), msg, hint)
		return nil
	}
	d.ok(d.t("cli.doctor.label.config_valid"), d.f("cli.doctor.config.profiles_defined", len(cfg.VPNs)))
	return cfg
}

func (d *run) skipped() {
	d.warn(d.t("cli.doctor.label.skipped"), d.t("cli.doctor.skipped.no_valid_config"), d.t("doctor.hint.no_config"))
}

// ── binaries ──────────────────────────────────────────────────────────────────

// checkBinaries checks, per adapter type in use, the binary the adapter will
// actually execute on this OS (shared knowledge in internal/vpndetect), or the
// profile's binary_path override when set.
func (d *run) checkBinaries(cfg *config.Config) {
	if cfg == nil {
		d.skipped()
		return
	}
	overrides := make(map[string]string) // adapter type → first binary_path seen
	types := make([]string, 0, len(cfg.VPNs))
	for _, name := range SortedProfileNames(cfg) {
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
		label := adapterType + " " + d.t("cli.doctor.label.binary")
		// No CLI on this OS is not a failure: the provider adapter opens the
		// app and waits for the user to connect (attach mode).
		if adapterType == "protonvpn" && errors.Is(protonvpn.CheckPlatform(), protonvpn.ErrUnsupportedOS) {
			d.warn(label, d.t("cli.doctor.binary.proton_os"), "")
			continue
		}
		if adapterType == "nordvpn" && errors.Is(nordvpn.CheckPlatform(), nordvpn.ErrUnsupportedOS) {
			d.warn(label, d.t("cli.doctor.binary.nord_os"), "")
			continue
		}
		found, ok := findBinary(adapterType, overrides[adapterType])
		if !ok {
			if override := overrides[adapterType]; override != "" {
				d.fail(label, d.f("cli.doctor.file.not_found", override), d.t("doctor.hint.binary_override"))
			} else if vpndetect.Known(adapterType) {
				d.fail(label, d.t("cli.doctor.binary.not_found"), d.t("doctor.hint.binary_missing"))
			}
			continue
		}
		ver := binaryVersion(adapterType, found)
		if r := []rune(ver); len(r) > 48 {
			ver = string(r[:48]) + "…"
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

func findBinary(adapterType, override string) (string, bool) {
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

// binaryVersion runs the binary's version command with a timeout, or reads
// the file version for GUI apps that must not be launched.
func binaryVersion(adapterType, path string) string {
	args := vpndetect.VersionArgs(adapterType)
	if len(args) == 0 {
		return vpndetect.FileVersion(path)
	}
	ctx, cancel := context.WithTimeout(context.Background(), versionCheckTimeout)
	defer cancel()
	out, err := exec.CommandContext(ctx, path, args...).Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(strings.Split(string(out), "\n")[0])
}

// SortedProfileNames returns cfg's profile names in a stable order.
func SortedProfileNames(cfg *config.Config) []string {
	names := make([]string, 0, len(cfg.VPNs))
	for name := range cfg.VPNs {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// ── certs & keys ──────────────────────────────────────────────────────────────

func (d *run) checkCerts(cfg *config.Config) {
	if cfg == nil {
		d.skipped()
		return
	}
	any := false
	for _, name := range SortedProfileNames(cfg) {
		v := cfg.VPNs[name]
		if v.Auth.Cert != "" {
			any = true
			d.checkFile(name+": "+d.t("cli.doctor.label.cert"), v.Auth.Cert)
		}
		if v.Auth.Key != "" {
			any = true
			d.checkFile(name+": "+d.t("cli.doctor.label.key"), v.Auth.Key)
		}
		if v.ConfigFile != "" {
			any = true
			d.checkFile(name+": "+d.t("cli.doctor.label.config_path"), v.ConfigFile)
		}
	}
	if !any {
		d.ok(d.t("cli.doctor.label.no_file_paths"), d.t("cli.doctor.value.skipped"))
	}
}

func (d *run) checkFile(label, path string) {
	info, err := os.Stat(path)
	if err != nil {
		d.fail(label, d.f("cli.doctor.file.not_found", path), d.t("doctor.hint.file_missing"))
		return
	}
	if info.IsDir() {
		d.fail(label, d.f("cli.doctor.file.is_dir", path), d.t("doctor.hint.file_missing"))
		return
	}
	d.ok(label, path)
}

// ── keychain ──────────────────────────────────────────────────────────────────

// getCredential reads the OS keychain; tests replace it.
var getCredential = config.GetCredential

func (d *run) checkKeychain(cfg *config.Config) {
	if cfg == nil {
		d.skipped()
		return
	}
	any := false
	for _, name := range SortedProfileNames(cfg) {
		v := cfg.VPNs[name]
		for _, field := range []struct {
			set  bool
			kind string
		}{{v.Auth.PasswordKeychain != "", "password"}, {v.Auth.UsernameKeychain != "", "username"}} {
			if !field.set {
				continue
			}
			any = true
			if _, err := getCredential(name, field.kind); err != nil {
				d.fail(name+": "+field.kind, d.t("cli.doctor.keychain.not_found"), d.t("doctor.hint.keychain_missing"))
			} else {
				d.ok(name+": "+field.kind, d.t("cli.doctor.keychain.found"))
			}
		}
	}
	if !any {
		d.ok(d.t("cli.doctor.label.no_keychain"), d.t("cli.doctor.value.skipped"))
	}
}

// ── permissions ───────────────────────────────────────────────────────────────

// checkPermissions reports whether privileged network changes (kill switch,
// DNS guard, routes, WireGuard/OpenVPN tunnels) can run: this process is
// elevated, or the Kongtrol helper is installed, answering and the same
// version. The kill switch / DNS guard lines only warn when neither holds.
func (d *run) checkPermissions() {
	ready := d.checkHelper()
	if !ready {
		detail, hint := d.t("doctor.permission.needs_helper"), d.t("doctor.hint.helper_install")
		d.warnFix(d.t("cli.doctor.label.kill_switch"), detail, hint, FixHelperInstall)
		d.warnFix(d.t("cli.doctor.label.dns_guard"), detail, hint, FixHelperInstall)
	} else {
		d.ok(d.t("cli.doctor.label.kill_switch"), d.t("cli.doctor.permission.available"))
		dnsDetail := d.t("cli.doctor.permission.available")
		if d.dnsGuardActive() {
			dnsDetail = d.t("cli.doctor.dns.already_active")
		}
		d.ok(d.t("cli.doctor.label.dns_guard"), dnsDetail)
	}
	if runtime.GOOS == "darwin" { // display note only
		d.warn(d.t("cli.doctor.label.kill_switch"), d.t("doctor.killswitch.macos_beta"), "")
	}
}

// checkHelper reports the helper line and whether privileged operations
// are available.
func (d *run) checkHelper() bool {
	label := d.t("cli.doctor.label.helper")
	if isElevated() {
		d.ok(label, d.t("cli.doctor.helper.elevated"))
		return true
	}
	install := d.t("doctor.hint.helper_install")
	st := helperQueryStatus()
	if !st.Installed {
		d.warnFix(label, d.t("doctor.helper.missing"), install, FixHelperInstall)
		return false
	}
	if !st.Running {
		d.warnFix(label, d.t("service.status.stopped"), install, FixHelperInstall)
		return false
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	res, err := helperPing(ctx)
	update := d.t("doctor.hint.helper_update")
	switch {
	case errors.Is(err, privops.ErrDenied):
		d.warnFix(label, d.t("doctor.helper.denied"), install, FixHelperInstall)
		return false
	case err != nil:
		d.warnFix(label, d.t("service.install.no_response"), install, FixHelperInstall)
		return false
	case res.Protocol != privops.ProtocolVersion:
		d.warnFix(label, d.f("service.status.version_mismatch", res.Version, d.opt.Version), update, FixHelperInstall)
		return false
	case res.Version != d.opt.Version:
		// Same protocol: everything works, but suggest updating.
		d.warnFix(label, d.f("service.status.version_mismatch", res.Version, d.opt.Version), update, FixHelperInstall)
		return true
	}
	d.ok(label, d.f("cli.doctor.helper.ok", res.Version))
	return true
}

// ── runtime ───────────────────────────────────────────────────────────────────

// probes holds what the runtime checks need. Adapters are fresh and never
// connected: Status() only probes tunnels started elsewhere.
type probes struct {
	adapters map[string]vpn.VPNAdapter
	initErrs map[string]error
}

func newProbes(cfg *config.Config) *probes {
	p := &probes{adapters: map[string]vpn.VPNAdapter{}, initErrs: map[string]error{}}
	if cfg == nil {
		return p
	}
	for name, v := range cfg.VPNs {
		a, err := vpn.New(v.Type)
		if err != nil {
			p.initErrs[name] = err
			continue
		}
		// Same pre-seed as the daemon so Status() can probe tunnels started
		// outside this process (no credentials are read).
		if c, ok := a.(vpn.Configurable); ok {
			c.Configure(vpn.AdapterConfig{
				Host:       v.Host,
				Port:       v.Port,
				TunnelName: v.TunnelName,
				CertPath:   v.Auth.Cert,
				KeyPath:    v.Auth.Key,
				ConfigPath: v.ConfigFile,
				Username:   v.Auth.Username,
				Extra:      v.AdapterExtra(),
			})
		}
		p.adapters[name] = a
	}
	return p
}

func (d *run) checkRuntimeNetworking(cfg *config.Config) {
	label := d.t("cli.doctor.label.routes_probe")
	if routes, err := newRouteLister().List(); err != nil {
		d.warn(label, d.f("cli.doctor.routes.read_failed", err), d.t("doctor.hint.retry"))
	} else {
		d.ok(label, d.f("cli.doctor.routes.visible", len(routes)))
	}

	if cfg == nil {
		d.skipped()
		return
	}
	if dm := d.opt.Daemon; dm != nil {
		d.ok(d.t("cli.doctor.label.kill_switch_runtime"), d.killSwitchLabel(dm))
	}
	dnsLabel := d.t("cli.doctor.label.dns_guard_runtime")
	switch {
	case !cfg.Security.DNSGuard.Enabled:
		d.ok(dnsLabel, d.t("cli.doctor.dns.disabled"))
	case d.dnsGuardActive():
		d.ok(dnsLabel, d.t("cli.doctor.dns.active"))
	default:
		d.warn(dnsLabel, d.t("cli.doctor.dns.idle"), "")
	}
}

// killSwitchLabel: "ARMADO (protegiendo 2 túneles)" with the daemon's
// detail, else the plain state.
func (d *run) killSwitchLabel(dm *Daemon) string {
	if dm.KillSwitch != nil {
		key, args := dm.KillSwitch.StateKey()
		return d.f(key, args...)
	}
	switch {
	case dm.KillSwitchActive:
		return d.t("cli.status.security.active_caps")
	case dm.KillSwitchEnabled:
		return d.t("cli.status.security.armed")
	default:
		return d.t("cli.status.security.off_caps")
	}
}

// dnsGuardActive prefers the daemon's answer; with no daemon it asks the OS
// whether a guard is in place.
func (d *run) dnsGuardActive() bool {
	if d.opt.Daemon != nil {
		return d.opt.Daemon.DNSGuardActive
	}
	return dnsGuardActive()
}

func (d *run) checkAdapterStatus(cfg *config.Config) {
	if cfg == nil {
		d.warn(d.t("cli.doctor.label.adapter_status"), d.t("cli.doctor.skipped.no_valid_config"), d.t("doctor.hint.no_config"))
		return
	}
	for _, name := range SortedProfileNames(cfg) {
		label := name + ": " + d.t("cli.doctor.label.adapter_status")
		adapter, ok := d.rt.adapters[name]
		if !ok {
			detail := d.t("cli.doctor.adapter.not_initialized")
			if err := d.rt.initErrs[name]; err != nil {
				detail += " (" + err.Error() + ")"
			}
			d.fail(label, detail, d.t("doctor.hint.update"))
			continue
		}
		var st vpn.Status
		if s, ok := d.opt.Daemon.status(name); ok {
			st = s.Normalize()
		} else {
			st = adapter.Status().Normalize()
		}
		if st == "" {
			d.fail(label, d.t("cli.doctor.adapter.empty_status"), d.t("doctor.hint.update"))
			continue
		}
		if st == vpn.StatusError {
			d.warn(label, d.t("doctor.adapter.error_state"), d.f("doctor.hint.adapter_error", name))
			continue
		}
		d.ok(label, string(st))
	}
}

func (dm *Daemon) status(name string) (vpn.Status, bool) {
	if dm == nil {
		return "", false
	}
	st, ok := dm.Tunnels[name]
	return st, ok
}

// ── adapters ──────────────────────────────────────────────────────────────────

func (d *run) checkAdapters(cfg *config.Config) {
	registered := vpn.Registered()
	d.ok(d.t("cli.doctor.label.registered_adapters"), strings.Join(registered, ", "))
	if cfg == nil {
		return
	}
	for _, name := range SortedProfileNames(cfg) {
		v := cfg.VPNs[name]
		if !slices.Contains(registered, v.Type) {
			d.fail(name+": "+d.t("cli.doctor.label.adapter"), d.f("cli.doctor.adapter.not_registered", v.Type), d.t("doctor.hint.update"))
		}
	}
}
