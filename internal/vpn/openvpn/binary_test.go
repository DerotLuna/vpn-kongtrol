package openvpn

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestFindBinary_Override(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "openvpn-bin")
	if err := os.WriteFile(file, []byte("x"), 0o755); err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name     string
		override string
		want     string
		wantErr  bool
	}{
		{"regular file", file, file, false},
		{"missing path", filepath.Join(dir, "nope"), "", true},
		{"directory", dir, "", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := FindBinary(tt.override)
			if tt.wantErr {
				if !errors.Is(err, ErrBinaryNotFound) {
					t.Fatalf("err = %v, want ErrBinaryNotFound", err)
				}
				if got != "" {
					t.Errorf("path = %q, want empty", got)
				}
				return
			}
			if err != nil || got != tt.want {
				t.Fatalf("FindBinary = %q, %v; want %q", got, err, tt.want)
			}
		})
	}
}
