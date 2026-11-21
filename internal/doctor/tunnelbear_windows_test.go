package doctor

import (
	"strings"
	"testing"

	"github.com/vpn-kongtrol/kongtrol/internal/i18n"
)

func TestTunnelBearDriverReportsMissingAndBrokenSeparately(t *testing.T) {
	d := &run{opt: Options{Lang: i18n.EN}}
	d.reportTunnelBearDriver(true, nil, nil)
	if d.rep.Warnings != 1 || !strings.Contains(d.rep.Checks[0].Detail, "shows no adapter") {
		t.Fatalf("missing adapter report = %+v", d.rep)
	}
	code := uint32(28)
	d = &run{opt: Options{Lang: i18n.ES}}
	d.reportTunnelBearDriver(false, []tunnelBearDevice{{Name: "TunnelBear Adapter v9", ConfigManagerErrorCode: &code}}, nil)
	if d.rep.Warnings != 1 || len(d.rep.Checks) != 1 {
		t.Fatalf("driver report = %+v", d.rep)
	}
	check := d.rep.Checks[0]
	if !strings.Contains(check.Detail, "28") || !strings.Contains(check.Detail, "TunnelBear Adapter v9") || check.Help != "tunnelbear_driver" {
		t.Fatalf("driver report = %+v", check)
	}
}
