// Kongtrol UI — "＋ Agregar": move a direct file's rules to the provider app · v1.0.0 · 2026-11-23
//
// ADR-0104. After a provider-app profile P is saved (e.g. Proton VPN), the
// Done step finds the direct profiles made from that provider's files
// (GET /api/v1/vpns `provider`, ADR-0103) that have rules and offers, one
// card each, to point those rules at P (POST /api/v1/policies/move). The
// sheet finds the candidates itself: the tray and notices cannot pass them.
import { useMemo, useState } from 'react'
import { usePreferences } from '../../app/preferences-context'
import type { AdapterType, VPNProfile } from '../../data/profiles'
import { useMovePolicies, usePoliciesQuery } from '../../data/rules'
import { Button } from '../ui/Button'
import { Banner } from '../ui/States'
import { specOf } from '../profiles/catalog'
import { ErrorBanner } from './parts'

interface Candidate {
  from: string
  rules: number
}

/** Direct profiles from `type`'s files (not `to` itself) with at least one rule, by name. */
function moveCandidates(
  to: string,
  type: AdapterType | undefined,
  profiles: readonly VPNProfile[],
  policies: readonly { via: string }[],
): Candidate[] {
  if (!type) return []
  return profiles
    .filter((p) => p.name !== to && !!p.provider && p.provider === type)
    .map((p) => ({ from: p.name, rules: policies.filter((r) => r.via === p.name).length }))
    .filter((c) => c.rules > 0)
    .sort((a, b) => a.from.localeCompare(b.from))
}

export function MoveRules({ to, type, profiles }: { to: string; type: AdapterType | undefined; profiles: readonly VPNProfile[] }) {
  const { t } = usePreferences()
  const policiesQ = usePoliciesQuery()
  const move = useMovePolicies()
  // Answered cards stay answered; the list itself is taken once the rules load.
  const [done, setDone] = useState<ReadonlySet<string>>(new Set())
  const [failed, setFailed] = useState<{ from: string; error: unknown } | null>(null)
  const [pending, setPending] = useState<string | null>(null)
  const candidates = useMemo(
    () => moveCandidates(to, type, profiles, policiesQ.data ?? []),
    [to, type, profiles, policiesQ.data],
  )
  const shown = candidates.filter((c) => !done.has(c.from))
  if (shown.length === 0 || !type) return null
  const provider = specOf(type).label
  const answer = (from: string) => setDone((d) => new Set(d).add(from))

  const doMove = async (from: string) => {
    setFailed(null)
    setPending(from)
    try {
      await move.mutateAsync({ from, to })
      answer(from)
    } catch (error) {
      setFailed({ from, error })
    } finally {
      setPending(null)
    }
  }

  return (
    <div className="k-add-move" aria-live="polite">
      {shown.map(({ from, rules }) => (
        <div key={from}>
          <Banner
            tone="info"
            title={t('add.move.title', { from, to })}
            actions={
              <>
                <Button size="sm" loading={pending === from} disabled={pending !== null && pending !== from} onClick={() => void doMove(from)}>
                  {t('add.move.move')}
                </Button>
                <Button variant="outlined" size="sm" disabled={pending !== null} onClick={() => answer(from)}>
                  {t('add.move.keep', { from })}
                </Button>
              </>
            }
          >
            <span>{t(rules === 1 ? 'add.move.body.one' : 'add.move.body.other', { n: rules, from, to, provider })}</span>
          </Banner>
          {failed?.from === from && <ErrorBanner error={failed.error} />}
        </div>
      ))}
    </div>
  )
}
