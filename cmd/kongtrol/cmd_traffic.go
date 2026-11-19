package main

import (
	"bytes"
	"fmt"
	"os"
	"strconv"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"

	"github.com/vpn-kongtrol/kongtrol/internal/config"
	"github.com/vpn-kongtrol/kongtrol/internal/daemonctl"
)

// Recent destinations (ADR-0054): `kongtrol traffic recent` reads the
// daemon's in-memory list; `kongtrol config recent-connections` turns it
// on or off. Same paths as the dashboard: GET /api/v1/connections/recent
// and PUT /api/v1/settings.

var trafficCmd = &cobra.Command{
	Use:  "traffic",
	Args: cobra.NoArgs,
}

var recentLimit int

// daemonRecentFn is replaced in tests.
var daemonRecentFn = func(base string, limit int) (daemonctl.RecentConnections, error) {
	return daemonClient(base).RecentConnections(limit)
}

var trafficRecentCmd = &cobra.Command{
	Use:  "recent",
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, _ []string) error {
		if recentLimit < 0 {
			return guidedError("api.err.recent_limit", nil, "api.err.recent_limit.fix")
		}
		base := daemonAPIBase()
		if !probeDaemonAPI(base) {
			return guidedError("cli.recent.no_daemon", nil, "cli.recent.no_daemon_do")
		}
		res, err := daemonRecentFn(base, recentLimit)
		if err != nil {
			return err
		}
		if outputJSON {
			return emitJSON(res)
		}
		switch {
		case !res.Supported:
			fmt.Println(tuiWarn(ct("cli.recent.unsupported")))
			return nil
		case !res.Enabled:
			fmt.Println(tuiInfo(ct("cli.recent.disabled")))
			fmt.Println(tuiInfo(ct("cli.recent.disabled_do")))
			return nil
		case len(res.Connections) == 0:
			fmt.Println(tuiInfo(ct("cli.recent.empty")))
			return nil
		}
		tw := tabwriter.NewWriter(os.Stdout, 0, 2, 2, ' ', 0)
		fmt.Fprintln(tw, strings.Join([]string{
			ct("cli.recent.col.dest"), ct("cli.recent.col.port"), ct("cli.recent.col.process"),
			ct("cli.recent.col.via"), ct("cli.recent.col.count"), ct("cli.recent.col.last"),
		}, "\t"))
		for _, c := range res.Connections {
			dest := c.RemoteIP
			if c.Host != "" {
				dest = c.Host + " (" + c.RemoteIP + ")"
			}
			via := c.Via
			if via == "" {
				via = ct("cli.recent.direct")
			}
			if c.Interface != "" {
				via += " · " + c.Interface
			}
			proc := c.Process
			if proc == "" {
				proc = "—"
			}
			fmt.Fprintln(tw, strings.Join([]string{
				dest, strconv.Itoa(int(c.Port)), proc, via, strconv.Itoa(c.Count),
				c.LastSeen.Local().Format(time.TimeOnly),
			}, "\t"))
		}
		return tw.Flush()
	},
}

// configRecentCmd: on / off / status. Through the daemon when it runs
// (applies at once and erases the list when off), else in kongtrol.yaml.
var configRecentCmd = &cobra.Command{
	Use:  "recent-connections <on|off|status>",
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		var enabled bool
		switch strings.ToLower(strings.TrimSpace(args[0])) {
		case "on", "true", "si", "sí", "yes":
			enabled = true
		case "off", "false", "no":
		case "status":
			if cfg.RecentConnectionsEnabled() {
				fmt.Println(tuiInfo(ct("cli.recent.status_on")))
			} else {
				fmt.Println(tuiInfo(ct("cli.recent.status_off")))
			}
			return nil
		default:
			return fmt.Errorf("%s", cf("cli.recent.invalid", args[0]))
		}
		if base := daemonAPIBase(); probeDaemonAPI(base) {
			if err := daemonClient(base).SetRecentConnections(enabled); err != nil {
				return err
			}
		} else if err := setRecentConnectionsInFile(activeCfgPath, enabled); err != nil {
			return err
		}
		if !enabled {
			fmt.Println(tuiOK(ct("cli.recent.off")))
			return nil
		}
		fmt.Println(tuiOK(ct("cli.recent.on")))
		fmt.Println(tuiInfo(paint(cDim, ct("cli.recent.notice"))))
		return nil
	},
}

// setRecentConnectionsInFile sets monitor.recent_connections.enabled in
// the YAML file, keeping comments.
func setRecentConnectionsInFile(path string, enabled bool) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	var doc yaml.Node
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return err
	}
	if doc.Kind != yaml.DocumentNode || len(doc.Content) == 0 || doc.Content[0].Kind != yaml.MappingNode {
		return fmt.Errorf("%s: not a YAML mapping", path)
	}
	root := doc.Content[0]
	mon := mappingKey(root, "monitor")
	if mon == nil || mon.Kind != yaml.MappingNode {
		mon = &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}
		setMapping(root, "monitor", mon)
	}
	rc := mappingKey(mon, "recent_connections")
	if rc == nil || rc.Kind != yaml.MappingNode {
		rc = &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}
		setMapping(mon, "recent_connections", rc)
	}
	setMapping(rc, "enabled", boolNode(enabled))
	var buf bytes.Buffer
	enc := yaml.NewEncoder(&buf)
	enc.SetIndent(2)
	if err := enc.Encode(&doc); err != nil {
		return err
	}
	_ = enc.Close()
	return config.WriteFileAtomic(path, buf.Bytes(), 0o600)
}

func init() {
	trafficRecentCmd.Flags().IntVar(&recentLimit, "limit", 0, ct("cli.recent.flag.limit"))
	trafficCmd.AddCommand(trafficRecentCmd)
	rootCmd.AddCommand(trafficCmd)
	configCmd.AddCommand(configRecentCmd)
}
