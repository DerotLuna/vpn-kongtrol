// Kongtrol UI — API types · v0.1.0 · 2026-11-17
//
// Hand-written mirrors of the Go structs this UI reads. Keep field names
// identical to the JSON the daemon sends (source in the comment above each).
// Phase 2: generate these from Go (see the report / AGENTS.md proposal).

/** internal/vpn/adapter.go `Status` (always normalized, never ""). */
export type TunnelStatus = 'connected' | 'connecting' | 'disconnected' | 'error'

/** internal/vpn/adapter.go `AttachState` (json tags). */
interface AttachState {
  phase: string
  app: string
  location?: string
  reason?: string
  timeout_s?: number
  since: string
}

/**
 * internal/monitor/collector.go `TunnelMetrics`. Most fields have no json
 * tag, so Go encodes them with their Go names (PascalCase).
 * GET /api/v1/tunnels returns an array; the WebSocket sends a name-keyed map.
 */
export interface TunnelMetrics {
  Name: string
  Status: TunnelStatus
  InterfaceName: string
  AssignedIP: string
  ConnectedAt: string
  UptimeSeconds: number
  BytesSent: number
  BytesReceived: number
  LatencyMS: number
  UpdatedAt: string
  last_error?: string
  last_error_at?: string
  last_error_key?: string
  attach?: AttachState
}

/** internal/api/handlers_helper.go `HelperStatus`. */
export interface HelperStatus {
  installed: boolean
  running: boolean
  available: boolean
  elevated: boolean
  helper_version: string
  cli_version: string
  needs_install: boolean
}

/** internal/api/handlers_security.go `preferenceDTO`. Empty = no explicit choice. */
export interface Preferences {
  language: string
  theme: string
}

/** Error bodies: writeError {error, code}; localized {error, hint, error_key}. */
export interface ApiErrorBody {
  error?: string
  code?: string
  hint?: string
  error_key?: string
  detail?: string
}

/** internal/api/handlers_security.go `leakCheckStatus`. */
export interface LeakCheck {
  state: 'clean' | 'leak' | 'error'
  has_leak: boolean
  public_ip?: string
  reason?: string
  checked_at: string
}

/**
 * Public IP fields of GET /api/v1/security/status (and the answers of
 * POST /api/v1/security/publicip[/check]). The lookup is opt-in: off by
 * default, and the daemon never looks the IP up just to answer a GET.
 */
export interface PublicIPStatus {
  public_ip_check_enabled: boolean
  public_ip_check_host: string
  leak_check?: LeakCheck | null
}
