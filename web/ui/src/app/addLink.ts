// Kongtrol UI — "＋ Agregar" deep link · v1.0.0 · 2026-11-19
//
// One sheet hosted by the shell (components/add/AddSheet.tsx, ADR-0052)
// answers every "add" entry in the panel. It opens from the URL, so any
// page, the tray or a doc can link to it:
//   ?add=menu|vpn|file|rule[&via=<perfil>&target=<sitio>&app=<exe>&type=<tipo>]
// `type` (an adapter type) is an extra the Perfiles first-run tiles use to
// skip straight to "Usar <app>". Opening changes only the query string of
// the current page, so it never counts as leaving a form
// (UnsavedChangesGuard) and Back closes the sheet.
import { useCallback } from 'react'
import { useLocation, useNavigate } from 'react-router'

export type AddMode = 'menu' | 'vpn' | 'file' | 'rule'

export interface AddRequest {
  mode: AddMode
  /** Profile the rule should use ("sale por"). */
  via?: string
  /** Site, URL, IP or CIDR to prefill the rule with. */
  target?: string
  /** Program to prefill the rule with (app:<exe>). */
  app?: string
  /** Adapter type to start "Conectar una VPN que ya uso" with. */
  type?: string
}

const MODES: readonly AddMode[] = ['menu', 'vpn', 'file', 'rule']
const EXTRA = ['via', 'target', 'app', 'type'] as const
export const ADD_PARAMS: readonly string[] = ['add', ...EXTRA]

/** Longest value kept from the URL (a DNS name is ≤ 253; paths fit too). */
const MAX_VALUE = 260

/** The sheet's request in this URL, or null when it is closed. Unknown modes open the menu. */
export function parseAddParams(params: URLSearchParams): AddRequest | null {
  const add = params.get('add')
  if (add === null) return null
  const mode = (MODES as readonly string[]).includes(add.trim()) ? (add.trim() as AddMode) : 'menu'
  const req: AddRequest = { mode }
  for (const key of EXTRA) {
    const value = params.get(key)?.trim()
    if (value) req[key] = value.slice(0, MAX_VALUE)
  }
  return req
}

/** The query string for `req`, keeping the page's own parameters. */
export function addSearch(current: URLSearchParams, req: AddRequest): string {
  const next = new URLSearchParams(current)
  for (const key of ADD_PARAMS) next.delete(key)
  next.set('add', req.mode)
  for (const key of EXTRA) {
    const value = req[key]
    if (value) next.set(key, value)
  }
  return `?${next.toString()}`
}

/** The query string with the sheet closed. */
export function withoutAdd(current: URLSearchParams): string {
  const next = new URLSearchParams(current)
  for (const key of ADD_PARAMS) next.delete(key)
  const s = next.toString()
  return s ? `?${s}` : ''
}

/** Opens / closes the hosted "＋ Agregar" sheet on the current page. */
export function useAddSheet() {
  const navigate = useNavigate()
  const location = useLocation()
  const open = useCallback(
    (req: AddRequest) =>
      void navigate({ pathname: location.pathname, search: addSearch(new URLSearchParams(location.search), req), hash: location.hash }),
    [navigate, location.pathname, location.search, location.hash],
  )
  const close = useCallback(
    () =>
      void navigate(
        { pathname: location.pathname, search: withoutAdd(new URLSearchParams(location.search)), hash: location.hash },
        { replace: true },
      ),
    [navigate, location.pathname, location.search, location.hash],
  )
  return { open, close }
}
