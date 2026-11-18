package main

import (
	"bytes"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"

	"github.com/vpn-kongtrol/kongtrol/internal/config"
)

// configPublicIPCmd turns the opt-in public IP lookup on or off
// (security.public_ip_check.enabled). With a running daemon it goes through
// its API, so the change applies at once; otherwise kongtrol.yaml is edited
// in place, keeping comments.
var configPublicIPCmd = &cobra.Command{
	Use:  "public-ip <on|off>",
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		var enabled bool
		switch strings.ToLower(strings.TrimSpace(args[0])) {
		case "on", "true", "si", "sí", "yes":
			enabled = true
		case "off", "false", "no":
		default:
			return fmt.Errorf("%s", cf("cli.public_ip.invalid", args[0]))
		}
		host := cfg.PublicIPHost()
		if base := daemonAPIBase(); probeDaemonAPI(base) {
			st, err := daemonClient(base).SetPublicIPCheck(enabled)
			if err != nil {
				return err
			}
			if st.Host != "" {
				host = st.Host
			}
		} else if err := setPublicIPInFile(activeCfgPath, enabled); err != nil {
			return err
		}
		if !enabled {
			fmt.Println(tuiOK(ct("cli.public_ip.off")))
			return nil
		}
		fmt.Println(tuiOK(ct("cli.public_ip.on")))
		fmt.Println(tuiInfo(paint(cDim, cf("cli.public_ip.notice", host))))
		fmt.Println(tuiInfo(paint(cDim, ct("cli.public_ip.next"))))
		return nil
	},
}

// setPublicIPInFile edits security.public_ip_check.enabled in the YAML
// file (and clears the legacy leak_detection.enabled when turning off).
func setPublicIPInFile(path string, enabled bool) error {
	return editConfigYAML(path, func(root *yaml.Node) {
		sec := childMapping(root, "security")
		setMapping(childMapping(sec, "public_ip_check"), "enabled", boolNode(enabled))
		if ld := mappingKey(sec, "leak_detection"); !enabled && ld != nil && ld.Kind == yaml.MappingNode {
			setMapping(ld, "enabled", boolNode(false))
		}
	})
}

// childMapping returns parent's mapping under key, creating it (or
// replacing a non-mapping value) when needed.
func childMapping(parent *yaml.Node, key string) *yaml.Node {
	n := mappingKey(parent, key)
	if n == nil || n.Kind != yaml.MappingNode {
		n = &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}
		setMapping(parent, key, n)
	}
	return n
}

// editConfigYAML applies edit to kongtrol.yaml's root mapping and writes it
// back atomically, keeping comments and key order (yaml.v3 node tree). Used
// by the config toggles when no daemon is running.
func editConfigYAML(path string, edit func(root *yaml.Node)) error {
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
	edit(doc.Content[0])
	var buf bytes.Buffer
	enc := yaml.NewEncoder(&buf)
	enc.SetIndent(2)
	if err := enc.Encode(&doc); err != nil {
		return err
	}
	_ = enc.Close()
	// Only booleans change, so the file stays as valid as it was.
	return config.WriteFileAtomic(path, buf.Bytes(), 0o600)
}

func init() {
	configCmd.AddCommand(configPublicIPCmd)
}
