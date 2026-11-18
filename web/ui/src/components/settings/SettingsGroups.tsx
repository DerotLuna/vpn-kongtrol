// Kongtrol UI — Ajustes: appearance, privacy, help links · v1.0.0 · 2026-11-18
//
// Language and theme go through PreferencesProvider (the same
// ~/.kongtrol/preferences.json the tray and CLI read). Privacy only links the
// public IP lookup to Seguridad, where its single switch and "uses an external
// service" notice live (legacy had a second, notice-less switch here).
import { useId } from 'react'
import { Link } from 'react-router'
import { usePreferences } from '../../app/preferences-context'
import { useSecurityQuery } from '../../data/resources'
import type { Lang } from '../../i18n/messages'
import { guideLink, siteLink } from '../../links'
import type { Theme } from '../../theme/theme'
import { ExternalLink } from '../ui/ExternalLink'
import { StatusPill } from '../ui/StatusPill'
import { SettingRow, SettingsSection } from './SettingsSection'

interface Option<V extends string> {
  value: V
  label: string
  lang?: string
}

/** Native radios styled as a segmented control: arrows move, Space selects. */
function Segmented<V extends string>({
  legend,
  value,
  options,
  onChange,
  hint,
}: {
  legend: string
  value: V
  options: Option<V>[]
  onChange: (v: V) => void
  hint?: string
}) {
  const name = useId()
  return (
    <fieldset className="k-seg-field" aria-describedby={hint ? `${name}-h` : undefined}>
      <legend className="k-set__label">{legend}</legend>
      <div className="k-seg">
        {options.map((o) => (
          <label key={o.value} className="k-seg__opt" lang={o.lang}>
            <input
              type="radio"
              name={name}
              value={o.value}
              checked={value === o.value}
              onChange={() => onChange(o.value)}
            />
            <span>{o.label}</span>
          </label>
        ))}
      </div>
      {hint && (
        <p id={`${name}-h`} className="k-set__hint">
          {hint}
        </p>
      )}
    </fieldset>
  )
}

export function AppearanceSection() {
  const { t, lang, setLang, theme, toggleTheme } = usePreferences()
  return (
    <SettingsSection id="look" title={t('settings.look.title')} what={t('settings.look.what')}>
      <div className="k-set__pair">
        <Segmented<Lang>
          legend={t('settings.lang')}
          value={lang}
          onChange={setLang}
          options={[
            { value: 'es', label: t('settings.lang.es'), lang: 'es' },
            { value: 'en', label: t('settings.lang.en'), lang: 'en' },
          ]}
        />
        <Segmented<Theme>
          legend={t('settings.theme')}
          value={theme}
          onChange={(v) => {
            if (v !== theme) toggleTheme()
          }}
          hint={t('settings.theme.hint')}
          options={[
            { value: 'light', label: t('settings.theme.light') },
            { value: 'dark', label: t('settings.theme.dark') },
          ]}
        />
      </div>
    </SettingsSection>
  )
}

export function PublicIPRow() {
  const { t } = usePreferences()
  const { data } = useSecurityQuery()
  const on = data?.public_ip_check_enabled
  return (
    <SettingRow label={t('settings.publicip')} hint={t('settings.publicip.what')}>
      {data ? (
        <StatusPill tone={on ? 'on' : 'idle'}>{on ? t('settings.publicip.on') : t('settings.publicip.off')}</StatusPill>
      ) : (
        <StatusPill tone="unknown">…</StatusPill>
      )}
      <Link className="k-set__link" to="/security">
        {t('settings.publicip.link')}
      </Link>
    </SettingRow>
  )
}

export function PrivacySection() {
  const { t } = usePreferences()
  return (
    <SettingsSection id="privacy" title={t('settings.privacy.title')} what={t('settings.privacy.what')}>
      <PublicIPRow />
    </SettingsSection>
  )
}

export function AboutSection() {
  const { t, lang } = usePreferences()
  return (
    <SettingsSection id="about" title={t('settings.about.title')}>
      <ul className="k-set__links">
        <li>
          <ExternalLink href={siteLink('guide', lang)}>{t('settings.about.guide')}</ExternalLink>
        </li>
        <li>
          <ExternalLink href={guideLink('trouble')}>{t('settings.about.trouble')}</ExternalLink>
        </li>
        <li>
          <ExternalLink href={siteLink('privacy', lang)}>{t('settings.about.privacy')}</ExternalLink>
        </li>
        <li>
          <ExternalLink href={siteLink('security', lang)}>{t('settings.about.security')}</ExternalLink>
        </li>
      </ul>
    </SettingsSection>
  )
}
