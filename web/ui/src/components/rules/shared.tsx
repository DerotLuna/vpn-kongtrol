// Reglas + Horarios — pieces both pages share · v1.0.0 · 2026-11-18
import type { ReactNode } from 'react'
import { usePreferences } from '../../app/preferences-context'
import type { TunnelState } from '../../data/resources'
import { useProfileState } from './hooks'
import type { MessageKey } from '../../i18n/messages'
import { PageHeader } from '../layout/PageHeader'
import { Button } from '../ui/Button'
import { Dialog } from '../ui/Dialog'
import { StatusPill, type PillTone } from '../ui/StatusPill'
import { TabLinks } from '../ui/Tabs'

export function RulesHeader() {
  const { t } = usePreferences()
  return (
    <PageHeader title={t('nav.rules')} lead={t('page.rules.lead')}>
      <TabLinks
        label={t('nav.rules')}
        items={[
          { to: '/rules', label: t('page.rules.tab.policies') },
          { to: '/rules/schedule', label: t('page.rules.tab.schedule') },
        ]}
      />
    </PageHeader>
  )
}

const TONE: Record<TunnelState, PillTone> = {
  connected: 'ok',
  connecting: 'connecting',
  no_response: 'caution',
  error: 'bad',
  disconnected: 'idle',
}

const WORD: Record<TunnelState, MessageKey> = {
  connected: 'tunnels.status.connected',
  connecting: 'tunnels.status.connecting',
  no_response: 'tunnels.status.noResponse',
  error: 'tunnels.status.error',
  disconnected: 'tunnels.status.disconnected',
}

/** The VPN's name in mono plus its live state (Four States Rule: word + color + dot shape). */
export function VpnState({ name }: { name: string }) {
  const { t } = usePreferences()
  const state = useProfileState(name)
  return (
    <span className="r-vpn">
      <span className="mono">{name}</span>
      {state ? (
        <StatusPill tone={TONE[state]}>{t(WORD[state])}</StatusPill>
      ) : (
        <StatusPill tone="unknown">{t('rules.vpn.unknown')}</StatusPill>
      )}
    </span>
  )
}

interface ConfirmDeleteProps {
  open: boolean
  title: string
  body: ReactNode
  confirm: string
  onConfirm: () => void
  onClose: () => void
}

/** Destructive confirmation: Cancel first (the safe default), then the red action. */
export function ConfirmDelete({ open, title, body, confirm, onConfirm, onClose }: ConfirmDeleteProps) {
  const { t } = usePreferences()
  return (
    <Dialog
      open={open}
      onClose={onClose}
      title={title}
      description={body}
      alert
      actions={
        <>
          <Button variant="text" onClick={onClose}>
            {t('rules.action.cancel')}
          </Button>
          <Button
            variant="danger"
            onClick={() => {
              onConfirm()
              onClose()
            }}
          >
            {confirm}
          </Button>
        </>
      }
    />
  )
}
