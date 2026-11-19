// Kongtrol UI — Ajustes: appearance, privacy, help links · v1.1.0 · 2026-11-19
//
// Language and theme go through PreferencesProvider (the same
// ~/.kongtrol/preferences.json the tray and CLI read). Privacy only links the
// public IP lookup to Seguridad, where its single switch and "uses an external
// service" notice live (legacy had a second, notice-less switch here); the
// recent connections switch (memory only) lives here.
import { useId } from 'react'
import { Link } from 'react-router'
import { usePreferences } from '../../app/preferences-context'
import { useQueryClient } from '@tanstack/react-query'
import { useSecurityQuery } from '../../data/resources'
import { useSaveSettings, useSettingsQuery } from '../../data/settings'
import type { Lang } from '../../i18n/messages'
import { guideLink, siteLink } from '../../links'
import type { Theme } from '../../theme/theme'
import { ExternalLink } from '../ui/ExternalLink'
import { StatusPill } from '../ui/StatusPill'
import { Switch } from '../ui/Switch'
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

/**
 * Tráfico › Recientes (ADR-0054): the daemon keeps the list in memory only
 * while this is on (settings `recent_connections_enabled`, on by default).
 * An older daemon that does not send the field shows no switch.
 */
export function RecentConnectionsRow() {
  const { t } = usePreferences()
  const settings = useSettingsQuery()
  const save = useSaveSettings()
  const client = useQueryClient()
  const value = settings.data?.recent_connections_enabled
  if (typeof value !== 'boolean') return null
  return (
    <Switch
      checked={value}
      onChange={(next) =>
        save.mutate({ recent_connections_enabled: next }, { onSettled: () => void client.invalidateQueries({ queryKey: ['connections'] }) })
      }
      label={t('settings.recent')}
      description={t('settings.recent.what')}
      onText={t('settings.publicip.on')}
      offText={t('settings.publicip.off')}
      pending={save.isPending}
    />
  )
}

export function PrivacySection() {
  const { t } = usePreferences()
  return (
    <SettingsSection id="privacy" title={t('settings.privacy.title')} what={t('settings.privacy.what')}>
      <PublicIPRow />
      <RecentConnectionsRow />
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
