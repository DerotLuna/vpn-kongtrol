// Kongtrol UI — profile and group errors in the user's language · v1.1.0 · 2026-11-18
//
// Validation errors already arrive translated (config.ValidationError:
// error + hint + error_key), and so do the profile and group handler errors
// since ADR-0042. The few still in plain English (env-var references,
// restart unsupported) are matched here and rewritten as
// "what happened + what to do". Anything else passes through unchanged and
// reads like every other error (api/errors.ts describeError).
import { ApiError } from '../../api/client'
import type { Translate } from '../../api/errors'
import type { MessageKey } from '../../i18n/messages'

interface Rule {
  match: RegExp
  message: MessageKey
  fix: MessageKey
}

const RULES: Rule[] = [
  { match: /^(?<a>\w+) must not contain environment-variable references$/, message: 'profiles.check.envRef', fix: 'profiles.err.envRef.fix' },
  { match: /^restart not supported by this daemon instance$/, message: 'profiles.restart.err.unsupported', fix: 'profiles.restart.failed.fix' },
]

export interface DeleteRefusal {
  message: string
  fix: string
  /** Where the profile is still used: label (already translated) + names. */
  refs: { label: string; names: string[] }[]
}

/**
 * A delete the daemon refused because something still uses the profile
 * (api.err.vpn_in_use) or it is the last one (api.err.vpn_last), ready for
 * the dialog. The daemon sends one translated line, "headline · label: a,
 * b · label: —"; the parts are split here so each list reads on its own
 * row and empty ones ("—") are left out. Anything else → null.
 */
export function deleteRefusal(err: unknown): DeleteRefusal | null {
  if (!(err instanceof ApiError) || err.status !== 409) return null
  if (err.key === 'api.err.vpn_last') return { message: err.message, fix: err.hint, refs: [] }
  if (err.key !== 'api.err.vpn_in_use') return null
  const [head = err.message, ...parts] = err.message.split(' · ')
  const refs: DeleteRefusal['refs'] = []
  for (const part of parts) {
    const at = part.lastIndexOf(': ')
    if (at < 0) continue
    const names = part
      .slice(at + 2)
      .replace(/[.。]$/, '')
      .split(', ')
      .map((n) => n.trim())
      .filter((n) => n && n !== '—')
    if (names.length) refs.push({ label: part.slice(0, at), names })
  }
  return { message: refs.length ? head.replace(/[.。]$/, '') : err.message, fix: err.hint, refs }
}

/** The same error, with a translated message and fix when we know it. */
export function localizeError(err: unknown, t: Translate): unknown {
  if (!(err instanceof ApiError) || err.kind !== 'server' || err.body.error_key) return err
  for (const rule of RULES) {
    const m = rule.match.exec(err.message)
    if (!m) continue
    const vars = { a: m.groups?.a ?? '', b: m.groups?.b ?? '' }
    return new ApiError('server', err.status, {
      error: t(rule.message, vars),
      hint: t(rule.fix, vars),
      error_key: `kongtrol.ui.${rule.message}`,
    })
  }
  return err
}
