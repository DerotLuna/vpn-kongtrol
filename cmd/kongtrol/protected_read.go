package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"
	"github.com/vpn-kongtrol/kongtrol/internal/config"
	"github.com/vpn-kongtrol/kongtrol/internal/profileimport"
)

// Root-only profile files (ADR-0058). A WireGuard config in /etc/wireguard
// is root:root 0600 inside a 0700 folder; Kongtrol runs as the user, so it
// can't read it. Instead of a helper operation that would hand root-only
// secrets to any process of the user without asking, the user gives one
// OS permission prompt (sudo in a terminal, the desktop's pkexec dialog
// from the dashboard) to a short-lived copy of kongtrol that releases the
// file only if it passes the helper's own checks
// (profileimport.ReadProtected); the content is then imported like any
// other file into ~/.kongtrol/configs (0600).

// errElevatedReadUnavailable: no way to ask for permission here (Windows,
// or macOS without a terminal).
var errElevatedReadUnavailable = errors.New("elevated read unavailable")

// errElevatedReadCancelled: the user dismissed the permission prompt.
var errElevatedReadCancelled = errors.New("elevated read cancelled")

// readProtectedFn reads a root-only profile file with one permission
// prompt (a variable so tests stub it).
var readProtectedFn = readProtectedElevated

// elevatedReadTimeout bounds the prompt + read (the user types a password).
const elevatedReadTimeout = 3 * time.Minute

// profileReadProtectedCmd is the elevated half: run as root by sudo or
// pkexec, it prints the validated file on stdout and nothing else.
var profileReadProtectedCmd = &cobra.Command{
	Use:          "read-protected <file>",
	Hidden:       true,
	Args:         cobra.ExactArgs(1),
	SilenceUsage: true,
	Annotations:  map[string]string{annotationPrefsOnly: "true"},
	RunE: func(cmd *cobra.Command, args []string) error {
		data, err := profileimport.ReadProtected(args[0])
		if err != nil {
			// Running as root: print only the generic refusal (path the
			// caller typed + what to do), never a detail of the file.
			return localizeConfigErr(&config.ValidationError{Key: profileimport.ErrKeyProtectedPath, Args: []any{args[0]}})
		}
		_, err = os.Stdout.Write(data)
		return err
	},
}

// importRootOnly asks once, reads path with permission and imports it.
// The question names the file; without a terminal and without --yes it
// only prints what to do (confirmAction rules, ADR-0033).
func importRootOnly(path, name string, overwrite, yes bool) (profileimport.Result, error) {
	rootOnly := &config.ValidationError{Key: config.ErrKeyFileRootOnly, Args: []any{path}}
	if !confirmAction(cf("cli.profile.import.root_only_ask", path), yes, true) {
		return profileimport.Result{}, localizeConfigErr(rootOnly)
	}
	ctx, cancel := context.WithTimeout(context.Background(), elevatedReadTimeout)
	defer cancel()
	content, err := readProtectedFn(ctx, path)
	if err != nil {
		return profileimport.Result{}, elevatedReadError(err, rootOnly)
	}
	return storeImport(path, name, profileimport.TypeForContent(path, content), content, overwrite)
}

// elevatedReadError turns a failed elevated read into the line to show.
func elevatedReadError(err error, rootOnly error) error {
	switch {
	case errors.Is(err, errElevatedReadCancelled):
		return fmt.Errorf("%s\n    %s", ct("cli.profile.import.root_only_cancelled"), localizedHint(rootOnly))
	case errors.Is(err, errElevatedReadUnavailable):
		return localizeConfigErr(rootOnly)
	}
	var ve *config.ValidationError
	if errors.As(err, &ve) {
		return localizeConfigErr(err)
	}
	return fmt.Errorf("%s\n    %s", cf("cli.profile.import.root_only_failed", err), localizedHint(rootOnly))
}

// localizedHint is the "what to do" line of a ValidationError in the CLI
// language.
func localizedHint(err error) string {
	_, hint, _ := config.LocalizeError(err, cliLang())
	return hint
}

func init() {
	profileCmd.AddCommand(profileReadProtectedCmd)
}
