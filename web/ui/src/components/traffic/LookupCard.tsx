// Kongtrol UI — "¿Por dónde sale?" + the rules it filters · v1.1.0 · 2026-11-24
//
// Classic panel's traffic map (index.html:68, app.js:892-1014): a site, IP,
// URL, rule/profile name or app:<program> → GET /api/v1/resolve, explained
// in plain words against what is up right now (a rule's VPN may be off).
// A program is only recognised, never routed (ADR-0113): its answer says so.
// For a domain sent through a VPN, the IPs it resolves to through that
// VPN's DNS are one click away (GET /api/v1/dns/resolve). Below, the rules
// filtered by the same query, each with the IPs Kongtrol found and when
// they expire.
import { memo, useId, useMemo, useState, type FormEvent } from 'react'
import { Link } from 'react-router'
import type { TunnelMetrics } from '../../api/types'
import { usePreferences } from '../../app/preferences-context'
import {
  explainLookup,
  filterPolicies,
  formatRelative,
  isDomain,
  parseLookup,
  policyResolved,
  useDnsResolveQuery,
  useNetworkOverviewQuery,
  usePoliciesQuery,
  useResolveQuery,
  type LookupVerdict,
  type Policy,
  type ResolveResult,
  type TunnelRoutesResponse,
} from '../../data/traffic'
import { Button } from '../ui/Button'
import { Card } from '../ui/Card'
import { DataList, type Column } from '../ui/DataList'
import { TextField } from '../ui/Field'
import { CheckIcon, AlertIcon, InfoIcon, RefreshIcon } from '../ui/icons'
import { EmptyState, Skeleton } from '../ui/States'
import { SectionError } from './SectionError'
import { AddButton } from '../add/entries'

interface Props {
  tunnels: readonly TunnelMetrics[]
  routes: TunnelRoutesResponse | undefined
}

export function LookupCard({ tunnels, routes }: Props) {
  const { t } = usePreferences()
  const [draft, setDraft] = useState('')
  const [query, setQuery] = useState('')
  const resolve = useResolveQuery(query)
  const policies = usePoliciesQuery()

  const submit = (e: FormEvent) => {
    e.preventDefault()
    setQuery(draft.trim())
    if (draft.trim() && draft.trim() === query) void resolve.refetch()
  }
  const clear = () => {
    setDraft('')
    setQuery('')
  }

  return (
    <Card title={t('traffic.lookup.title')} className="tr-lookup">
      <form className="tr-lookup__form" onSubmit={submit} role="search" aria-label={t('traffic.lookup.title')}>
        <TextField
          label={t('traffic.lookup.label')}
          hint={t('traffic.lookup.hint')}
          value={draft}
          onChange={(e) => setDraft(e.target.value)}
          mono
          autoComplete="off"
          spellCheck={false}
          inputMode="url"
          enterKeyHint="search"
        />
        <div className="tr-lookup__actions">
          <Button type="submit" variant="filled" loading={resolve.isFetching && query !== ''}>
            {t('traffic.lookup.check')}
          </Button>
          {(draft || query) && (
            <Button type="button" variant="text" onClick={clear}>
              {t('traffic.lookup.clear')}
            </Button>
          )}
        </div>
      </form>

      <div aria-live="polite" className="tr-lookup__live">
        {query && resolve.isError && <SectionError error={resolve.error} onRetry={() => void resolve.refetch()} />}
        {query && resolve.data && (
          <LookupResult
            result={resolve.data}
            input={query}
            verdict={explainLookup(resolve.data, { tunnels, policies: policies.data ?? [], routes })}
          />
        )}
      </div>

      <RulesList query={query} policies={policies} />
    </Card>
  )
}

function verdictText(v: LookupVerdict, lhs: string, t: ReturnType<typeof usePreferences>['t']): { title: string; detail: string } {
  switch (v.kind) {
    case 'tunnel':
      return {
        title: t('traffic.lookup.tunnel', { lhs, via: v.via }),
        detail: v.rule ? t('traffic.lookup.tunnel.detail', { rule: v.rule }) : t('traffic.lookup.tunnel.detailNoRule', { via: v.via }),
      }
    case 'tunnel_down':
      return { title: t('traffic.lookup.down', { lhs, via: v.via }), detail: t('traffic.lookup.down.detail') }
    case 'blocked':
      return { title: t('traffic.lookup.blocked', { lhs, via: v.via }), detail: t('traffic.lookup.blocked.detail', { rule: v.rule }) }
    case 'all_traffic':
      return { title: t('traffic.lookup.allTraffic', { lhs, via: v.via }), detail: t('traffic.lookup.allTraffic.detail', { via: v.via }) }
    case 'normal':
      return {
        title: t('traffic.lookup.normal', { lhs }),
        detail: v.iface ? t('traffic.lookup.normal.detail', { iface: v.iface }) : t('traffic.lookup.normal.detailNoIface'),
      }
    case 'app':
      return {
        title: v.rule ? t('traffic.lookup.app', { lhs, rule: v.rule }) : t('traffic.lookup.appNoRule', { lhs }),
        detail: t('policy.app.notApplied'),
      }
  }
}

const TONE: Record<LookupVerdict['kind'], string> = {
  app: 'caution',
  tunnel: 'ok',
  all_traffic: 'ok',
  tunnel_down: 'caution',
  blocked: 'bad',
  normal: 'neutral',
}

function LookupResult({ result, input, verdict }: { result: ResolveResult; input: string; verdict: LookupVerdict }) {
  const { t } = usePreferences()
  const lhs = result.app ? `app:${result.app}` : result.target || input
  const { title, detail } = verdictText(verdict, lhs, t)
  const tone = TONE[verdict.kind]
  const icon = tone === 'ok' ? <CheckIcon /> : tone === 'neutral' ? <InfoIcon /> : <AlertIcon />
  const overview = useNetworkOverviewQuery(verdict.kind === 'normal')
  const showDns = (verdict.kind === 'tunnel' || verdict.kind === 'all_traffic') && !result.app && isDomain(result.target)
  return (
    <div className={`tr-result tr-result--${tone}`}>
      <span className="tr-result__icon">{icon}</span>
      <div className="tr-result__text">
        <strong>{title}</strong>
        <p>{detail}</p>
        {verdict.kind === 'normal' && overview.data?.public_ip && (
          <p className="mono small">{t('traffic.lookup.publicIp', { ip: overview.data.public_ip })}</p>
        )}
        {(verdict.kind === 'tunnel_down' || verdict.kind === 'blocked') && (
          <Link className="k-btn k-btn--outlined k-btn--sm tr-result__action" to="/">
            <span className="k-btn__label">{t('traffic.lookup.goHome')}</span>
          </Link>
        )}
        {verdict.kind !== 'app' && (
          <AddButton req={{ mode: 'rule', target: result.target || input }} variant="text" className="tr-result__action">
            {t('add.thisVia')}
          </AddButton>
        )}
        {showDns && 'via' in verdict && <DnsDetail key={`${result.target}|${verdict.via}`} domain={result.target} via={verdict.via} />}
      </div>
    </div>
  )
}

function DnsDetail({ domain, via }: { domain: string; via: string }) {
  const { t } = usePreferences()
  const [open, setOpen] = useState(false)
  const dns = useDnsResolveQuery(domain, via, open)
  const id = useId()
  return (
    <div className="tr-dns">
      <button type="button" className="tr-disclosure" aria-expanded={open} aria-controls={id} onClick={() => setOpen((o) => !o)}>
        <span className="tr-disclosure__chev" aria-hidden="true" />
        {t('traffic.dns.show', { via })}
      </button>
      {open && (
        <div id={id} className="tr-dns__body">
          {dns.isPending && <Skeleton lines={1} label={t('traffic.lookup.checking')} />}
          {dns.isError && <SectionError error={dns.error} onRetry={() => void dns.refetch()} />}
          {dns.data &&
            (dns.data.ips.length === 0 ? (
              <p className="muted">{t('traffic.dns.none', { via, domain })}</p>
            ) : (
              <>
                <p className="small muted">{t('traffic.dns.title', { domain, via })}</p>
                <ul className="tr-chips">
                  {dns.data.ips.map((ip) => (
                    <li key={ip} className="mono">
                      {ip}
                    </li>
                  ))}
                </ul>
              </>
            ))}
        </div>
      )}
    </div>
  )
}

// ── Rules ─────────────────────────────────────────────────────────────────

function RulesList({ query, policies }: { query: string; policies: ReturnType<typeof usePoliciesQuery> }) {
  const { t } = usePreferences()
  const columns = useRuleColumns()
  const all = policies.data
  const lookup = parseLookup(query)
  const filterText = lookup?.app ?? lookup?.target ?? ''
  const shown = useMemo(() => (all ? filterPolicies(all, filterText) : []), [all, filterText])

  return (
    <section className="tr-rules" aria-labelledby="tr-rules-title" aria-busy={policies.isFetching || undefined}>
      <div className="tr-section-head">
        <h3 id="tr-rules-title" className="tr-subtitle">
          {t('traffic.rules.title')}
        </h3>
        <Button variant="text" size="sm" icon={<RefreshIcon />} onClick={() => void policies.refetch()} loading={policies.isRefetching}>
          {t('traffic.rules.refresh')}
        </Button>
      </div>
      {policies.isPending && <Skeleton lines={3} label={t('traffic.rules.title')} />}
      {policies.isError && !all && <SectionError error={policies.error} onRetry={() => void policies.refetch()} />}
      {all && all.length === 0 && (
        <EmptyState
          title={t('traffic.rules.empty.title')}
          body={t('traffic.rules.empty.body')}
          action={
            <Link className="k-btn k-btn--outlined k-btn--md" to="/rules">
              <span className="k-btn__label">{t('traffic.rules.empty.action')}</span>
            </Link>
          }
        />
      )}
      {all && all.length > 0 && (
        <>
          <p className="tr-note" role="status">
            {filterText
              ? t('traffic.rules.showing', { shown: shown.length, total: all.length, query: filterText })
              : t('traffic.rules.showingAll', { total: all.length })}
          </p>
          <DataList
            caption={t('traffic.rules.title')}
            columns={columns}
            rows={shown}
            rowKey={(p) => p.name}
            empty={<p className="muted">{t('traffic.rules.noMatch', { query: filterText })}</p>}
          />
          <p className="small muted">{t('traffic.rules.ips.explain')}</p>
        </>
      )}
    </section>
  )
}

function useRuleColumns(): Column<Policy>[] {
  const { t, lang } = usePreferences()
  return useMemo(
    () => [
      {
        id: 'name',
        header: t('traffic.rules.col.rule'),
        cell: (p) => (
          <span className="tr-rule-name">
            <span className="mono">{p.name}</span>
            {(p.on_tunnel_down === 'block' || p.on_tunnel_down === 'connect') && <span className="tr-tag tr-tag--block">{t('traffic.rules.blockTag')}</span>}
          </span>
        ),
      },
      {
        id: 'match',
        header: t('traffic.rules.col.match'),
        cell: (p) => <MatchTags policy={p} />,
      },
      { id: 'via', header: t('traffic.rules.col.via'), cell: (p) => <span className="mono">{p.via}</span> },
      {
        id: 'ips',
        header: t('traffic.rules.col.ips'),
        cell: (p) => <ResolvedCell policy={p} lang={lang} />,
      },
    ],
    [t, lang],
  )
}

function MatchTags({ policy }: { policy: Policy }) {
  const { t } = usePreferences()
  const tags = [
    ...(policy.domains ?? []).map((v) => ({ kind: 'domain' as const, v })),
    ...(policy.ip_ranges ?? []).map((v) => ({ kind: 'ip' as const, v })),
    ...(policy.apps ?? []).map((v) => ({ kind: 'app' as const, v })),
  ]
  return (
    <ul className="tr-chips tr-chips--match">
      {tags.map(({ kind, v }) => (
        <li key={`${kind}:${v}`}>
          <span className="tr-chips__kind">{t(`traffic.match.${kind}`)}</span> <span className="mono">{v}</span>
        </li>
      ))}
    </ul>
  )
}

const ResolvedCell = memo(function ResolvedCell({ policy, lang }: { policy: Policy; lang: 'es' | 'en' }) {
  const { t } = usePreferences()
  const [open, setOpen] = useState(false)
  const id = useId()
  const ips = policyResolved(policy)
  if (ips.length === 0) return <span className="muted">{t('traffic.rules.ipsNone')}</span>
  return (
    <div className="tr-ips">
      <button
        type="button"
        className="tr-disclosure tr-disclosure--inline"
        aria-expanded={open}
        aria-controls={id}
        aria-label={open ? t('traffic.rules.ips.hide', { rule: policy.name }) : t('traffic.rules.ips.show', { n: ips.length, rule: policy.name })}
        onClick={() => setOpen((o) => !o)}
      >
        <span className="tr-disclosure__chev" aria-hidden="true" />
        {t('traffic.rules.ips', { n: ips.length })}
      </button>
      {open && (
        <ul id={id} className="tr-ips__list">
          {ips.map((ip) => (
            <li key={ip.cidr}>
              <span className="mono">{ip.cidr}</span>
              {ip.host && <span className="muted"> · {ip.host}</span>}
              {ip.expires_at && (
                <span className="tr-ips__exp"> · {t('traffic.rules.ips.expires', { when: formatRelative(ip.expires_at, lang) })}</span>
              )}
            </li>
          ))}
        </ul>
      )}
    </div>
  )
})
