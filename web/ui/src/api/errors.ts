// Kongtrol UI — one error = what happened + one "what to do" line · v0.1.0 · 2026-11-18
//
// Shared by ErrorNotice, ErrorState and the mutation toasts so an error reads
// the same everywhere (AGENTS.md "User guidance"). Uses the daemon's
// translated message and hint when it sent them; otherwise our own text.
// Never shows a raw transport error alone.
import type { MessageKey } from '../i18n/messages'
import { ApiError } from './client'

export interface ErrorText {
  message: string
  fix: string
}

export type Translate = (key: MessageKey, vars?: Record<string, string | number>) => string

export function toApiError(err: unknown): ApiError {
  return err instanceof ApiError ? err : new ApiError('server', 0, { error: String(err) })
}

export function describeError(err: unknown, t: Translate): ErrorText {
  const error = toApiError(err)
  switch (error.kind) {
    case 'offline':
      return { message: t('error.offline'), fix: t('error.offline.fix') }
    case 'unauthorized':
      return { message: t('error.unauthorized'), fix: t('error.unauthorized.fix') }
    default:
      return { message: error.message, fix: error.hint || t('error.generic.fix') }
  }
}
