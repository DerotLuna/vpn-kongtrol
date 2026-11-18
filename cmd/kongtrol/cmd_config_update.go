package main

import (
	"bytes"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"

	"github.com/vpn-kongtrol/kongtrol/internal/config"
	"github.com/vpn-kongtrol/kongtrol/internal/update"
)

// configUpdateCheckCmd turns the opt-in update notice on or off
// (updates.check_enabled, ADR-0032), or shows its state. Same path as
// `config public-ip`: through the daemon's API when it runs (applies at
// once), otherwise editing kongtrol.yaml in place, keeping comments.
var configUpdateCheckCmd = &cobra.Command{
	Use:  "update-check <on|off|status>",
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		var enabled bool
		switch strings.ToLower(strings.TrimSpace(args[0])) {
		case "on", "true", "si", "sí", "yes":
			enabled = true
		case "off", "false", "no":
		case "status":
			return printUpdateCheckStatus()
		default:
			return fmt.Errorf("%s", cf("cli.update.invalid", args[0]))
		}
		if base := daemonAPIBase(); probeDaemonAPI(base) {
			if _, err := daemonClient(base).SetUpdateCheck(enabled); err != nil {
				return err
			}
		} else if err := setUpdateCheckInFile(activeCfgPath, enabled); err != nil {
			return err
		}
		if !enabled {
			fmt.Println(tuiOK(ct("cli.update.off")))
			return nil
		}
		fmt.Println(tuiOK(ct("cli.update.on")))
		fmt.Println(tuiInfo(paint(cDim, ct("cli.update.notice"))))
		return nil
	},
}

func printUpdateCheckStatus() error {
	if !cfg.UpdateCheckEnabled() {
		fmt.Println(tuiInfo(ct("cli.update.status_off")))
		return nil
	}
	r := update.LastFrom(version, update.DefaultStatePath())
	when := ct("cli.update.never")
	if !r.CheckedAt.IsZero() {
		when = r.CheckedAt.Local().Format(time.DateTime)
	}
	fmt.Println(tuiInfo(cf("cli.update.status_on", when)))
	if r.Newer {
		fmt.Println(tuiInfo(cf("cli.update.available", r.Latest, r.Current)))
	}
	return nil
}

// setUpdateCheckInFile sets updates.check_enabled in the YAML file.
func setUpdateCheckInFile(path string, enabled bool) error {
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
	up := mappingKey(root, "updates")
	if up == nil || up.Kind != yaml.MappingNode {
		up = &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}
		setMapping(root, "updates", up)
	}
	setMapping(up, "check_enabled", boolNode(enabled))
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
	configCmd.AddCommand(configUpdateCheckCmd)
}
