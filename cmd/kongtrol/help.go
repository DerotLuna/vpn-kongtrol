package main

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/vpn-kongtrol/kongtrol/internal/i18n"
)

// Translated, grouped help built on cobra's own extension points:
// Command.SetUsageTemplate / SetHelpTemplate with cobra.AddTemplateFunc,
// Command.AddGroup + GroupID, SetHelpCommand / SetHelpCommandGroupID and a
// pre-registered "help" flag (cobra's InitDefaultHelpFlag only adds its
// English one when the flag is missing). See
// https://pkg.go.dev/github.com/spf13/cobra#Command and
// https://github.com/spf13/cobra/blob/main/site/content/user_guide.md
// ("Grouping commands in help", "Defining your own help", "Defining your own usage").
//
// Every user-visible text comes from internal/i18n with keys derived from the
// command path: help.<path>.short / .long / .example and the optional
// help.<path>.args (the arguments part of Use), e.g. help.kongtrol.config.lang.long.

const (
	groupConnect     = "connect"
	groupSetup       = "setup"
	groupDiagnostics = "diagnostics"
	groupSystem      = "system"
)

// rootLayout is the order and group of the top-level commands in --help.
var rootLayout = []struct {
	name, group string
}{
	{"up", groupConnect}, {"down", groupConnect}, {"status", groupConnect},
	{"routes", groupConnect}, {"dashboard", groupConnect}, {"tray", groupConnect},
	{"init", groupSetup}, {"config", groupSetup}, {"profile", groupSetup}, {"policy", groupSetup},
	{"reload", groupSetup}, {"restart", groupSetup}, {"export", groupSetup},
	{"doctor", groupDiagnostics}, {"ip", groupDiagnostics}, {"traffic", groupDiagnostics}, {"logs", groupDiagnostics},
	{"service", groupSystem}, {"uninstall", groupSystem}, {"version", groupSystem}, {"help", groupSystem},
}

var helpCmd = &cobra.Command{
	Use: "help [command]",
	// No config load or banner: help works before `kongtrol init`.
	PersistentPreRunE: func(*cobra.Command, []string) error { return nil },
	RunE: func(c *cobra.Command, args []string) error {
		cmd, _, err := c.Root().Find(args)
		if cmd == nil || err != nil {
			return fmt.Errorf("%s", cf("cli.error.unknown_command", strings.Join(args, " ")))
		}
		return cmd.Help()
	},
}

// helpKey is the i18n key prefix for cmd: "help." + its path with dots.
func helpKey(cmd *cobra.Command) string {
	return "help." + strings.ReplaceAll(cmd.CommandPath(), " ", ".")
}

// setupHelp translates and groups the whole command tree for lang. It runs
// once every init() registered its commands, and is safe to call again.
func setupHelp(root *cobra.Command, lang i18n.Lang) {
	t := func(key string) string { return i18n.T(lang, key) }

	root.SetHelpCommand(helpCmd)
	root.InitDefaultHelpCmd()

	groups := map[string]string{
		groupConnect:     "help.group.connect",
		groupSetup:       "help.group.setup",
		groupDiagnostics: "help.group.diagnostics",
		groupSystem:      "help.group.system",
	}
	for _, id := range []string{groupConnect, groupSetup, groupDiagnostics, groupSystem} {
		if !root.ContainsGroup(id) {
			root.AddGroup(&cobra.Group{ID: id})
		}
	}
	for _, g := range root.Groups() {
		g.Title = t(groups[g.ID])
	}

	// Re-add the top-level commands in rootLayout order (cobra keeps
	// insertion order with EnableCommandSorting = false); unknown ones last.
	existing := root.Commands()
	byName := map[string]*cobra.Command{}
	for _, c := range existing {
		byName[c.Name()] = c
	}
	root.RemoveCommand(existing...)
	placed := map[string]bool{}
	for _, l := range rootLayout {
		if c, ok := byName[l.name]; ok {
			c.GroupID = l.group
			root.AddCommand(c)
			placed[l.name] = true
		}
	}
	for _, c := range existing {
		if !placed[c.Name()] {
			root.AddCommand(c)
		}
	}
	root.SetHelpCommandGroupID(groupSystem)

	walkVisible(root, func(c *cobra.Command) {
		key := helpKey(c)
		c.Short = t(key + ".short")
		c.Long = t(key + ".long")
		c.Example = t(key + ".example")
		if args := key + ".args"; t(args) != args {
			c.Use = c.Name() + " " + t(args)
		}
		usage := i18n.F(lang, "help.flag.help", c.Name())
		if f := c.Flags().Lookup("help"); f != nil {
			f.Usage = usage
		} else {
			c.Flags().BoolP("help", "h", false, usage)
		}
		c.LocalFlags().VisitAll(func(f *pflag.Flag) {
			if f.Name != "help" {
				translateFlag(c, f, lang)
			}
		})
	})

	cobra.AddTemplateFunc("ht", t)
	cobra.AddTemplateFunc("hf", func(key string, args ...any) string { return i18n.F(lang, key, args...) })
	cobra.AddTemplateFunc("useLine", func(c *cobra.Command) string {
		return strings.Replace(c.UseLine(), "[flags]", t("help.tpl.flags_arg"), 1)
	})
	defaults := t("help.tpl.default")
	cobra.AddTemplateFunc("localDefaults", func(s string) string {
		return strings.ReplaceAll(s, "(default ", "("+defaults+" ")
	})
	root.SetUsageTemplate(usageTemplate)
	root.SetHelpTemplate(helpTemplate)
}

// annotationFlagKey stores the i18n key of a flag's usage text.
const annotationFlagKey = "kongtrol.i18n-key"

// translateFlag sets f.Usage in lang. Flags are declared in init() with
// ct(key), i.e. already rendered in the start-up language; the key is
// recovered once from that text (preferring a key that names the command,
// then the flag, then the first in sorted order, so the choice is stable)
// and kept in the flag's annotations. Hidden flags have no usage text.
func translateFlag(c *cobra.Command, f *pflag.Flag, lang i18n.Lang) {
	if f.Hidden {
		return
	}
	key := ""
	if a := f.Annotations[annotationFlagKey]; len(a) == 1 {
		key = a[0]
	} else {
		best := -1
		for _, k := range i18n.KeysFor(f.Usage) {
			score := 0
			if strings.Contains(k, "."+c.Name()+".") {
				score += 2
			}
			if strings.Contains(k, strings.ReplaceAll(f.Name, "-", "_")) {
				score++
			}
			if score > best {
				key, best = k, score
			}
		}
		if key == "" {
			return
		}
		if f.Annotations == nil {
			f.Annotations = map[string][]string{}
		}
		f.Annotations[annotationFlagKey] = []string{key}
	}
	f.Usage = i18n.T(lang, key)
}

// walkVisible calls fn for c and every descendant, skipping hidden
// subtrees (daemon, helper, …): they never show in help.
func walkVisible(c *cobra.Command, fn func(*cobra.Command)) {
	if c.Hidden {
		return
	}
	fn(c)
	for _, sub := range c.Commands() {
		walkVisible(sub, fn)
	}
}

// usageTemplate is cobra's default usage template (v1.8) with translated
// headings, localized "[flags]" / "(default …)" and "Commands:" for
// subcommands outside any group.
const usageTemplate = `{{ht "help.tpl.usage"}}{{if .Runnable}}
  {{useLine .}}{{end}}{{if .HasAvailableSubCommands}}
  {{.CommandPath}} {{ht "help.tpl.command_arg"}}{{end}}{{if gt (len .Aliases) 0}}

{{ht "help.tpl.aliases"}}
  {{.NameAndAliases}}{{end}}{{if .HasExample}}

{{ht "help.tpl.examples"}}
{{.Example}}{{end}}{{if .HasAvailableSubCommands}}{{$cmds := .Commands}}{{if eq (len .Groups) 0}}

{{ht "help.tpl.commands"}}{{range $cmds}}{{if (or .IsAvailableCommand (eq .Name "help"))}}
  {{rpad .Name .NamePadding }} {{.Short}}{{end}}{{end}}{{else}}{{range $group := .Groups}}

{{.Title}}{{range $cmds}}{{if (and (eq .GroupID $group.ID) (or .IsAvailableCommand (eq .Name "help")))}}
  {{rpad .Name .NamePadding }} {{.Short}}{{end}}{{end}}{{end}}{{if not .AllChildCommandsHaveGroup}}

{{ht "help.tpl.commands"}}{{range $cmds}}{{if (and (eq .GroupID "") (or .IsAvailableCommand (eq .Name "help")))}}
  {{rpad .Name .NamePadding }} {{.Short}}{{end}}{{end}}{{end}}{{end}}{{end}}{{if .HasAvailableLocalFlags}}

{{ht "help.tpl.flags"}}
{{.LocalFlags.FlagUsages | trimTrailingWhitespaces | localDefaults}}{{end}}{{if .HasAvailableInheritedFlags}}

{{ht "help.tpl.global_flags"}}
{{.InheritedFlags.FlagUsages | trimTrailingWhitespaces | localDefaults}}{{end}}{{if .HasHelpSubCommands}}

{{ht "help.tpl.topics"}}{{range .Commands}}{{if .IsAdditionalHelpTopicCommand}}
  {{rpad .CommandPath .CommandPathPadding}} {{.Short}}{{end}}{{end}}{{end}}{{if .HasAvailableSubCommands}}

{{hf "help.tpl.more" .CommandPath}}{{end}}
`

// helpTemplate is cobra's default help template plus the guide link.
const helpTemplate = `{{with (or .Long .Short)}}{{. | trimTrailingWhitespaces}}

{{end}}{{if or .Runnable .HasSubCommands}}{{.UsageString}}{{end}}
{{ht "help.tpl.guide"}}
`
