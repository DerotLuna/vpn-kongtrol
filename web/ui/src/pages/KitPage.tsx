// Kongtrol UI — component kit review page (/_kit) · v0.2.0 · 2026-11-19
//
// Development builds only (App.tsx loads it behind import.meta.env.DEV), so
// it never ships and its sample text lives here, not in messages.ts. Shows
// every kit component in every state for review in both themes and
// languages (header toggles).
import { useState } from 'react'
import { usePreferences } from '../app/preferences-context'
import { PageHeader } from '../components/layout/PageHeader'
import { Button, IconButton } from '../components/ui/Button'
import { Card } from '../components/ui/Card'
import { DataList, type Column } from '../components/ui/DataList'
import { Dialog } from '../components/ui/Dialog'
import { DateRangeField, MultiSelectField, SelectField, TextAreaField, TextField } from '../components/ui/Field'
import { PlugIcon, PlusIcon, RefreshIcon, SettingsIcon } from '../components/ui/icons'
import { PowerIcon, TrashIcon } from '../components/profiles/icons'
import { Banner, EmptyState, ErrorState, Kbd, Skeleton } from '../components/ui/States'
import { StatusPill } from '../components/ui/StatusPill'
import { Switch } from '../components/ui/Switch'
import { Tabs } from '../components/ui/Tabs'
import { Tooltip } from '../components/ui/Tooltip'
import { ApiError } from '../api/client'
import { toasts } from '../data/toast'

const TEXT = {
  es: {
    title: 'Kit de componentes',
    lead: 'Solo en desarrollo. Cada componente en cada estado, con los tokens de DESIGN.md.',
    buttons: 'Botones', connect: 'Conectar', details: 'Ver detalles', cancel: 'Cancelar', del: 'Borrar perfil', saving: 'Guardando…',
    states: 'Estados (regla de los cuatro estados)', connected: 'Conectada', connecting: 'Conectando', noResp: 'Sin respuesta',
    error: 'Con error', off: 'Desconectada', unknown: 'Sin verificar', on: 'Activado',
    switches: 'Interruptores', ks: 'Kill switch', ksDesc: 'Bloquea el tráfico si una VPN protegida se cae.', dns: 'Protección DNS',
    dnsDesc: 'Falla a propósito para probar la vuelta atrás.', onT: 'Activado', offT: 'Desactivado',
    list: 'Lista de datos', name: 'Nombre', state: 'Estado', traffic: 'Tráfico', actions: 'Acciones', tick: 'Simular tick en vivo',
    fields: 'Campos', profile: 'Nombre del perfil', profileHint: 'Solo letras, números y guiones.', cidr: 'Destino (IP o red)',
    cidrErr: 'No es una red válida.', cidrFix: 'Qué hacer: escribe algo como 10.0.0.0/8.', type: 'Tipo', notes: 'Notas',
    feedback: 'Avisos', toastOk: 'Toast de éxito', toastErr: 'Toast de error', dialog: 'Abrir diálogo', sheet: 'Abrir hoja',
    dTitle: '¿Borrar el perfil «oficina»?', dBody: 'Se borra de kongtrol.yaml y del llavero. No se puede deshacer.',
    sTitle: 'Más opciones', loading: 'Cargando perfiles…', emptyT: 'Todavía no tienes reglas', emptyB: 'Una regla decide por qué VPN sale un sitio. Crea la primera para empezar.',
    newRule: 'Nueva regla', bannerT: 'Falta el ayudante del sistema', bannerB: 'Sin él, Kongtrol no puede conectar tus VPN.', install: 'Instalar ayudante',
    infoT: 'Hay una versión nueva', infoB: 'Kongtrol 0.7.0 está disponible.', tabs: 'Pestañas', t1: 'General', t2: 'Avanzado', p1: 'Panel general.', p2: 'Panel avanzado.',
    kbd: 'Atajos', kbdLine: 'Cierra con', tip: 'Ajustes del perfil',
    controls: 'Selección, filtros y fechas', search: 'Buscar perfil', chosen: 'Perfiles elegidos', disconnect: 'Desconectar oficina', level: 'Nivel', levelAll: 'Todos', profileF: 'Perfil', profileAll: 'Todos los perfiles',
    dates: 'Fechas', reconnect: 'Reconectar', addSite: 'Hacer que un sitio salga por oficina', remove: 'Borrar',
    edge: 'Tooltip junto al borde: se corre para no salirse de la pantalla.', checks: 'Casillas y opciones', check: 'Incluir subdominios', r1: 'Usar internet normal', r2: 'Bloquear',
  },
  en: {
    title: 'Component kit',
    lead: 'Development only. Every component in every state, with the DESIGN.md tokens.',
    buttons: 'Buttons', connect: 'Connect', details: 'View details', cancel: 'Cancel', del: 'Delete profile', saving: 'Saving…',
    states: 'States (Four States Rule)', connected: 'Connected', connecting: 'Connecting', noResp: 'No response',
    error: 'Error', off: 'Disconnected', unknown: 'Not verified', on: 'On',
    switches: 'Switches', ks: 'Kill switch', ksDesc: 'Blocks traffic if a protected VPN drops.', dns: 'DNS protection',
    dnsDesc: 'Fails on purpose to show the rollback.', onT: 'On', offT: 'Off',
    list: 'Data list', name: 'Name', state: 'State', traffic: 'Traffic', actions: 'Actions', tick: 'Simulate a live tick',
    fields: 'Fields', profile: 'Profile name', profileHint: 'Letters, numbers and dashes only.', cidr: 'Destination (IP or network)',
    cidrErr: 'This is not a valid network.', cidrFix: 'What to do: type something like 10.0.0.0/8.', type: 'Type', notes: 'Notes',
    feedback: 'Feedback', toastOk: 'Success toast', toastErr: 'Error toast', dialog: 'Open dialog', sheet: 'Open sheet',
    dTitle: 'Delete the “office” profile?', dBody: 'It is removed from kongtrol.yaml and the keychain. This cannot be undone.',
    sTitle: 'More options', loading: 'Loading profiles…', emptyT: 'You have no rules yet', emptyB: 'A rule decides which VPN a site goes through. Create the first one to start.',
    newRule: 'New rule', bannerT: 'The system helper is missing', bannerB: 'Without it, Kongtrol cannot connect your VPNs.', install: 'Install helper',
    infoT: 'A new version is out', infoB: 'Kongtrol 0.7.0 is available.', tabs: 'Tabs', t1: 'General', t2: 'Advanced', p1: 'General panel.', p2: 'Advanced panel.',
    kbd: 'Shortcuts', kbdLine: 'Close with', tip: 'Profile settings',
    controls: 'Select, filters and dates', search: 'Search profile', chosen: 'Chosen profiles', disconnect: 'Disconnect office', level: 'Level', levelAll: 'All', profileF: 'Profile', profileAll: 'All profiles',
    dates: 'Dates', reconnect: 'Reconnect', addSite: 'Send a site through office', remove: 'Delete',
    edge: 'Tooltip at the edge: it shifts to stay on screen.', checks: 'Check boxes and radios', check: 'Include subdomains', r1: 'Use the normal internet', r2: 'Block',
  },
} as const

interface Row {
  name: string
  state: 'ok' | 'connecting' | 'caution' | 'bad' | 'idle'
  mb: number
}

export default function KitPage() {
  const { lang } = usePreferences()
  const x = TEXT[lang]
  const [ks, setKs] = useState(true)
  const [dns, setDns] = useState(false)
  const [dnsPending, setDnsPending] = useState(false)
  const [dialog, setDialog] = useState(false)
  const [sheet, setSheet] = useState(false)
  const [tab, setTab] = useState('a')
  const [type, setType] = useState('wireguard')
  const [level, setLevel] = useState('')
  const [profiles, setProfiles] = useState<string[]>(['oficina'])
  const [range, setRange] = useState({ from: '', to: '' })
  const [sub, setSub] = useState(true)
  const [down, setDown] = useState('allow')
  const [rows, setRows] = useState<Row[]>([
    { name: 'oficina', state: 'ok', mb: 412.4 },
    { name: 'casa-wg', state: 'connecting', mb: 0 },
    { name: 'proton', state: 'caution', mb: 88.1 },
    { name: 'cliente-a', state: 'bad', mb: 3.2 },
  ])
  const label = { ok: x.connected, connecting: x.connecting, caution: x.noResp, bad: x.error, idle: x.off }
  const columns: Column<Row>[] = [
    { id: 'name', header: x.name, cell: (r) => <span className="mono">{r.name}</span> },
    { id: 'state', header: x.state, cell: (r) => <StatusPill tone={r.state}>{label[r.state]}</StatusPill> },
    { id: 'mb', header: x.traffic, numeric: true, cell: (r) => `${r.mb.toFixed(1)} MB` },
  ]

  // Optimistic switch that the "daemon" refuses: flips, then rolls back + toast.
  const toggleDns = (next: boolean) => {
    setDns(next)
    setDnsPending(true)
    setTimeout(() => {
      setDns(!next)
      setDnsPending(false)
      toasts.show({ tone: 'bad', message: x.dns, detail: x.cidrFix })
    }, 900)
  }

  return (
    <>
      <PageHeader title={x.title} lead={x.lead} />
      <div className="kit-grid">
        <Card title={x.buttons}>
          <div className="kit-row">
            <Button icon={<PlusIcon />}>{x.connect}</Button>
            <Button variant="tonal">{x.details}</Button>
            <Button variant="outlined">{x.cancel}</Button>
            <Button variant="text">{x.details}</Button>
            <Button variant="danger">{x.del}</Button>
          </div>
          <div className="kit-row">
            <Button loading>{x.saving}</Button>
            <Button disabled>{x.connect}</Button>
            <Button size="sm" variant="outlined" icon={<RefreshIcon />}>
              {x.details}
            </Button>
            <IconButton label={x.tip} icon={<SettingsIcon />} />
            <IconButton label={x.tip} icon={<SettingsIcon />} variant="outlined" />
          </div>
        </Card>

        <Card title={x.states}>
          <div className="kit-row">
            <StatusPill tone="ok">{x.connected}</StatusPill>
            <StatusPill tone="connecting">{x.connecting}</StatusPill>
            <StatusPill tone="caution">{x.noResp}</StatusPill>
            <StatusPill tone="bad">{x.error}</StatusPill>
            <StatusPill tone="idle">{x.off}</StatusPill>
            <StatusPill tone="unknown">{x.unknown}</StatusPill>
            <StatusPill tone="on">{x.on}</StatusPill>
          </div>
        </Card>

        <Card title={x.switches}>
          <Switch checked={ks} onChange={setKs} label={x.ks} description={x.ksDesc} onText={x.onT} offText={x.offT} />
          <Switch
            checked={dns}
            onChange={toggleDns}
            pending={dnsPending}
            label={x.dns}
            description={x.dnsDesc}
            onText={x.onT}
            offText={x.offT}
          />
        </Card>

        <Card
          title={x.list}
          aside={
            <Button
              size="sm"
              variant="outlined"
              onClick={() => setRows((rs) => rs.map((r, i) => (i === 0 ? { ...r, mb: r.mb + 1.3 } : r)))}
            >
              {x.tick}
            </Button>
          }
        >
          <DataList caption={x.list} columns={columns} rows={rows} rowKey={(r) => r.name} />
        </Card>

        <Card title={x.fields}>
          <TextField label={x.profile} hint={x.profileHint} defaultValue="oficina" required />
          <TextField label={x.cidr} mono defaultValue="10.0.0/8" error={{ message: x.cidrErr, fix: x.cidrFix }} />
          <SelectField
            label={x.type}
            value={type}
            onChange={setType}
            options={[
              { value: 'wireguard', label: 'WireGuard', group: 'Directa' },
              { value: 'openvpn', label: 'OpenVPN', group: 'Directa' },
              { value: 'nordvpn', label: 'NordVPN', group: 'App', hint: 'Usa la app oficial' },
            ]}
          />
          <TextAreaField label={x.notes} rows={2} />
        </Card>

        <Card title={x.controls}>
          <SelectField
            label={x.level}
            value={level}
            onChange={setLevel}
            options={[{ value: '', label: x.levelAll }, ...['INFO', 'WARN', 'ERROR', 'SECURITY'].map((l) => ({ value: l, label: l }))]}
          />
          <MultiSelectField
            label={x.profileF}
            mono
            value={profiles}
            onChange={setProfiles}
            placeholder={x.profileAll}
            searchLabel={x.search}
            chipsLabel={x.chosen}
            options={['oficina', 'casa-wg', 'proton', 'cliente-a', 'cliente-b', 'laboratorio'].map((p) => ({ value: p, label: p }))}
          />
          <DateRangeField label={x.dates} value={range} onChange={setRange} />
          <div className="kit-row">
            <IconButton size="sm" variant="filled" label={x.connect} icon={<PlugIcon />} />
            <IconButton size="sm" variant="strong" label={x.disconnect} icon={<PowerIcon />} />
            <IconButton size="sm" variant="filled" label={x.connect} icon={<PlugIcon />} loading />
            <IconButton size="sm" variant="outlined" label={x.reconnect} icon={<RefreshIcon />} />
            <IconButton size="sm" variant="outlined" label={x.addSite} icon={<PlusIcon />} />
            <IconButton size="sm" label={x.remove} icon={<TrashIcon />} className="kp-danger-icon" />
            <span className="kit-edge">
              <IconButton label={x.edge} icon={<SettingsIcon />} />
            </span>
          </div>
        </Card>

        <Card title={x.checks}>
          <label className="kit-check">
            <input type="checkbox" checked={sub} onChange={(e) => setSub(e.target.checked)} />
            <span>{x.check}</span>
          </label>
          <fieldset className="kit-radios">
            {(['allow', 'block'] as const).map((v) => (
              <label key={v} className="kit-check">
                <input type="radio" name="kit-down" checked={down === v} onChange={() => setDown(v)} />
                <span>{v === 'allow' ? x.r1 : x.r2}</span>
              </label>
            ))}
          </fieldset>
        </Card>

        <Card title={x.feedback}>
          <div className="kit-row">
            <Button variant="outlined" onClick={() => toasts.show({ tone: 'ok', message: x.toastOk })}>
              {x.toastOk}
            </Button>
            <Button
              variant="outlined"
              onClick={() => toasts.show({ tone: 'bad', message: x.toastErr, detail: x.cidrFix, action: { label: x.connect, run: () => {} } })}
            >
              {x.toastErr}
            </Button>
            <Button variant="danger" onClick={() => setDialog(true)}>
              {x.dialog}
            </Button>
            <Button variant="tonal" onClick={() => setSheet(true)}>
              {x.sheet}
            </Button>
          </div>
          <Banner tone="caution" title={x.bannerT} actions={<Button size="sm">{x.install}</Button>}>
            <p>{x.bannerB}</p>
          </Banner>
          <Banner tone="info" title={x.infoT}>
            <p>{x.infoB}</p>
          </Banner>
          <ErrorState error={new ApiError('offline', 0)} onRetry={() => {}} />
          <Skeleton label={x.loading} />
          <EmptyState title={x.emptyT} body={x.emptyB} action={<Button icon={<PlusIcon />}>{x.newRule}</Button>} />
        </Card>

        <Card title={x.tabs}>
          <Tabs
            label={x.tabs}
            value={tab}
            onChange={setTab}
            items={[
              { id: 'a', label: x.t1, panel: <p>{x.p1}</p> },
              { id: 'b', label: x.t2, panel: <p>{x.p2}</p> },
            ]}
          />
          <p>
            {x.kbdLine} <Kbd>Esc</Kbd> ·{' '}
            <Tooltip content={x.tip} describe>
              <a href="#kit">{x.kbd}</a>
            </Tooltip>
          </p>
        </Card>
      </div>

      <Dialog
        open={dialog}
        onClose={() => setDialog(false)}
        title={x.dTitle}
        description={x.dBody}
        alert
        actions={
          <>
            <Button variant="outlined" onClick={() => setDialog(false)}>
              {x.cancel}
            </Button>
            <Button variant="danger" onClick={() => setDialog(false)}>
              {x.del}
            </Button>
          </>
        }
      />
      <Dialog open={sheet} onClose={() => setSheet(false)} title={x.sTitle} variant="sheet">
        <p>{x.p2}</p>
      </Dialog>
    </>
  )
}
