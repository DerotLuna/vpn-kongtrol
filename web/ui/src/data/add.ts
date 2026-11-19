// Kongtrol UI — "＋ Agregar" sheet: the little it adds to the data layer · v1.0.0 · 2026-11-19
//
// The sheet (components/add, ADR-0052) creates profiles and rules through
// the same hooks as Perfiles and Reglas (data/profiles.ts, data/rules.ts),
// so caches, toasts and rollbacks stay one. This module only holds:
// - the live "Después" preview: POST /api/v1/policies/test as a query keyed
//   by the draft, so typing asks once per settled value and never races;
// - the file dropped anywhere on the window, handed to the sheet.
import { useQuery } from '@tanstack/react-query'
import type { PolicyPayload } from '../components/rules/model'
import { rulesApi } from './rules'
import { createStore } from './store'

/** Dry run of an unsaved rule for one destination or program (no side effects on the daemon). */
export function useDraftRuleTest(rule: PolicyPayload | null, target: string, app: string) {
  return useQuery({
    queryKey: ['policy-test', rule, target, app],
    queryFn: () => rulesApi.testPolicy(rule as PolicyPayload, target, app),
    enabled: rule !== null && (target !== '' || app !== ''),
    staleTime: 30_000,
    retry: false,
  })
}

const dropped = createStore<File | null>(null)

/** A .conf / .ovpn dropped on the window, waiting for the sheet to take it. */
export const droppedFile = {
  set: (file: File | null): void => dropped.set(file),
  /** Hands the file over once. */
  take: (): File | null => {
    const file = dropped.get()
    dropped.set(null)
    return file
  },
  subscribe: (listener: () => void): (() => void) => dropped.subscribe(listener),
}
