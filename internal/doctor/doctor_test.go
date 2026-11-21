package doctor

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/vpn-kongtrol/kongtrol/internal/i18n"
	"github.com/vpn-kongtrol/kongtrol/internal/privhelper"
	"github.com/vpn-kongtrol/kongtrol/internal/privops"
	"github.com/vpn-kongtrol/kongtrol/internal/routing"
	"github.com/vpn-kongtrol/kongtrol/internal/vpn"
	"github.com/vpn-kongtrol/kongtrol/internal/vpn/appattach"
)

type fakeRoutes struct{}

func (fakeRoutes) List() ([]routing.Route, error) { return nil, nil }

type fakeAdapter struct{ st vpn.Status }

func (f *fakeAdapter) Connect(context.Context, vpn.AdapterConfig) error { return nil }
func (f *fakeAdapter) Disconnect(context.Context) error                 { return nil }
func (f *fakeAdapter) Reconnect(context.Context) error                  { return nil }
func (f *fakeAdapter) Status() vpn.Status                               { return f.st }
func (f *fakeAdapter) TunnelInfo() (*vpn.TunnelInfo, error)             { return nil, nil }
func (f *fakeAdapter) Name() string                                     { return "tailscale" }
func (f *fakeAdapter) Version() string                                  { return "v0" }
func (f *fakeAdapter) Capabilities() vpn.Capabilities                   { return vpn.Capabilities{} }

func init() {
	// tailscale's real adapter is not linked into this test binary.
	vpn.Register("tailscale", func() vpn.VPNAdapter { return &fakeAdapter{st: vpn.StatusDisconnected} })
}

// stubOS replaces every OS probe. helperOK: the helper answers with our
// protocol and version.
func stubOS(t *testing.T, helperOK bool) {
	t.Helper()
	prevQ, prevP, prevE, prevD, prevR, prevC := helperQueryStatus, helperPing, isElevated, dnsGuardActive, newRouteLister, getCredential
	t.Cleanup(func() {
		helperQueryStatus, helperPing, isElevated, dnsGuardActive, newRouteLister, getCredential = prevQ, prevP, prevE, prevD, prevR, prevC
	})
	prevProvider, prevIfaces, prevRoutes := activeProviderTunnel, listNetworkInterfaces, listSystemRoutes
	t.Cleanup(func() {
		activeProviderTunnel, listNetworkInterfaces, listSystemRoutes = prevProvider, prevIfaces, prevRoutes
	})
	activeProviderTunnel = func(string) (appattach.Tunnel, bool, bool) { return appattach.Tunnel{}, false, false }
	listNetworkInterfaces = func() []appattach.Iface { return nil }
	listSystemRoutes = func() ([]routing.Route, error) { return nil, nil }
	isElevated = func() bool { return false }
	prevPath := pathEnv
	t.Cleanup(func() { pathEnv = prevPath })
	pathEnv = func() string { return "" }
	dnsGuardActive = func() bool { return false }
	newRouteLister = func() routeLister { return fakeRoutes{} }
	getCredential = func(string, string) (string, error) { return "", errors.New("missing") }
	if helperOK {
		helperQueryStatus = func() privhelper.State { return privhelper.State{Installed: true, Running: true} }
		helperPing = func(context.Context) (privops.PingResult, error) {
			return privops.PingResult{Version: "v1", Protocol: privops.ProtocolVersion}, nil
		}
		return
	}
	helperQueryStatus = func() privhelper.State { return privhelper.State{} }
	helperPing = func(context.Context) (privops.PingResult, error) { return privops.PingResult{}, errors.New("down") }
}

func writeConfig(t *testing.T, body string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "kongtrol.yaml")
	if err := os.WriteFile(p, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	return p
}

func find(rep Report, label string) (Check, bool) {
	for _, c := range rep.Checks {
		if c.Label == label {
			return c, true
		}
	}
	return Check{}, false
}

func TestRunMissingConfigFailsWithHint(t *testing.T) {
	stubOS(t, true)
	var sections, checks int
	rep := Run(Options{
		Lang:       i18n.ES,
		ConfigPath: filepath.Join(t.TempDir(), "missing.yaml"),
		Version:    "v1",
		OnSection:  func(string) { sections++ },
		OnCheck:    func(Check) { checks++ },
	})
	if rep.Failures == 0 {
		t.Fatalf("missing config must fail: %+v", rep)
	}
	c := rep.Checks[0]
	if c.Status != StatusFail || c.Hint != i18n.T(i18n.ES, "doctor.hint.config_missing") {
		t.Fatalf("first check = %+v", c)
	}
	if sections != 8 || checks != len(rep.Checks) {
		t.Fatalf("callbacks: sections=%d checks=%d/%d", sections, checks, len(rep.Checks))
	}
	for _, c := range rep.Checks {
		if c.Status == StatusOK && (c.Hint != "" || c.Fix != "") {
			t.Fatalf("ok check with hint/fix: %+v", c)
		}
		if c.Status == StatusFail && c.Hint == "" {
			t.Fatalf("failure without a what-to-do line: %+v", c)
		}
	}
}

func TestRunHelperMissingOffersFix(t *testing.T) {
	stubOS(t, false)
	rep := Run(Options{Lang: i18n.EN, ConfigPath: filepath.Join(t.TempDir(), "x.yaml"), Version: "v1"})
	n := 0
	for _, c := range rep.Checks {
		if c.Fix == FixHelperInstall {
			n++
			if c.Status != StatusWarn || c.Hint == "" {
				t.Fatalf("fix on %+v", c)
			}
		}
	}
	// helper + kill switch + DNS guard lines.
	if n != 3 {
		t.Fatalf("want 3 checks offering helper_install, got %d: %+v", n, rep.Checks)
	}
}

func TestRunHelperReadyNoFix(t *testing.T) {
	stubOS(t, true)
	rep := Run(Options{Lang: i18n.EN, ConfigPath: filepath.Join(t.TempDir(), "x.yaml"), Version: "v1"})
	for _, c := range rep.Checks {
		if c.Fix != "" {
			t.Fatalf("no fix expected with a working helper: %+v", c)
		}
	}
	if c, ok := find(rep, i18n.T(i18n.EN, "cli.doctor.label.helper")); !ok || c.Status != StatusOK {
		t.Fatalf("helper line: %+v", c)
	}
}

const tailscaleCfg = `vpns:
  office:
    type: tailscale
    auth:
      method: token
      password_keychain: office
security:
  dns_guard:
    enabled: true
`

func TestRunPrefersDaemonState(t *testing.T) {
	stubOS(t, true)
	path := writeConfig(t, tailscaleCfg)
	rep := Run(Options{
		Lang: i18n.EN, ConfigPath: path, Version: "v1",
		Daemon: &Daemon{Tunnels: map[string]vpn.Status{"office": vpn.StatusConnected}, DNSGuardActive: true, KillSwitchEnabled: true},
	})
	c, ok := find(rep, "office: "+i18n.T(i18n.EN, "cli.doctor.label.adapter_status"))
	if !ok || c.Detail != string(vpn.StatusConnected) {
		t.Fatalf("adapter status must come from the daemon: %+v", c)
	}
	dns, _ := find(rep, i18n.T(i18n.EN, "cli.doctor.label.dns_guard_runtime"))
	if dns.Status != StatusOK || dns.Detail != i18n.T(i18n.EN, "cli.doctor.dns.active") {
		t.Fatalf("DNS guard must come from the daemon: %+v", dns)
	}
	ks, ok := find(rep, i18n.T(i18n.EN, "cli.doctor.label.kill_switch_runtime"))
	if !ok || ks.Detail != i18n.T(i18n.EN, "cli.status.security.armed") {
		t.Fatalf("kill switch line: %+v", ks)
	}
	cred, _ := find(rep, "office: password")
	if cred.Status != StatusFail || cred.Hint != i18n.T(i18n.EN, "doctor.hint.keychain_missing") {
		t.Fatalf("keychain line: %+v", cred)
	}
}

func TestRunNoDaemonUsesAdapter(t *testing.T) {
	stubOS(t, true)
	rep := Run(Options{Lang: i18n.EN, ConfigPath: writeConfig(t, tailscaleCfg), Version: "v1"})
	c, _ := find(rep, "office: "+i18n.T(i18n.EN, "cli.doctor.label.adapter_status"))
	if c.Detail != string(vpn.StatusDisconnected) {
		t.Fatalf("adapter status: %+v", c)
	}
	if _, ok := find(rep, i18n.T(i18n.EN, "cli.doctor.label.kill_switch_runtime")); ok {
		t.Fatal("no daemon: no live kill switch line")
	}
}
