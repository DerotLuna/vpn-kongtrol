//go:build windows

package appattach

// osLaunch starts the first installed Windows executable of the app.
func osLaunch(app App) error {
	if p, ok := firstExisting(app.WindowsPaths); ok {
		return startDetached(p)
	}
	return ErrAppNotFound
}
