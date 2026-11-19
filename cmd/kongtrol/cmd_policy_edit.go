package main

import (
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/spf13/cobra"

	"github.com/vpn-kongtrol/kongtrol/internal/config"
	"github.com/vpn-kongtrol/kongtrol/internal/configedit"
	"github.com/vpn-kongtrol/kongtrol/internal/daemonctl"
	"github.com/vpn-kongtrol/kongtrol/internal/policy"
)

// policy add / remove: the CLI side of the dashboard's "＋ Agregar" and
// policy delete. With a daemon they go through its API (POST/DELETE
// /api/v1/policies); without one they apply the same internal/configedit
// edit to kongtrol.yaml (the API's load → mutate → config.Validate → save
// path), so both check and store a policy the same way.

var (
	policyAddVia           string
	policyAddName          string
	policyAddBlockWhenDown bool
	policyAddNoSubdomains  bool
	policyRemoveYes        bool
)

var policyAddCmd = &cobra.Command{
	Use:  "add <site|ip|cidr|app:exe>...",
	Args: cobra.MinimumNArgs(1),
	RunE: runPolicyAdd,
}

var policyRemoveCmd = &cobra.Command{
	Use:  "remove <name>",
	Args: cobra.ExactArgs(1),
	RunE: runPolicyRemove,
}

// policyAddResult is the --json answer of policy add.
type policyAddResult struct {
	Name         string   `json:"name"`
	Via          string   `json:"via"`
	Domains      []string `json:"domains"`
	IPRanges     []string `json:"ip_ranges"`
	Apps         []string `json:"apps"`
	OnTunnelDown string   `json:"on_tunnel_down"`
	ViaDaemon    bool     `json:"via_daemon"`
}

func runPolicyAdd(cmd *cobra.Command, args []string) error {
	via := strings.TrimSpace(policyAddVia)
	if via == "" {
		return fmt.Errorf("%s", cf("cli.policy.add.via_required", profileList()))
	}
	if cfg != nil {
		if _, ok := cfg.VPNs[via]; !ok {
			// The daemon refuses it too; here the hint can list the profiles.
			return fmt.Errorf("%s", cf("cli.policy.add.via_unknown", via, profileList()))
		}
	}
	targets, invalid := policy.ClassifyTargets(args, !policyAddNoSubdomains)
	if len(invalid) > 0 {
		return fmt.Errorf("%s", cf("cli.policy.add.invalid", invalid[0]))
	}
	name := strings.TrimSpace(policyAddName)
	if name == "" {
		name = policy.AutoName(ct("cli.policy.add.auto_name"), targets.First, via, policyNames())
	}
	rule := config.PolicyRule{
		Name: name,
		Via:  via,
		Match: config.MatchSpec{
			Domains:  targets.Domains,
			IPRanges: targets.IPRanges,
			Apps:     targets.Apps,
		},
	}
	if policyAddBlockWhenDown {
		rule.OnTunnelDown = config.OnTunnelDownBlock
	}

	viaDaemon, err := savePolicy(rule)
	if err != nil {
		return err
	}
	res := policyAddResult{
		Name: rule.Name, Via: via, ViaDaemon: viaDaemon,
		Domains: nonNilStrings(targets.Domains), IPRanges: nonNilStrings(targets.IPRanges), Apps: nonNilStrings(targets.Apps),
		OnTunnelDown: config.OnTunnelDownAllow,
	}
	if policyAddBlockWhenDown {
		res.OnTunnelDown = config.OnTunnelDownBlock
	}
	if outputJSON {
		return emitJSON(res)
	}
	if outputQuiet {
		return nil
	}
	fmt.Println(tuiOK(cf("cli.policy.add.done", rule.Name, via)))
	for _, line := range []struct {
		key  string
		list []string
	}{{"cli.policy.add.sites", targets.Domains}, {"cli.policy.add.ranges", targets.IPRanges}, {"cli.policy.add.apps", targets.Apps}} {
		if len(line.list) > 0 {
			fmt.Println("  " + styleDim.Render(cf(line.key, strings.Join(line.list, ", "))))
		}
	}
	if policyAddBlockWhenDown {
		fmt.Println("  " + styleDim.Render(ct("cli.policy.add.blocked_when_down")))
	} else {
		fmt.Println("  " + styleDim.Render(ct("cli.policy.add.allowed_when_down")))
	}
	if !viaDaemon {
		fmt.Println(tuiInfo(ct("cli.policy.add.saved_offline")))
	}
	fmt.Println(tuiInfo(cf("cli.policy.add.next", policyTestTarget(targets))))
	return nil
}

// policyTestTarget is what the next-step line suggests testing: the first
// site, IP or app ("app:x"), as `kongtrol policy show` takes it.
func policyTestTarget(t policy.Targets) string {
	switch {
	case len(t.Domains) > 0:
		return t.Domains[0]
	case len(t.IPRanges) > 0:
		return strings.SplitN(t.IPRanges[0], "/", 2)[0]
	case len(t.Apps) > 0:
		return "app:" + t.Apps[0]
	}
	return t.First
}

func runPolicyRemove(cmd *cobra.Command, args []string) error {
	name := strings.TrimSpace(args[0])
	if i := policyIndex(name); i >= 0 {
		name = cfg.Policies[i].Name
	} else {
		return localizeConfigErr(&config.ValidationError{Key: configedit.ErrKeyPolicyNotFound, Args: []any{name}})
	}
	if !confirmAction(cf("cli.policy.remove.ask", name), policyRemoveYes, false) {
		if !outputQuiet && !outputJSON {
			fmt.Println(tuiInfo(cf("cli.policy.remove.hint", name)))
		}
		if outputJSON {
			return emitJSON(map[string]any{"name": name, "removed": false})
		}
		return nil
	}
	viaDaemon, err := deletePolicy(name)
	if err != nil {
		return err
	}
	if outputJSON {
		return emitJSON(map[string]any{"name": name, "removed": true, "via_daemon": viaDaemon})
	}
	if !outputQuiet {
		fmt.Println(tuiOK(cf("cli.policy.remove.done", name)))
		fmt.Println(tuiInfo(ct("cli.policy.remove.next")))
	}
	return nil
}

// savePolicy adds rule through the running daemon, or to kongtrol.yaml.
func savePolicy(rule config.PolicyRule) (viaDaemon bool, err error) {
	if base := daemonAPIBase(); daemonProbeFn(base) {
		return true, remoteUserErr(configClient(base).CreatePolicy(rule))
	}
	return false, editConfigOffline(func(c *config.Config) error {
		_, err := configedit.AddPolicy(c, rule)
		return err
	})
}

// deletePolicy removes the policy named name through the running daemon, or
// from kongtrol.yaml.
func deletePolicy(name string) (viaDaemon bool, err error) {
	if base := daemonAPIBase(); daemonProbeFn(base) {
		return true, remoteUserErr(configClient(base).DeletePolicy(name))
	}
	return false, editConfigOffline(func(c *config.Config) error { return configedit.RemovePolicy(c, name) })
}

// editConfigOffline is the daemon API's save path for when no daemon runs:
// kongtrol.yaml is loaded fresh (no machine-local preferences mixed in),
// edit changes it, configedit.Save validates and writes it.
func editConfigOffline(edit func(*config.Config) error) error {
	c, err := config.Load(activeCfgPath)
	if err != nil {
		return localizeConfigErr(err)
	}
	if err := edit(c); err != nil {
		return localizeConfigErr(err)
	}
	_, err = configedit.Save(activeCfgPath, c)
	return localizeConfigErr(err)
}

// configClient is the daemon client for config edits: it asks for errors
// in the CLI's language.
func configClient(base string) *daemonctl.Client {
	c := daemonClient(base)
	c.Lang = string(cliLang())
	return c
}

// remoteUserErr turns a translated daemon refusal into the message plus its
// "what to do" line; other errors stay as they are.
func remoteUserErr(err error) error {
	var re *daemonctl.RemoteError
	if errors.As(err, &re) {
		if text, ok := re.UserText(); ok {
			return errors.New(text)
		}
	}
	return err
}

func policyIndex(name string) int {
	if cfg == nil {
		return -1
	}
	return configedit.FindPolicy(cfg, name)
}

func policyNames() []string {
	if cfg == nil {
		return nil
	}
	out := make([]string, 0, len(cfg.Policies))
	for _, p := range cfg.Policies {
		out = append(out, p.Name)
	}
	return out
}

// profileList is the configured profile names, comma-separated, for hints.
func profileList() string {
	if cfg == nil || len(cfg.VPNs) == 0 {
		return ct("cli.placeholder.profile")
	}
	names := make([]string, 0, len(cfg.VPNs))
	for n := range cfg.VPNs {
		names = append(names, n)
	}
	sort.Strings(names)
	return strings.Join(names, ", ")
}

func nonNilStrings(s []string) []string {
	if s == nil {
		return []string{}
	}
	return s
}

func init() {
	policyAddCmd.Flags().StringVar(&policyAddVia, "via", "", ct("cli.policy.add.flag.via"))
	policyAddCmd.Flags().StringVar(&policyAddName, "name", "", ct("cli.policy.add.flag.name"))
	policyAddCmd.Flags().BoolVar(&policyAddBlockWhenDown, "block-when-down", false, ct("cli.policy.add.flag.block_when_down"))
	policyAddCmd.Flags().BoolVar(&policyAddNoSubdomains, "no-subdomains", false, ct("cli.policy.add.flag.no_subdomains"))
	policyRemoveCmd.Flags().BoolVarP(&policyRemoveYes, "yes", "y", false, ct("cli.policy.remove.flag.yes"))
	policyCmd.AddCommand(policyAddCmd, policyRemoveCmd)
}
