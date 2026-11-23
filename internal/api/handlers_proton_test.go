package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/vpn-kongtrol/kongtrol/internal/config"
)

const protonConfAPI = "[Interface]\n# NetShield = 1\n# VPN Accelerator = on\nPrivateKey = aaaa\nAddress = 10.2.0.2/32\nDNS = 10.2.0.1\n\n[Peer]\nPublicKey = bbbb\nAllowedIPs = 0.0.0.0/0\nEndpoint = 203.0.113.7:51820\n"

// A file from a Proton account says once, after the import, that it
// clashes with Proton's app (ADR-0103); other files carry no hint.
func TestImportVPNFile_ProtonHint(t *testing.T) {
	s, _ := newReloadTestServer(t, baseTestYAML, nil)
	s.SetImportDir(filepath.Join(t.TempDir(), "configs"))

	code, out := doImport(t, s, importBody("proton-ch", "wireguard", protonConfAPI, false))
	if code != http.StatusCreated || out["provider"] != "protonvpn" || out["hint_key"] != "profile.import.proton_hint" ||
		out["hint"] != "Este archivo es de tu cuenta de Proton. No lo conectes mientras la app de Proton esté conectada: chocan." {
		t.Fatalf("%d %v", code, out)
	}
	code, out = doImport(t, s, importBody("office", "wireguard", wgConf, false))
	if code != http.StatusCreated || out["provider"] != nil || out["hint"] != nil {
		t.Fatalf("plain file: %d %v", code, out)
	}
}

// The ADR-0103 refusals reach the panel like ADR-0099's: 409 with the
// translated reason and "what to do".
func TestHandleConnect_RefusedSameProtonAccount(t *testing.T) {
	for _, c := range []struct {
		err      *config.ValidationError
		wantHint string
	}{
		{&config.ValidationError{Key: "api.err.connect_proton_app", Args: []any{"office"}},
			"Tu tráfico ya sale por Proton. Para usar office, desconecta Proton en su app y vuelve a conectar office."},
		{&config.ValidationError{Key: "api.err.connect_same_address", Args: []any{"office", "proton-ch", "10.2.0.2"}},
			"Desconecta proton-ch, o cambia la dirección de uno en su archivo (por ejemplo 10.3.0.2 y DNS 10.3.0.1, como indica Proton)."},
	} {
		s, done := newLastErrorServer(t, func() error { return nil })
		s.SetConnectCheck(func(string) error { return c.err })
		req := httptest.NewRequest(http.MethodPost, "/api/v1/tunnels/office/connect", nil)
		req.SetPathValue("name", "office")
		req.Header.Set("X-Kongtrol-Lang", "es")
		rec := httptest.NewRecorder()
		s.handleConnect(rec, req)
		var body map[string]string
		_ = json.Unmarshal(rec.Body.Bytes(), &body)
		if rec.Code != http.StatusConflict || body["error_key"] != c.err.Key || body["hint"] != c.wantHint {
			t.Fatalf("%s: %d %v", c.err.Key, rec.Code, body)
		}
		select {
		case <-done:
			t.Fatal("connect ran despite the refusal")
		default:
		}
	}
}
