// Kongtrol UI — a group's connect / disconnect squares (Inicio "Grupos" and /profiles/groups) · v1.0.0 · 2026-11-19
//
// The same icons and sizes as a tunnel row (owner feedback 2026-11-19):
// 36px squares with a tooltip and an accessible name that carries the group.
// What shows depends on how many members are up, and the tooltip says why:
// - none up: Conectar (plug, filled amber);
// - all up: Desconectar (power, ink outline);
// - some up: both — "Conectar las que faltan" (filled: the group is not
//   complete yet) and "Desconectar todas" (outlined), with the counts.
// A square in progress keeps its spinner until every member it touched
// reached its end state (data/transition.ts, passed in as `busy`).
import { usePreferences } from '../../app/preferences-context'
import { PowerIcon } from '../profiles/icons'
import { IconButton } from '../ui/Button'
import { PlugIcon } from '../ui/icons'

type GroupAction = 'connect' | 'disconnect'

interface Props {
  name: string
  total: number
  up: number
  /** The group action in progress (pending request or members still moving). */
  busy: GroupAction | undefined
  onRun: (action: GroupAction) => void
}

export function GroupConnect({ name, total, up, busy, onRun }: Props) {
  const { t } = usePreferences()
  const showConnect = up < total
  const showDisconnect = up > 0
  const partial = showConnect && showDisconnect
  const connectLabel = busy === 'connect'
    ? t('home.groups.act.connecting', { name })
    : partial
      ? t('home.groups.act.connectRest', { name, missing: total - up, total })
      : t('home.groups.act.connect', { name })
  const disconnectLabel = busy === 'disconnect'
    ? t('home.groups.act.disconnecting', { name })
    : partial
      ? t('home.groups.act.disconnectAll', { name, up, total })
      : t('home.groups.act.disconnect', { name })
  return (
    <>
      {showConnect && (
        <IconButton
          size="sm"
          variant="filled"
          className="home-main-act"
          label={connectLabel}
          icon={<PlugIcon />}
          loading={busy === 'connect'}
          aria-disabled={busy ? true : undefined}
          onClick={busy ? undefined : () => onRun('connect')}
        />
      )}
      {showDisconnect && (
        <IconButton
          size="sm"
          variant={partial ? 'outlined' : 'strong'}
          label={disconnectLabel}
          icon={<PowerIcon />}
          loading={busy === 'disconnect'}
          aria-disabled={busy ? true : undefined}
          onClick={busy ? undefined : () => onRun('disconnect')}
        />
      )}
    </>
  )
}
