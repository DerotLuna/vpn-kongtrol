package docgen

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/vpn-kongtrol/kongtrol/internal/i18n"
)

// CLIFlag is one flag as shown in help: "-f, --follow", "--tail N".
type CLIFlag struct {
	Flag    string `json:"flag"`
	Usage   string `json:"usage"`
	Default string `json:"default,omitempty"`
}

// CLICommand is one visible command.
type CLICommand struct {
	Path    string    `json:"path"`
	Usage   string    `json:"usage"`
	Short   string    `json:"short"`
	Long    string    `json:"long"`
	Example string    `json:"example,omitempty"`
	Aliases []string  `json:"aliases,omitempty"`
	Flags   []CLIFlag `json:"flags,omitempty"`
}

// CLIGroup is a help group ("Conexión:") with its commands in help order.
type CLIGroup struct {
	ID       string       `json:"id"`
	Title    string       `json:"title"`
	Commands []CLICommand `json:"commands"`
}

// CLIRef is the whole reference in one language.
type CLIRef struct {
	Lang        i18n.Lang         `json:"-"`
	Labels      map[string]string `json:"labels"`
	Groups      []CLIGroup        `json:"groups"`
	GlobalFlags []CLIFlag         `json:"globalFlags"`
}

// BuildCLI reads the command tree. The caller must have translated it into
// lang first (cmd/kongtrol setupHelp), exactly as `kongtrol --help` shows it.
func BuildCLI(root *cobra.Command, lang i18n.Lang) CLIRef {
	t := func(k string) string { return strings.TrimSuffix(i18n.T(lang, k), ":") }
	ref := CLIRef{Lang: lang, Labels: map[string]string{
		"usage":       t("help.tpl.usage"),
		"flags":       t("help.tpl.flags"),
		"globalFlags": t("help.tpl.global_flags"),
		"examples":    t("help.tpl.examples"),
		"aliases":     t("help.tpl.aliases"),
		"default":     t("help.tpl.default"),
		"command":     text(lang, "cli.col.command"),
		"description": text(lang, "cli.col.description"),
	}}
	for _, g := range root.Groups() {
		group := CLIGroup{ID: g.ID, Title: strings.TrimSuffix(g.Title, ":")}
		for _, c := range root.Commands() {
			if c.GroupID == g.ID {
				walkCommands(c, func(c *cobra.Command) { group.Commands = append(group.Commands, buildCommand(c, lang)) })
			}
		}
		ref.Groups = append(ref.Groups, group)
	}
	root.LocalFlags().VisitAll(func(f *pflag.Flag) {
		if !f.Hidden {
			ref.GlobalFlags = append(ref.GlobalFlags, buildFlag(f))
		}
	})
	return ref
}

// walkCommands visits c and its visible descendants in help order.
func walkCommands(c *cobra.Command, fn func(*cobra.Command)) {
	if c.Hidden || (!c.IsAvailableCommand() && c.Name() != "help") {
		return
	}
	fn(c)
	for _, sub := range c.Commands() {
		walkCommands(sub, fn)
	}
}

func buildCommand(c *cobra.Command, lang i18n.Lang) CLICommand {
	usage := c.CommandPath() + " " + i18n.T(lang, "help.tpl.command_arg")
	if c.Runnable() {
		usage = strings.Replace(c.UseLine(), "[flags]", i18n.T(lang, "help.tpl.flags_arg"), 1)
	}
	var lines []string
	for _, l := range strings.Split(strings.TrimSpace(c.Example), "\n") {
		lines = append(lines, strings.TrimSpace(l))
	}
	cmd := CLICommand{
		Path: c.CommandPath(), Usage: usage, Short: c.Short, Long: c.Long,
		Example: strings.Join(lines, "\n"), Aliases: c.Aliases,
	}
	if cmd.Long == "" {
		cmd.Long = cmd.Short
	}
	c.LocalNonPersistentFlags().VisitAll(func(f *pflag.Flag) {
		if !f.Hidden && f.Name != "help" {
			cmd.Flags = append(cmd.Flags, buildFlag(f))
		}
	})
	return cmd
}

func buildFlag(f *pflag.Flag) CLIFlag {
	name, usage := pflag.UnquoteUsage(f)
	flag := "--" + f.Name
	if f.Shorthand != "" {
		flag = "-" + f.Shorthand + ", " + flag
	}
	if name != "" {
		flag += " " + name
	}
	def := f.DefValue
	switch def {
	case "", "false", "0", "[]":
		def = ""
	}
	return CLIFlag{Flag: flag, Usage: usage, Default: def}
}

// CLIMarkdown renders docs/CLI.md (es) or docs/CLI.en.md (en).
func CLIMarkdown(ref CLIRef, s Stamp) []byte {
	lang := ref.Lang
	var b bytes.Buffer
	fmt.Fprintf(&b, "<!-- %s -->\n\n", text(lang, "generated.cli"))
	b.WriteString(logoLine)
	fmt.Fprintf(&b, "# %s\n\n", text(lang, "cli.title"))
	fmt.Fprintf(&b, "%s %s · %s\n\n", text(lang, "version"), s.Version, s.Date)
	fmt.Fprintf(&b, "%s\n\n", text(lang, "cli.intro"))
	for _, g := range ref.Groups {
		fmt.Fprintf(&b, "## %s\n\n", g.Title)
		for _, c := range g.Commands {
			fmt.Fprintf(&b, "### `%s`\n\n%s\n\n```text\n%s\n```\n\n", c.Path, c.Long, c.Usage)
			if len(c.Aliases) > 0 {
				fmt.Fprintf(&b, "%s: `%s`\n\n", ref.Labels["aliases"], strings.Join(c.Aliases, "`, `"))
			}
			if len(c.Flags) > 0 {
				fmt.Fprintf(&b, "**%s**\n\n", ref.Labels["flags"])
				flagTable(&b, ref, c.Flags)
			}
			if c.Example != "" {
				fmt.Fprintf(&b, "**%s**\n\n```bash\n%s\n```\n\n", ref.Labels["examples"], c.Example)
			}
		}
	}
	fmt.Fprintf(&b, "## %s\n\n", ref.Labels["globalFlags"])
	flagTable(&b, ref, ref.GlobalFlags)
	fmt.Fprintf(&b, "## %s\n\n%s\n\n", text(lang, "cli.lang.title"), text(lang, "cli.lang.body"))
	fmt.Fprintf(&b, "## %s\n\n%s\n", text(lang, "cli.more.title"), text(lang, "cli.more.body"))
	return b.Bytes()
}

func flagTable(b *bytes.Buffer, ref CLIRef, flags []CLIFlag) {
	fmt.Fprintf(b, "| %s | %s |\n|---|---|\n", ref.Labels["flags"], ref.Labels["description"])
	for _, f := range flags {
		usage := f.Usage
		if f.Default != "" {
			usage += fmt.Sprintf(" (%s `%s`)", ref.Labels["default"], f.Default)
		}
		fmt.Fprintf(b, "| `%s` | %s |\n", cell(f.Flag), cell(usage))
	}
	b.WriteString("\n")
}

// cell escapes a Markdown table cell (GFM: "\|" also inside code spans).
func cell(s string) string { return strings.ReplaceAll(s, "|", `\|`) }

// CLIJSON renders docs/cli.json: both languages, consumed by the landing
// guide's command reference at build time.
func CLIJSON(refs []CLIRef, s Stamp) ([]byte, error) {
	type doc struct {
		Comment string            `json:"$comment"`
		Version string            `json:"version"`
		Date    string            `json:"date"`
		Langs   map[string]CLIRef `json:"langs"`
	}
	d := doc{Comment: text(i18n.EN, "generated.cli"), Version: s.Version, Date: s.Date, Langs: map[string]CLIRef{}}
	for _, r := range refs {
		d.Langs[string(r.Lang)] = r
	}
	var b bytes.Buffer
	enc := json.NewEncoder(&b)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	if err := enc.Encode(d); err != nil {
		return nil, err
	}
	return b.Bytes(), nil
}
