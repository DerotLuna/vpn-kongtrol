// Kongtrol UI kit — form fields · v0.1.0 · 2026-11-18
//
// Visible label above the control (never placeholder-only), optional hint,
// and an inline error that names the problem plus one "what to do" line
// (WCAG 3.3.1 / 3.3.3; the daemon's `hint` fits the fix slot as is). The
// error is linked with aria-describedby and the control gets aria-invalid.
// Validation runs on blur / submit, not on every keystroke.
import { useId, type InputHTMLAttributes, type ReactNode, type SelectHTMLAttributes, type TextareaHTMLAttributes } from 'react'

export interface FieldError {
  message: string
  fix?: string
}

interface FieldShellProps {
  label: string
  hint?: ReactNode
  error?: FieldError | null
  required?: boolean
  /** Machine-like values (IPs, CIDRs, paths) read in mono (DESIGN.md). */
  mono?: boolean
  children: (ids: { id: string; describedBy: string | undefined; invalid: boolean }) => ReactNode
}

function FieldShell({ label, hint, error, required, children }: FieldShellProps) {
  const id = useId()
  const hintId = hint ? `${id}-hint` : undefined
  const errId = error ? `${id}-err` : undefined
  const describedBy = [errId, hintId].filter(Boolean).join(' ') || undefined
  return (
    <div className="k-field" data-invalid={error ? true : undefined}>
      <label htmlFor={id} className="k-field__label">
        {label}
        {required && <span aria-hidden="true"> *</span>}
      </label>
      {children({ id, describedBy, invalid: !!error })}
      {error && (
        <p id={errId} className="k-field__error">
          <strong>{error.message}</strong>
          {error.fix && ' '}
          {error.fix && <span>{error.fix}</span>}
        </p>
      )}
      {hint && (
        <p id={hintId} className="k-field__hint">
          {hint}
        </p>
      )}
    </div>
  )
}

type Shared = Omit<FieldShellProps, 'children'>

export function TextField({ label, hint, error, required, mono, className = '', ...rest }: Shared & InputHTMLAttributes<HTMLInputElement>) {
  return (
    <FieldShell label={label} hint={hint} error={error ?? null} {...(required ? { required } : {})}>
      {({ id, describedBy, invalid }) => (
        <input
          id={id}
          className={`k-input ${mono ? 'mono' : ''} ${className}`.trim()}
          aria-describedby={describedBy}
          aria-invalid={invalid || undefined}
          required={required}
          {...rest}
        />
      )}
    </FieldShell>
  )
}

export function SelectField({
  label,
  hint,
  error,
  required,
  className = '',
  children,
  ...rest
}: Shared & SelectHTMLAttributes<HTMLSelectElement> & { children: ReactNode }) {
  return (
    <FieldShell label={label} hint={hint} error={error ?? null} {...(required ? { required } : {})}>
      {({ id, describedBy, invalid }) => (
        <select
          id={id}
          className={`k-input k-select ${className}`.trim()}
          aria-describedby={describedBy}
          aria-invalid={invalid || undefined}
          required={required}
          {...rest}
        >
          {children}
        </select>
      )}
    </FieldShell>
  )
}

export function TextAreaField({ label, hint, error, required, mono, className = '', ...rest }: Shared & TextareaHTMLAttributes<HTMLTextAreaElement>) {
  return (
    <FieldShell label={label} hint={hint} error={error ?? null} {...(required ? { required } : {})}>
      {({ id, describedBy, invalid }) => (
        <textarea
          id={id}
          className={`k-input k-textarea ${mono ? 'mono' : ''} ${className}`.trim()}
          aria-describedby={describedBy}
          aria-invalid={invalid || undefined}
          required={required}
          {...rest}
        />
      )}
    </FieldShell>
  )
}
