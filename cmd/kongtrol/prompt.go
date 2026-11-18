package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"
)

// Yes/no questions in the terminal (first run, up, reload). Following
// clig.dev "Interactivity": only ask when stdin and stdout are an
// interactive terminal, never with --json or -q, and let scripts answer
// up front with --yes. Without a terminal nothing waits for input: the
// caller prints its "what to do" line instead.

// Test seams.
var (
	promptReader        io.Reader = os.Stdin
	promptInteractiveFn           = func() bool { return isTerminal() && stdinIsTerminal() }
)

// stdinIsTerminal reports whether stdin is a character device (a console),
// not a pipe or a file.
func stdinIsTerminal() bool {
	fi, err := os.Stdin.Stat()
	if err != nil {
		return false
	}
	return fi.Mode()&os.ModeCharDevice != 0
}

// canPrompt reports whether a question can be asked now.
func canPrompt() bool {
	return !outputJSON && !outputQuiet && promptInteractiveFn()
}

// askYesNo prints question with its default ([S/n] or [s/N]) and reads one
// line. An empty answer takes the default; end of input answers no.
func askYesNo(question string, defaultYes bool) bool {
	hint := ct("cli.prompt.default_yes")
	if !defaultYes {
		hint = ct("cli.prompt.default_no")
	}
	fmt.Print(tuiInfo(question+" "+paint(cDim, hint)) + " ")
	line, err := bufio.NewReader(promptReader).ReadString('\n')
	if err != nil && line == "" {
		fmt.Println()
		return false
	}
	switch strings.ToLower(strings.TrimSpace(line)) {
	case "":
		return defaultYes
	case "s", "si", "sí", "y", "yes":
		return true
	}
	return false
}

// confirmAction decides whether to go ahead: --yes (yes) goes ahead without
// asking; without an interactive terminal it does not (the caller prints
// how to do it by hand); otherwise it asks.
func confirmAction(question string, yes, defaultYes bool) bool {
	if yes {
		return true
	}
	if !canPrompt() {
		return false
	}
	return askYesNo(question, defaultYes)
}
