package doctor

import (
	"runtime"

	"github.com/vpn-kongtrol/kongtrol/internal/config"
	"github.com/vpn-kongtrol/kongtrol/internal/vpn/forticlient"
)

// checkFortiSupport shows the installed FortiClient version, the automatic
// connect strategy Kongtrol picks for it and how well that is tested
// (docs/forticlient-support.md). Anything not verified is a warning: it
// may fall back to "open FortiClient and wait for the user".
func (d *run) checkFortiSupport(cfg *config.Config) {
	appPath := ""
	for _, name := range SortedProfileNames(cfg) {
		if v := cfg.VPNs[name]; v.Type == "forticlient" && v.AppPath != "" {
			appPath = v.AppPath
			break
		}
	}
	in := forticlient.DetectInstall(appPath)
	plan := forticlient.SelectPlan(runtime.GOOS, in)
	ver := in.Version
	if ver == "" {
		ver = d.t("cli.doctor.forti.version_unknown")
	}
	support := "cli.doctor.forti.support." + string(plan.Support)
	if plan.Strategy == forticlient.StrategyAttach {
		support = "cli.doctor.forti.support.attach"
	}
	detail := d.f("cli.doctor.forti.detail", ver, d.t("cli.doctor.forti.strategy."+string(plan.Strategy)), d.t(support))
	label := d.t("cli.doctor.label.forti_auto")
	if plan.Support == forticlient.SupportVerified {
		d.ok(label, detail)
	} else {
		d.warn(label, detail, d.t("doctor.hint.forti_fallback"))
	}
}
