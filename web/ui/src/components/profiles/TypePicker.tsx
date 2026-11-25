// Kongtrol UI — step 1 of "Nueva VPN": which VPN is it? · v1.0.1 · 2026-11-25
//
// Two groups in plain words (the app you already use / a file or client
// Kongtrol drives), each VPN as a monogram tile with how it works on this
// computer. VPN apps found on this computer are marked and listed first,
// so the non-technical path is "pick the one you have".
import type { AdapterType, DetectedClient } from '../../data/profiles'
import { usePreferences } from '../../app/preferences-context'
import { CATALOG, type AdapterSpec } from './catalog'
import { Monogram, SupportChip } from './Bits'
import './picker.css'

interface Props {
  detected: readonly DetectedClient[]
  onPick: (type: AdapterType) => void
}

function Tile({ spec, found, onPick }: { spec: AdapterSpec; found: boolean; onPick: (t: AdapterType) => void }) {
  const { t } = usePreferences()
  return (
    <li>
      <button type="button" className="kp-tile" onClick={() => onPick(spec.type)}>
        <Monogram spec={spec} />
        <span className="kp-tile__text">
          <span className="kp-tile__name">{spec.label}</span>
          <span className="kp-tile__meta">
            {found && <span className="kp-found">{t('profiles.picker.found')}</span>}
            <SupportChip spec={spec} />
          </span>
        </span>
      </button>
    </li>
  )
}

export function TypePicker({ detected, onPick }: Props) {
  const { t } = usePreferences()
  const found = new Set(detected.map((d) => d.adapter_type).filter(Boolean))
  const order = (list: AdapterSpec[]) => [...list].sort((a, b) => Number(found.has(b.type)) - Number(found.has(a.type)))
  const groups = [
    { id: 'attach', title: t('profiles.picker.attach'), body: t('profiles.picker.attach.body'), items: order(CATALOG.filter((s) => s.group === 'attach')) },
    { id: 'direct', title: t('profiles.picker.direct'), body: t('profiles.picker.direct.body'), items: order(CATALOG.filter((s) => s.group === 'direct')) },
  ]
  return (
    <div className="kp-picker">
      {groups.map((g) => (
        <section key={g.id} aria-labelledby={`kp-pick-${g.id}`} className="kp-picker__group">
          <h3 id={`kp-pick-${g.id}`}>{g.title}</h3>
          <p className="kp-muted">{g.body}</p>
          <ul className="kp-tiles">
            {g.items.map((s) => (
              <Tile key={s.type} spec={s} found={found.has(s.type)} onPick={onPick} />
            ))}
          </ul>
        </section>
      ))}
      <p className="kp-muted kp-small">{t('profiles.picker.noAffiliation')}</p>
    </div>
  )
}
