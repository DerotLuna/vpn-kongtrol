// Kongtrol UI kit — Switch · v0.1.0 · 2026-11-18
//
// role="switch" on a native button (WAI-ARIA APG switch pattern): Space and
// Enter toggle, the name comes from the visible label, and the state is also
// written as a word (Apple HIG: a switch applies at once; DESIGN.md: color
// is never the only signal). Optimistic by design: the caller flips `checked`
// immediately (useOptimisticMutation) and passes `pending` while the daemon
// confirms; a refusal rolls `checked` back.
import { useId, type ReactNode } from 'react'

interface SwitchProps {
  checked: boolean
  onChange: (next: boolean) => void
  label: ReactNode
  description?: ReactNode
  /** Words for the two states, e.g. "Activado" / "Desactivado". */
  onText: string
  offText: string
  pending?: boolean
  disabled?: boolean
}

export function Switch({ checked, onChange, label, description, onText, offText, pending = false, disabled = false }: SwitchProps) {
  const id = useId()
  return (
    <div className="k-switch-row">
      <div className="k-switch-text">
        <span id={`${id}-label`} className="k-switch-label">
          {label}
        </span>
        {description && (
          <span id={`${id}-desc`} className="k-switch-desc">
            {description}
          </span>
        )}
      </div>
      <button
        type="button"
        role="switch"
        aria-checked={checked}
        aria-labelledby={`${id}-label`}
        aria-describedby={description ? `${id}-desc` : undefined}
        aria-busy={pending || undefined}
        disabled={disabled}
        className="k-switch"
        data-pending={pending || undefined}
        onClick={() => onChange(!checked)}
      >
        <span className="k-switch__track" aria-hidden="true">
          <span className="k-switch__thumb" />
        </span>
        <span className="k-switch__state">{checked ? onText : offText}</span>
      </button>
    </div>
  )
}
