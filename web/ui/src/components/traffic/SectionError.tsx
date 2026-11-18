// Kongtrol UI — per-section error on the Traffic page · v1.0.0 · 2026-11-18
//
// "Kongtrol is not answering" and "open the panel from Kongtrol" are app
// state with one banner for the whole page (ADR-0038 §4); repeating them in
// every section would bury the one line that matters. Sections only show
// their own failures (the daemon answered with an error).
import { toApiError } from '../../api/errors'
import { ErrorState } from '../ui/States'

export function SectionError({ error, onRetry }: { error: unknown; onRetry: () => void }) {
  if (toApiError(error).kind !== 'server') return null
  return <ErrorState error={error} onRetry={onRetry} />
}
