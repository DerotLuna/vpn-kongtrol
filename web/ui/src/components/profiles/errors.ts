// Kongtrol UI — profile and group errors in the user's language · v1.0.0 · 2026-11-18
//
// Validation errors already arrive translated (config.ValidationError:
// error + hint + error_key). Conflicts and missing-field errors from the
// profile and group handlers are still plain English (docs/dashboard-parity.md
// §5.2 item 16), so the known ones are matched here and rewritten as
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
  { match: /^vpn profile already exists$/, message: 'profiles.err.exists', fix: 'profiles.err.exists.fix' },
  { match: /^profile name is required$/, message: 'profiles.check.nameRequired', fix: 'profiles.err.name.fix' },
  { match: /^cannot delete the last remaining VPN profile$/, message: 'profiles.err.last', fix: 'profiles.err.last.fix' },
  { match: /^profile "(?<a>.+)" is referenced by policy "(?<b>.+)"$/, message: 'profiles.err.inPolicy', fix: 'profiles.err.inPolicy.fix' },
  { match: /^profile "(?<a>.+)" is referenced by group "(?<b>.+)"$/, message: 'profiles.err.inGroup', fix: 'profiles.err.inGroup.fix' },
  { match: /^(?<a>\w+) must not contain environment-variable references$/, message: 'profiles.check.envRef', fix: 'profiles.err.envRef.fix' },
  { match: /^vpn profile not found$/, message: 'profiles.err.gone', fix: 'profiles.err.gone.fix' },
  { match: /^group already exists$/, message: 'groups.err.exists', fix: 'groups.err.exists.fix' },
  { match: /^group name is required$/, message: 'groups.check.nameRequired', fix: 'groups.err.name.fix' },
  { match: /^group references unknown profile "(?<a>.+)"$/, message: 'groups.err.unknownProfile', fix: 'groups.err.unknownProfile.fix' },
  { match: /^group must include at least one profile$/, message: 'groups.check.profilesRequired', fix: 'groups.err.profiles.fix' },
  { match: /^group not found$/, message: 'groups.err.gone', fix: 'groups.err.gone.fix' },
  { match: /^restart not supported by this daemon instance$/, message: 'profiles.restart.err.unsupported', fix: 'profiles.restart.failed.fix' },
]

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
