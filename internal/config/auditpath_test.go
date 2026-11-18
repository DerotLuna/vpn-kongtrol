package config

import (
	"errors"
	"path/filepath"
	"testing"
)

// Regression: purge used security.audit_log.path as an "extra file" and
// removed it recursively; a folder must not be accepted there.
func TestValidateAuditLogPath(t *testing.T) {
	dir := t.TempDir()
	cases := []struct {
		path    string
		wantErr bool
	}{
		{"", false},
		{filepath.Join(dir, "audit.log"), false},
		{dir, true},
		{dir + string(filepath.Separator), true},
		{"~/", true},
	}
	for _, c := range cases {
		err := ValidateAuditLogPath(c.path)
		if (err != nil) != c.wantErr {
			t.Errorf("ValidateAuditLogPath(%q) = %v, wantErr %v", c.path, err, c.wantErr)
		}
		var ve *ValidationError
		if err != nil && (!errors.As(err, &ve) || ve.Key != "config.err.audit_log_dir") {
			t.Errorf("%q: want config.err.audit_log_dir, got %v", c.path, err)
		}
	}
}
