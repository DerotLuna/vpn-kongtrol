package main

import (
	"fmt"
)

// The question after the first VPN the user connects (ADR-0101): "¿Volver
// a conectar tus VPN al encender el equipo?". The panel and the tray
// flyout ask it from GET /api/v1/startup (`ask`); `kongtrol up` asks it on
// an interactive terminal (confirmAction rules, ADR-0033) and otherwise
// prints how to turn it on. Never with --json or -q.

// restoreAnswerer saves the answer (daemonctl.Client.AnswerRestoreQuestion).
type restoreAnswerer interface {
	AnswerRestoreQuestion(on bool) error
}

// offerRestoreAtStart asks once, after a successful `kongtrol up`, unless
// the user already chose. A connect that just succeeded is enough: the
// daemon marks the question pending at the same moment.
func offerRestoreAtStart(d restoreAnswerer) {
	if outputJSON || outputQuiet {
		return
	}
	prefs, err := loadPreferences()
	if err != nil || prefs.RestoreAnswered() {
		return
	}
	if !canPrompt() {
		fmt.Println(tuiInfo(paint(cDim, ct("cli.onboarding.restore.hint"))))
		return
	}
	fmt.Println(tuiInfo(ct("onboarding.restore.body")))
	on := askYesNo(ct("onboarding.restore.title"), true)
	if err := d.AnswerRestoreQuestion(on); err != nil {
		fmt.Println(tuiWarn(remoteUserErr(err).Error()))
		fmt.Println(tuiInfo(paint(cDim, ct("cli.onboarding.restore.hint"))))
		return
	}
	if on {
		fmt.Println(tuiOK(ct("cli.onboarding.restore.on")))
	} else {
		fmt.Println(tuiInfo(ct("cli.onboarding.restore.off")))
	}
}
