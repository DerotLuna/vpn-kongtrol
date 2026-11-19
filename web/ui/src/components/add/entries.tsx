// Kongtrol UI — "＋ Agregar" entry points used across pages · v1.0.0 · 2026-11-19
//
// Every page that offers "add" opens the shell's one sheet (ADR-0052)
// through the deep link (app/addLink.ts), prefilled from where it was
// pressed: a tunnel row (via=<perfil>), a lookup result (target / app),
// an Actividad notice (via=<perfil>).
import type { ReactNode } from 'react'
import { useAddSheet, type AddRequest } from '../../app/addLink'
import { usePreferences } from '../../app/preferences-context'
import { Button, IconButton, type ButtonProps } from '../ui/Button'
import { PlusIcon } from '../ui/icons'

/** A button that opens the sheet with `req`. */
export function AddButton({ req, children, ...rest }: { req: AddRequest; children: ReactNode } & Omit<ButtonProps, 'onClick'>) {
  const { open } = useAddSheet()
  return (
    <Button icon={<PlusIcon />} aria-haspopup="dialog" onClick={() => open(req)} {...rest}>
      {children}
    </Button>
  )
}

/** "＋" on a tunnel row: "Hacer que un sitio salga por {vpn}" (name + tooltip). */
export function AddRuleIcon({ via }: { via: string }) {
  const { t } = usePreferences()
  const { open } = useAddSheet()
  return (
    <IconButton
      label={t('add.siteVia', { vpn: via })}
      icon={<PlusIcon />}
      aria-haspopup="dialog"
      onClick={() => open({ mode: 'rule', via })}
    />
  )
}
