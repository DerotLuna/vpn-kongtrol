// Kongtrol UI — cache keys, one per daemon resource · v0.1.0 · 2026-11-18
//
// Keys name resources, not pages: two pages reading the tunnels share one
// request and one cache entry. Phase 2 adds keys here (vpns, policies,
// groups, schedule, settings, audit, routes…) next to their endpoint.
export const keys = {
  tunnels: ['tunnels'] as const,
  helper: ['helper'] as const,
  version: ['version'] as const,
  security: ['security'] as const,
  updates: ['updates'] as const,
} as const

/** Mutation keys, so any component can show "in progress" for a shared action. */
export const mutationKeys = {
  helperInstall: ['helper', 'install'] as const,
} as const
