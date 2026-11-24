// Kongtrol UI — B: "Importar un archivo de VPN" · v1.2.0 · 2026-11-24
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
// certificates). Whatever is still missing comes back by name and each gets
// an "Agregar <name>" button (or a drop) that imports again with it.
import { useEffect, useMemo, useRef, useState, type FormEvent } from 'react'
import { ApiError } from '../../api/client'
import { describeError } from '../../api/errors'
import { usePreferences } from '../../app/preferences-context'
import { useImportProfileFile, usePickAvailable, usePickProfileFile, type ImportRequest, type VPNProfile } from '../../data/profiles'
import type { MessageKey } from '../../i18n/messages'
import { Button } from '../ui/Button'
import { SelectField, TextField, type FieldError } from '../ui/Field'
import { Banner } from '../ui/States'
import { AUTH_METHODS, specOf, type AuthMethod } from '../profiles/catalog'
import { emptyDraft, fieldForErrorKey, toRequest } from '../profiles/draft'
import { localizeError } from '../profiles/errors'
import { useDropZone } from '../profiles/dropZone'
import { UploadIcon } from '../profiles/icons'
import { MAX_EXTRA_BYTES, MAX_EXTRA_FILES, MAX_IMPORT_BYTES, fromBase64, importName, toBase64 } from '../profiles/importModel'
import { useSaveAndActivate } from './activate'
import { isValidName, ovpnNeedsCredentials, profileNameFromFile, typeForContent, typeForFile } from './model'
import { ActivateFooter, ErrorBanner } from './parts'

interface Props {
  profiles: readonly VPNProfile[]
  /** Files dropped on the window or handed over by the caller (the .conf / .ovpn, maybe with its certificates). */
  incoming: File[] | null
  /** Shown on top when the user came from the type picker ("WireGuard se agrega con su archivo"). */
  note?: string | undefined
  onBack: () => void
  /** provider: whose account issued the file (POST /api/v1/vpns/import, ADR-0103). */
  onDone: (name: string, path: string, reauth: boolean, provider?: string) => void
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
}

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
  const [tried, setTried] = useState(false)
  const [nameError, setNameError] = useState<FieldError | null>(null)
  const [refusal, setRefusal] = useState<{ error: unknown; exists: boolean } | null>(null)
  const [stored, setStored] = useState<Stored | null>(null)
  const dropZone = useDropZone()
  const drag = dropZone.drag
  const [noPicker, setNoPicker] = useState(false)
  const importFile = useImportProfileFile()
  const pickAvailable = usePickAvailable()
  const pickFile = usePickProfileFile()
  const act = useSaveAndActivate()
  const nativePicker = pickAvailable.data === true && !noPicker

  useEffect(() => onDirty(chosen !== null), [chosen, onDirty])

  const select = (c: Omit<Chosen, 'id'>, added: Extra[]) => {
    const next = { ...c, id: ++choiceSeq }
    chosenId.current = next.id
    setRefusal(null)
    setStored(null)
    setMissing([])
    setFileError(null)
    setChosen(next)
    setExtras(added)
    setName(profileNameFromFile(c.fileName, names))
    setAuthMethod(c.needsCreds ? 'credentials' : specOf(c.type).defaultAuth)
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
        if (missing.length > 0) void submit(undefined, false, added)
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
      const res = await pickFile.mutateAsync()
      if (res.cancelled) return
      const text = new TextDecoder().decode(fromBase64(res.content_base64))
      const type = typeForContent(res.name, text) ?? res.type
      select({ fileName: res.name, type, content: res.content_base64, needsCreds: type === 'openvpn' && ovpnNeedsCredentials(text), path: res.path }, [])
    } catch (err) {
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

  const submit = async (e?: Pick<FormEvent, 'preventDefault'>, overwrite = false, files: Extra[] = extras) => {
    e?.preventDefault()
    if (act.busy || importFile.isPending) return
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
      const body: ImportRequest = chosen.path
        ? { name: as, type: chosen.type, path: chosen.path, overwrite }
        : { name: as, type: chosen.type, content_base64: chosen.content, overwrite }
      if (files.length > 0) body.files = files.map((f) => ({ name: f.name, content_base64: f.content }))
      try {
        const res = await importFile.mutateAsync(body)
        // Another file was chosen while this one was being stored: drop this answer.
        if (chosenId.current !== chosen.id) return
        path = res.path
        provider = res.provider
        setMissing([])
        setStored({ for: chosen.id, as, path, provider, included: res.included })
      } catch (err) {
        if (chosenId.current !== chosen.id) return
        const list = err instanceof ApiError && err.key === MISSING_KEY && Array.isArray(err.body.missing) ? (err.body.missing as string[]) : []
        setMissing(list)
        setRefusal({ error: err, exists: err instanceof ApiError && err.key === 'api.err.import_exists' })
        return
      }
    }
    const draft = { ...emptyDraft(chosen.type), name: n, config: path, authMethod, username, password }
    try {
      const outcome = await act.run(toRequest(draft, 'create'))
      onDirty(false)
      if (outcome !== 'failed') onDone(n, path, outcome === 'reauth', provider)
    } catch (err) {
      const local = localizeError(err, t)
      if (local instanceof ApiError && fieldForErrorKey(local.key) === 'name') {
        setNameError(describeError(local, t))
      } else setRefusal({ error: local, exists: false })
    }
  }

  const showCreds = chosen?.type === 'openvpn' && authMethod !== 'certificate'
  const stillMissing = missing.filter((m) => !extras.some((x) => sameName(x.name, m)))

  return (
    <form className="k-add__step" onSubmit={(e) => void submit(e)} noValidate>
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
          {pickFile.isPending ? t('add.file.picking') : drag ? t('add.file.drop') : t('add.file.what')}
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
        </div>
        {fileError && (
          <p className="k-field__error" role="alert">
            <strong>{fileError.message}</strong> <span>{fileError.fix}</span>
          </p>
        )}
      </div>

      {refusal &&
        (refusal.exists ? (
          <Banner
            tone="caution"
            title={describeError(refusal.error, t).message}
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
            <span>{describeError(refusal.error, t).fix}</span>
          </Banner>
        ) : missing.length > 0 ? (
          <Banner
            tone="caution"
            title={describeError(refusal.error, t).message}
            actions={stillMissing.map((m) => (
              <Button key={m} variant="outlined" size="sm" icon={<UploadIcon />} onClick={() => extraInput.current?.click()}>
                {t('add.file.addMissing', { file: m })}
              </Button>
            ))}
          >
            <span>{describeError(refusal.error, t).fix}</span>
          </Banner>
        ) : (
          <ErrorBanner error={refusal.error} />
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
          {chosen.needsCreds && <p className="k-add__lead">{t('add.file.creds')}</p>}
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
        onBack={onBack}
        onRetry={() =>
          void act.retryRestart().then((outcome) => {
            if (outcome !== 'failed' && stored) onDone(name.trim(), stored.path, outcome === 'reauth', stored.provider)
          })
        }
      />
    </form>
  )
}
