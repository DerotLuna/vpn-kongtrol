package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/zalando/go-keyring"

	"github.com/vpn-kongtrol/kongtrol/internal/config"
	"github.com/vpn-kongtrol/kongtrol/internal/configedit"
	"github.com/vpn-kongtrol/kongtrol/internal/vpn/appattach"
	"github.com/vpn-kongtrol/kongtrol/internal/vpndetect"
)

func stubPassword(t *testing.T, pwd string, asked *int) {
	t.Helper()
	old := promptPasswordFn
	t.Cleanup(func() { promptPasswordFn = old })
	promptPasswordFn = func(string) (string, error) { *asked++; return pwd, nil }
}

func TestProfileAdd_OfflineStoresProfileAndPassword(t *testing.T) {
	path := withEditConfig(t)
	keyring.MockInit()
	asked := 0
	stubPassword(t, "s3cret", &asked)

	// No terminal: no question, the profile is created and the line says
	// where to add the password.
	out, err := runFlags(t, profileAddCmd, map[string]string{"name": "trabajo", "host": "vpn.example.com", "username": "ana"}, "forticlient")
	if err != nil {
		t.Fatal(err)
	}
	if asked != 0 || !strings.Contains(out, "Sin contraseña") || !strings.Contains(out, "kongtrol up trabajo") {
		t.Fatalf("asked=%d output:\n%s", asked, out)
	}
	v := reloadEditCfg(t, path).VPNs["trabajo"]
	if v.Type != "forticlient" || v.Host != "vpn.example.com" || v.Auth.Method != "credentials" || v.Auth.PasswordKeychain != "" {
		t.Fatalf("profile = %+v", v)
	}

	// A terminal: the password is asked (never a flag) and goes to the keychain.
	promptInteractiveFn = func() bool { return true }
	if _, err := runFlags(t, profileAddCmd, map[string]string{"name": "cisco", "host": "vpn.example.com", "username": "ana"}, "ciscoanyconnect"); err != nil {
		t.Fatal(err)
	}
	if asked != 1 {
		t.Fatalf("password asked %d times", asked)
	}
	if v := reloadEditCfg(t, path).VPNs["cisco"]; v.Auth.PasswordKeychain != "cisco.password" {
		t.Fatalf("profile = %+v", v)
	}
	if pwd, err := config.GetCredential("cisco", "password"); err != nil || pwd != "s3cret" {
		t.Fatalf("keychain: %q %v", pwd, err)
	}
	if profileAddCmd.Flags().Lookup("password") != nil {
		t.Fatal("a password flag would put it in the shell history")
	}

	// Provider: location fields; --json.
	promptInteractiveFn = func() bool { return false }
	outputJSON = true
	out, err = runFlags(t, profileAddCmd, map[string]string{"name": "nord", "country": "Spain", "no-auto-connect": "true"}, "NordVPN")
	outputJSON = false
	if err != nil {
		t.Fatal(err)
	}
	var res profileAddResult
	if json.Unmarshal([]byte(out), &res) != nil || res.Name != "nord" || res.Type != "nordvpn" || res.ViaDaemon {
		t.Fatalf("json = %s", out)
	}
	if v := reloadEditCfg(t, path).VPNs["nord"]; v.Country != "Spain" || v.AutoConnect == nil || *v.AutoConnect {
		t.Fatalf("nord = %+v", v)
	}
}

func TestProfileAdd_Errors(t *testing.T) {
	withEditConfig(t)
	keyring.MockInit()
	cases := []struct {
		flags map[string]string
		typ   string
		want  string
	}{
		{map[string]string{"name": "x"}, "pptp", "Tipos:"},
		{nil, "wireguard", "--name"},
		{map[string]string{"name": "oficina"}, "wireguard", "Ya hay un perfil"},
		// surfshark cannot pick a location: the config check answers, translated.
		{map[string]string{"name": "s", "country": "US"}, "surfshark", "\n    "},
	}
	for _, c := range cases {
		_, err := runFlags(t, profileAddCmd, c.flags, c.typ)
		if err == nil || !strings.Contains(err.Error(), c.want) || !strings.Contains(err.Error(), "\n    ") {
			t.Fatalf("%s %v: %v", c.typ, c.flags, err)
		}
	}
	if err := profileAddCmd.Args(profileAddCmd, nil); err == nil {
		t.Fatal("profile add with no type accepted")
	}
}

func TestProfileAdd_DaemonOffersRestart(t *testing.T) {
	withEditConfig(t)
	var body configedit.VPNProfile
	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/v1/vpns", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&body)
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(map[string]any{"status": "created", "profile": "nord2", "restart_required": true})
	})
	ts := httptest.NewServer(mux)
	defer ts.Close()
	host, port := fakeDaemonBase(t, ts)
	cfg.Monitor.Dashboard.Bind, cfg.Monitor.Dashboard.Port = host, port
	daemonProbeFn = func(string) bool { return true }
	restarts := 0
	old := restartDaemonFn
	t.Cleanup(func() { restartDaemonFn = old })
	restartDaemonFn = func(string) error { restarts++; return nil }

	// No terminal, no --yes: no restart, the line says how.
	out, err := runFlags(t, profileAddCmd, map[string]string{"name": "nord2"}, "nordvpn")
	if err != nil || restarts != 0 || !strings.Contains(out, "kongtrol restart") {
		t.Fatalf("restarts=%d err=%v\n%s", restarts, err, out)
	}
	if configedit.Str(body.Name) != "nord2" || configedit.Str(body.Type) != "nordvpn" || body.Password != "" {
		t.Fatalf("body = %+v", body)
	}
	if _, err := runFlags(t, profileAddCmd, map[string]string{"name": "nord3", "yes": "true"}, "nordvpn"); err != nil || restarts != 1 {
		t.Fatalf("--yes: restarts=%d err=%v", restarts, err)
	}
}

func TestProfileImportCreate(t *testing.T) {
	path := withEditConfig(t)
	keyring.MockInit()
	src := filepath.Join(t.TempDir(), "Mi Oficina (2).conf")
	if err := os.WriteFile(src, []byte("[Interface]\nPrivateKey = a\nAddress = 10.0.0.2/32\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	// importAndCreate reloads the config itself (import alone needs none).
	oldCfgPath := cfgPath
	t.Cleanup(func() { cfgPath = oldCfgPath })
	cfgPath = path
	out, err := runFlags(t, profileImportCmd, map[string]string{"create": "true"}, src)
	if err != nil {
		t.Fatal(err)
	}
	v, ok := reloadEditCfg(t, path).VPNs["Mi-Oficina-2"]
	if !ok || v.Type != "wireguard" || filepath.Base(v.ConfigFile) != "Mi-Oficina-2.conf" || v.Auth.Method != "certificate" {
		t.Fatalf("profiles = %+v\n%s", reloadEditCfg(t, path).VPNs, out)
	}
	if !strings.Contains(out, "kongtrol up Mi-Oficina-2") {
		t.Fatalf("output:\n%s", out)
	}
	// runFlags resets every flag after a run, the inherited --config too
	// (cfgPath): point it at the test config again, never at the real one.
	cfgPath = path
	// The profile name is taken: refused before storing anything.
	if _, err := runFlags(t, profileImportCmd, map[string]string{"create": "true", "name": "oficina"}, src); err == nil || !strings.Contains(err.Error(), "Ya hay un perfil") {
		t.Fatalf("taken name: %v", err)
	}
}

func TestProfileDetect(t *testing.T) {
	withEditConfig(t)
	old := detectClientsFn
	t.Cleanup(func() { detectClientsFn = old })
	oldActive := activeTunnelFn
	t.Cleanup(func() { activeTunnelFn = oldActive })
	activeTunnelFn = func(typ string) (appattach.Tunnel, bool, bool) {
		return appattach.Tunnel{Name: "NordLynx"}, typ == "nordvpn", typ == "nordvpn"
	}
	detectClientsFn = func() []vpndetect.Client {
		return []vpndetect.Client{{Label: "NordVPN", AdapterType: "nordvpn", Path: "/opt/nord"}, {Label: "Otra", Path: "/opt/x"}}
	}
	out, err := runFlags(t, profileDetectCmd, nil)
	if err != nil || !strings.Contains(out, "NordVPN") || !strings.Contains(out, "sin perfil en Kongtrol") || !strings.Contains(out, "kongtrol profile add nordvpn") || !strings.Contains(out, "conectada ahora (NordLynx)") {
		t.Fatalf("%v\n%s", err, out)
	}
	outputJSON = true
	out, err = runFlags(t, profileDetectCmd, nil)
	outputJSON = false
	var list []detectedClient
	if err != nil || json.Unmarshal([]byte(out), &list) != nil || len(list) != 2 || list[1].AdapterType != "" || list[0].Active == nil || !*list[0].Active || list[1].Active != nil {
		t.Fatalf("json: %v\n%s", err, out)
	}
	detectClientsFn = func() []vpndetect.Client { return nil }
	outputJSON = true
	out, _ = runFlags(t, profileDetectCmd, nil)
	outputJSON = false
	if strings.TrimSpace(out) != "[]" {
		t.Fatalf("empty json = %q", out)
	}
}
