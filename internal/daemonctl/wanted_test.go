package daemonctl

import (
	"os"
	"path/filepath"
	"slices"
	"testing"
)

func TestWanted_AddRemoveClearPersist(t *testing.T) {
	path := filepath.Join(t.TempDir(), "run", "wanted.json")
	w := NewWanted(path)
	if got := w.List(); len(got) != 0 {
		t.Fatalf("new set = %v", got)
	}
	for _, n := range []string{"work", "home", "work", ""} {
		if err := w.Add(n); err != nil {
			t.Fatal(err)
		}
	}
	// A new daemon (after a crash or a reboot) reads the same file.
	if got := NewWanted(path).List(); !slices.Equal(got, []string{"home", "work"}) {
		t.Fatalf("after restart = %v", got)
	}
	if fi, err := os.Stat(path); err != nil || (fi.Mode().Perm()&0o077 != 0 && os.PathSeparator == '/') {
		t.Fatalf("file %v, mode %v", err, fi)
	}
	if err := w.Remove("work"); err != nil {
		t.Fatal(err)
	}
	if got := w.List(); !slices.Equal(got, []string{"home"}) {
		t.Fatalf("after remove = %v", got)
	}
	if err := w.Clear(); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("an empty set must leave no file: %v", err)
	}
	if err := w.Remove("ghost"); err != nil { // nothing to do, no file
		t.Fatal(err)
	}
}

func TestWanted_KeepDropsDeletedProfiles(t *testing.T) {
	w := NewWanted(filepath.Join(t.TempDir(), "wanted.json"))
	_ = w.Add("old")
	_ = w.Add("work")
	kept, err := w.Keep(func(n string) bool { return n == "work" })
	if err != nil || !slices.Equal(kept, []string{"work"}) || !slices.Equal(w.List(), []string{"work"}) {
		t.Fatalf("Keep = %v, %v; list %v", kept, err, w.List())
	}
}

// A damaged or foreign file is an empty set, never an error that stops
// the daemon; the next change rewrites it.
func TestWanted_DamagedFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "wanted.json")
	for _, content := range []string{"", "{", `{"version":2,"profiles":["x"]}`, `[1,2]`} {
		if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
		w := NewWanted(path)
		if got := w.List(); len(got) != 0 {
			t.Errorf("%q read as %v", content, got)
		}
		if err := w.Add("work"); err != nil || !slices.Equal(w.List(), []string{"work"}) {
			t.Errorf("%q: Add = %v, list %v", content, err, w.List())
		}
	}
}

func TestWanted_NilIsHarmless(t *testing.T) {
	var w *Wanted
	if w.List() != nil || w.Add("x") != nil || w.Clear() != nil {
		t.Fatal("a nil set must do nothing")
	}
}
