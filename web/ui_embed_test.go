package assets

import (
	"io/fs"
	"testing"
)

// UI must work with and without a UI build: with one, index.html is served;
// without one (fresh clone, no Node), only dist/.gitkeep is embedded.
func TestUI(t *testing.T) {
	fsys, built := UI()
	if fsys == nil {
		t.Fatal("UI() returned a nil FS")
	}
	_, err := fs.Stat(fsys, "index.html")
	if built != (err == nil) {
		t.Fatalf("built=%v but stat index.html err=%v", built, err)
	}
}
