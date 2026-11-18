// Kongtrol UI kit — Button / IconButton · v0.1.0 · 2026-11-18
//
// Variants follow Material 3's emphasis ladder mapped to DESIGN.md:
// filled = the one primary action (amber), tonal = secondary emphasis,
// outlined = neutral, text = lowest, danger = destructive (outlined red).
// Loading keeps the label (and width) and adds a spinner + aria-busy, so the
// layout never jumps and screen readers hear the same name.
import { forwardRef, type ButtonHTMLAttributes, type ReactNode } from 'react'
import { Tooltip } from './Tooltip'

export type ButtonVariant = 'filled' | 'tonal' | 'outlined' | 'text' | 'danger'
export type ButtonSize = 'md' | 'sm'

export interface ButtonProps extends ButtonHTMLAttributes<HTMLButtonElement> {
  variant?: ButtonVariant
  size?: ButtonSize
  loading?: boolean
  icon?: ReactNode
}

export const Button = forwardRef<HTMLButtonElement, ButtonProps>(function Button(
  { variant = 'filled', size = 'md', loading = false, icon, children, className = '', disabled, type = 'button', ...rest },
  ref,
) {
  return (
    <button
      ref={ref}
      type={type}
      className={`k-btn k-btn--${variant} k-btn--${size} ${className}`.trim()}
      disabled={disabled || loading}
      aria-busy={loading || undefined}
      {...rest}
    >
      {loading ? <span className="k-spinner" aria-hidden="true" /> : icon}
      <span className="k-btn__label">{children}</span>
    </button>
  )
})

export interface IconButtonProps extends Omit<ButtonHTMLAttributes<HTMLButtonElement>, 'children'> {
  /** Accessible name; also the tooltip text. */
  label: string
  icon: ReactNode
  variant?: 'standard' | 'outlined'
  pressed?: boolean
}

/** 44px square; the label is the accessible name and the tooltip (WCAG 1.4.13). */
export function IconButton({ label, icon, variant = 'standard', pressed, className = '', type = 'button', ...rest }: IconButtonProps) {
  return (
    <Tooltip content={label}>
      <button
        type={type}
        aria-label={label}
        aria-pressed={pressed}
        className={`k-icon-btn k-icon-btn--${variant} ${className}`.trim()}
        {...rest}
      >
        {icon}
      </button>
    </Tooltip>
  )
}
