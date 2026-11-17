package openvpn

import (
	"path/filepath"
	"slices"
	"testing"
)

func argValue(args []string, flag string) (string, int) {
	for i, a := range args {
		if a == flag && i+1 < len(args) {
			return args[i+1], i
		}
	}
	return "", -1
}

func TestBuildArgs(t *testing.T) {
	absCfg := filepath.Join(t.TempDir(), "profiles", "office.ovpn")
	tests := []struct {
		name     string
		opts     startOptions
		wantCert bool
	}{
		{"absolute minimal", startOptions{ConfigPath: absCfg}, false},
		{"relative path", startOptions{ConfigPath: filepath.Join("conf", "rel.ovpn")}, false},
		{"cert and key", startOptions{ConfigPath: absCfg, CertPath: "c.crt", KeyPath: "c.key"}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			args, err := buildArgs(tt.opts, 54321)
			if err != nil {
				t.Fatal(err)
			}
			want, _ := filepath.Abs(tt.opts.ConfigPath)
			cfg, cfgIdx := argValue(args, "--config")
			if cfg != want || !filepath.IsAbs(cfg) {
				t.Errorf("--config = %q, want absolute %q", cfg, want)
			}
			cd, cdIdx := argValue(args, "--cd")
			if cd != filepath.Dir(want) {
				t.Errorf("--cd = %q, want %q", cd, filepath.Dir(want))
			}
			if cdIdx < 0 || cdIdx > cfgIdx {
				t.Errorf("--cd at %d must precede --config at %d", cdIdx, cfgIdx)
			}
			host, mIdx := argValue(args, "--management")
			if host != "127.0.0.1" || mIdx+2 >= len(args) || args[mIdx+2] != "54321" {
				t.Errorf("management args wrong: %v", args)
			}
			if slices.Contains(args, "--route-nopull") {
				t.Errorf("--route-nopull must never be passed: %v", args)
			}
			cert, _ := argValue(args, "--cert")
			key, _ := argValue(args, "--key")
			if tt.wantCert {
				if cert != tt.opts.CertPath || key != tt.opts.KeyPath {
					t.Errorf("--cert/--key = %q/%q", cert, key)
				}
			} else if slices.Contains(args, "--cert") || slices.Contains(args, "--key") {
				t.Errorf("unexpected --cert/--key: %v", args)
			}
		})
	}
}
