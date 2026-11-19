package main

import (
	"bufio"
	"fmt"
	"io"

	"github.com/spf13/cobra"

	"github.com/vpn-kongtrol/kongtrol/internal/i18n"
)

// Double click on kongtrol.exe (Windows Explorer, no console of its own):
// the window would open and close at once. Instead it shows a translated
// notice pointing to the installer and waits for Enter (ADR-0045).
//
// cobra has the same guard (cobra.MousetrapHelpText, via
// github.com/inconshreveable/mousetrap), but with MousetrapDisplayDuration 0
// it adds a fixed English "Press return to continue..." line, so it is turned
// off and the same mousetrap check runs here with our own text.
func init() {
	cobra.MousetrapHelpText = ""
}

// startedFromExplorer reports a double click: no arguments and Explorer as
// the parent process. A shortcut with arguments (e.g. "kongtrol dashboard")
// also has Explorer as parent, so it must still run the command.
func startedFromExplorer(args []string) bool {
	return len(args) <= 1 && startedByExplorer()
}

// showExplorerNotice prints the notice and waits for Enter (or EOF).
func showExplorerNotice(w io.Writer, r io.Reader, lang i18n.Lang) {
	fmt.Fprintln(w, i18n.T(lang, "cli.explorer_notice"))
	_, _ = bufio.NewReader(r).ReadString('\n')
}
