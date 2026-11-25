// Kongtrol UI tests — "Importar un archivo de VPN": found original, folder pick, missing-files layout (ADR-0112), abort on close, "Guardar" (ADR-0115), automatic check (ADR-0118) · v1.2.0 · 2026-11-25
import { fireEvent, render, screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { App } from '../../App'
import { RESTART_TIMING, resetRestartPhase, type VPNProfile } from '../../data/profiles'
import { FakeSocket, helperReady, isImportCheck, json, passedCheck, stubFetch, type Routes } from '../../test/fakes'

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

/** The automatic check passed (ADR-0118): the save buttons are on. */
const ready = (sheet: HTMLElement) => within(sheet).findByText(/^Ready: /)

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

/** Bodies sent to that route; the automatic checks (ADR-0118) only with `checks`. */
function bodies(fetchMock: ReturnType<typeof stubFetch>, key: string, checks = false): Record<string, unknown>[] {
  return fetchMock.mock.calls
    .filter(([url, init]) => `${init?.method ?? 'GET'} ${String(url)}` === key && isImportCheck(init) === checks)
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
    await ready(sheet)
    await userEvent.click(within(sheet).getByRole('button', { name: 'Save and activate' }))
    expect(await within(sheet).findByText('Found the certificates next to rluna.conf in C:\\Users\\ana\\OpenVPN\\config')).toBeInTheDocument()
    expect(bodies(fetchMock, 'POST /api/v1/vpns/import')[0]).toMatchObject({ file_name: 'rluna.conf' })
  })

  it('missing files: short title, names as chips, buttons stacked under the text', async () => {
    const { sheet } = await openFileStep({ 'POST /api/v1/vpns/import?check': () => json(missingAll, 400) })
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

  it('"Choose the folder they are in" opens the OS folder dialog, checks again with ref_dir, then saves with it', async () => {
    const { sheet, fetchMock } = await openFileStep({
      'GET /api/v1/vpns/import/pick': () => json({ available: true }),
      'POST /api/v1/vpns/import/pick-folder': () => json({ path: 'D:\\certs', name: 'certs' }),
      'POST /api/v1/vpns/import?check': (init) =>
        (JSON.parse(String(init?.body)) as { ref_dir?: string }).ref_dir ? json(passedCheck(init)) : json(missingAll, 400),
      'POST /api/v1/vpns/import': () => json({ status: 'imported', path: '/k/rluna.ovpn', type: 'openvpn', overwritten: false }, 201),
      'POST /api/v1/vpns': () => json({ status: 'created', profile: 'rluna', restart_required: false }, 201),
    })
    await waitFor(() => expect(fetchMock.mock.calls.some(([url]) => String(url) === '/api/v1/vpns/import/pick')).toBe(true))
    await within(sheet).findByText('This .ovpn needs 4 more files')
    const banner = sheet.querySelector<HTMLElement>('.k-add-missing')!
    expect(within(banner).getByText('Choose the folder they are in, or add them one by one.')).toBeInTheDocument()
    await userEvent.click(within(banner).getByRole('button', { name: 'Choose the folder they are in' }))
    await ready(sheet)
    expect(bodies(fetchMock, 'POST /api/v1/vpns/import', true).at(-1)).toMatchObject({ ref_dir: 'D:\\certs', check: true })
    // "Save" (ADR-0115): nothing connects.
    await userEvent.click(within(sheet).getByRole('button', { name: 'Save' }))
    expect(await within(sheet).findByText('“rluna” is ready.')).toBeInTheDocument()
    const sent = bodies(fetchMock, 'POST /api/v1/vpns/import')
    expect(sent).toHaveLength(1)
    expect(sent[0]).toMatchObject({ ref_dir: 'D:\\certs', file_name: 'rluna.conf' })
    expect(fetchMock.mock.calls.some(([url]) => String(url).endsWith('/connect'))).toBe(false)
  })
})

describe('B: the file is checked as soon as it is chosen (ADR-0118)', () => {
  it('shows "Checking the file…" in the result area while the daemon checks, and keeps both save buttons off', async () => {
    let answer: (r: Response) => void = () => {}
    const { sheet, fetchMock } = await openFileStep({
      'POST /api/v1/vpns/import?check': () => new Promise<Response>((resolve) => (answer = resolve)),
    })
    const area = sheet.querySelector<HTMLElement>('.k-add-file__result')!
    expect(await within(area).findByText('Checking the file…')).toBeInTheDocument()
    expect(area).toHaveAttribute('aria-busy', 'true')
    expect(area).toHaveAttribute('aria-live', 'polite')
    expect(within(sheet).getByRole('button', { name: 'Save' })).toBeDisabled()
    expect(within(sheet).getByRole('button', { name: 'Save and activate' })).toBeDisabled()
    expect(within(sheet).getByText('The save buttons turn on when the check is done.')).toBeInTheDocument()
    // The check is the same import, marked as a check, with the dropped file's own name.
    expect(bodies(fetchMock, 'POST /api/v1/vpns/import', true)[0]).toMatchObject({ check: true, file_name: 'rluna.conf', type: 'openvpn' })
    answer(new Response(JSON.stringify({ status: 'checked', type: 'openvpn', file: 'rluna.ovpn', exists: false, included: [], missing: [], needs_credentials: false })))
    await ready(sheet)
    expect(area).not.toHaveAttribute('aria-busy')
    expect(within(sheet).getByRole('button', { name: 'Save' })).toBeEnabled()
  })

  it('a passed check says what saving will do: files inside, where they were found, a name already taken, credentials', async () => {
    const { sheet, fetchMock } = await openFileStep({
      'POST /api/v1/vpns/import?check': (init) =>
        json(passedCheck(init, { included: ['ca.crt', 'ta.key'], found_in: 'C:\\Users\\ana\\OpenVPN\\config', exists: true, needs_credentials: true })),
    })
    expect(await within(sheet).findByText('Ready: OpenVPN. You can save it now.')).toBeInTheDocument()
    expect(within(sheet).getByText('They will go inside the profile: ca.crt, ta.key')).toBeInTheDocument()
    expect(within(sheet).getByText('Found the certificates next to rluna.conf in C:\\Users\\ana\\OpenVPN\\config')).toBeInTheDocument()
    expect(within(sheet).getByText('A file is already saved as rluna.ovpn. When you save you can replace it, or change the name.')).toBeInTheDocument()
    expect(within(sheet).getByText('This file asks for a username and password.')).toBeInTheDocument()
    expect(within(sheet).getByRole('textbox', { name: 'Username' })).toBeInTheDocument()
    expect(within(sheet).getByRole('button', { name: 'Save and activate' })).toBeEnabled()
    // Nothing was imported: only checks went out.
    expect(bodies(fetchMock, 'POST /api/v1/vpns/import')).toHaveLength(0)
  })

  it('a refused file shows why and what to do at once, and saving stays off', async () => {
    const { sheet } = await openFileStep({
      'POST /api/v1/vpns/import?check': () =>
        json({ error: 'the file runs commands (up, line 3)', hint: 'Delete that line and choose the file again.', error_key: 'api.err.import_directive' }, 400),
    })
    expect(await within(sheet).findByText('the file runs commands (up, line 3)')).toBeInTheDocument()
    expect(within(sheet).getByText('Delete that line and choose the file again.')).toBeInTheDocument()
    expect(within(sheet).getByRole('button', { name: 'Save' })).toBeDisabled()
    expect(within(sheet).getByRole('button', { name: 'Save and activate' })).toBeDisabled()
    expect(within(sheet).getByText('Solve what is shown above to save.')).toBeInTheDocument()
  })

  it('a newer file aborts the check still running for the old one; only the newest answer counts', async () => {
    const signals: (AbortSignal | null | undefined)[] = []
    const { sheet } = await openFileStep({
      'POST /api/v1/vpns/import?check': (init) => {
        signals.push(init?.signal)
        if (signals.length === 1) {
          return new Promise<Response>((_, reject) => {
            init?.signal?.addEventListener('abort', () => reject(new DOMException('aborted', 'AbortError')))
          })
        }
        return json(passedCheck(init))
      },
    })
    await waitFor(() => expect(signals).toHaveLength(1))
    const alerts = screen.queryAllByRole('alert').length
    const zone = sheet.querySelector<HTMLElement>('.k-add-drop')!
    fireEvent.drop(zone, { dataTransfer: { files: [new File(['[Interface]\nPrivateKey = a\n'], 'home.conf')], types: ['Files'] } })
    await within(sheet).findByText('home.conf · WireGuard')
    await waitFor(() => expect(signals[0]?.aborted).toBe(true))
    expect(await within(sheet).findByText('Ready: WireGuard. You can save it now.')).toBeInTheDocument()
    // The abort is not an error.
    expect(screen.queryAllByRole('alert').length).toBeLessThanOrEqual(alerts)
  })

  it('closing the sheet aborts a running check', async () => {
    let signal: AbortSignal | null | undefined
    const { sheet } = await openFileStep({
      'POST /api/v1/vpns/import?check': (init) => {
        signal = init?.signal
        return new Promise<Response>((_, reject) => {
          init?.signal?.addEventListener('abort', () => reject(new DOMException('aborted', 'AbortError')))
        })
      },
    })
    await waitFor(() => expect(signal).toBeDefined())
    await userEvent.click(within(sheet).getByRole('button', { name: 'Close' }))
    // The draft guard asks first: discard it.
    const discard = await screen.findByRole('button', { name: 'Discard' }).catch(() => null)
    if (discard) await userEvent.click(discard)
    await waitFor(() => expect(signal?.aborted).toBe(true))
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
