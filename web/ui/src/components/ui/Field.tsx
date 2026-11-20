// Kongtrol UI kit — form fields · v0.3.0 · 2026-11-19
//
// Visible label above the control (never placeholder-only), optional hint,
// and an inline error that names the problem plus one "what to do" line
// (WCAG 3.3.1 / 3.3.3; the daemon's `hint` fits the fix slot as is). The
// error is linked with aria-describedby and the control gets aria-invalid.
// Validation runs on blur / submit, not on every keystroke.
import { useId, type InputHTMLAttributes, type ReactNode, type TextareaHTMLAttributes } from 'react'
import type { DateRange } from './dates'
import { DateRangePicker } from './DateRangePicker'
import { MultiSelect } from './MultiSelect'
import { Select, type SelectOption } from './Select'

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
  children: (ids: { id: string; labelId: string; describedBy: string | undefined; invalid: boolean }) => ReactNode
}

function FieldShell({ label, hint, error, required, children }: FieldShellProps) {
  const id = useId()
  const hintId = hint ? `${id}-hint` : undefined
  const errId = error ? `${id}-err` : undefined
  const describedBy = [errId, hintId].filter(Boolean).join(' ') || undefined
  return (
    <div className="k-field" data-invalid={error ? true : undefined}>
      <label htmlFor={id} id={`${id}-label`} className="k-field__label">
        {label}
        {required && <span aria-hidden="true"> *</span>}
      </label>
      {children({ id, labelId: `${id}-label`, describedBy, invalid: !!error })}
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

/** The kit Select (own listbox, not the browser's <select>) with label, hint and error. */
export function SelectField<V extends string>({
  label,
  hint,
  error,
  required,
  mono,
  ...rest
}: Shared & {
  value: V
  onChange: (value: V) => void
  options: readonly SelectOption<V>[]
  disabled?: boolean
  placeholder?: string
  className?: string
  name?: string
}) {
  return (
    <FieldShell label={label} hint={hint} error={error ?? null} {...(required ? { required } : {})}>
      {({ id, labelId, describedBy, invalid }) => (
        <Select id={id} aria-labelledby={labelId} aria-describedby={describedBy} aria-invalid={invalid || undefined} mono={mono} {...rest} />
      )}
    </FieldShell>
  )
}

/** The kit MultiSelect (search a list, pick several, chips) with label, hint and error. */
export function MultiSelectField({
  label,
  hint,
  error,
  mono,
  ...rest
}: Omit<Shared, 'required'> & {
  value: readonly string[]
  onChange: (value: string[]) => void
  options: readonly SelectOption[]
  placeholder: string
  searchLabel: string
  chipsLabel: string
  emptyText?: string
  className?: string
}) {
  return (
    <FieldShell label={label} hint={hint} error={error ?? null}>
      {({ id, labelId, describedBy }) => <MultiSelect id={id} aria-labelledby={labelId} aria-describedby={describedBy} mono={mono} {...rest} />}
    </FieldShell>
  )
}

/** The kit DateRangePicker (one "Desde–Hasta" control with presets) with label, hint and error. */
export function DateRangeField({
  label,
  hint,
  error,
  ...rest
}: Omit<Shared, 'mono' | 'required'> & { value: DateRange; onChange: (value: DateRange) => void; max?: string }) {
  return (
    <FieldShell label={label} hint={hint} error={error ?? null}>
      {({ id, labelId, describedBy, invalid }) => (
        <DateRangePicker id={id} aria-labelledby={labelId} aria-describedby={describedBy} aria-invalid={invalid || undefined} {...rest} />
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
