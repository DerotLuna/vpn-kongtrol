package main

import (
	"errors"
	"os"
	"sort"
	"strings"

	"github.com/vpn-kongtrol/kongtrol/internal/config"
	"github.com/vpn-kongtrol/kongtrol/internal/i18n"
	"github.com/vpn-kongtrol/kongtrol/internal/monitor"
	"github.com/vpn-kongtrol/kongtrol/internal/privops"
)

func cliLang() i18n.Lang {
	lang, _ := resolveCLILang()
	return lang
}

// resolveCLILang picks the display language and says where it came from:
// KONGTROL_LANG, then the saved choice in preferences.json
// (`kongtrol config lang`), then the OS language (es* -> ES, else EN).
func resolveCLILang() (i18n.Lang, i18n.Source) {
	if lang := strings.ToLower(strings.TrimSpace(os.Getenv("KONGTROL_LANG"))); lang != "" {
		if strings.HasPrefix(lang, "es") {
			return i18n.ES, i18n.SourceEnv
		}
		if strings.HasPrefix(lang, "en") {
			return i18n.EN, i18n.SourceEnv
		}
	}
	if p, err := loadPreferences(); err == nil {
		switch strings.ToLower(strings.TrimSpace(p.Language)) {
		case "es":
			return i18n.ES, i18n.SourceSaved
		case "en":
			return i18n.EN, i18n.SourceSaved
		}
	}
	return i18n.System(), i18n.SourceSystem
}

func ct(key string) string {
	return i18n.T(cliLang(), key)
}

func cf(key string, args ...any) string {
	return i18n.F(cliLang(), key, args...)
}

// localizedErrText renders err for lang: a config.ValidationError becomes
// its translated message plus one "what to do" line; anything else keeps
// its existing text.
func localizedErrText(lang i18n.Lang, err error) string {
	if msg, hint, ok := config.LocalizeError(err, lang); ok {
		return msg + "\n    " + hint
	}
	return err.Error()
}

// localizeConfigErr is localizedErrText for CLI commands that return the
// error to cobra (printed by main): validation errors arrive translated.
func localizeConfigErr(err error) error {
	if err == nil {
		return nil
	}
	if _, _, ok := config.LocalizeError(err, cliLang()); ok {
		return errors.New(localizedErrText(cliLang(), err))
	}
	return err
}

// nextStepsProfile picks the profile name shown in the wizard's
// "kongtrol up <profile>" hint: the first one added in this run, else the
// first configured one, else a translated placeholder.
func nextStepsProfile(lang i18n.Lang, added []profileSummary, cfg *config.Config) string {
	if len(added) > 0 {
		return added[0].name
	}
	if cfg != nil && len(cfg.VPNs) > 0 {
		names := make([]string, 0, len(cfg.VPNs))
		for n := range cfg.VPNs {
			names = append(names, n)
		}
		sort.Strings(names)
		return names[0]
	}
	return i18n.T(lang, "nextsteps.profile_placeholder")
}

// tunnelErrorText is the line shown for a tunnel's last connect failure:
// the translated guidance when the daemon tagged it with an i18n key (e.g.
// the privileged helper is missing), else the daemon's message.
func tunnelErrorText(t monitor.TunnelMetrics) string {
	if t.LastErrorKey != "" {
		return ct(t.LastErrorKey)
	}
	return t.LastError
}

// errorGuidance returns the translated one-line "what to do" for err when
// it carries a guidance key: a privops error (helper missing, tunnel not
// Kongtrol's, config file not yours, …) in this process, or the
// error_key the daemon sent back (daemonctl.RemoteError).
func errorGuidance(err error) (string, bool) {
	if k, ok := privops.GuidanceKey(err); ok {
		return ct(k), true
	}
	var keyed interface{ ErrorKey() string }
	if errors.As(err, &keyed) && keyed.ErrorKey() != "" {
		return ct(keyed.ErrorKey()), true
	}
	return "", false
}
