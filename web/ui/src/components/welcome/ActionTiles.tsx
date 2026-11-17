import type { ReactNode } from 'react'
import { usePreferences } from '../../app/preferences-context'
import type { MessageKey } from '../../i18n/messages'
import { LEGACY } from '../../links'
import { PlugIcon, PlusIcon, PulseIcon } from '../ui/icons'

interface Tile {
  href: string
  title: MessageKey
  body: MessageKey
  icon: ReactNode
}

// Phase 1: every action still lives in the classic dashboard. Each tile
// moves to an /app/ route as its page migrates.
const TILES: Tile[] = [
  { href: LEGACY.studio, title: 'welcome.actions.add.title', body: 'welcome.actions.add.body', icon: <PlusIcon /> },
  { href: LEGACY.home, title: 'welcome.actions.connect.title', body: 'welcome.actions.connect.body', icon: <PlugIcon /> },
  { href: LEGACY.home, title: 'welcome.actions.status.title', body: 'welcome.actions.status.body', icon: <PulseIcon /> },
]

/** The big "what do you want to do" actions. Primary = Add VPN when none exist. */
export function ActionTiles({ emphasizeAdd }: { emphasizeAdd: boolean }) {
  const { t } = usePreferences()
  return (
    <section aria-labelledby="actions-title">
      <h2 id="actions-title" className="section-title">
        {t('welcome.actions.title')}
      </h2>
      <div className="grid-3">
        {TILES.map((tile, i) => {
          const primary = emphasizeAdd ? i === 0 : i === 1
          return (
            <a key={tile.title} href={tile.href} className={primary ? 'action primary' : 'action'}>
              <span className="action-icon">{tile.icon}</span>
              <strong>{t(tile.title)}</strong>
              <span className="muted">{t(tile.body)}</span>
              <span className="note">{t('welcome.actions.opensClassic')}</span>
            </a>
          )
        })}
      </div>
    </section>
  )
}
