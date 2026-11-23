package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/zalando/go-keyring"
)

const protonMoveConf = "[Interface]\n# NetShield = 1\n# VPN Accelerator = on\nPrivateKey = x\nAddress = 10.2.0.2/32\nDNS = 10.2.0.1\n\n[Peer]\nPublicKey = y\nEndpoint = 203.0.113.7:51820\n"

// withProtonFileConfig: a direct profile made from a Proton file with two
// rules, and a profile that is not Proton's with one.
func withProtonFileConfig(t *testing.T) string {
	t.Helper()
	path := withEditConfig(t)
	conf := filepath.Join(t.TempDir(), "proton-ch.conf")
	if err := os.WriteFile(conf, []byte(protonMoveConf), 0o600); err != nil {
		t.Fatal(err)
	}
	body := editTestYAML + "  proton-ch:\n    type: wireguard\n    config: \"" + filepath.ToSlash(conf) + "\"\n    auth:\n      method: certificate\n" +
		"policies:\n" +
		"  - name: news\n    via: proton-ch\n    match:\n      domains: [news.example]\n" +
		"  - name: shop\n    via: proton-ch\n    match:\n      domains: [shop.example]\n" +
		"  - name: corp\n    via: oficina\n    match:\n      domains: [corp.example]\n"
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	reloadEditCfg(t, path)
	keyring.MockInit()
	asked := 0
	stubPassword(t, "", &asked)
	return path
}

func viaOf(t *testing.T, path, rule string) string {
	t.Helper()
	for _, p := range reloadEditCfg(t, path).Policies {
		if p.Name == rule {
			return p.Via
		}
	}
	return ""
}

// Adding the Proton app profile with no terminal: nothing moves, the line
// says which command does it (ADR-0104).
func TestProfileAdd_ProviderOffersRuleMove_NoTerminal(t *testing.T) {
	path := withProtonFileConfig(t)
	out, err := runFlags(t, profileAddCmd, map[string]string{"name": "Proton", "yes": "true"}, "protonvpn")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "2 reglas usan proton-ch, tu archivo de Proton VPN") ||
		!strings.Contains(out, `kongtrol policy move "proton-ch" "Proton"`) {
		t.Fatalf("output:\n%s", out)
	}
	if viaOf(t, path, "news") != "proton-ch" {
		t.Fatal("--yes moved the rules: moving is its own choice")
	}
}

// On a terminal that answers yes, both rules move; the other profile's stays.
func TestProfileAdd_ProviderOffersRuleMove_Yes(t *testing.T) {
	path := withProtonFileConfig(t)
	promptInteractiveFn = func() bool { return true }
	promptReader = strings.NewReader("s\n")
	t.Cleanup(func() { promptReader = os.Stdin })
	out, err := runFlags(t, profileAddCmd, map[string]string{"name": "Proton"}, "protonvpn")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "¿Usar Proton en tus reglas de proton-ch?") || !strings.Contains(out, "Listo: 2 reglas ahora usan Proton.") {
		t.Fatalf("output:\n%s", out)
	}
	if viaOf(t, path, "news") != "Proton" || viaOf(t, path, "shop") != "Proton" || viaOf(t, path, "corp") != "oficina" {
		t.Fatalf("policies = %+v", reloadEditCfg(t, path).Policies)
	}
}

func TestPolicyMove_Offline(t *testing.T) {
	path := withProtonFileConfig(t)
	out, err := runFlags(t, policyMoveCmd, nil, "oficina", "casa")
	if err != nil || !strings.Contains(out, "Listo: 1 regla ahora usa casa.") || viaOf(t, path, "corp") != "casa" {
		t.Fatalf("%v\n%s", err, out)
	}
	out, err = runFlags(t, policyMoveCmd, nil, "oficina", "casa")
	if err != nil || !strings.Contains(out, "oficina no tiene reglas") {
		t.Fatalf("none: %v\n%s", err, out)
	}
	if _, err := runFlags(t, policyMoveCmd, nil, "casa", "casa"); err == nil || !strings.Contains(err.Error(), "Elige otro perfil") {
		t.Fatalf("same: %v", err)
	}
}
