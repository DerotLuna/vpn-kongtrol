// Kongtrol UI kit — Button / IconButton · v0.3.0 · 2026-11-19
//
// Variants follow Material 3's emphasis ladder mapped to DESIGN.md:
// filled = the one primary action (gold), tonal = secondary emphasis,
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
  /**
   * standard = ghost, outlined = hairline, filled = the row's one primary
   * action (gold: Conectar), strong = the state-relevant action that must
   * not be gold (Desconectar: ink outline, DESIGN.md).
   */
  variant?: 'standard' | 'outlined' | 'filled' | 'strong'
  /** md = 44px (default), sm = 36px: the same height as a small Button beside it. */
  size?: ButtonSize
  pressed?: boolean
  /** Busy: a spinner replaces the icon (aria-busy); the name stays. */
  loading?: boolean
  /** A longer tooltip than the name (it then describes the button). */
  tip?: string
}

/** Square, 44px (sm 36px); the label is the accessible name and the tooltip (WCAG 1.4.13). */
export function IconButton({ label, icon, variant = 'standard', size = 'md', pressed, loading = false, tip, className = '', type = 'button', ...rest }: IconButtonProps) {
  return (
    <Tooltip content={tip ?? label} {...(tip ? { describe: true } : {})}>
      <button
        type={type}
        aria-label={label}
        aria-pressed={pressed}
        aria-busy={loading || undefined}
        className={`k-icon-btn k-icon-btn--${variant} k-icon-btn--${size} ${className}`.trim()}
        {...rest}
      >
        {loading ? <span className="k-spinner" aria-hidden="true" /> : icon}
      </button>
    </Tooltip>
  )
}
