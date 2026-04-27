package main

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"

	"github.com/spf13/cobra"

	"github.com/vpn-kongtrol/kongtrol/internal/config"
	"github.com/vpn-kongtrol/kongtrol/internal/security"
	"github.com/vpn-kongtrol/kongtrol/internal/vpn"
)

var doctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "Check your environment — binaries, certs, keychain, permissions",
	Long: `doctor validates your full Kongtrol stack without connecting to any VPN.
Run this before your first connection, or when diagnosing a teammate's setup.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		d := &doctor{}
		d.run()
		if d.failures > 0 {
			fmt.Printf("\n%d check(s) failed. Fix the issues above and run 'kongtrol doctor' again.\n", d.failures)
			os.Exit(1)
		}
		fmt.Println("\nAll checks passed. You're good to go.")
		return nil
	},
}

func init() {
	rootCmd.AddCommand(doctorCmd)
}

// ── doctor ────────────────────────────────────────────────────────────────────

type doctor struct {
	failures int
}

func (d *doctor) run() {
	fmt.Println("Kongtrol Doctor")
	fmt.Println(strings.Repeat("─", 52))

	d.section("Configuration")
	cfgPath, cfg := d.checkConfig()

	d.section("VPN Binaries")
	d.checkBinaries(cfg)

	d.section("Certificates & Keys")
	d.checkCerts(cfg)

	d.section("Keychain Credentials")
	d.checkKeychain(cfg)

	d.section("Permissions")
	d.checkPermissions()

	d.section("Registered Adapters")
	d.checkAdapters(cfg)

	_ = cfgPath
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
		d.fail("config file", "not found — run 'kongtrol init' to create one")
		return "", nil
	}
	d.ok("config file", path)

	cfg, err := config.Load(path)
	if err != nil {
		d.fail("config valid", err.Error())
		return path, nil
	}
	d.ok("config valid", fmt.Sprintf("%d profile(s) defined", len(cfg.VPNs)))
	return path, cfg
}

// ── binaries ──────────────────────────────────────────────────────────────────

type binarySpec struct {
	adapterType string
	candidates  []string
	versionArg  string
}

func vpnBinarySpecs() []binarySpec {
	specs := []binarySpec{
		{
			adapterType: "openvpn",
			candidates:  []string{"openvpn"},
			versionArg:  "--version",
		},
		{
			adapterType: "protonvpn",
			candidates:  []string{"protonvpn-cli"},
			versionArg:  "--version",
		},
		{
			adapterType: "tailscale",
			candidates:  []string{"tailscale"},
			versionArg:  "version",
		},
		{
			adapterType: "cloudflarewarp",
			candidates:  []string{"warp-cli"},
			versionArg:  "--version",
		},
		{
			adapterType: "wireguard",
			candidates:  []string{"wg", "wg-quick"},
			versionArg:  "--version",
		},
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
		d.warn("skipped", "no valid config loaded")
		return
	}

	// Which adapter types are actually used?
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
			d.fail(spec.adapterType+" binary", "not found — install the VPN client")
			continue
		}
		ver := ""
		if out, err := exec.Command(found, spec.versionArg).Output(); err == nil {
			ver = strings.TrimSpace(strings.Split(string(out), "\n")[0])
			if len(ver) > 48 {
				ver = ver[:48] + "…"
			}
		}
		if ver == "" {
			d.ok(spec.adapterType+" binary", found)
		} else {
			d.ok(spec.adapterType+" binary", ver)
		}
	}
}

// ── certs & keys ──────────────────────────────────────────────────────────────

func (d *doctor) checkCerts(cfg *config.Config) {
	if cfg == nil {
		d.warn("skipped", "no valid config loaded")
		return
	}
	any := false
	for name, v := range cfg.VPNs {
		if v.Auth.Cert != "" {
			any = true
			d.checkFile(name+": cert", v.Auth.Cert)
		}
		if v.Auth.Key != "" {
			any = true
			d.checkFile(name+": key", v.Auth.Key)
		}
		if v.ConfigFile != "" {
			any = true
			d.checkFile(name+": config file", v.ConfigFile)
		}
	}
	if !any {
		d.ok("no file paths configured", "skipped")
	}
}

func (d *doctor) checkFile(label, path string) {
	info, err := os.Stat(path)
	if err != nil {
		d.fail(label, fmt.Sprintf("not found: %s", path))
		return
	}
	if info.IsDir() {
		d.fail(label, fmt.Sprintf("is a directory, expected a file: %s", path))
		return
	}
	d.ok(label, path)
}

// ── keychain ──────────────────────────────────────────────────────────────────

func (d *doctor) checkKeychain(cfg *config.Config) {
	if cfg == nil {
		d.warn("skipped", "no valid config loaded")
		return
	}
	any := false
	for name, v := range cfg.VPNs {
		if v.Auth.PasswordKeychain != "" {
			any = true
			_, err := config.GetCredential(name, "password")
			if err != nil {
				d.fail(name+": password", "not in keychain — run 'kongtrol init' to store it")
			} else {
				d.ok(name+": password", "found in OS keychain")
			}
		}
		if v.Auth.UsernameKeychain != "" {
			any = true
			_, err := config.GetCredential(name, "username")
			if err != nil {
				d.fail(name+": username", "not in keychain — run 'kongtrol init' to store it")
			} else {
				d.ok(name+": username", "found in OS keychain")
			}
		}
	}
	if !any {
		d.ok("no keychain credentials configured", "skipped")
	}
}

// ── permissions ───────────────────────────────────────────────────────────────

func (d *doctor) checkPermissions() {
	// Kill switch: probe by calling Disable() (a no-op when inactive).
	// A real Enable() requires a live tunnel interface name, so we just verify
	// the platform implementation is callable without panicking.
	ks := security.NewKillSwitch()
	if err := ks.Disable(); err != nil {
		d.warn("kill switch", fmt.Sprintf("may need elevated privileges: %v", err))
	} else {
		d.ok("kill switch", "platform implementation available")
	}

	// Check DNS guard write access.
	dnsGuard := security.NewDNSGuard()
	if dnsGuard.IsActive() {
		d.ok("dns guard", "already active")
	} else {
		// On Linux, check /etc/resolv.conf is writable.
		if runtime.GOOS == "linux" {
			f, err := os.OpenFile("/etc/resolv.conf", os.O_WRONLY|os.O_APPEND, 0)
			if err != nil {
				d.warn("dns guard", "/etc/resolv.conf not writable — may need sudo")
			} else {
				f.Close()
				d.ok("dns guard", "/etc/resolv.conf writable")
			}
		} else {
			d.ok("dns guard", "available")
		}
	}
}

// ── adapters ──────────────────────────────────────────────────────────────────

func (d *doctor) checkAdapters(cfg *config.Config) {
	registered := vpn.Registered()
	d.ok("registered adapters", strings.Join(registered, ", "))

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
			d.fail(name+": adapter", fmt.Sprintf("type %q not registered — check blank imports in main.go", v.Type))
		}
	}
}

// ── output helpers ────────────────────────────────────────────────────────────

const (
	checkWidth = 36
)

func (d *doctor) section(name string) {
	fmt.Printf("\n  %s\n", name)
}

func (d *doctor) ok(label, detail string) {
	fmt.Printf("  ✓  %-*s  %s\n", checkWidth, label, detail)
}

func (d *doctor) warn(label, detail string) {
	fmt.Printf("  ⚠  %-*s  %s\n", checkWidth, label, detail)
}

func (d *doctor) fail(label, detail string) {
	d.failures++
	fmt.Printf("  ✗  %-*s  %s\n", checkWidth, label, detail)
}
