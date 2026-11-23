import { matchOf, readDestinations } from '../add/model'
import { onDownOf, type OnTunnelDown, type Policy, type PolicyPayload, type TunnelDownChoice } from './model'

export interface PolicyDraft {
  name: string
  via: string
  destinations: string
  onDown: TunnelDownChoice
}

export function draftFromPolicy(p: Policy): PolicyDraft {
  return {
    name: p.name,
    via: p.via,
    destinations: [...(p.domains ?? []), ...(p.ip_ranges ?? []), ...(p.apps ?? []).map((app) => `app:${app}`)].join('\n'),
    onDown: p.on_tunnel_down_override === '' ? 'inherit' : onDownOf({ on_tunnel_down: p.on_tunnel_down_override ?? p.on_tunnel_down }),
  }
}

/** Field order for moving focus to the first problem. */
export const POLICY_FIELDS = ['name', 'via', 'destinations', 'onDown'] as const
type PolicyField = (typeof POLICY_FIELDS)[number]

export interface PolicyError {
  code: 'required' | 'duplicate' | 'noProfiles' | 'noMatch' | 'invalid' | 'blockAppsOnly'
  value?: string
  line?: number
}

export type PolicyErrors = Partial<Record<PolicyField, PolicyError>>

interface ValidateContext {
  names: readonly string[]
  editing?: string | undefined
  profiles: readonly string[]
  profileAction?: OnTunnelDown
}

export function validatePolicy(d: PolicyDraft, ctx: ValidateContext) {
  const errors: PolicyErrors = {}
  const name = d.name.trim()
  if (!name) errors.name = { code: 'required' }
  else if (name.toLowerCase() !== ctx.editing?.toLowerCase() &&
    ctx.names.some((n) => n.toLowerCase() === name.toLowerCase())) {
    errors.name = { code: 'duplicate', value: name }
  }

  if (ctx.profiles.length === 0) errors.via = { code: 'noProfiles' }
  else if (!d.via) errors.via = { code: 'required' }

  const checked = readDestinations(d.destinations)
  const invalid = checked.find(({ result }) => !result.ok)
  const entries = checked.flatMap(({ result }) => result.ok ? [result.entry] : [])
  const hasApps = entries.some((entry) => entry.kind === 'app')
  const action = d.onDown === 'inherit' ? (ctx.profileAction ?? 'allow') : d.onDown
  if (invalid && !invalid.result.ok) errors.destinations = { code: 'invalid', value: invalid.result.value, line: invalid.line }
  else if (!entries.length) errors.destinations = { code: 'noMatch' }
  else if (action !== 'allow' && entries.every((entry) => entry.kind === 'app')) errors.onDown = { code: 'blockAppsOnly' }

  const payload: PolicyPayload | null = Object.keys(errors).length > 0 ? null : {
    name,
    via: d.via,
    on_tunnel_down: d.onDown === 'inherit' ? '' : d.onDown,
    match: matchOf(entries, false),
  }
  return { errors, payload, hasApps }
}
