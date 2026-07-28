package main

import (
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"

	"github.com/vpn-kongtrol/kongtrol/internal/i18n"
)

// ── Wizard field validators ───────────────────────────────────────────────────
//
// Pure functions plugged into huh.Input.Validate(...) — no TTY, no i18n
// lookups baked into the message beyond what's passed in, so they're testable
// in isolation (see wizard_validate_test.go).

// validateHost rejects empty hosts and common typos (a URL scheme, embedded
// spaces). Anything that parses as an IP is accepted outright; otherwise it
// must look like a bare hostname/domain.
func validateHost(lang i18n.Lang) func(string) error {
	return func(s string) error {
		s = strings.TrimSpace(s)
		if s == "" {
			return fmt.Errorf("%s", i18n.T(lang, "validate.host_empty"))
		}
		if net.ParseIP(s) != nil {
			return nil
		}
		if strings.ContainsAny(s, " \t") || strings.Contains(s, "://") {
			return fmt.Errorf("%s", i18n.T(lang, "validate.host_invalid"))
		}
		for label := range strings.SplitSeq(s, ".") {
			if label == "" {
				return fmt.Errorf("%s", i18n.T(lang, "validate.host_invalid"))
			}
		}
		return nil
	}
}

// validatePort accepts empty (caller applies its own default) and any integer
// in the valid TCP/UDP port range.
func validatePort(lang i18n.Lang) func(string) error {
	return func(s string) error {
		s = strings.TrimSpace(s)
		if s == "" {
			return nil
		}
		n, err := strconv.Atoi(s)
		if err != nil || n < 1 || n > 65535 {
			return fmt.Errorf("%s", i18n.T(lang, "validate.port_invalid"))
		}
		return nil
	}
}

// validateExistingPath accepts empty (field is optional/embedded elsewhere)
// and otherwise requires the path to exist on disk.
func validateExistingPath(lang i18n.Lang) func(string) error {
	return func(s string) error {
		s = strings.TrimSpace(s)
		if s == "" {
			return nil
		}
		if _, err := os.Stat(s); err != nil {
			return fmt.Errorf(i18n.T(lang, "validate.path_not_found"), s)
		}
		return nil
	}
}

// validateProfileName requires a non-empty name; normalization to
// lowercase-with-dashes happens in the caller after this passes.
func validateProfileName(lang i18n.Lang) func(string) error {
	return func(s string) error {
		if strings.TrimSpace(s) == "" {
			return fmt.Errorf("%s", i18n.T(lang, "collect.profile_name_empty"))
		}
		return nil
	}
}
