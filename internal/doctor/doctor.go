// Package doctor runs Kongtrol's read-only diagnostics ("kongtrol doctor",
// GET /api/v1/doctor, the dashboard's Diagnostics card). It returns
// translated checks; each client only renders them.
//
// Nothing here connects a tunnel, starts goroutines, creates the API token
// or opens the audit log: a diagnostic must not change the machine.
package doctor

import (
	"github.com/vpn-kongtrol/kongtrol/internal/i18n"
	"github.com/vpn-kongtrol/kongtrol/internal/security"
	"github.com/vpn-kongtrol/kongtrol/internal/vpn"
)

// Check statuses.
const (
	StatusOK   = "ok"
	StatusWarn = "warn"
	StatusFail = "fail"
)

// FixHelperInstall is the only fix a GUI may offer: install the privileged
// helper (one OS permission prompt, POST /api/v1/helper/install).
const FixHelperInstall = "helper_install"

// Check is one diagnostic line, already translated.
type Check struct {
	Section string `json:"section"`
	Label   string `json:"label"`
	Status  string `json:"status"` // ok|warn|fail
	Detail  string `json:"detail"`
	// Hint is one "what to do" line (warnings and failures only).
	Hint string `json:"hint,omitempty"`
	// Fix names an automatic fix a client may offer (FixHelperInstall).
	Fix string `json:"fix,omitempty"`
}

// Report is the result of Run.
type Report struct {
	Checks   []Check `json:"checks"`
	Failures int     `json:"failures"`
	Warnings int     `json:"warnings"`
}

// Daemon is the running daemon's own view. The daemon owns the tunnels,
// kill switch and DNS guard, so its state wins over this process's fresh,
// never-connected adapters.
type Daemon struct {
	Tunnels           map[string]vpn.Status
	KillSwitchEnabled bool
	KillSwitchActive  bool
	KillSwitch        *security.KillSwitchStatus // detail, may be nil
	DNSGuardActive    bool
}

// Options configures Run.
type Options struct {
	Lang i18n.Lang
	// ConfigPath is kongtrol.yaml; "" searches config.DefaultPaths().
	ConfigPath string
	// Version is this Kongtrol's version (compared with the helper's).
	Version string
	// Daemon is the daemon's state; nil when no daemon answers.
	Daemon *Daemon
	// OnSection / OnCheck stream progress (CLI output); both optional.
	OnSection func(name string)
	OnCheck   func(Check)
}

type run struct {
	opt     Options
	section string
	rep     Report
	rt      *probes
}

// Run executes every check in order and returns the report.
func Run(opt Options) Report {
	d := &run{opt: opt, rep: Report{Checks: []Check{}}}

	d.beginSection("cli.doctor.section.configuration")
	cfg := d.checkConfig()
	d.rt = newProbes(cfg)

	d.beginSection("cli.doctor.section.binaries")
	d.checkBinaries(cfg)

	d.beginSection("cli.doctor.section.certs")
	d.checkCerts(cfg)

	d.beginSection("cli.doctor.section.keychain")
	d.checkKeychain(cfg)

	d.beginSection("cli.doctor.section.permissions")
	d.checkPermissions()

	d.beginSection("cli.doctor.section.runtime")
	d.checkRuntimeNetworking(cfg)

	d.beginSection("cli.doctor.section.adapters_status")
	d.checkAdapterStatus(cfg)

	d.beginSection("cli.doctor.section.adapters_registered")
	d.checkAdapters(cfg)

	return d.rep
}

func (d *run) t(key string) string           { return i18n.T(d.opt.Lang, key) }
func (d *run) f(key string, a ...any) string { return i18n.F(d.opt.Lang, key, a...) }

func (d *run) beginSection(key string) {
	d.section = d.t(key)
	if d.opt.OnSection != nil {
		d.opt.OnSection(d.section)
	}
}

func (d *run) add(c Check) {
	c.Section = d.section
	switch c.Status {
	case StatusWarn:
		d.rep.Warnings++
	case StatusFail:
		d.rep.Failures++
	default:
		c.Hint, c.Fix = "", ""
	}
	d.rep.Checks = append(d.rep.Checks, c)
	if d.opt.OnCheck != nil {
		d.opt.OnCheck(c)
	}
}

func (d *run) ok(label, detail string) {
	d.add(Check{Label: label, Status: StatusOK, Detail: detail})
}

// warn / fail take the translated "what to do" line ("" when the detail
// already says it all).
func (d *run) warn(label, detail, hint string) {
	d.add(Check{Label: label, Status: StatusWarn, Detail: detail, Hint: hint})
}

func (d *run) fail(label, detail, hint string) {
	d.add(Check{Label: label, Status: StatusFail, Detail: detail, Hint: hint})
}

// warnFix is warn with an automatic fix a client may offer.
func (d *run) warnFix(label, detail, hint, fix string) {
	d.add(Check{Label: label, Status: StatusWarn, Detail: detail, Hint: hint, Fix: fix})
}
