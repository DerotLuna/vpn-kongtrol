// Reglas + Horarios — page icons, same stroke system as components/ui/icons.tsx
// (24px grid, 1.8 stroke, currentColor, always aria-hidden).
import type { ReactNode } from 'react'

function Icon({ children }: { children: ReactNode }) {
  return (
    <svg
      viewBox="0 0 24 24"
      fill="none"
      stroke="currentColor"
      strokeWidth="1.8"
      strokeLinecap="round"
      strokeLinejoin="round"
      aria-hidden="true"
      focusable="false"
    >
      {children}
    </svg>
  )
}

export const EditIcon = () => (
  <Icon>
    <path d="M4 20h4L19 9l-4-4L4 16v4z" />
    <path d="M13.5 6.5l4 4" />
  </Icon>
)

export const TrashIcon = () => (
  <Icon>
    <path d="M4 7h16M10 11v6M14 11v6M6 7l1 13h10l1-13M9 7V4h6v3" />
  </Icon>
)

export const RouteIcon = () => (
  <Icon>
    <circle cx="6" cy="6" r="2.5" />
    <circle cx="18" cy="18" r="2.5" />
    <path d="M8.5 6H15a3 3 0 0 1 0 6H9a3 3 0 0 0 0 6h6.5" />
  </Icon>
)

export const ClockIcon = () => (
  <Icon>
    <circle cx="12" cy="12" r="8.5" />
    <path d="M12 7.5V12l3 2" />
  </Icon>
)

export const TargetIcon = () => (
  <Icon>
    <circle cx="11" cy="11" r="6.5" />
    <path d="M20 20l-4.4-4.4" />
  </Icon>
)

export const FileIcon = () => (
  <Icon>
    <path d="M14 3H6v18h12V7l-4-4z" />
    <path d="M14 3v4h4" />
  </Icon>
)

/** Connector between the steps of a route (points right; rotated when stacked). */
export const StepArrow = () => (
  <svg viewBox="0 0 40 12" aria-hidden="true" focusable="false" preserveAspectRatio="none">
    <path d="M0 6h36" stroke="currentColor" strokeWidth="1.5" vectorEffect="non-scaling-stroke" />
    <path d="M31 1.5L37 6l-6 4.5" fill="none" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" strokeLinejoin="round" />
  </svg>
)
