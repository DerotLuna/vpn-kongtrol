// Kongtrol UI — the guided profile form · v1.3.0 · 2026-11-24
//
// Only the fields the chosen VPN type uses, in short sections: what it is,
// how it connects, how it signs in, what happens if it drops, and the
// order of its rules (priority, visible since ADR-0108: one small number,
// empty = by position). Rarely needed settings (network connection name, wait time,
// app folder, sign-in for VPNs that hold their own session) sit under
// "Más opciones" with the technical names next to them, so the
// non-technical path stays short and developers still reach everything.
import type { ReactNode } from 'react'
import { Link } from 'react-router'
import { usePreferences } from '../../app/preferences-context'
import type { AdapterType, VPNProfile } from '../../data/profiles'
import type { MessageKey } from '../../i18n/messages'
import { SelectField, TextField, type FieldError } from '../ui/Field'
import { StatusPill } from '../ui/StatusPill'
import { Switch } from '../ui/Switch'
import { ON_TUNNEL_DOWN, type OnTunnelDown } from '../rules/model'
import { Monogram, SupportChip } from './Bits'
import { AUTH_METHODS, CATALOG, specOf, type AuthMethod } from './catalog'
import type { Draft, DraftField, Problems } from './draft'
import { FailoverEditor } from './FailoverEditor'
import { ImportFile } from './ImportFile'

interface Props {
  draft: Draft
  onChange: (patch: Partial<Draft>) => void
  onTypeChange: (type: AdapterType) => void
  problems: Problems
  /** A daemon validation error placed on its field. */
  serverErrors: Partial<Record<DraftField, FieldError>>
  profiles: readonly VPNProfile[]
  /** The stored profile when editing (credentials state shown; renaming allowed). */
  editing?: VPNProfile | undefined
  /** Its tunnel is up or connecting: a rename would be refused (api.err.vpn_rename_busy). */
  busy?: boolean
}

const AUTH_LABEL: Record<AuthMethod, MessageKey> = {
  certificate: 'profiles.auth.certificate',
  credentials: 'profiles.auth.credentials',
  'certificate+credentials': 'profiles.auth.both',
}

const OVERRIDE: Record<VPNProfile['kill_switch_override'], MessageKey> = {
  inherit: 'profiles.ks.inherit',
  on: 'profiles.ks.on',
  off: 'profiles.ks.off',
}

function Section({ title, children }: { title: string; children: ReactNode }) {
  return (
    <fieldset className="kp-section">
      <legend>{title}</legend>
      <div className="kp-section__body">{children}</div>
    </fieldset>
  )
}

/** "Etiqueta" plus its YAML key in mono, for people who also edit kongtrol.yaml. */
function Tech({ yaml }: { yaml: string }) {
  return <span className="kp-yaml mono"> · {yaml}</span>
}

export function ProfileForm({ draft, onChange, onTypeChange, problems, serverErrors, profiles, editing, busy = false }: Props) {
  const { t } = usePreferences()
  const spec = specOf(draft.type)
  const err = (f: DraftField): FieldError | null => {
    const p = problems[f]
    if (p) return { message: t(p.key, p.vars), fix: t(`${p.key}.fix` as MessageKey, p.vars) }
    return serverErrors[f] ?? null
  }
  const text = (f: DraftField) => ({
    value: draft[f] as string,
    onChange: (e: { target: { value: string } }) => onChange({ [f]: e.target.value } as Partial<Draft>),
    error: err(f),
  })
  const hasCert = draft.authMethod !== 'credentials'
  const hasCreds = draft.authMethod !== 'certificate'
  const candidates = profiles.map((p) => p.name).filter((n) => n !== (editing?.name ?? draft.name.trim()))

  const auth = (
    <>
      <SelectField
        label={t('profiles.field.authMethod')}
        hint={t('profiles.hint.authMethod')}
        value={draft.authMethod}
        onChange={(authMethod: AuthMethod) => onChange({ authMethod })}
        options={AUTH_METHODS.map((m) => ({ value: m, label: t(AUTH_LABEL[m]) }))}
      />
      {hasCert && (
        <>
          <TextField label={t('profiles.field.cert')} hint={t('profiles.hint.cert')} mono autoComplete="off" spellCheck={false} placeholder="~/.kongtrol/certs/client.crt" {...text('cert')} />
          <TextField label={t('profiles.field.key')} mono autoComplete="off" spellCheck={false} placeholder="~/.kongtrol/certs/client.key" {...text('key')} />
        </>
      )}
      {hasCreds && (
        <>
          <TextField label={t('profiles.field.username')} autoComplete="off" spellCheck={false} {...text('username')} />
          <TextField
            label={t('profiles.field.password')}
            type="password"
            autoComplete="new-password"
            placeholder={editing?.has_password_credential ? t('profiles.password.keep') : ''}
            hint={
              <>
                {t('profiles.hint.password')}{' '}
                {editing?.has_password_credential && <strong>{t('profiles.password.stored')}</strong>}
              </>
            }
            {...text('password')}
          />
        </>
      )}
    </>
  )

  const showAttach = spec.attach
  const showAdvancedAuth = spec.auth === 'advanced'
  const advanced: DraftField[] = ['iface', 'waitTimeout', 'appPath', ...(showAdvancedAuth ? (['cert', 'key', 'username', 'password'] as DraftField[]) : [])]
  const advancedProblem = advanced.some((f) => problems[f] || serverErrors[f])

  return (
    <div className="kp-form">
      <Section title={t('profiles.section.what')}>
        <div className="kp-typehead">
          <Monogram spec={spec} size="lg" />
          <div>
            <p className="kp-typehead__name">{spec.label}</p>
            <p className="kp-muted kp-small">{t(spec.group === 'attach' ? 'profiles.picker.attach.body' : 'profiles.picker.direct.body')}</p>
            <SupportChip spec={spec} long />
          </div>
        </div>
        <SelectField
          label={t('profiles.field.type')}
          value={draft.type}
          onChange={(type: AdapterType) => onTypeChange(type)}
          options={(['attach', 'direct'] as const).flatMap((g) =>
            CATALOG.filter((s) => s.group === g).map((s) => ({
              value: s.type,
              label: s.label,
              group: t(g === 'attach' ? 'profiles.picker.attach' : 'profiles.picker.direct'),
            })),
          )}
        />
        <TextField
          label={t('profiles.field.name')}
          required
          autoComplete="off"
          spellCheck={false}
          placeholder={t('profiles.placeholder.name')}
          hint={
            !editing
              ? t('profiles.hint.name')
              : busy && draft.name.trim() !== editing.name
                ? <strong>{t('profiles.hint.nameBusy', { name: editing.name })}</strong>
                : t('profiles.hint.nameEdit')
          }
          {...text('name')}
        />
      </Section>

      {(spec.config || spec.host || spec.location || spec.autoConnect || spec.serverGroup || spec.protocol) && (
        <Section title={t('profiles.section.connect')}>
          {spec.config && (
            <TextField
              label={t(spec.config === 'ovpn' ? 'profiles.field.ovpn' : 'profiles.field.conf')}
              required
              mono
              spellCheck={false}
              placeholder={t(spec.config === 'ovpn' ? 'profiles.placeholder.ovpn' : 'profiles.placeholder.conf')}
              hint={
                <>
                  {t(spec.config === 'ovpn' ? 'profiles.hint.ovpn' : 'profiles.hint.conf')}
                  {spec.config === 'ovpn' && (
                    <>
                      {' '}
                      {t('profiles.hint.ovpnRoutes')}
                    </>
                  )}
                </>
              }
              {...text('config')}
            />
          )}
          {spec.config && (draft.type === 'wireguard' || draft.type === 'openvpn') && (
            <ImportFile
              type={draft.type}
              profileName={draft.name}
              onImported={(path) => onChange({ config: path })}
              protectedPath={serverErrors.config && draft.config.startsWith('/') ? draft.config : undefined}
            />
          )}
          {spec.host && (
            <TextField
              label={t(spec.host === 'required' ? 'profiles.field.host' : 'profiles.field.exitNode')}
              required={spec.host === 'required'}
              mono
              spellCheck={false}
              placeholder={spec.host === 'required' ? t('profiles.placeholder.host') : ''}
              hint={t(spec.host === 'required' ? 'profiles.hint.host' : 'profiles.hint.exitNode')}
              {...text('host')}
            />
          )}
          {spec.autoConnect && (
            <Switch
              checked={draft.autoConnect}
              onChange={(v) => onChange({ autoConnect: v })}
              label={t('profiles.field.autoConnect')}
              description={t(draft.type === 'forticlient' ? 'profiles.hint.autoConnectForti' : 'profiles.hint.autoConnect')}
              onText={t('profiles.yes')}
              offText={t('profiles.no')}
            />
          )}
          {spec.location && (
            <div className="kp-row">
              <TextField label={t('profiles.field.country')} placeholder={t('profiles.placeholder.country')} hint={t('profiles.hint.country')} {...text('country')} />
              <TextField label={t('profiles.field.city')} placeholder={t('profiles.placeholder.city')} {...text('city')} />
            </div>
          )}
          {spec.serverGroup && (
            <TextField label={t('profiles.field.group')} placeholder={t('profiles.placeholder.group')} {...text('group')} />
          )}
          {spec.location && (
            <TextField
              label={t('profiles.field.server')}
              mono
              spellCheck={false}
              placeholder={t('profiles.placeholder.server')}
              hint={t('profiles.hint.server')}
              {...text('server')}
            />
          )}
          {spec.protocol && (
            <TextField label={t('profiles.field.protocol')} mono placeholder="wireguard" hint={t('profiles.hint.protocol')} {...text('protocol')} />
          )}
        </Section>
      )}

      {spec.auth === 'main' && <Section title={t('profiles.section.signin')}>{auth}</Section>}
      {spec.auth === 'none' && (
        <p className="kp-muted kp-small kp-note">{t('profiles.signin.app', { app: spec.label })}</p>
      )}

      <Section title={t('profiles.section.drops')}>
        <FailoverEditor
          value={draft.failover}
          onChange={(failover) => onChange({ failover })}
          candidates={candidates}
          error={err('failover')}
        />
        <div className="kp-readonly">
          <span className="k-field__label">{t('profiles.field.killSwitch')}</span>
          <span className="kp-readonly__row">
            <StatusPill tone={editing?.kill_switch_override === 'on' ? 'on' : 'idle'}>
              {t(OVERRIDE[editing?.kill_switch_override ?? 'inherit'])}
            </StatusPill>
            <Link to="/security#profile-overrides">{t('profiles.ks.change')}</Link>
          </span>
          <span className="k-field__hint">{t('profiles.ks.hint')}</span>
        </div>
        <SelectField
          label={t('profiles.onDown.label')}
          hint={t(`rules.onDown.${draft.onTunnelDown}.hint`)}
          value={draft.onTunnelDown}
          options={ON_TUNNEL_DOWN.map((value) => ({ value, label: t(`rules.onDown.${value}`) }))}
          onChange={(value) => onChange({ onTunnelDown: value as OnTunnelDown })}
        />
        <p className="k-field__hint">{t('profiles.onDown.hint')}</p>
      </Section>

      <Section title={t('profiles.section.order')}>
        <TextField
          label={t('profiles.field.priority')}
          name="priority"
          inputMode="numeric"
          autoComplete="off"
          className="kp-priority"
          placeholder={t('profiles.placeholder.priority')}
          hint={<>{t('profiles.hint.priority')}<Tech yaml="priority" /></>}
          {...text('priority')}
        />
      </Section>

      <details className="kp-more" open={advancedProblem ? true : undefined}>
        <summary>{t('profiles.section.more')}</summary>
        <div className="kp-section__body">
          {showAttach && (
            <>
              <TextField
                label={t('profiles.field.interface')}
                hint={<>{t('profiles.hint.interface')}<Tech yaml="interface" /></>}
                placeholder={t('profiles.placeholder.interface')}
                mono
                spellCheck={false}
                {...text('iface')}
              />
              <TextField
                label={t('profiles.field.waitTimeout')}
                hint={<>{t('profiles.hint.waitTimeout')}<Tech yaml="wait_timeout" /></>}
                placeholder="5m"
                mono
                spellCheck={false}
                {...text('waitTimeout')}
              />
              <TextField
                label={t('profiles.field.appPath')}
                hint={<>{t('profiles.hint.appPath')}<Tech yaml="app_path" /></>}
                mono
                spellCheck={false}
                {...text('appPath')}
              />
            </>
          )}
          {showAdvancedAuth && (
            <>
              <p className="kp-muted kp-small">{t('profiles.signin.advanced', { app: spec.label })}</p>
              {auth}
            </>
          )}
          <p className="kp-muted kp-small">
            {t('profiles.tech.type')} <code className="mono">type: {draft.type}</code>
          </p>
        </div>
      </details>
    </div>
  )
}
