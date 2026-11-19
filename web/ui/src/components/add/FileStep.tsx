// Kongtrol UI — B: "Importar un archivo de VPN" · v1.0.0 · 2026-11-19
//
// The type comes from the extension (profileimport.TypeForFile), the name
// from the file name (cleaned to a valid, unused name). Saving stores the
// file with POST /api/v1/vpns/import (checked with the helper's own rules)
// and then creates the profile with POST /api/v1/vpns, then restarts. An
// .ovpn with `auth-user-pass` shows user and password at once, with that
// sign-in preselected (a client-side hint; OpenVPN stays the authority).
// The stored path is a detail: only under "Más opciones" and in Done.
import { useEffect, useMemo, useRef, useState, type FormEvent } from 'react'
import { ApiError } from '../../api/client'
import { describeError } from '../../api/errors'
import { usePreferences } from '../../app/preferences-context'
import { useImportProfileFile, type VPNProfile } from '../../data/profiles'
import type { MessageKey } from '../../i18n/messages'
import { Button } from '../ui/Button'
import { SelectField, TextField, type FieldError } from '../ui/Field'
import { Banner } from '../ui/States'
import { AUTH_METHODS, specOf, type AuthMethod } from '../profiles/catalog'
import { emptyDraft, fieldForErrorKey, toRequest } from '../profiles/draft'
import { localizeError } from '../profiles/errors'
import { UploadIcon } from '../profiles/icons'
import { MAX_IMPORT_BYTES, importName, toBase64 } from '../profiles/importModel'
import { useSaveAndActivate } from './activate'
import { isValidName, ovpnNeedsCredentials, profileNameFromFile, typeForFile } from './model'
import { ActivateFooter, ErrorBanner } from './parts'

interface Props {
  profiles: readonly VPNProfile[]
  /** A file dropped on the window or handed over by the caller. */
  incoming: File | null
  /** Shown on top when the user came from the type picker ("WireGuard se agrega con su archivo"). */
  note?: string | undefined
  onBack: () => void
  onDone: (name: string, path: string) => void
  onDirty: (dirty: boolean) => void
}

interface Chosen {
  file: File
  type: 'wireguard' | 'openvpn'
  content: string
  needsCreds: boolean
}

const AUTH_LABEL: Record<AuthMethod, MessageKey> = {
  certificate: 'profiles.auth.certificate',
  credentials: 'profiles.auth.credentials',
  'certificate+credentials': 'profiles.auth.both',
}

export function FileStep({ profiles, incoming, note, onBack, onDone, onDirty }: Props) {
  const { t } = usePreferences()
  const names = useMemo(() => profiles.map((p) => p.name), [profiles])
  const input = useRef<HTMLInputElement>(null)
  const [chosen, setChosen] = useState<Chosen | null>(null)
  const [fileError, setFileError] = useState<FieldError | null>(null)
  const [name, setName] = useState('')
  const [authMethod, setAuthMethod] = useState<AuthMethod>('certificate')
  const [username, setUsername] = useState('')
  const [password, setPassword] = useState('')
  const [tried, setTried] = useState(false)
  const [nameError, setNameError] = useState<FieldError | null>(null)
  const [refusal, setRefusal] = useState<{ error: unknown; exists: boolean } | null>(null)
  const [stored, setStored] = useState<{ as: string; path: string } | null>(null)
  const [drag, setDrag] = useState(false)
  const importFile = useImportProfileFile()
  const act = useSaveAndActivate()

  useEffect(() => onDirty(chosen !== null), [chosen, onDirty])

  const take = async (file: File) => {
    setRefusal(null)
    setStored(null)
    const type = typeForFile(file.name)
    if (!type) {
      setFileError({ message: t('add.file.wrongExt', { file: file.name }), fix: t('add.file.wrongExt.fix') })
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
    const needsCreds = type === 'openvpn' && ovpnNeedsCredentials(new TextDecoder().decode(bytes))
    setFileError(null)
    setChosen({ file, type, content: toBase64(bytes), needsCreds })
    setName(profileNameFromFile(file.name, names))
    setAuthMethod(needsCreds ? 'credentials' : specOf(type).defaultAuth)
  }

  // A file dropped on the window while this step is open (or that opened it).
  useEffect(() => {
    if (incoming) void take(incoming)
    // take is recreated each render; only a new file matters here.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [incoming])

  const pick = (files: FileList | null) => {
    const file = files?.[0]
    if (file) void take(file)
    if (input.current) input.current.value = ''
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

  const submit = async (e: Pick<FormEvent, 'preventDefault'>, overwrite = false) => {
    e.preventDefault()
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
    const as = importName(n, chosen.file.name)
    let path = stored?.as === as ? stored.path : ''
    if (!path) {
      try {
        const res = await importFile.mutateAsync({ name: as, type: chosen.type, content_base64: chosen.content, overwrite })
        path = res.path
        setStored({ as, path })
      } catch (err) {
        setRefusal({ error: err, exists: err instanceof ApiError && err.key === 'api.err.import_exists' })
        return
      }
    }
    const draft = { ...emptyDraft(chosen.type), name: n, config: path, authMethod, username, password }
    try {
      const ok = await act.run(toRequest(draft, 'create'))
      onDirty(false)
      if (ok) onDone(n, path)
    } catch (err) {
      const local = localizeError(err, t)
      if (local instanceof ApiError && fieldForErrorKey(local.key) === 'name') {
        setNameError(describeError(local, t))
      } else setRefusal({ error: local, exists: false })
    }
  }

  const showCreds = chosen?.type === 'openvpn' && authMethod !== 'certificate'

  return (
    <form className="k-add__step" onSubmit={(e) => void submit(e)} noValidate>
      {note && <p className="k-add__muted">{note}</p>}
      <div
        className="k-add-drop"
        data-drag={drag || undefined}
        onDragOver={(e) => {
          e.preventDefault()
          setDrag(true)
        }}
        onDragLeave={() => setDrag(false)}
        onDrop={(e) => {
          e.preventDefault()
          setDrag(false)
          pick(e.dataTransfer.files)
        }}
      >
        <input
          ref={input}
          type="file"
          accept=".conf,.ovpn"
          className="k-add-drop__input"
          tabIndex={-1}
          aria-hidden="true"
          onChange={(e) => pick(e.target.files)}
        />
        <Button variant={chosen ? 'outlined' : 'filled'} icon={<UploadIcon />} onClick={() => input.current?.click()} data-autofocus="">
          {chosen ? t('add.file.change') : t('add.file.choose')}
        </Button>
        <p className="k-add__muted k-add__small">{drag ? t('add.file.drop') : t('add.file.what')}</p>
        <div aria-live="polite">
          {chosen && (
            <p className="k-add-drop__chosen">
              <span className="mono">{t('add.file.chosen', { file: chosen.file.name, type: specOf(chosen.type).label })}</span>
            </p>
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
                <SelectField label={t('profiles.field.authMethod')} value={authMethod} onChange={(e) => setAuthMethod(e.target.value as AuthMethod)}>
                  {AUTH_METHODS.map((m) => (
                    <option key={m} value={m}>
                      {t(AUTH_LABEL[m])}
                    </option>
                  ))}
                </SelectField>
              )}
              <p className="k-add__muted k-add__small">{t('add.file.where')}</p>
              {stored && (
                <p className="k-add__small">
                  {t('add.file.stored')} <code className="mono">{stored.path}</code>
                </p>
              )}
            </div>
          </details>
        </>
      )}

      <ActivateFooter
        phase={act.phase}
        onBack={onBack}
        onRetry={() =>
          void act.retryRestart().then((ok) => {
            if (ok && stored) onDone(name.trim(), stored.path)
          })
        }
      />
    </form>
  )
}
