import type { RuleDraft } from './model'
import { TUNNEL_DOWN_CHOICES } from '../rules/model'

export interface AddRecovery {
  profile: string
  draft: RuleDraft | null
}

const KEY = 'kongtrol.add-recovery'
const TTL_MS = 10 * 60_000

/** Only the rule draft and the name of the profile that was already saved. */
export function saveAddRecovery(recovery: AddRecovery): boolean {
  try {
    localStorage.setItem(KEY, JSON.stringify({ ...recovery, expires: Date.now() + TTL_MS }))
    return true
  } catch {
    return false
  }
}

/** Called after a new local sign-in; consume this origin's draft once. */
export function takeAddRecovery(): AddRecovery | null {
  try {
    const raw = localStorage.getItem(KEY)
    if (!raw) return null
    localStorage.removeItem(KEY)
    const saved = JSON.parse(raw) as AddRecovery & { expires: number }
    if (typeof saved.expires !== 'number' || saved.expires <= Date.now() || typeof saved.profile !== 'string') return null
    const draft = saved.draft as (RuleDraft & { more?: string }) | null
    if (draft !== null && (!draft || typeof draft.input !== 'string'
      || typeof draft.name !== 'string' || typeof draft.subs !== 'boolean' || !TUNNEL_DOWN_CHOICES.includes(draft.onDown))) return null
    if (!draft) return { profile: saved.profile, draft: null }
    const { more, ...rule } = draft
    const input = more ? [rule.input, more].filter(Boolean).join('\n') : rule.input
    return { profile: saved.profile, draft: { ...rule, input, via: saved.profile } }
  } catch {
    return null
  }
}
