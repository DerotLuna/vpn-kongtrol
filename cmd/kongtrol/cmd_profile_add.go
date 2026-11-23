package main

import (
	"fmt"
	"slices"
	"strings"

	"github.com/charmbracelet/huh"
	"github.com/spf13/cobra"

	"github.com/vpn-kongtrol/kongtrol/internal/config"
	"github.com/vpn-kongtrol/kongtrol/internal/configedit"
	"github.com/vpn-kongtrol/kongtrol/internal/profileimport"
	"github.com/vpn-kongtrol/kongtrol/internal/vpn"
	"github.com/vpn-kongtrol/kongtrol/internal/vpn/appattach"
	"github.com/vpn-kongtrol/kongtrol/internal/vpndetect"
)

// profile add / detect and profile import --create: the CLI side of the
// dashboard's "＋ Agregar" for VPNs. A profile is created through the
// daemon's API (POST /api/v1/vpns) when it runs, else with the same
// internal/configedit path on kongtrol.yaml. Passwords never go on the
// command line: they are asked without echo on a terminal, or the user is
// told where to add them.

// profileAddFlags are the profile fields; empty ones are not sent.
var profileAddFlags struct {
	name, host, server, protocol, config, authMethod, cert, key, username string
	country, city, group, iface, waitTimeout, appPath                     string
	onTunnelDown                                                          string
	port                                                                  int
	noAutoConnect, yes                                                    bool
}

var (
	profileImportCreate bool
	profileImportYes    bool
)

var profileAddCmd = &cobra.Command{
	Use:  "add <type>",
	Args: cobra.ExactArgs(1),
	RunE: runProfileAdd,
}

var profileDetectCmd = &cobra.Command{
	Use:         "detect",
	Args:        cobra.NoArgs,
	Annotations: map[string]string{annotationPrefsOnly: "true"},
	RunE:        runProfileDetect,
}

// Test seam: the secure password prompt.
var promptPasswordFn = func(profile string) (string, error) {
	var pwd string
	err := runForm(newForm(huh.NewGroup(
		huh.NewInput().
			Title(cf("cli.profile.add.password", profile)).
			EchoMode(huh.EchoModePassword).
			Value(&pwd),
	)))
	return pwd, err
}

// profileAddResult is the --json answer of profile add / import --create.
type profileAddResult struct {
	Name            string `json:"name"`
	Type            string `json:"type"`
	ViaDaemon       bool   `json:"via_daemon"`
	RestartRequired bool   `json:"restart_required"`
	CredentialSaved bool   `json:"credential_saved"`
	ConfigPath      string `json:"config,omitempty"`
}

func runProfileAdd(cmd *cobra.Command, args []string) error {
	typ := strings.ToLower(strings.TrimSpace(args[0]))
	if !slices.Contains(vpn.Registered(), typ) {
		return fmt.Errorf("%s", cf("cli.profile.add.type_unknown", args[0], strings.Join(vpn.Registered(), ", ")))
	}
	f := profileAddFlags
	if strings.TrimSpace(f.name) == "" {
		return fmt.Errorf("%s", cf("cli.profile.add.name_required", typ))
	}
	p := configedit.VPNProfile{Name: strPtr(f.name), Type: &typ}
	for _, s := range []struct {
		dst **string
		v   string
	}{
		{&p.Host, f.host}, {&p.Server, f.server}, {&p.Protocol, f.protocol}, {&p.ConfigFile, f.config},
		{&p.Cert, f.cert}, {&p.Key, f.key}, {&p.Username, f.username}, {&p.Country, f.country},
		{&p.City, f.city}, {&p.Group, f.group}, {&p.Interface, f.iface}, {&p.WaitTimeout, f.waitTimeout},
		{&p.AppPath, f.appPath},
		{&p.OnTunnelDown, f.onTunnelDown},
	} {
		if strings.TrimSpace(s.v) != "" {
			*s.dst = strPtr(s.v)
		}
	}
	if f.port != 0 {
		p.Port = &f.port
	}
	if f.noAutoConnect {
		off := false
		p.AutoConnect = &off
	}
	return createProfile(p, f.yes)
}

// createProfile fills the auth method, asks for the password when the type
// needs one, creates the profile and offers the restart a new profile needs.
func createProfile(p configedit.VPNProfile, yes bool) error {
	typ, name := configedit.Str(p.Type), configedit.Str(p.Name)
	method := configedit.Str(p.AuthMethod)
	if method == "" {
		method = configedit.DefaultAuthMethod(typ)
		if typ == "openvpn" && configedit.Str(p.Username) != "" {
			method = "certificate+credentials"
		}
		p.AuthMethod = &method
	}
	if configedit.NeedsPassword(typ, method) && canPrompt() {
		pwd, err := promptPasswordFn(name)
		if err != nil {
			return err
		}
		p.Password = pwd
	}
	res := profileAddResult{Name: name, Type: typ, CredentialSaved: p.Password != "", ConfigPath: configedit.Str(p.ConfigFile)}
	var err error
	res.ViaDaemon, res.RestartRequired, err = saveProfile(p)
	p.Password = ""
	if err != nil {
		return err
	}
	if outputJSON {
		return emitJSON(res)
	}
	if !outputQuiet {
		fmt.Println(tuiOK(cf("cli.profile.add.done", name, typ)))
		if configedit.NeedsPassword(typ, method) && !res.CredentialSaved {
			fmt.Println(tuiWarn(cf("cli.profile.add.password_hint", name)))
		}
		// A provider-app profile: offer it the rules of direct profiles
		// made from that provider's files (ADR-0104).
		if typ != "wireguard" && typ != "openvpn" {
			offerRuleMove(name, typ)
		}
	}
	if res.RestartRequired {
		base := daemonAPIBase()
		if confirmAction(ct("cli.reload.ask_restart"), yes, true) {
			if err := restartDaemonFn(base); err != nil {
				return err
			}
		} else if !outputQuiet {
			fmt.Println(tuiInfo(ct("cli.profile.add.restart_hint")))
			return nil
		}
	}
	if !outputQuiet {
		fmt.Println(tuiInfo(cf("cli.profile.add.next", name)))
	}
	return nil
}

// saveProfile creates p through the running daemon, or in kongtrol.yaml
// (+ keychain) with the API's own configedit path.
func saveProfile(p configedit.VPNProfile) (viaDaemon, restartRequired bool, err error) {
	if base := daemonAPIBase(); daemonProbeFn(base) {
		restart, err := configClient(base).CreateVPN(p)
		return true, restart, remoteUserErr(err)
	}
	// As the API: loaded fresh, credentials put back if the save fails.
	c, err := config.Load(activeCfgPath)
	if err != nil {
		return false, false, localizeConfigErr(err)
	}
	_, err = configedit.AddVPNProfile(c, p, func(c *config.Config) error {
		_, err := configedit.Save(activeCfgPath, c)
		return err
	})
	return false, false, localizeConfigErr(err)
}

func strPtr(s string) *string {
	s = strings.TrimSpace(s)
	return &s
}

// ── profile detect ───────────────────────────────────────────────────────────

// detectClientsFn is replaced in tests.
var detectClientsFn = vpndetect.DetectAll

// detectedClient is one VPN app found here: the same shape as
// GET /api/v1/vpns/detect (adapter_type "" = found, no adapter).
type detectedClient struct {
	Label       string `json:"label"`
	AdapterType string `json:"adapter_type"`
	Path        string `json:"path"`
	// Active: the app's tunnel is up now (provider apps; null = cannot tell).
	Active    *bool  `json:"active"`
	Interface string `json:"interface,omitempty"`
}

// activeTunnelFn is replaced in tests.
var activeTunnelFn = appattach.ActiveTunnel

// runProfileDetect lists the VPN apps installed here with the same
// internal/vpndetect probes as the dashboard, the wizard and doctor. It
// only looks for files and network interfaces (is the app's tunnel up?):
// nothing runs, no admin, nothing leaves the machine (ADR-0053).
func runProfileDetect(cmd *cobra.Command, args []string) error {
	found := detectClientsFn()
	out := make([]detectedClient, 0, len(found))
	for _, c := range found {
		d := detectedClient{Label: c.Label, AdapterType: c.AdapterType, Path: c.Path}
		if c.AdapterType != "" {
			if t, up, known := activeTunnelFn(c.AdapterType); known {
				d.Active = &up
				if up {
					d.Interface = t.Name
				}
			}
		}
		out = append(out, d)
	}
	if outputJSON {
		return emitJSON(out)
	}
	if outputQuiet {
		return nil
	}
	if len(out) == 0 {
		fmt.Println(tuiInfo(ct("cli.profile.detect.none")))
		fmt.Println(tuiInfo(ct("cli.profile.detect.none_next")))
		return nil
	}
	fmt.Println(tuiInfo(cf("cli.profile.detect.found", len(out))))
	example := ""
	for _, c := range out {
		kind := c.AdapterType
		if kind == "" {
			kind = ct("cli.profile.detect.unsupported")
		} else if example == "" {
			example = c.AdapterType
		}
		line := "  " + styleBright.Render(c.Label) + "  " + styleDim.Render(kind+" · "+c.Path)
		if c.Active != nil && *c.Active {
			line += "  " + tuiOK(cf("cli.profile.detect.active", c.Interface))
		}
		fmt.Println(line)
	}
	if example != "" {
		fmt.Println(tuiInfo(cf("cli.profile.detect.next", example)))
	}
	return nil
}

// ── profile import --create ──────────────────────────────────────────────────

// importAndCreate imports file (as runProfileImport) and creates a profile
// of its type that uses it, named like the stored file.
func importAndCreate(file, name string, overwrite bool) error {
	if strings.TrimSpace(name) == "" {
		name = profileimport.CleanName(file)
	}
	// The profile needs kongtrol.yaml: load it now (import alone does not).
	if err := loadConfig(); err != nil {
		return err
	}
	if _, exists := cfg.VPNs[name]; exists {
		return localizeConfigErr(&config.ValidationError{Key: configedit.ErrKeyVPNExists, Args: []any{name}})
	}
	res, err := runProfileImport(file, name, overwrite)
	if err != nil {
		return err
	}
	if !outputJSON && !outputQuiet {
		fmt.Println(tuiOK(cf("cli.profile.import.done", res.Path)))
		printImportHint(res)
	}
	p := configedit.VPNProfile{Name: strPtr(name), Type: strPtr(res.Type), ConfigFile: strPtr(res.Path)}
	return createProfile(p, profileImportYes)
}

func init() {
	fl := profileAddCmd.Flags()
	f := &profileAddFlags
	fl.StringVar(&f.name, "name", "", ct("cli.profile.add.flag.name"))
	fl.StringVar(&f.host, "host", "", ct("cli.profile.add.flag.host"))
	fl.IntVar(&f.port, "port", 0, ct("cli.profile.add.flag.port"))
	fl.StringVar(&f.config, "config-file", "", ct("cli.profile.add.flag.config_file"))
	fl.StringVar(&f.authMethod, "auth-method", "", ct("cli.profile.add.flag.auth_method"))
	fl.StringVar(&f.cert, "cert", "", ct("cli.profile.add.flag.cert"))
	fl.StringVar(&f.key, "key", "", ct("cli.profile.add.flag.key"))
	fl.StringVar(&f.username, "username", "", ct("cli.profile.add.flag.username"))
	fl.StringVar(&f.server, "server", "", ct("cli.profile.add.flag.server"))
	fl.StringVar(&f.country, "country", "", ct("cli.profile.add.flag.country"))
	fl.StringVar(&f.city, "city", "", ct("cli.profile.add.flag.city"))
	fl.StringVar(&f.group, "group", "", ct("cli.profile.add.flag.group"))
	fl.StringVar(&f.protocol, "protocol", "", ct("cli.profile.add.flag.protocol"))
	fl.StringVar(&f.iface, "interface", "", ct("cli.profile.add.flag.interface"))
	fl.StringVar(&f.waitTimeout, "wait-timeout", "", ct("cli.profile.add.flag.wait_timeout"))
	fl.StringVar(&f.appPath, "app-path", "", ct("cli.profile.add.flag.app_path"))
	fl.StringVar(&f.onTunnelDown, "on-tunnel-down", "", ct("cli.profile.add.flag.on_tunnel_down"))
	fl.BoolVar(&f.noAutoConnect, "no-auto-connect", false, ct("cli.profile.add.flag.no_auto_connect"))
	fl.BoolVarP(&f.yes, "yes", "y", false, ct("cli.profile.add.flag.yes"))

	profileImportCmd.Flags().BoolVar(&profileImportCreate, "create", false, ct("cli.profile.import.flag.create"))
	profileImportCmd.Flags().BoolVarP(&profileImportYes, "yes", "y", false, ct("cli.profile.import.flag.yes"))
	profileCmd.AddCommand(profileAddCmd, profileDetectCmd)
}
