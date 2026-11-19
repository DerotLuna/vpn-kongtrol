// Kongtrol UI — "Crea tu primera VPN" (no profiles yet) · v1.1.0 · 2026-11-19
//
// The empty state teaches the next step (NN/g; DESIGN.md Operate): three
// short steps, then the VPN apps found on this computer as one-click
// starting points, then "another VPN". Each opens the shell's "＋ Agregar"
// sheet (ADR-0055), which saves and then restarts Kongtrol on its own.
import type { AdapterType, DetectedClient } from '../../data/profiles'
import { usePreferences } from '../../app/preferences-context'
import { guideLink } from '../../links'
import { Button } from '../ui/Button'
import { ExternalLink } from '../ui/ExternalLink'
import { PlusIcon } from '../ui/icons'
import { Monogram } from './Bits'
import { isKnownType, specOf } from './catalog'

interface Props {
  setupMode: boolean
  detected: readonly DetectedClient[]
  detecting: boolean
  onStart: (type?: AdapterType) => void
}

export function FirstRun({ setupMode, detected, detecting, onStart }: Props) {
  const { t } = usePreferences()
  const supported = detected.filter((d) => isKnownType(d.adapter_type))
  const unsupported = detected.filter((d) => !isKnownType(d.adapter_type))
  // One button per adapter type even if two installs were found.
  const types = [...new Set(supported.map((d) => d.adapter_type as AdapterType))]

  return (
    <section className="kp-first" aria-labelledby="kp-first-title">
      <h2 id="kp-first-title">{t(setupMode ? 'profiles.setup.title' : 'profiles.first.title')}</h2>
      <p className="kp-first__lead">{t('profiles.first.lead')}</p>
      <ol className="kp-first__steps">
        <li>{t('profiles.first.step1')}</li>
        <li>{t('profiles.first.step2')}</li>
        <li>{t('profiles.first.step3setup')}</li>
      </ol>

      <div className="kp-first__found" aria-live="polite" aria-busy={detecting || undefined}>
        {detecting ? (
          <p className="kp-muted">{t('profiles.first.detecting')}</p>
        ) : types.length > 0 ? (
          <>
            <h3>{t('profiles.first.found')}</h3>
            <ul className="kp-tiles">
              {types.map((type) => {
                const spec = specOf(type)
                return (
                  <li key={type}>
                    <button type="button" className="kp-tile kp-tile--found" onClick={() => onStart(type)}>
                      <Monogram spec={spec} />
                      <span className="kp-tile__text">
                        <span className="kp-tile__name">{t('profiles.first.use', { app: spec.label })}</span>
                        <span className="kp-tile__meta kp-muted">{t(spec.group === 'attach' ? 'profiles.first.attachShort' : 'profiles.first.directShort')}</span>
                      </span>
                    </button>
                  </li>
                )
              })}
            </ul>
          </>
        ) : (
          <p className="kp-muted">{t('profiles.first.noneFound')}</p>
        )}
        {unsupported.length > 0 && (
          <p className="kp-muted kp-small">
            {t('profiles.first.unsupported', { apps: unsupported.map((d) => d.label).join(', ') })}
          </p>
        )}
      </div>

      <div className="kp-first__actions">
        <Button variant={types.length > 0 ? 'outlined' : 'filled'} icon={<PlusIcon />} onClick={() => onStart()}>
          {t(types.length > 0 ? 'profiles.first.other' : 'profiles.first.choose')}
        </Button>
        <ExternalLink href={guideLink('connect')}>{t('help.connect')}</ExternalLink>
      </div>
    </section>
  )
}
