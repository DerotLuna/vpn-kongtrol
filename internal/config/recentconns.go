package config

// RecentConnectionsEnabled reports whether the daemon keeps its in-memory
// list of recent destinations (monitor.recent_connections.enabled, default
// on; ADR-0054).
func (c *Config) RecentConnectionsEnabled() bool {
	if c == nil {
		return true
	}
	e := c.Monitor.RecentConnections.Enabled
	return e == nil || *e
}

// SetRecentConnections stores the choice; on is stored as "unset" so the
// YAML stays minimal.
func (c *Config) SetRecentConnections(on bool) {
	if on {
		c.Monitor.RecentConnections.Enabled = nil
		return
	}
	off := false
	c.Monitor.RecentConnections.Enabled = &off
}
