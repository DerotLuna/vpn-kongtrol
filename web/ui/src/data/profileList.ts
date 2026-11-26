import type { TunnelMetrics } from '../api/types'
import { useEffect, useState } from 'react'
import { isKnownType, specOf } from '../components/profiles/catalog'
import type { VPNProfile } from './profiles'
import { useMetricsHistoryQuery, type ProfileHistory } from './traffic'

export function profileDescription(profile: Pick<VPNProfile, 'type' | 'provider'> | undefined, app: string): string {
  if (!profile || !isKnownType(profile.type)) return ''
  const spec = specOf(profile.type)
  if (profile.provider && isKnownType(profile.provider) && profile.provider !== profile.type) {
    return `${specOf(profile.provider).label} · ${spec.label}`
  }
  return spec.group === 'attach' ? `${spec.label} · ${app}` : spec.label
}

export function profileOrder(tunnels: readonly TunnelMetrics[], history: Record<string, ProfileHistory>): string[] {
  return [...tunnels].sort((a, b) => {
    const connected = Number(b.Status === 'connected') - Number(a.Status === 'connected')
    if (connected) return connected
    const aTime = Date.parse(history[a.Name]?.last_connected_at ?? '') || 0
    const bTime = Date.parse(history[b.Name]?.last_connected_at ?? '') || 0
    return bTime - aTime || a.Name.localeCompare(b.Name)
  }).map((tunnel) => tunnel.Name)
}

export function useStableProfileOrder(tunnels: readonly TunnelMetrics[] | undefined, revision = 0) {
  const history = useMetricsHistoryQuery()
  const [saved, setSaved] = useState<{ revision: number; names: string[] } | null>(null)
  useEffect(() => {
    if (!tunnels || history.isPending) return
    // oxlint-disable-next-line react/set-state-in-effect
    setSaved((previous) => {
      const sorted = profileOrder(tunnels, history.data ?? {})
      if (!previous || previous.revision !== revision) return { revision, names: sorted }
      const current = new Set(sorted)
      const names = previous.names.filter((name) => current.has(name))
      const known = new Set(names)
      names.push(...sorted.filter((name) => !known.has(name)))
      return names.join('\n') === previous.names.join('\n') ? previous : { revision, names }
    })
  }, [tunnels, history.data, history.isPending, revision])
  const names = saved?.revision === revision ? saved.names : (tunnels ?? []).map((tunnel) => tunnel.Name)
  const freeze = () => {
    if (!tunnels || !history.isPending) return
    setSaved((previous) => previous?.revision === revision ? previous : { revision, names })
  }
  return { names, freeze }
}

export function orderedProfiles(tunnels: readonly TunnelMetrics[], names: readonly string[]): TunnelMetrics[] {
  const rank = new Map(names.map((name, index) => [name, index]))
  return [...tunnels].sort((a, b) => (rank.get(a.Name) ?? names.length) - (rank.get(b.Name) ?? names.length))
}
