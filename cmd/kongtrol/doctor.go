package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/spf13/cobra"

	"github.com/vpn-kongtrol/kongtrol/internal/config"
	"github.com/vpn-kongtrol/kongtrol/internal/security"
	"github.com/vpn-kongtrol/kongtrol/internal/vpn"
)

// doctorVersionCheckTimeout bounds each VPN client binary's --version-style
// probe. doctor is meant to be a quick, safe diagnostic — a misbehaving or
// hung client binary (or a PowerShell VersionInfo lookup that never
// returns) must not be able to make it hang indefinitely.
const doctorVersionCheckTimeout = 5 * time.Second

var doctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "Check your environment — binaries, certs, keychain, permissions",
	Long: `doctor validates your full Kongtrol stack without connecting to any VPN.
Run this before your first connection, or when diagnosing a teammate's setup.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if doctorFix {
			n, err := applyDoctorFixes()
			if err != nil {
				return err
			}
			if !outputJSON {
				fmt.Println(tuiInfo(cf("cli.doctor.fix.applied", n)))
			}
		}
		d := &doctor{}
		d.run()
		if outputJSON {
			if err := emitJSON(d.report()); err != nil {
				return err
			}
		} else {
			if d.warnings > 0 {
				fmt.Printf("\n%s\n", tuiWarn(cf("cli.doctor.summary.warn", d.warnings)))
			}
			if d.failures == 0 {
				fmt.Println("\n" + tuiOK(ct("cli.doctor.summary.ok")))
			}
		}
		if d.failures > 0 {
			return fmt.Errorf("%s", cf("cli.doctor.error.failed_checks", d.failures))
		}
		return nil
	},
}

var doctorFix bool

func init() {
	doctorCmd.Short = ct("cli.doctor.short")
	doctorCmd.Long = ct("cli.doctor.long")
	doctorCmd.Example = ct("cli.doctor.examples")
	doctorCmd.Flags().BoolVar(&doctorFix, "fix", false, ct("cli.doctor.flag.fix"))
	rootCmd.AddCommand(doctorCmd)
}

func applyDoctorFixes() (int, error) {
	applied := 0
	runDir := filepath.Join(filepath.Dir(pidFilePath()))
	if err := os.MkdirAll(runDir, 0o700); err == nil {
		applied++
	}
	paths := config.DefaultPaths()
	for _, p := range paths {
		if p == "" {
			continue
		}
		if err := os.MkdirAll(filepath.Dir(p), 0o700); err == nil {
			applied++
		}
	}
	var cfgLoaded *config.Config
	if cfgPath != "" {
		if c, err := config.Load(cfgPath); err == nil {
			cfgLoaded = c
		}
	} else {
		for _, p := range paths {
			if c, err := config.Load(p); err == nil {
				cfgLoaded = c
				break
			}
		}
	}
	if cfgLoaded != nil && cfgLoaded.Security.AuditLog.Path != "" {
		if err := os.MkdirAll(filepath.Dir(cfgLoaded.Security.AuditLog.Path), 0o700); err == nil {
			applied++
		}
		todayAudit := security.DailyAuditLogPath(cfgLoaded.Security.AuditLog.Path, time.Now())
		if _, err := os.Stat(todayAudit); os.IsNotExist(err) {
			if err := os.WriteFile(todayAudit, []byte{}, 0o600); err == nil {
				applied++
			}
		}
	}
	return applied, nil
}

type doctorCheck struct {
	Section string `json:"section"`
	Label   string `json:"label"`
	Status  string `json:"status"` // ok|warn|fail
	Detail  string `json:"detail"`
}

type doctorReport struct {
	Checks   []doctorCheck `json:"checks"`
	Failures int           `json:"failures"`
	Warnings int           `json:"warnings"`
}

// ── doctor ────────────────────────────────────────────────────────────────────

type doctor struct {
	failures int
	warnings int
	sectionN string
	checks   []doctorCheck
}

func (d *doctor) run() {
	if !outputJSON {
		fmt.Println(stylePrompt.Render(sym("◎", "#")) + "  " + styleBright.Render(ct("cli.doctor.title")))
		fmt.Println(styleDim.Render(doctorRule()))
	}

	d.section(ct("cli.doctor.section.configuration"))
	_, cfg := d.checkConfig()

	d.section(ct("cli.doctor.section.binaries"))
	d.checkBinaries(cfg)

	d.section(ct("cli.doctor.section.certs"))
	d.checkCerts(cfg)

	d.section(ct("cli.doctor.section.keychain"))
	d.checkKeychain(cfg)

	d.section(ct("cli.doctor.section.permissions"))
	d.checkPermissions()

	d.section(ct("cli.doctor.section.runtime"))
	d.checkRuntimeNetworking(cfg)

	d.section(ct("cli.doctor.section.adapters_status"))
	d.checkAdapterStatus(cfg)

	d.section(ct("cli.doctor.section.adapters_registered"))
	d.checkAdapters(cfg)
}

func (d *doctor) report() doctorReport {
	return doctorReport{
		Checks:   d.checks,
		Failures: d.failures,
		Warnings: d.warnings,
	}
}

// ── config ────────────────────────────────────────────────────────────────────

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
		d.fail(ct("cli.doctor.label.config_valid"), err.Error())
		return path, nil
	}
	d.ok(ct("cli.doctor.label.config_valid"), cf("cli.doctor.config.profiles_defined", len(cfg.VPNs)))
	return path, cfg
}

// ── binaries ──────────────────────────────────────────────────────────────────

type binarySpec struct {
	adapterType string
	candidates  []string
	versionArg  string
	guiApp      bool // if true, read file version via PowerShell instead of launching
}

func vpnBinarySpecs() []binarySpec {
	specs := []binarySpec{
		{adapterType: "openvpn", candidates: []string{"openvpn"}, versionArg: "--version"},
		{adapterType: "protonvpn", candidates: []string{"protonvpn-cli"}, versionArg: "--version"},
		{adapterType: "tailscale", candidates: []string{"tailscale"}, versionArg: "version"},
		{adapterType: "cloudflarewarp", candidates: []string{"warp-cli"}, versionArg: "--version"},
		{adapterType: "wireguard", candidates: []string{"wg", "wg-quick"}, versionArg: "--version"},
	}

	switch runtime.GOOS {
	case "windows":
		specs = append(specs,
			binarySpec{
				adapterType: "forticlient",
				candidates: []string{
					`C:\Program Files\Fortinet\FortiClient\FortiClient.exe`,
					`C:\Program Files (x86)\Fortinet\FortiClient\FortiClient.exe`,
				},
				versionArg: "--version",
				guiApp:     true,
			},
			binarySpec{
				adapterType: "ciscoanyconnect",
				candidates: []string{
					`C:\Program Files (x86)\Cisco\Cisco AnyConnect Secure Mobility Client\vpncli.exe`,
					`C:\Program Files\Cisco\Cisco AnyConnect Secure Mobility Client\vpncli.exe`,
				},
				versionArg: "-v",
			},
			binarySpec{
				adapterType: "globalprotect",
				candidates: []string{
					`C:\Program Files\Palo Alto Networks\GlobalProtect\PanGPS.exe`,
				},
				versionArg: "--version",
			},
		)
	case "darwin":
		specs = append(specs,
			binarySpec{
				adapterType: "forticlient",
				candidates:  []string{"/Applications/FortiClient.app/Contents/MacOS/FortiClient"},
				versionArg:  "--version",
			},
			binarySpec{
				adapterType: "ciscoanyconnect",
				candidates:  []string{"/opt/cisco/anyconnect/bin/vpn"},
				versionArg:  "-v",
			},
			binarySpec{
				adapterType: "globalprotect",
				candidates:  []string{"/Applications/GlobalProtect.app/Contents/MacOS/GlobalProtect"},
				versionArg:  "--version",
			},
		)
	case "linux":
		specs = append(specs,
			binarySpec{
				adapterType: "forticlient",
				candidates:  []string{"forticlientsslvpn", "/usr/lib/forticlient/forticlientsslvpn"},
				versionArg:  "--version",
			},
			binarySpec{
				adapterType: "ciscoanyconnect",
				candidates:  []string{"/opt/cisco/anyconnect/bin/vpn"},
				versionArg:  "-v",
			},
		)
	}
	return specs
}

func (d *doctor) checkBinaries(cfg *config.Config) {
	if cfg == nil {
		d.warn(ct("cli.doctor.label.skipped"), ct("cli.doctor.skipped.no_valid_config"))
		return
	}

	used := make(map[string]bool)
	for _, v := range cfg.VPNs {
		used[v.Type] = true
	}

	for _, spec := range vpnBinarySpecs() {
		if !used[spec.adapterType] {
			continue
		}
		found := ""
		for _, candidate := range spec.candidates {
			if _, err := os.Stat(candidate); err == nil {
				found = candidate
				break
			}
			if path, err := exec.LookPath(candidate); err == nil {
				found = path
				break
			}
		}
		if found == "" {
			d.fail(spec.adapterType+" "+ct("cli.doctor.label.binary"), ct("cli.doctor.binary.not_found"))
			continue
		}
		ver := ""
		versionCtx, cancelVersion := context.WithTimeout(context.Background(), doctorVersionCheckTimeout)
		if spec.guiApp {
			if out, err := exec.CommandContext(versionCtx, "powershell", "-NoProfile", "-Command",
				fmt.Sprintf(`(Get-Item '%s').VersionInfo.ProductVersion`, found)).Output(); err == nil {
				ver = strings.TrimSpace(string(out))
			}
		} else if out, err := exec.CommandContext(versionCtx, found, spec.versionArg).Output(); err == nil {
			ver = strings.TrimSpace(strings.Split(string(out), "\n")[0])
		}
		cancelVersion()
		if len(ver) > 48 {
			ver = ver[:48] + "…"
		}
		if ver == "" {
			d.ok(spec.adapterType+" "+ct("cli.doctor.label.binary"), found)
		} else {
			d.ok(spec.adapterType+" "+ct("cli.doctor.label.binary"), ver)
		}
	}
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

func (d *doctor) checkPermissions() {
	if hasNetworkAdminPrivileges() {
		d.ok(ct("cli.doctor.label.kill_switch"), ct("cli.doctor.permission.available"))
	} else {
		d.warn(ct("cli.doctor.label.kill_switch"), fmt.Sprintf(ct("cli.doctor.permission.maybe_elevated"), ct("cli.doctor.permission.required")))
	}

	dnsGuard := security.NewDNSGuard()
	if dnsGuard.IsActive() {
		d.ok(ct("cli.doctor.label.dns_guard"), ct("cli.doctor.dns.already_active"))
		return
	}
	if runtime.GOOS == "linux" {
		f, err := os.OpenFile("/etc/resolv.conf", os.O_WRONLY|os.O_APPEND, 0)
		if err != nil {
			d.warn(ct("cli.doctor.label.dns_guard"), ct("cli.doctor.dns.needs_sudo"))
			return
		}
		f.Close()
		d.ok(ct("cli.doctor.label.dns_guard"), ct("cli.doctor.dns.writable"))
		return
	}
	d.ok(ct("cli.doctor.label.dns_guard"), ct("cli.doctor.permission.available"))
}

func (d *doctor) checkRuntimeNetworking(cfg *config.Config) {
	if routeMgr == nil {
		d.warn(ct("cli.doctor.label.routes_probe"), ct("cli.doctor.routes.unavailable"))
	} else if routes, err := routeMgr.List(); err != nil {
		d.warn(ct("cli.doctor.label.routes_probe"), fmt.Sprintf(ct("cli.doctor.routes.read_failed"), err))
	} else {
		d.ok(ct("cli.doctor.label.routes_probe"), fmt.Sprintf(ct("cli.doctor.routes.visible"), len(routes)))
	}

	if cfg == nil {
		d.warn(ct("cli.doctor.label.dns_guard_runtime"), ct("cli.doctor.skipped.no_valid_config"))
		return
	}
	if !cfg.Security.DNSGuard.Enabled {
		d.ok(ct("cli.doctor.label.dns_guard_runtime"), ct("cli.doctor.dns.disabled"))
		return
	}
	if dnsMgr != nil && dnsMgr.IsActive() {
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
	for name := range cfg.VPNs {
		adapter, ok := adapters[name]
		if !ok {
			d.fail(name+": "+ct("cli.doctor.label.adapter_status"), ct("cli.doctor.adapter.not_initialized"))
			continue
		}
		st := adapter.Status().Normalize()
		if st == "" {
			d.fail(name+": "+ct("cli.doctor.label.adapter_status"), ct("cli.doctor.adapter.empty_status"))
			continue
		}
		if st == vpn.StatusError {
			d.warn(name+": "+ct("cli.doctor.label.adapter_status"), cf("cli.doctor.adapter.error_state", name))
			continue
		}
		d.ok(name+": "+ct("cli.doctor.label.adapter_status"), string(st))
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

func (d *doctor) section(name string) {
	d.sectionN = name
	if !outputJSON {
		fmt.Printf("\n  %s\n", styleBright.Render(name))
		fmt.Printf("  %s\n", styleDim.Render(doctorRule()))
	}
}

func (d *doctor) record(status, label, detail string) {
	d.checks = append(d.checks, doctorCheck{
		Section: d.sectionN,
		Label:   label,
		Status:  status,
		Detail:  detail,
	})
}

func (d *doctor) ok(label, detail string) {
	d.record("ok", label, detail)
	if !outputJSON {
		d.printCheckLine("OK", label, detail)
	}
}

func (d *doctor) warn(label, detail string) {
	d.warnings++
	d.record("warn", label, detail)
	if !outputJSON {
		d.printCheckLine("WARN", label, detail)
	}
}

func (d *doctor) fail(label, detail string) {
	d.failures++
	d.record("fail", label, detail)
	if !outputJSON {
		d.printCheckLine("ERROR", label, detail)
	}
}

func (d *doctor) printCheckLine(level, label, detail string) {
	lines := wrapText(detail, doctorDetailWidth())
	if len(lines) == 0 {
		lines = []string{""}
	}
	badge := renderLevelBadge(level)
	fmt.Printf("  %s  %-*s  %s\n", badge, checkWidth, label, lines[0])
	for i := 1; i < len(lines); i++ {
		fmt.Printf("  %s  %-*s  %s\n", styleDim.Render(sym("│", "|")), checkWidth, "", styleDim.Render(lines[i]))
	}
}

func doctorRule() string {
	w := terminalWidth() - 4
	if w < 32 {
		w = 32
	}
	return strings.Repeat("─", w)
}

func doctorDetailWidth() int {
	// "  <mark>  <label(36)>  <detail>"
	w := terminalWidth() - (2 + 3 + 2 + checkWidth + 2)
	if w < 24 {
		w = 24
	}
	return w
}

func wrapText(s string, width int) []string {
	if width <= 0 || s == "" {
		return []string{s}
	}
	words := strings.Fields(s)
	if len(words) == 0 {
		return []string{""}
	}
	lines := make([]string, 0, 4)
	current := words[0]
	for _, w := range words[1:] {
		candidate := current + " " + w
		if utf8.RuneCountInString(candidate) <= width {
			current = candidate
			continue
		}
		lines = append(lines, current)
		if utf8.RuneCountInString(w) <= width {
			current = w
			continue
		}
		// hard-wrap long tokens (paths/URLs)
		for utf8.RuneCountInString(w) > width {
			lines = append(lines, truncateRunes(w, width))
			w = trimRunesPrefix(w, width)
		}
		current = w
	}
	if current != "" {
		lines = append(lines, current)
	}
	if len(lines) == 0 {
		return []string{""}
	}
	return lines
}

func truncateRunes(s string, n int) string {
	if n <= 0 {
		return ""
	}
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n])
}

func trimRunesPrefix(s string, n int) string {
	if n <= 0 {
		return s
	}
	r := []rune(s)
	if len(r) <= n {
		return ""
	}
	return string(r[n:])
}
