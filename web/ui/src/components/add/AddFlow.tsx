// Kongtrol UI — "＋ Agregar" sheet · v1.4.0 · 2026-11-25
//
// One sheet for every "add" in the panel (ADR-0055): a menu with three
// plain-language choices, then
//   A "Conectar una VPN que ya uso"  (VpnStep: found apps first, "Usar")
//   B "Importar un archivo de VPN"   (FileStep: .conf / .ovpn, drop)
//   C "Hacer que un sitio salga por una VPN" (RuleStep: one sentence)
// and a Done step with the next useful actions. A and B save (restarting
// only when the daemon asks): "Guardar y activar" also connects the new
// profile, "Guardar" only saves it (ADR-0115); C saves paused. C with no profile runs A first
// and comes back with the draft kept.
//
// Native <dialog> sheet (kit Dialog): Esc and the close button ask first
// when there is a draft; focus starts on each step's first control
// ([data-autofocus]) and returns to whatever opened the sheet.
import { useCallback, useEffect, useMemo, useRef, useState } from 'react'
import { useNavigate } from 'react-router'
import { useAuthState } from '../../api/auth'
import type { AddRequest } from '../../app/addLink'
import { usePreferences } from '../../app/preferences-context'
import { useUnsavedChanges } from '../../app/unsaved'
import { droppedFile } from '../../data/add'
import { useTunnelActions } from '../../data/home'
import { useDetectedClients, useProfilesQuery, type AdapterType, type VPNProfile } from '../../data/profiles'
import { useExplain, useTogglePolicy } from '../../data/rules'
import { useTunnelsQuery } from '../../data/resources'
import type { MessageKey } from '../../i18n/messages'
import { Button } from '../ui/Button'
import { Dialog } from '../ui/Dialog'
import { CheckIcon, HomeIcon, PlugIcon, PlusIcon } from '../ui/icons'
import { Banner, ErrorState, Skeleton } from '../ui/States'
import { FileIcon } from '../profiles/icons'
import { RouteIcon } from '../rules/icons'
import { isKnownType, specOf } from '../profiles/catalog'
import { FileStep } from './FileStep'
import { defaultVia, emptyRuleDraft } from './model'
import { MoveRules } from './MoveRules'
import { Choice } from './parts'
import { saveAddRecovery, type AddRecovery } from './recovery'
import { RuleStep, type RuleDraft, type SavedRule } from './RuleStep'
import { VpnStep } from './VpnStep'

type View =
  | { k: 'menu' }
  | { k: 'vpn'; type?: AdapterType | undefined }
  | { k: 'file'; note?: string | undefined }
  | { k: 'rule' }
  | { k: 'doneVpn'; name: string; path?: string | undefined; provider?: string | undefined; type?: AdapterType | undefined; connecting?: boolean | undefined }
  | { k: 'doneRule'; rule: SavedRule }

const TITLE: Record<View['k'], MessageKey> = {
  menu: 'add.title.menu',
  vpn: 'add.title.vpn',
  file: 'add.title.file',
  rule: 'add.title.rule',
  doneVpn: 'add.title.done',
  doneRule: 'add.title.done',
}

function initialView(req: AddRequest): View {
  if (req.mode === 'vpn') {
    if (req.type === 'wireguard' || req.type === 'openvpn') return { k: 'file' }
    return { k: 'vpn', type: req.type && isKnownType(req.type) ? req.type : undefined }
  }
  if (req.mode === 'file') return { k: 'file' }
  if (req.mode === 'rule') return { k: 'rule' }
  return { k: 'menu' }
}

export function AddFlow({ req, recovery, onClose }: { req: AddRequest; recovery: AddRecovery | null; onClose: () => void }) {
  const { t } = usePreferences()
  const auth = useAuthState()
  const navigate = useNavigate()
  const profilesQ = useProfilesQuery()
  const tunnelsQ = useTunnelsQuery()
  const profiles = useMemo(() => profilesQ.data ?? [], [profilesQ.data])
  const tunnels = useMemo(() => tunnelsQ.data ?? [], [tunnelsQ.data])
  const names = useMemo(() => profiles.map((p) => p.name), [profiles])

  const [view, setView] = useState<View>(() =>
    recovery ? (recovery.draft ? { k: 'rule' } : { k: 'doneVpn', name: recovery.profile }) : initialView(req),
  )
  const [rule, setRule] = useState<RuleDraft>(() => recovery?.draft ?? emptyRuleDraft(req.app ? `app:${req.app}` : (req.target ?? ''), ''))
  const [viaChosen, setViaChosen] = useState(!!recovery?.draft)
  const [backToRule, setBackToRule] = useState(false)
  const [backNote, setBackNote] = useState<string | undefined>(() =>
    recovery?.draft ? t('add.rule.back', { name: recovery.profile }) : undefined,
  )
  const [resume, setResume] = useState<{ saved: boolean; target: string } | null>(null)
  const [stepDirty, setStepDirty] = useState(false)
  const [confirmClose, setConfirmClose] = useState(false)
  const runTunnel = useTunnelActions()
  const body = useRef<HTMLDivElement>(null)
  const [opener] = useState(() => document.activeElement as HTMLElement | null)

  // "sale por" follows the deep link / only profile / first connected until the user picks.
  const via = viaChosen ? rule.via : defaultVia(names, tunnels, req.via)
  const ruleDraft = useMemo(() => ({ ...rule, via }), [rule, via])
  const ruleDirty = view.k === 'rule' && (rule.input !== emptyRuleDraft(req.app ? `app:${req.app}` : (req.target ?? '')).input || rule.name !== '' || !rule.subs || rule.onDown !== 'inherit' || viaChosen)
  const dirty = !resume?.saved && ((view.k === 'vpn' || view.k === 'file') ? stepDirty : ruleDirty || (backToRule && rule.input !== ''))
  useUnsavedChanges(dirty)

  // Focus the first control of each step; give focus back on close.
  const focusFirst = useCallback(() => body.current?.querySelector<HTMLElement>('[data-autofocus]')?.focus(), [])
  useEffect(() => {
    focusFirst()
  }, [view.k, focusFirst, profilesQ.isPending])
  useEffect(
    () => () => {
      if (opener?.isConnected) opener.focus()
    },
    [opener],
  )

  // Files dropped on the window while the menu or the import step is open.
  const [incoming, setIncoming] = useState<File[] | null>(() => droppedFile.take())
  const viewKind = useRef(view.k)
  useEffect(() => {
    viewKind.current = view.k
  })
  useEffect(
    () =>
      droppedFile.subscribe(() => {
        const files = droppedFile.take()
        if (!files || (viewKind.current !== 'menu' && viewKind.current !== 'file')) return
        setIncoming(files)
        setView({ k: 'file' })
      }),
    [],
  )

  const requestClose = () => {
    if (dirty && !confirmClose) setConfirmClose(true)
    else onClose()
  }
  const go = (next: View) => {
    setConfirmClose(false)
    setStepDirty(false)
    setView(next)
  }
  const onDirty = useCallback((d: boolean) => setStepDirty(d), [])

  const vpnDone = (name: string, reauth: boolean, connect: boolean, path?: string, provider?: string, type?: AdapterType) => {
    // "Guardar y activar": connect now. After a restart that needs a new
    // session (reauth) the page reloads, so it is left to the user.
    if (connect && !reauth) runTunnel(name, 'connect')
    if (reauth) {
      const draft = backToRule ? { ...rule, via: name } : null
      setResume({ saved: saveAddRecovery({ profile: name, draft }), target: draft?.input ?? '' })
      setStepDirty(false)
      return
    }
    if (backToRule) {
      setBackToRule(false)
      setRule((r) => ({ ...r, via: name }))
      setViaChosen(true)
      setBackNote(t('add.rule.back', { name }))
      go({ k: 'rule' })
      return
    }
    go({ k: 'doneVpn', name, path, provider, type, connecting: connect })
  }

  const finish = (to?: string) => {
    onClose()
    if (to) void navigate(to)
  }

  let content
  if (auth === 'unauthorized') {
    content = (
      <Banner tone="bad" urgent title={t(resume ? 'add.reauth.saved' : 'error.unauthorized')}>
        <p>{t('error.unauthorized.fix')}</p>
        {resume?.saved && <p>{t('add.reauth.resume')}</p>}
        {resume?.target && !resume.saved && <p>{t('add.reauth.copy')}</p>}
        {!!resume?.target && <p><code className="mono">{resume.target}</code></p>}
      </Banner>
    )
  } else if (profilesQ.isPending && view.k !== 'menu') {
    content = <Skeleton lines={3} label={t('add.title.menu')} />
  } else if (profilesQ.isError && !profilesQ.data && view.k !== 'menu') {
    content = <ErrorState error={profilesQ.error} onRetry={() => void profilesQ.refetch()} />
  } else {
    switch (view.k) {
      case 'menu':
        content = <Menu onPick={(k) => go(k === 'vpn' ? { k: 'vpn' } : k === 'file' ? { k: 'file' } : { k: 'rule' })} />
        break
      case 'vpn':
        content = (
          <VpnStep
            profiles={profiles}
            initialType={view.type}
            onBack={() => go(backToRule ? { k: 'rule' } : { k: 'menu' })}
            onNeedsFile={(type) => go({ k: 'file', note: t('add.file.fromPicker', { app: specOf(type).label }) })}
            onDone={(name, reauth, type, connect) => vpnDone(name, reauth, connect, undefined, undefined, type)}
            onDirty={onDirty}
            focusFirst={focusFirst}
          />
        )
        break
      case 'file':
        content = (
          <FileStep
            profiles={profiles}
            incoming={incoming}
            note={view.note}
            onBack={() => go(backToRule ? { k: 'rule' } : { k: 'menu' })}
            onDone={(name, path, reauth, provider, connect) => vpnDone(name, reauth, connect, path, provider)}
            onDirty={onDirty}
          />
        )
        break
      case 'rule':
        content = (
          <RuleStep
            draft={ruleDraft}
            profileAction={profiles.find((p) => p.name === ruleDraft.via)?.on_tunnel_down ?? 'allow'}
            onChange={(patch) => {
              if (patch.via !== undefined) setViaChosen(true)
              setRule((r) => ({ ...r, ...patch }))
            }}
            profiles={names}
            tunnels={tunnels}
            backNote={backNote}
            onBack={() => go({ k: 'menu' })}
            onNeedVpn={() => {
              setBackToRule(true)
              go({ k: 'vpn' })
            }}
            onDone={(saved) => {
              setBackNote(undefined)
              go({ k: 'doneRule', rule: saved })
            }}
          />
        )
        break
      case 'doneVpn':
        content = (
          <DoneVpn
            name={view.name}
            path={view.path}
            provider={view.provider}
            type={view.type ?? profiles.find((p) => p.name === view.name)?.type}
            connecting={view.connecting === true}
            profiles={profiles}
            onRule={() => {
              setRule(emptyRuleDraft('', view.name))
              setViaChosen(true)
              go({ k: 'rule' })
            }}
            onHome={() => finish('/')}
          />
        )
        break
      case 'doneRule':
        content = (
          <DoneRule
            rule={view.rule}
            connected={tunnels.some((tn) => tn.Name === view.rule.via && tn.Status === 'connected')}
            onAnother={() => {
              setRule(emptyRuleDraft('', view.rule.via))
              setViaChosen(true)
              go({ k: 'rule' })
            }}
            onFinish={() => finish()}
          />
        )
        break
    }
  }

  return (
    <Dialog open onClose={requestClose} title={t(TITLE[view.k])} variant="sheet" className="k-add-sheet">
      <div ref={body} className="k-add">
        {confirmClose && (
          <Banner
            tone="caution"
            title={t('unsaved.title')}
            urgent
            actions={
              <>
                <Button variant="filled" size="sm" onClick={() => setConfirmClose(false)} data-autofocus="">
                  {t('unsaved.keep')}
                </Button>
                <Button variant="danger" size="sm" onClick={onClose}>
                  {t('unsaved.discard')}
                </Button>
              </>
            }
          >
            <span>{t('unsaved.body')}</span>
          </Banner>
        )}
        {content}
      </div>
    </Dialog>
  )
}

function Menu({ onPick }: { onPick: (k: 'vpn' | 'file' | 'rule') => void }) {
  const { t } = usePreferences()
  const detected = useDetectedClients()
  // The app connected right now first: it is the one the user is most likely adding.
  const known = (detected.data ?? []).filter((d) => isKnownType(d.adapter_type))
  const apps = [...new Set([...known].sort((a, b) => Number(b.active === true) - Number(a.active === true)).map((d) => d.label || specOf(d.adapter_type).label))]
  return (
    <div className="k-add__step">
      <p className="k-add__muted">{t('add.menu.lead')}</p>
      <ul className="k-add-choices">
        <Choice
          icon={<PlugIcon />}
          title={t('add.menu.vpn')}
          body={
            <>
              {t('add.menu.vpn.what')}
              {apps.length > 0 && <span className="k-add-choice__found">{t('add.menu.vpn.found', { apps: apps.join(', ') })}</span>}
            </>
          }
          onClick={() => onPick('vpn')}
          autoFocus
        />
        <Choice icon={<FileIcon />} title={t('add.menu.file')} body={t('add.menu.file.what')} onClick={() => onPick('file')} />
        <Choice icon={<RouteIcon />} title={t('add.menu.rule')} body={t('add.menu.rule.what')} onClick={() => onPick('rule')} />
      </ul>
    </div>
  )
}

function DoneVpn({
  name,
  path,
  provider,
  type,
  connecting,
  profiles,
  onRule,
  onHome,
}: {
  name: string
  path?: string | undefined
  /** "Guardar y activar" already asked to connect it (ADR-0115): no "Conectar" choice. */
  connecting: boolean
  /** The new profile's type: a provider app may take over its files' rules (ADR-0104). */
  type?: AdapterType | undefined
  profiles: readonly VPNProfile[]
  /** Whose account issued the imported file (ADR-0103): Proton clashes with its own app. */
  provider?: string | undefined
  onRule: () => void
  onHome: () => void
}) {
  const { t } = usePreferences()
  const run = useTunnelActions()
  return (
    <div className="k-add__step">
      <Banner tone="ok" title={t(connecting ? 'add.done.vpn.connecting' : 'add.done.vpn', { name })}>
        <span>{t(connecting ? 'add.done.vpn.connecting.what' : 'add.done.vpn.what')}</span>
      </Banner>
      {provider === 'protonvpn' && <Banner tone="caution" title={t('add.done.protonHint')} />}
      <MoveRules to={name} type={type} profiles={profiles} />
      {path && (
        <p className="k-add__small">
          {t('add.file.stored')} <code className="mono">{path}</code>
        </p>
      )}
      <ul className="k-add-choices">
        {!connecting && (
          <Choice icon={<PlugIcon />} title={t('add.done.connect', { name })} onClick={() => {
            run(name, 'connect')
            onHome()
          }} autoFocus />
        )}
        <Choice icon={<RouteIcon />} title={t('add.done.rule')} onClick={onRule} />
        <Choice icon={<HomeIcon />} title={t('add.done.home')} onClick={onHome} autoFocus={connecting} />
      </ul>
    </div>
  )
}

function DoneRule({ rule, connected, onAnother, onFinish }: { rule: SavedRule; connected: boolean; onAnother: () => void; onFinish: () => void }) {
  const { t } = usePreferences()
  const run = useTunnelActions()
  const activation = useTogglePolicy()
  const [active, setActive] = useState(false)
  const [asked, setAsked] = useState(false)
  const result = useExplain(active && asked ? rule.probe : '')
  const activate = async () => {
    try {
      await activation.mutateAsync({ name: rule.name, disabled: false, inline: true })
      setActive(true)
    } catch {
      // The error stays by the activation button, which can be used again.
    }
  }
  // ADR-0113: a program is only recognised, never routed: no "Probado: …
  // sale por …" for it, no test, the honest line instead.
  const isApp = /^app:/i.test(rule.probe)
  const target = rule.probe.replace(/^app:/i, '')
  let tested: string | null = null
  if (!isApp && asked && result.data) {
    const ex = result.data
    if (!ex.Matched || ex.Via !== rule.via) tested = t('add.done.tested.other', { target })
    else if (connected) tested = t('add.done.tested.via', { target, via: ex.Via, rule: ex.RuleName })
    else tested = t(rule.onDown === 'allow' ? 'add.done.tested.offAllow' : 'add.done.tested.offBlock', { target, via: ex.Via })
  }
  return (
    <div className="k-add__step">
      <Banner tone="ok" title={t('add.done.rule.title', { name: rule.name })}>
        <span>{t(active ? 'add.done.rule.active' : 'add.done.rule.paused')}</span>
      </Banner>
      {activation.isError && <ErrorState error={activation.error} />}
      {isApp && <Banner tone="caution" title={t('policy.app.notApplied')} />}
      <div aria-live="polite">{tested && <p className="k-add-preview">{tested}</p>}</div>
      <footer className="k-dialog__actions k-add__actions">
        <Button variant="text" className="k-add__back" icon={<PlusIcon />} onClick={onAnother}>
          {t('add.done.another')}
        </Button>
        {!active && (
          <Button variant="outlined" loading={activation.isPending} onClick={() => void activate()}>
            {t('add.done.activate')}
          </Button>
        )}
        {active && !connected && !isApp && (
          <Button variant="outlined" icon={<PlugIcon />} onClick={() => run(rule.via, 'connect')}>
            {t('add.done.connectNow', { name: rule.via })}
          </Button>
        )}
        {active && !isApp && (
          <Button variant="outlined" icon={<CheckIcon />} loading={asked && result.isFetching} onClick={() => (asked ? void result.refetch() : setAsked(true))}>
            {t('add.done.test')}
          </Button>
        )}
        <Button onClick={onFinish} data-autofocus="">{t('add.finish')}</Button>
      </footer>
    </div>
  )
}
