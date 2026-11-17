package main

import (
	"strings"
	"testing"

	"github.com/vpn-kongtrol/kongtrol/internal/i18n"
)

func TestServiceCommandsRegistered(t *testing.T) {
	for _, path := range [][]string{{"service", "install"}, {"service", "uninstall"}, {"service", "status"}, {"helper", "run"}} {
		c, _, err := rootCmd.Find(path)
		if err != nil || c.Name() != path[1] {
			t.Errorf("%v not registered: %v", path, err)
		}
	}
	if c, _, _ := rootCmd.Find([]string{"helper"}); !c.Hidden {
		t.Error("helper command must be hidden")
	}
}

func TestServiceMessagesTranslated(t *testing.T) {
	keys := []string{
		"help.kongtrol.service.short", "help.kongtrol.service.install.short", "help.kongtrol.service.uninstall.short", "help.kongtrol.service.status.short",
		"service.install.start", "service.install.uac", "service.install.sudo", "service.install.verifying",
		"service.install.ok", "service.install.next", "service.install.failed_detail", "service.install.failed_hint",
		"service.install.no_response", "service.install.no_response_hint", "service.elevation.cancelled",
		"service.elevation.cancelled_hint", "service.elevated.exit_code", "service.uninstall.start",
		"service.uninstall.ok", "service.uninstall.next", "service.uninstall.failed_detail", "service.uninstall.failed_hint",
		"service.status.not_installed", "service.status.installed", "service.status.stopped", "service.status.stopped_hint",
		"service.status.denied", "service.status.denied_hint", "service.status.running", "service.status.version_mismatch",
		"service.status.version_mismatch_hint", "service.status.ok", "service.error.not_installed",
		"service.error.outdated", "service.error.denied",
	}
	for _, k := range keys {
		es, en := i18n.T(i18n.ES, k), i18n.T(i18n.EN, k)
		if es == k || en == k || es == en {
			t.Errorf("%s: missing or untranslated (es=%q en=%q)", k, es, en)
		}
		if strings.Contains(es, "\n") && !strings.HasPrefix(k, "cli.") {
			t.Errorf("%s: guidance must be one line", k)
		}
	}
}
