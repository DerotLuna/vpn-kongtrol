package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"

	"github.com/vpn-kongtrol/kongtrol/internal/config"
	"github.com/vpn-kongtrol/kongtrol/internal/security"
	"github.com/vpn-kongtrol/kongtrol/internal/vpn"
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
