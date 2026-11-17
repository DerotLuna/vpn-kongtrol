package config

import (
	"net/url"
	"strings"
)

// DefaultPublicIPEndpoint is the public IP service used when
// security.public_ip_check.endpoints is empty (plain-text answer).
const DefaultPublicIPEndpoint = "https://api.ipify.org"

// PublicIPCheckEnabled reports whether the user opted in to the external
// public IP lookup: security.public_ip_check.enabled, or the legacy
// security.leak_detection.enabled (leak detection needs the lookup). The
// kill switch and DNS guard never enable it.
func (c *Config) PublicIPCheckEnabled() bool {
	if c == nil {
		return false
	}
	return c.Security.PublicIPCheck.Enabled || c.Security.LeakDetection.Enabled
}

// SetPublicIPCheck turns the lookup on or off. Turning it off also clears
// the legacy leak_detection.enabled, which would otherwise keep it on.
func (c *Config) SetPublicIPCheck(enabled bool) {
	c.Security.PublicIPCheck.Enabled = enabled
	if !enabled {
		c.Security.LeakDetection.Enabled = false
	}
}

// PublicIPEndpoints returns the configured endpoints, or the default.
func (c *Config) PublicIPEndpoints() []string {
	if c != nil && len(c.Security.PublicIPCheck.Endpoints) > 0 {
		out := make([]string, 0, len(c.Security.PublicIPCheck.Endpoints))
		for _, ep := range c.Security.PublicIPCheck.Endpoints {
			if ep = strings.TrimSpace(ep); ep != "" {
				out = append(out, ep)
			}
		}
		if len(out) > 0 {
			return out
		}
	}
	return []string{DefaultPublicIPEndpoint}
}

// PublicIPHost returns the host of the first endpoint, for the one-line
// "uses an external service (<host>)" notice.
func (c *Config) PublicIPHost() string {
	return EndpointHost(c.PublicIPEndpoints()[0])
}

// EndpointHost returns the host of an endpoint URL (the raw string when it
// does not parse).
func EndpointHost(raw string) string {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || u.Host == "" {
		return raw
	}
	return u.Hostname()
}

// ValidatePublicIPEndpoint accepts only absolute https:// URLs with a host
// and no credentials: the lookup must never go out in clear text or carry
// secrets.
func ValidatePublicIPEndpoint(raw string) error {
	raw = strings.TrimSpace(raw)
	u, err := url.Parse(raw)
	if err != nil || !strings.EqualFold(u.Scheme, "https") || u.Hostname() == "" || u.User != nil || u.Opaque != "" {
		return newValidationError("config.err.public_ip_endpoint", raw)
	}
	return nil
}
