// Kongtrol UI — optimistic mutations with rollback · v0.2.0 · 2026-11-18
//
// The pattern from TanStack Query's "Optimistic Updates" guide: cancel
// in-flight reads of the resource, snapshot it, apply the expected result at
// once, roll back to the snapshot if the daemon refuses, and re-read the
// resource either way so the screen ends on the daemon's truth. A refusal
// becomes one toast with the "what to do" line and a Retry action.
import { useMutation, useQueryClient, type QueryKey } from '@tanstack/react-query'
import { describeError } from '../api/errors'
import { ApiError } from '../api/client'
import { usePreferences } from '../app/preferences-context'
import { toasts } from './toast'

export interface OptimisticOptions<TVars, TData, TCache> {
  queryKey: QueryKey
  mutationFn: (vars: TVars) => Promise<TData>
  /** The cache as it should look if the daemon accepts. */
  apply: (current: TCache, vars: TVars) => TCache
  /** Toast after success (info), or undefined when the change is visible in place. */
  successMessage?: (vars: TVars, data: TData) => string | undefined
  /**
   * A refusal the page shows in place (e.g. a dialog listing what blocks a
   * delete). Return true when handled: the cache still rolls back, but no toast.
   */
  onRefused?: (err: unknown, vars: TVars) => boolean
}

interface Snapshot<TCache> {
  previous: TCache | undefined
}

export function useOptimisticMutation<TVars, TData, TCache>(opts: OptimisticOptions<TVars, TData, TCache>) {
  const client = useQueryClient()
  const { t } = usePreferences()
  const { queryKey, apply, successMessage } = opts

  const mutation = useMutation<TData, unknown, TVars, Snapshot<TCache>>({
    mutationFn: opts.mutationFn,
    onMutate: async (vars) => {
      await client.cancelQueries({ queryKey })
      const previous = client.getQueryData<TCache>(queryKey)
      if (previous !== undefined) client.setQueryData<TCache>(queryKey, apply(previous, vars))
      return { previous }
    },
    onError: (err, vars, snapshot) => {
      if (snapshot?.previous !== undefined) client.setQueryData<TCache>(queryKey, snapshot.previous)
      if (opts.onRefused?.(err, vars)) return
      if (err instanceof ApiError && err.kind === 'unauthorized') return
      const { message, fix } = describeError(err, t)
      toasts.show({
        tone: 'bad',
        message,
        detail: fix,
        action: { label: t('error.retry'), run: () => mutation.mutate(vars) },
      })
    },
    onSuccess: (data, vars) => {
      const message = successMessage?.(vars, data)
      if (message) toasts.show({ tone: 'info', message })
    },
    onSettled: () => client.invalidateQueries({ queryKey }),
  })
  return mutation
}
