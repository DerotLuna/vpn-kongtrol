// Kongtrol UI — "＋ Agregar" entry points used across pages · v1.2.0 · 2026-11-24
//
// Every page that offers "add" opens the shell's one sheet (ADR-0055)
// through the deep link (app/addLink.ts), prefilled from where it was
// pressed: a tunnel row (via=<perfil>), a lookup result (target / app),
// an Actividad notice (via=<perfil>).
import { forwardRef, type ReactNode } from 'react'
import { useAddSheet, type AddRequest } from '../../app/addLink'
import { usePreferences } from '../../app/preferences-context'
import { Button, IconButton, type ButtonProps } from '../ui/Button'
import { PlusIcon } from '../ui/icons'

/** A button that opens the sheet with `req` (forwards its ref, so a kit Tooltip can wrap it). */
export const AddButton = forwardRef<HTMLButtonElement, { req: AddRequest; children: ReactNode } & Omit<ButtonProps, 'onClick'>>(
  function AddButton({ req, children, ...rest }, ref) {
    const { open } = useAddSheet()
    return (
      <Button ref={ref} icon={<PlusIcon />} aria-haspopup="dialog" onClick={() => open(req)} {...rest}>
        {children}
      </Button>
    )
  },
)

/** "＋" beside a notice: add a routing rule for this VPN. */
export function AddRuleIcon({ via }: { via: string }) {
  const { t } = usePreferences()
  const { open } = useAddSheet()
  return (
    <IconButton
      size="sm"
      variant="outlined"
      label={t('add.siteVia', { vpn: via })}
      icon={<PlusIcon />}
      aria-haspopup="dialog"
      onClick={() => open({ mode: 'rule', via })}
    />
  )
}
