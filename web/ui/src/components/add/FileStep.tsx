// Kongtrol UI — B: "Importar un archivo de VPN" · v1.5.0 · 2026-11-25
//
// The type comes from the extension and, for a .conf, from its content
// (profileimport.TypeForContent, ADR-0080: OpenVPN GUI uses .conf), the name
// from the file name (cleaned to a valid, unused name). Saving stores the
// file with POST /api/v1/vpns/import (checked with the helper's own rules)
// and then creates the profile with POST /api/v1/vpns, then restarts. An
// .ovpn with `auth-user-pass` shows user and password at once, with that
// sign-in preselected (a client-side hint; OpenVPN stays the authority).
// The stored path is a detail: only under "Más opciones" and in Done.
//
// Files an .ovpn names (ca ca.crt…, ADR-0105): "Elegir archivo" opens the
// OS's own dialog through the daemon when it can (a real path: the daemon
// reads them from the file's folder); otherwise the browser's input, which
// takes several files at once, like a drop (the .ovpn plus its
// certificates). A dropped file (no path) is matched by the daemon to its
// identical original in the usual folders, and the answer says where the
// certificates were found (`found_in`, ADR-0112). Whatever is still missing
// comes back by name: the banner lists them as chips, with "Elegir la
// carpeta donde están" (the OS's folder dialog, then `ref_dir`) when the
// daemon can show one, and "Agregar <name>" (or a drop) per file; the
// buttons sit under the text so the narrow sheet never squeezes it.
//
// A native pick waits for the user in the OS dialog: closing the sheet (or
// going back) aborts the request, the daemon's request context ends and
// the dialog closes; an abort reads as cancelled, never as an error.
// "Guardar y activar" saves and connects; "Guardar" only saves (ADR-0115).
//
// Automatic check (ADR-0118): every file chosen, dropped or picked, every
// certificate added and every folder picked runs the same import with
// `check: true` (nothing stored): a loader in the result area while it runs
// (same place, no jump), then the answer at once: ready (type, files that go
// inside, where they were found, Proton hint, a name already taken), the
// missing-files banner, or the refusal with what to do. Both save buttons
// stay off, with the reason under them, until the latest check passed. A
// newer check aborts the previous one, and leaving the step aborts it too.
// Saving still runs the real import (the daemon checks again).
import { useEffect, useMemo, useRef, useState, type FormEvent } from 'react'
import { ApiError, isAbort } from '../../api/client'
import { describeError } from '../../api/errors'
import { usePreferences } from '../../app/preferences-context'
import {
  useCheckProfileFile,
  useImportProfileFile,
  usePickAvailable,
  usePickImportFolder,
  usePickProfileFile,
  type ImportCheckResult,
  type ImportRequest,
  type VPNProfile,
} from '../../data/profiles'
import type { MessageKey } from '../../i18n/messages'
import { Button } from '../ui/Button'
import { SelectField, TextField, type FieldError } from '../ui/Field'
import { AlertIcon } from '../ui/icons'
import { Banner } from '../ui/States'
import { AUTH_METHODS, specOf, type AuthMethod } from '../profiles/catalog'
import { emptyDraft, fieldForErrorKey, toRequest } from '../profiles/draft'
import { localizeError } from '../profiles/errors'
import { useDropZone } from '../profiles/dropZone'
import { UploadIcon } from '../profiles/icons'
import type { OnTunnelDown } from '../rules/model'
import { MAX_EXTRA_BYTES, MAX_EXTRA_FILES, MAX_IMPORT_BYTES, fromBase64, importName, toBase64 } from '../profiles/importModel'
import { useSaveAndActivate, type SaveIntent } from './activate'
import { isValidName, ovpnNeedsCredentials, profileNameFromFile, typeForContent, typeForFile } from './model'
import { ActivateFooter, ErrorBanner, ProfileDownField } from './parts'

interface Props {
  profiles: readonly VPNProfile[]
  /** Files dropped on the window or handed over by the caller (the .conf / .ovpn, maybe with its certificates). */
  incoming: File[] | null
  /** Shown on top when the user came from the type picker ("WireGuard se agrega con su archivo"). */
  note?: string | undefined
  onBack: () => void
  /** provider: whose account issued the file (POST /api/v1/vpns/import, ADR-0103). */
  /** `connect`: "Guardar y activar" (true) or "Guardar" (false, ADR-0115). */
  onDone: (name: string, path: string, reauth: boolean, provider: string | undefined, connect: boolean) => void
  onDirty: (dirty: boolean) => void
}

interface Chosen {
  /** Changes with every file chosen: a stored copy belongs to one choice only. */
  id: number
  fileName: string
  type: 'wireguard' | 'openvpn'
  content: string
  needsCreds: boolean
  /** Real path from the OS's own dialog: the daemon reads the files it names from its folder. */
  path?: string | undefined
}

/** A certificate or key added by hand (sent as `files`). */
interface Extra {
  name: string
  content: string
}

interface Stored {
  for: number
  as: string
  path: string
  provider?: string | undefined
  included?: string[] | undefined
  /** Folder of the dropped file's original, where its certificates were read (ADR-0112). */
  found?: string | undefined
}

/** The latest automatic check of one choice (ADR-0118). */
type Check =
  | { for: number; state: 'running' }
  | { for: number; state: 'ok'; result: ImportCheckResult }
  | { for: number; state: 'failed'; error: unknown }

const AUTH_LABEL: Record<AuthMethod, MessageKey> = {
  certificate: 'profiles.auth.certificate',
  credentials: 'profiles.auth.credentials',
  'certificate+credentials': 'profiles.auth.both',
}

const MISSING_KEY = 'api.err.import_missing_files'
const PICKER_UNAVAILABLE = 'api.err.picker_unavailable'

let choiceSeq = 0

const sameName = (a: string, b: string) => a.toLowerCase() === b.toLowerCase()

export function FileStep({ profiles, incoming, note, onBack, onDone, onDirty }: Props) {
  const { t } = usePreferences()
  const names = useMemo(() => profiles.map((p) => p.name), [profiles])
  const input = useRef<HTMLInputElement>(null)
  const extraInput = useRef<HTMLInputElement>(null)
  const [chosen, setChosen] = useState<Chosen | null>(null)
  const chosenId = useRef<number | null>(null)
  const [extras, setExtras] = useState<Extra[]>([])
  const [missing, setMissing] = useState<string[]>([])
  const [fileError, setFileError] = useState<FieldError | null>(null)
  const [name, setName] = useState('')
  const [authMethod, setAuthMethod] = useState<AuthMethod>('certificate')
  const [username, setUsername] = useState('')
  const [password, setPassword] = useState('')
  const [onTunnelDown, setOnTunnelDown] = useState<OnTunnelDown>('allow')
  const [tried, setTried] = useState(false)
  const [nameError, setNameError] = useState<FieldError | null>(null)
  const [refusal, setRefusal] = useState<{ error: unknown; exists: boolean } | null>(null)
  const [stored, setStored] = useState<Stored | null>(null)
  const dropZone = useDropZone()
  const drag = dropZone.drag
  const [noPicker, setNoPicker] = useState(false)
  const importFile = useImportProfileFile()
  const checkFile = useCheckProfileFile()
  const [check, setCheck] = useState<Check | null>(null)
  /** The check in flight: a newer one aborts it, and so does leaving the step. */
  const checking = useRef<AbortController | null>(null)
  const checkSeq = useRef(0)
  useEffect(() => () => checking.current?.abort(), [])
  const pickAvailable = usePickAvailable()
  const pickFile = usePickProfileFile()
  const pickFolder = usePickImportFolder()
  /** Folder picked for the .ovpn's files ("Elegir la carpeta donde están"), for this choice only. */
  const [refDir, setRefDir] = useState<string | undefined>(undefined)
  const act = useSaveAndActivate()
  /** Which button saved: re-submits (a certificate added after a refusal) keep the user's choice. */
  const how = useRef<SaveIntent>('activate')
  /** The native pick in flight: aborted when the step goes away (sheet closed, Volver). */
  const picking = useRef<AbortController | null>(null)
  useEffect(() => () => picking.current?.abort(), [])
  const pickSignal = () => {
    picking.current?.abort()
    picking.current = new AbortController()
    return picking.current.signal
  }
  const nativePicker = pickAvailable.data === true && !noPicker

  useEffect(() => onDirty(chosen !== null), [chosen, onDirty])

  /** The import body for a choice (the check sends the same, plus `check`). */
  const bodyFor = (c: Chosen, as: string, files: Extra[], dir: string | undefined, overwrite: boolean): ImportRequest => {
    const body: ImportRequest = c.path
      ? { name: as, type: c.type, path: c.path, overwrite }
      : { name: as, type: c.type, content_base64: c.content, file_name: c.fileName, overwrite }
    if (files.length > 0) body.files = files.map((f) => ({ name: f.name, content_base64: f.content }))
    if (dir) body.ref_dir = dir
    return body
  }

  /** The automatic check (ADR-0118): the daemon evaluates everything and stores nothing. */
  const runCheck = async (c: Chosen, files: Extra[], dir: string | undefined, profileName: string) => {
    checking.current?.abort()
    const ctl = new AbortController()
    checking.current = ctl
    const seq = ++checkSeq.current
    setRefusal(null)
    setCheck({ for: c.id, state: 'running' })
    try {
      const result = await checkFile.mutateAsync({ body: bodyFor(c, importName(profileName, c.fileName), files, dir, false), signal: ctl.signal })
      if (seq !== checkSeq.current) return
      setMissing([])
      setCheck({ for: c.id, state: 'ok', result })
      if (result.needs_credentials) setAuthMethod((m) => (m === 'certificate' ? 'credentials' : m))
    } catch (err) {
      if (isAbort(err) || seq !== checkSeq.current) return
      setMissing(err instanceof ApiError && err.key === MISSING_KEY && Array.isArray(err.body.missing) ? (err.body.missing as string[]) : [])
      setCheck({ for: c.id, state: 'failed', error: err })
    }
  }

  const select = (c: Omit<Chosen, 'id'>, added: Extra[]) => {
    const next = { ...c, id: ++choiceSeq }
    chosenId.current = next.id
    setRefusal(null)
    setStored(null)
    setMissing([])
    setRefDir(undefined)
    setFileError(null)
    setChosen(next)
    setExtras(added)
    const proposed = profileNameFromFile(c.fileName, names)
    setName(proposed)
    setAuthMethod(c.needsCreds ? 'credentials' : specOf(c.type).defaultAuth)
    void runCheck(next, added, undefined, proposed)
  }

  /** Reads certificates / keys; null (and the reason shown) when one cannot go. */
  const readExtras = async (files: File[], have: Extra[]): Promise<Extra[] | null> => {
    const out = [...have]
    for (const file of files) {
      if (file.size > MAX_EXTRA_BYTES) {
        const vars = { file: file.name, size: Math.ceil(file.size / 1024), max: MAX_EXTRA_BYTES / 1024 }
        setFileError({ message: t('add.file.tooBig', vars), fix: t('add.file.extraTooBig.fix') })
        return null
      }
      let bytes: Uint8Array
      try {
        bytes = new Uint8Array(await file.arrayBuffer())
      } catch {
        setFileError({ message: t('add.file.readFail', { file: file.name }), fix: t('add.file.readFail.fix') })
        return null
      }
      const i = out.findIndex((x) => sameName(x.name, file.name))
      const extra = { name: file.name, content: toBase64(bytes) }
      if (i >= 0) out[i] = extra
      else out.push(extra)
    }
    if (out.length > MAX_EXTRA_FILES) {
      setFileError({ message: t('add.file.tooMany', { max: MAX_EXTRA_FILES }), fix: t('add.file.tooMany.fix') })
      return null
    }
    return out
  }

  /** Files dropped or picked in the browser: the first .conf / .ovpn is the VPN file, the rest its certificates. */
  const take = async (files: File[]) => {
    const file = files.find((f) => typeForFile(f.name))
    const others = files.filter((f) => f !== file)
    if (!file) {
      // Only certificates: they belong to the file already chosen.
      if (chosen && others.length > 0) {
        const added = await readExtras(others, extras)
        if (!added) return
        setFileError(null)
        setExtras(added)
        void runCheck(chosen, added, refDir, name)
        return
      }
      const first = files[0]
      if (first) setFileError({ message: t('add.file.wrongExt', { file: first.name }), fix: t('add.file.wrongExt.fix') })
      return
    }
    if (file.size > MAX_IMPORT_BYTES) {
      const vars = { file: file.name, size: Math.ceil(file.size / 1024), max: MAX_IMPORT_BYTES / 1024 }
      setFileError({ message: t('add.file.tooBig', vars), fix: t('add.file.tooBig.fix') })
      return
    }
    let bytes: Uint8Array
    try {
      bytes = new Uint8Array(await file.arrayBuffer())
    } catch {
      setFileError({ message: t('add.file.readFail', { file: file.name }), fix: t('add.file.readFail.fix') })
      return
    }
    const added = await readExtras(others, [])
    if (!added) return
    const text = new TextDecoder().decode(bytes)
    const type = typeForContent(file.name, text) ?? 'wireguard'
    select({ fileName: file.name, type, content: toBase64(bytes), needsCreds: type === 'openvpn' && ovpnNeedsCredentials(text) }, added)
  }

  // Files dropped on the window while this step is open (or that opened it).
  useEffect(() => {
    if (incoming) void take(incoming)
    // take is recreated each render; only new files matter here.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [incoming])

  const fromInput = (el: HTMLInputElement) => {
    const files = [...(el.files ?? [])]
    el.value = ''
    if (files.length > 0) void take(files)
  }

  /** "Elegir archivo": the OS's own dialog when the daemon can show one, else the browser's input. */
  const choose = async () => {
    if (!nativePicker) {
      input.current?.click()
      return
    }
    try {
      const res = await pickFile.mutateAsync(pickSignal())
      if (res.cancelled) return
      const text = new TextDecoder().decode(fromBase64(res.content_base64))
      const type = typeForContent(res.name, text) ?? res.type
      select({ fileName: res.name, type, content: res.content_base64, needsCreds: type === 'openvpn' && ovpnNeedsCredentials(text), path: res.path }, [])
    } catch (err) {
      if (isAbort(err)) return
      // No dialog here after all: the next click opens the browser's picker.
      if (err instanceof ApiError && err.key === PICKER_UNAVAILABLE) setNoPicker(true)
      setFileError(describeError(err, t))
    }
  }

  const checkName = (): FieldError | null => {
    const n = name.trim()
    if (!n) return { message: t('add.err.name.required'), fix: t('add.err.name.required.fix') }
    if (!isValidName(n)) return { message: t('add.err.name.chars'), fix: t('add.err.name.chars.fix') }
    if (names.some((x) => x.toLowerCase() === n.toLowerCase())) {
      return { message: t('profiles.check.nameTaken', { name: n }), fix: t('profiles.check.nameTaken.fix', { name: n }) }
    }
    return null
  }

  /** "Elegir la carpeta donde están": the OS's folder dialog, then import again reading the files there. */
  const chooseFolder = async () => {
    try {
      const res = await pickFolder.mutateAsync(pickSignal())
      if (res.cancelled) return
      setRefDir(res.path)
      if (chosen) void runCheck(chosen, extras, res.path, name)
    } catch (err) {
      if (isAbort(err)) return
      if (err instanceof ApiError && err.key === PICKER_UNAVAILABLE) setNoPicker(true)
      setFileError(describeError(err, t))
    }
  }

  const submit = async (e?: Pick<FormEvent, 'preventDefault'>, overwrite = false, files: Extra[] = extras, dir: string | undefined = refDir) => {
    e?.preventDefault()
    if (act.busy || importFile.isPending) return
    // Enter in a field submits too: only after the latest check passed.
    if (chosen && !checkPassed) return
    setTried(true)
    if (!chosen) {
      setFileError({ message: t('add.file.required'), fix: t('add.file.required.fix') })
      return
    }
    const problem = checkName()
    setNameError(problem)
    if (problem) return
    setRefusal(null)
    const n = name.trim()
    const as = importName(n, chosen.fileName)
    // A stored copy belongs to this choice and name only: a newer file never reuses it.
    const reuse = stored?.for === chosen.id && stored.as === as ? stored : null
    let path = reuse?.path ?? ''
    let provider = reuse?.provider
    if (!path) {
      const body = bodyFor(chosen, as, files, dir, overwrite)
      try {
        const res = await importFile.mutateAsync(body)
        // Another file was chosen while this one was being stored: drop this answer.
        if (chosenId.current !== chosen.id) return
        path = res.path
        provider = res.provider
        setMissing([])
        setStored({ for: chosen.id, as, path, provider, included: res.included, found: res.found_in })
      } catch (err) {
        if (chosenId.current !== chosen.id) return
        const list = err instanceof ApiError && err.key === MISSING_KEY && Array.isArray(err.body.missing) ? (err.body.missing as string[]) : []
        setMissing(list)
        setRefusal({ error: err, exists: err instanceof ApiError && err.key === 'api.err.import_exists' })
        return
      }
    }
    const draft = { ...emptyDraft(chosen.type), name: n, config: path, authMethod, username, password, onTunnelDown }
    try {
      const outcome = await act.run(toRequest(draft, 'create'), how.current)
      onDirty(false)
      if (outcome !== 'failed') onDone(n, path, outcome === 'reauth', provider, how.current === 'activate')
    } catch (err) {
      const local = localizeError(err, t)
      if (local instanceof ApiError && fieldForErrorKey(local.key) === 'name') {
        setNameError(describeError(local, t))
      } else setRefusal({ error: local, exists: false })
    }
  }

  const showCreds = chosen?.type === 'openvpn' && authMethod !== 'certificate'
  const stillMissing = missing.filter((m) => !extras.some((x) => sameName(x.name, m)))
  const current = chosen && check?.for === chosen.id ? check : null
  const checkRunning = current?.state === 'running'
  const checkOk = current?.state === 'ok' ? current.result : null
  const checkPassed = checkOk !== null
  // What the result area shows: a save refusal first, else a failed check.
  const shown = refusal ?? (current?.state === 'failed' ? { error: current.error, exists: false } : null)
  const needsCreds = (chosen?.needsCreds ?? false) || (checkOk?.needs_credentials ?? false)
  const found = checkOk?.found_in ?? (chosen && stored?.for === chosen.id ? stored.found : undefined)
  // A name already taken, for the name the check used (the user may have changed it since).
  const taken =
    chosen && checkOk?.exists && checkOk.file.replace(/\.[^.]*$/, '') === importName(name.trim(), chosen.fileName) ? checkOk.file : null
  const blocked = !chosen
    ? t('add.file.chooseFirst')
    : checkRunning || !current
      ? t('add.file.waitCheck')
      : checkPassed
        ? null
        : t('add.file.fixFirst')

  return (
    <form
      className="k-add__step k-add-file"
      onSubmit={(e) => {
        how.current = 'activate'
        void submit(e)
      }}
      noValidate
    >
      {note && <p className="k-add__muted">{note}</p>}
      <div
        className="k-add-drop"
        data-drag={drag || undefined}
        {...dropZone.handlers}
        onDrop={(e) => {
          e.preventDefault()
          dropZone.reset()
          void take([...e.dataTransfer.files])
        }}
      >
        <input
          ref={input}
          type="file"
          multiple
          accept=".conf,.ovpn,.crt,.pem,.key,.cer,.p12"
          className="k-add-drop__input"
          tabIndex={-1}
          aria-hidden="true"
          onChange={(e) => fromInput(e.target)}
        />
        <input
          ref={extraInput}
          type="file"
          multiple
          className="k-add-drop__input"
          tabIndex={-1}
          aria-hidden="true"
          onChange={(e) => fromInput(e.target)}
        />
        <Button
          variant={chosen ? 'outlined' : 'tonal'}
          icon={<UploadIcon />}
          loading={pickFile.isPending}
          onClick={() => void choose()}
          data-autofocus=""
        >
          {chosen ? t('add.file.change') : t('add.file.choose')}
        </Button>
        <p className="k-add__muted k-add__small">
          {pickFile.isPending
            ? t('add.file.picking')
            : pickFolder.isPending
              ? t('add.file.pickingFolder')
              : drag
                ? t('add.file.drop')
                : t('add.file.what')}
        </p>
        <div aria-live="polite">
          {chosen && (
            <p className="k-add-drop__chosen">
              <span className="mono">{t('add.file.chosen', { file: chosen.fileName, type: specOf(chosen.type).label })}</span>
            </p>
          )}
          {chosen && extras.length > 0 && (
            <p className="k-add__small">{t('add.file.extras', { files: extras.map((x) => x.name).join(', ') })}</p>
          )}
          {chosen && found && (
            <p className="k-add__small k-add-file__found">
              {t('add.file.found', { file: chosen.fileName, dir: found })}
            </p>
          )}
        </div>
        {fileError && (
          <p className="k-field__error" role="alert">
            <strong>{fileError.message}</strong> <span>{fileError.fix}</span>
          </p>
        )}
      </div>

      {chosen && (
        <div className="k-add-file__result" aria-live="polite" aria-busy={checkRunning || undefined}>
          {checkRunning ? (
            <p className="k-add-file__checking">
              <span className="k-spinner" aria-hidden="true" /> {t('add.file.checking')}
            </p>
          ) : !shown && checkOk ? (
            <div className="k-add-file__ok">
              <p className="k-add-file__ready">
                <strong>{t('add.file.checkOk', { type: specOf(checkOk.type).label })}</strong>
              </p>
              {checkOk.included.length > 0 && <p className="k-add__small">{t('add.file.checkIncluded', { files: checkOk.included.join(', ') })}</p>}
              {taken && <p className="k-add__small">{t('add.file.checkExists', { file: taken })}</p>}
              {checkOk.hint && <p className="k-add__small">{checkOk.hint}</p>}
            </div>
          ) : null}
        </div>
      )}

      {!checkRunning &&
        shown &&
        (shown.exists ? (
          <Banner
            tone="caution"
            title={describeError(shown.error, t).message}
            actions={
              <>
                <Button variant="outlined" size="sm" onClick={() => setRefusal(null)}>
                  {t('profiles.import.keep')}
                </Button>
                <Button variant="danger" size="sm" onClick={(e) => void submit(e, true)}>
                  {t('profiles.import.replace')}
                </Button>
              </>
            }
          >
            <span>{describeError(shown.error, t).fix}</span>
          </Banner>
        ) : stillMissing.length > 0 ? (
          // Not the kit Banner: its actions sit beside the text, which squeezed
          // the text into one word per line in the sheet. Here they wrap below.
          <div className="k-banner k-banner--caution k-add-missing" role="status">
            <span className="k-banner__icon">
              <AlertIcon />
            </span>
            <div className="k-banner__text">
              <strong>
                {stillMissing.length === 1 ? t('add.file.missingTitle.one') : t('add.file.missingTitle', { count: stillMissing.length })}
              </strong>
              <ul className="k-add-missing__chips" aria-label={t('add.file.missingList')}>
                {stillMissing.map((m) => (
                  <li key={m} className="k-add-missing__chip mono">
                    {m}
                  </li>
                ))}
              </ul>
              <span>{nativePicker ? t('add.file.missingWhat') : t('add.file.missingWhat.browser')}</span>
            </div>
            <div className="k-banner__actions k-add-missing__actions">
              {nativePicker && (
                <Button variant="tonal" size="sm" loading={pickFolder.isPending} onClick={() => void chooseFolder()}>
                  {t('add.file.pickFolder')}
                </Button>
              )}
              {stillMissing.map((m) => (
                <Button key={m} variant="outlined" size="sm" icon={<UploadIcon />} onClick={() => extraInput.current?.click()}>
                  {t('add.file.addMissing', { file: m })}
                </Button>
              ))}
            </div>
          </div>
        ) : (
          <ErrorBanner error={shown.error} />
        ))}

      {chosen && (
        <>
          <TextField
            label={t('add.field.name')}
            hint={t('add.field.name.hint')}
            required
            autoComplete="off"
            spellCheck={false}
            value={name}
            onChange={(e) => {
              setName(e.target.value)
              if (tried) setNameError(null)
            }}
            error={nameError}
          />
          <ProfileDownField value={onTunnelDown} onChange={setOnTunnelDown} />
          {needsCreds && <p className="k-add__lead">{t('add.file.creds')}</p>}
          {showCreds && (
            <>
              <TextField label={t('profiles.field.username')} autoComplete="off" spellCheck={false} value={username} onChange={(e) => setUsername(e.target.value)} />
              <TextField
                label={t('profiles.field.password')}
                hint={t('profiles.hint.password')}
                type="password"
                autoComplete="new-password"
                value={password}
                onChange={(e) => setPassword(e.target.value)}
              />
            </>
          )}
          <details className="k-add-more">
            <summary>{t('add.more')}</summary>
            <div className="k-add-more__body">
              {chosen.type === 'openvpn' && (
                <SelectField
                  label={t('profiles.field.authMethod')}
                  value={authMethod}
                  onChange={setAuthMethod}
                  options={AUTH_METHODS.map((m) => ({ value: m, label: t(AUTH_LABEL[m]) }))}
                />
              )}
              <p className="k-add__muted k-add__small">{t('add.file.where')}</p>
              {stored && (
                <p className="k-add__small">
                  {t('add.file.stored')} <code className="mono">{stored.path}</code>
                </p>
              )}
              {stored?.included && stored.included.length > 0 && (
                <p className="k-add__small">{t('add.file.included', { files: stored.included.join(', ') })}</p>
              )}
            </div>
          </details>
        </>
      )}

      <ActivateFooter
        phase={act.phase}
        intent={act.intent}
        blocked={blocked}
        onSaveOnly={() => {
          how.current = 'save'
          void submit()
        }}
        onBack={onBack}
        onRetry={() =>
          void act.retryRestart().then((outcome) => {
            if (outcome !== 'failed' && stored) onDone(name.trim(), stored.path, outcome === 'reauth', stored.provider, how.current === 'activate')
          })
        }
      />
    </form>
  )
}
