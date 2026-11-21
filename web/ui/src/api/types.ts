// Kongtrol UI — API types · v0.4.0 · 2026-11-18
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

/** internal/vpn/health.go `TunnelHealth` (json tags), ADR-0030. */
export interface TunnelHealth {
  state: 'ok' | 'idle' | 'no_response' | 'unknown'
  last_handshake?: string
  since?: string
  reason_key?: string
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
  health?: TunnelHealth
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
  /** Answers with an older protocol: "update", not "install" (absent on older daemons). */
  outdated?: boolean
}

/** GET /api/v1/version — internal/api `VersionInfo` (same as `kongtrol version`). */
export interface VersionInfo {
  version: string
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

/** internal/api/handlers_security.go `groupDTO` (GET /api/v1/groups). */
export interface Group {
  name: string
  profiles: string[]
}

/**
 * GET /api/v1/security/status, the fields the tray flyout shows. States come
 * from configuredRuntimeState: disabled (off in the config), idle (on,
 * nothing to do now), armed / active (kill switch), active (DNS guard).
 */
export interface SecurityStatus extends PublicIPStatus {
  /** Runtime: blocking right now (kill switch) / applied (DNS guard). */
  kill_switch?: boolean
  kill_switch_enabled: boolean
  kill_switch_state: 'disabled' | 'idle' | 'armed' | 'active'
  dns_guard?: boolean
  dns_guard_enabled: boolean
  dns_guard_state: 'disabled' | 'idle' | 'active'
}

/** Answers of POST /api/v1/tunnels/{name}/connect|disconnect. */
export interface TunnelActionResult {
  status: 'connecting' | 'already_connected' | 'disconnected' | 'waiting_user'
  tunnel: string
}

/** internal/api/handlers_lifecycle.go `UpdateStatus` (opt-in notice, ADR-0034). */
export interface UpdateStatus {
  check_enabled: boolean
  host: string
  current: string
  latest?: string
  url?: string
  newer: boolean
  checked_at?: string
}

/** Answer of POST /api/v1/data/purge (ADR-0034). */
export interface PurgeResult {
  status: 'purged'
  restarting: boolean
}

/** internal/doctor `Check`: already translated by the daemon. */
export interface DoctorCheck {
  section: string
  label: string
  status: 'ok' | 'warn' | 'fail'
  detail: string
  /** One "what to do" line (warnings and failures). */
  hint?: string
  /** The only fix the GUI offers: install the helper. */
  fix?: 'helper_install'
  help?: 'tunnelbear_driver'
}

/** GET /api/v1/doctor — internal/doctor `Report`. */
export interface DoctorReport {
  checks: DoctorCheck[]
  failures: number
  warnings: number
}
