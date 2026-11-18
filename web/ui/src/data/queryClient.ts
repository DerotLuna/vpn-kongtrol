// Kongtrol UI — server-state cache · v0.1.0 · 2026-11-18
//
// TanStack Query (ADR-0036) gives the cache keyed by resource,
// stale-while-revalidate, request de-duplication, retries and optimistic
// mutations with rollback, instead of hand-rolled hooks per page.
//
// Choices:
// - networkMode 'always': the API is local (127.0.0.1), so the browser's
//   online/offline flag says nothing about it; reachability comes from
//   data/connection.ts instead.
// - Retries only for "daemon not answering" and 5xx, with jittered backoff;
//   a 4xx is a real answer (validation, not found) and is shown at once.
// - staleTime 5s matches the daemon's own poll (collector, WebSocket
//   heartbeat): refetching sooner would only repeat the same data.
// - Structural sharing (default) keeps unchanged objects identical, so a
//   WebSocket tick re-renders only the rows that changed.
import { QueryClient } from '@tanstack/react-query'
import { ApiError } from '../api/client'
import { backoffDelay } from './backoff'

export const MAX_RETRIES = 2

export function shouldRetry(failureCount: number, error: unknown): boolean {
  if (failureCount >= MAX_RETRIES) return false
  if (!(error instanceof ApiError)) return false
  return error.kind === 'offline' || (error.kind === 'server' && error.status >= 500)
}

export function createQueryClient(): QueryClient {
  return new QueryClient({
    defaultOptions: {
      queries: {
        networkMode: 'always',
        staleTime: 5_000,
        gcTime: 5 * 60_000,
        refetchOnWindowFocus: true,
        retry: shouldRetry,
        retryDelay: (attempt) => backoffDelay(attempt),
      },
      mutations: {
        networkMode: 'always',
        // Never repeat a change on its own (connect, install, delete…).
        retry: false,
      },
    },
  })
}
