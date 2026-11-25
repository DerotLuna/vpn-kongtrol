// Kongtrol UI tests — "Importar un archivo de VPN": found original, folder pick, missing-files layout (ADR-0112), abort on close, "Guardar" (ADR-0115) · v1.1.0 · 2026-11-25
import { fireEvent, render, screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { App } from '../../App'
import { RESTART_TIMING, resetRestartPhase, type VPNProfile } from '../../data/profiles'
import { FakeSocket, helperReady, json, stubFetch, type Routes } from '../../test/fakes'

const work: VPNProfile = {
  name: 'work',
  type: 'nordvpn',
  priority: 10,
  failover: [],
  has_username_credential: false,
  has_password_credential: false,
  kill_switch_override: 'inherit',
  auto_connect: true,
}

const ovpn = 'client\nremote vpn.acme.test 1194\nca ca.crt\ncert rluna.crt\nkey rluna.key\ntls-crypt ta.key\n'
const missingAll = {
  error: 'files this .ovpn needs are missing: ca.crt, rluna.crt, rluna.key, ta.key',
  hint: 'Add them.',
  error_key: 'api.err.import_missing_files',
  missing: ['ca.crt', 'rluna.crt', 'rluna.key', 'ta.key'],
}

const timing = { ...RESTART_TIMING }

beforeEach(() => {
  Object.defineProperty(navigator, 'languages', { value: ['en-US'], configurable: true })
  FakeSocket.reset()
  vi.stubGlobal('WebSocket', FakeSocket)
  Object.assign(RESTART_TIMING, { stepMs: 1, stopSteps: 5, startSteps: 5 })
})

afterEach(() => {
  Object.assign(RESTART_TIMING, timing)
  resetRestartPhase()
})

function bodies(fetchMock: ReturnType<typeof stubFetch>, key: string): Record<string, unknown>[] {
  return fetchMock.mock.calls
    .filter(([url, init]) => `${init?.method ?? 'GET'} ${String(url)}` === key)
    .map(([, init]) => JSON.parse(String(init?.body)) as Record<string, unknown>)
}

async function openFileStep(extra: Routes) {
  window.history.replaceState(null, '', '/?add=file')
  const fetchMock = stubFetch({
    'GET /api/v1/preferences': () => json({ language: 'en', theme: '' }),
    'GET /api/v1/version': () => json({ version: 'v0.8.0' }),
    'GET /api/v1/helper/status': () => json(helperReady),
    'GET /api/v1/tunnels': () => json([]),
    'GET /api/v1/groups': () => json([]),
    'GET /api/v1/policies': () => json([]),
    'GET /api/v1/vpns': () => json([work]),
    'GET /api/v1/vpns/detect': () => json([]),
    'GET /api/v1/setup': () => json({ setup_mode: false, needs_setup: false, restart_pending: false, config_path: '' }),
    ...extra,
  })
  render(<App />)
  const sheet = await screen.findByRole('dialog', { name: 'Import a VPN file' })
  await within(sheet).findByRole('button', { name: 'Choose file' })
  const zone = sheet.querySelector<HTMLElement>('.k-add-drop')!
  fireEvent.drop(zone, { dataTransfer: { files: [new File([ovpn], 'rluna.conf')], types: ['Files'] } })
  await within(sheet).findByText('rluna.conf · OpenVPN')
  return { sheet, fetchMock }
}

describe('B: a dropped .ovpn whose certificates are elsewhere (ADR-0112)', () => {
  it('sends its own name and says where the daemon found the certificates', async () => {
    const { sheet, fetchMock } = await openFileStep({
      'POST /api/v1/vpns/import': () =>
        json({ status: 'imported', path: '/k/rluna.ovpn', type: 'openvpn', overwritten: false, included: ['ca.crt'], found_in: 'C:\\Users\\ana\\OpenVPN\\config' }, 201),
      // Stay on this step: the profile save never answers.
      'POST /api/v1/vpns': () => new Promise<Response>(() => {}),
    })
    // The import sheet is the wide variant (CSS :has(.k-add-file)).
    expect(sheet.querySelector('form.k-add-file')).not.toBeNull()
    await userEvent.click(within(sheet).getByRole('button', { name: 'Save and activate' }))
    expect(await within(sheet).findByText('Found the certificates next to rluna.conf in C:\\Users\\ana\\OpenVPN\\config')).toBeInTheDocument()
    expect(bodies(fetchMock, 'POST /api/v1/vpns/import')[0]).toMatchObject({ file_name: 'rluna.conf' })
  })

  it('missing files: short title, names as chips, buttons stacked under the text', async () => {
    const { sheet } = await openFileStep({ 'POST /api/v1/vpns/import': () => json(missingAll, 400) })
    await userEvent.click(within(sheet).getByRole('button', { name: 'Save and activate' }))
    expect(await within(sheet).findByText('This .ovpn needs 4 more files')).toBeInTheDocument()
    const banner = sheet.querySelector<HTMLElement>('.k-add-missing')!
    const chips = within(within(banner).getByRole('list', { name: 'Missing files' })).getAllByRole('listitem')
    expect(chips.map((c) => c.textContent)).toEqual(['ca.crt', 'rluna.crt', 'rluna.key', 'ta.key'])
    // The actions are their own row after the text (the stacked layout), not a column beside it.
    const actions = banner.querySelector('.k-add-missing__actions')!
    expect(banner.querySelector('.k-banner__text')!.nextElementSibling).toBe(actions)
    expect(within(banner).getByText('Add them one by one or drag them here. They usually come with the .ovpn.')).toBeInTheDocument()
    // No OS dialog in this session: no folder button.
    expect(within(banner).queryByRole('button', { name: 'Choose the folder they are in' })).toBeNull()
    expect(within(banner).getByRole('button', { name: 'Add ta.key' })).toBeInTheDocument()
  })

  it('"Choose the folder they are in" opens the OS folder dialog and imports again with ref_dir', async () => {
    let imports = 0
    const { sheet, fetchMock } = await openFileStep({
      'GET /api/v1/vpns/import/pick': () => json({ available: true }),
      'POST /api/v1/vpns/import/pick-folder': () => json({ path: 'D:\\certs', name: 'certs' }),
      'POST /api/v1/vpns/import': () => {
        imports++
        return imports === 1 ? json(missingAll, 400) : json({ status: 'imported', path: '/k/rluna.ovpn', type: 'openvpn', overwritten: false }, 201)
      },
      'POST /api/v1/vpns': () => json({ status: 'created', profile: 'rluna', restart_required: false }, 201),
    })
    await waitFor(() => expect(fetchMock.mock.calls.some(([url]) => String(url) === '/api/v1/vpns/import/pick')).toBe(true))
    // "Save" (ADR-0115): the import again after the folder pick keeps the choice, so nothing connects.
    await userEvent.click(within(sheet).getByRole('button', { name: 'Save' }))
    await within(sheet).findByText('This .ovpn needs 4 more files')
    const banner = sheet.querySelector<HTMLElement>('.k-add-missing')!
    expect(within(banner).getByText('Choose the folder they are in, or add them one by one.')).toBeInTheDocument()
    await userEvent.click(within(banner).getByRole('button', { name: 'Choose the folder they are in' }))
    expect(await within(sheet).findByText('“rluna” is ready.')).toBeInTheDocument()
    const sent = bodies(fetchMock, 'POST /api/v1/vpns/import')
    expect(sent).toHaveLength(2)
    expect(sent[1]).toMatchObject({ ref_dir: 'D:\\certs', file_name: 'rluna.conf' })
    expect(fetchMock.mock.calls.some(([url]) => String(url).endsWith('/connect'))).toBe(false)
  })
})

describe('B: the native pick ends with the sheet', () => {
  it('closing the sheet while the OS dialog is open aborts the request (the daemon closes the dialog) and shows no error', async () => {
    window.history.replaceState(null, '', '/?add=file')
    let signal: AbortSignal | null | undefined
    stubFetch({
      'GET /api/v1/preferences': () => json({ language: 'en', theme: '' }),
      'GET /api/v1/version': () => json({ version: 'v0.8.0' }),
      'GET /api/v1/helper/status': () => json(helperReady),
      'GET /api/v1/tunnels': () => json([]),
      'GET /api/v1/groups': () => json([]),
      'GET /api/v1/policies': () => json([]),
      'GET /api/v1/vpns': () => json([work]),
      'GET /api/v1/vpns/detect': () => json([]),
      'GET /api/v1/setup': () => json({ setup_mode: false, needs_setup: false, restart_pending: false, config_path: '' }),
      'GET /api/v1/vpns/import/pick': () => json({ available: true }),
      // The user has not picked yet: the answer never comes, unless aborted.
      'POST /api/v1/vpns/import/pick': (init) => {
        signal = init?.signal
        return new Promise<Response>((_, reject) => {
          init?.signal?.addEventListener('abort', () => reject(new DOMException('aborted', 'AbortError')))
        })
      },
    })
    render(<App />)
    const sheet = await screen.findByRole('dialog', { name: 'Import a VPN file' })
    const choose = await within(sheet).findByRole('button', { name: 'Choose file' })
    await waitFor(() => expect(choose).not.toBeDisabled())
    const alerts = screen.queryAllByRole('alert').length
    await userEvent.click(choose)
    await waitFor(() => expect(signal).toBeDefined())
    expect(signal?.aborted).toBe(false)
    await userEvent.click(within(sheet).getByRole('button', { name: 'Close' }))
    await waitFor(() => expect(signal?.aborted).toBe(true))
    expect(screen.queryByRole('dialog', { name: 'Import a VPN file' })).toBeNull()
    // No new error (an abort is a cancel, not "Kongtrol is not answering").
    expect(screen.queryAllByRole('alert').length).toBeLessThanOrEqual(alerts)
  })
})
