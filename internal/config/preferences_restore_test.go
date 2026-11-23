package config

import "testing"

// ADR-0101: restore is off until the user answers, the question is asked
// once, and an explicit choice (old or new) is kept.
func TestRestoreQuestion(t *testing.T) {
	var nilPrefs *Preferences
	if nilPrefs.RestoreTunnelsEnabled() || nilPrefs.RestoreQuestionPending() {
		t.Fatal("nil preferences: restore must be off and nothing pending")
	}

	p := &Preferences{}
	if p.RestoreTunnelsEnabled() || p.RestoreAnswered() || p.RestoreQuestionPending() {
		t.Fatalf("fresh: %+v", p)
	}
	if !p.MarkRestoreQuestion() || !p.RestoreQuestionPending() {
		t.Fatalf("after first connect: %+v", p)
	}
	if p.MarkRestoreQuestion() {
		t.Fatal("marked twice")
	}

	// "Ahora no": answered, off, never pending again.
	p.SetRestoreTunnels(false)
	if p.RestoreTunnelsEnabled() || !p.RestoreAnswered() || p.RestoreQuestionPending() || p.RestoreAsk {
		t.Fatalf("after Ahora no: %+v", p)
	}
	if p.MarkRestoreQuestion() {
		t.Fatal("asked again after an answer")
	}

	// A user who set it before (ADR-0098) keeps it and is not asked.
	on := true
	old := &Preferences{RestoreTunnels: &on}
	if !old.RestoreTunnelsEnabled() || old.MarkRestoreQuestion() {
		t.Fatalf("explicit choice: %+v", old)
	}
}
