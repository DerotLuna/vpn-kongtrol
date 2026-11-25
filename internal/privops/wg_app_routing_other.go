//go:build !linux

package privops

func supportsWGAppRouting() bool { return false }

func (l *Local) startWGAppRouting(string, int, WGAppRouting) (wgAppRoutingSession, error) {
	return wgAppRoutingSession{}, ErrUnsupported
}
