package config

// UpdateCheckEnabled reports whether the user opted in to the update notice
// (updates.check_enabled). Off by default and when c is nil.
func (c *Config) UpdateCheckEnabled() bool {
	return c != nil && c.Updates.CheckEnabled
}
