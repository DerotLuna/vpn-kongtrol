// Reglas + Horarios — small hooks shared by both pages · v1.0.0 · 2026-11-18
import { tunnelState, useTunnelsQuery, type TunnelState } from '../../data/resources'

/** State of one profile's tunnel, or undefined while unknown (not loaded / not running yet). */
export function useProfileState(name: string): TunnelState | undefined {
  const { data } = useTunnelsQuery()
  const tn = data?.find((x) => x.Name === name)
  return tn ? tunnelState(tn) : undefined
}
