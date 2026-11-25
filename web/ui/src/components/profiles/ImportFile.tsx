// Kongtrol UI — "Importar archivo" for WireGuard / OpenVPN profiles · v1.3.0 · 2026-11-24
//
// feature:profile_import
// The non-technical path to a .conf / .ovpn: pick (or drop) the file the
// provider gave you; the daemon checks it with the privileged helper's own
// rules and stores it in ~/.kongtrol/configs/ (POST /api/v1/vpns/import,
// ADR-0042); the form's path field is filled with where it landed. Typing
// a path by hand stays available for advanced users (the field above).
//
// Checked here first, with the same limits, so nothing large travels: the
// extension (.conf for WireGuard, .ovpn for OpenVPN) and the size (the
// helper's 512 KiB). A name already taken (409 api.err.import_exists) is
// answered inline with "Reemplazar" / "Conservar el que hay" — a confirm
// in place, never a dialog over the profile sheet. Every refusal reads as
// what happened + what to do (the daemon translates api.err.import_*).
// Drag and drop is an extra: the button is the accessible path. A dropped
// .ovpn sends its own name (`file_name`) so the daemon can find its
// identical original and read the certificates next to it (ADR-0112).
import { useId, useRef, useState, type DragEvent } from 'react'
import { ApiError } from '../../api/client'
import { describeError, type ErrorText } from '../../api/errors'
import { usePreferences } from '../../app/preferences-context'
import { useImportProfileFile, type ImportResult } from '../../data/profiles'
import { Button } from '../ui/Button'
import { Banner } from '../ui/States'
import { useDropZone } from './dropZone'
import { UploadIcon } from './icons'
import { IMPORT_EXT as EXT, MAX_EXTRA_BYTES, MAX_IMPORT_BYTES, importName, toBase64 } from './importModel'

interface Props {
  type: 'wireguard' | 'openvpn'
  /** The profile name typed so far; the stored file is named after it. */
  profileName: string
  onImported: (path: string) => void
  /**
   * A path typed in the field that only the administrator can read (the
   * save answered config.err.file_root_only, e.g. /etc/wireguard/wg0.conf):
   * offer to copy it with the OS permission window (ADR-0058).
   */
  protectedPath?: string | undefined
}

type Phase =
  | { kind: 'idle' }
  | { kind: 'done'; result: ImportResult; file?: string }
  | { kind: 'exists'; file: File; extras: File[]; error: ErrorText }
  | { kind: 'error'; error: ErrorText }

export function ImportFile({ type, profileName, onImported, protectedPath }: Props) {
  const { t } = usePreferences()
  const save = useImportProfileFile()
  const input = useRef<HTMLInputElement>(null)
  const id = useId()
  const [phase, setPhase] = useState<Phase>({ kind: 'idle' })
  const dropZone = useDropZone()
  const drag = dropZone.drag
  const ext = EXT[type]

  // extras: certificates / keys dropped or picked with the .ovpn, matched to its references by name (ADR-0105).
  const send = async (file: File, overwrite: boolean, extras: File[] = []) => {
    if (!file.name.toLowerCase().endsWith(ext)) {
      const vars = { file: file.name, ext }
      setPhase({ kind: 'error', error: { message: t('profiles.import.wrongExt', vars), fix: t('profiles.import.wrongExt.fix', vars) } })
      return
    }
    if (file.size > MAX_IMPORT_BYTES) {
      const vars = { file: file.name, ext, size: Math.ceil(file.size / 1024), max: MAX_IMPORT_BYTES / 1024 }
      setPhase({ kind: 'error', error: { message: t('profiles.import.tooBig', vars), fix: t('profiles.import.tooBig.fix', vars) } })
      return
    }
    let content: string
    const files: { name: string; content_base64: string }[] = []
    let current = file
    try {
      content = toBase64(new Uint8Array(await file.arrayBuffer()))
      for (const extra of extras) {
        current = extra
        if (extra.size > MAX_EXTRA_BYTES) {
          const vars = { file: extra.name, ext, size: Math.ceil(extra.size / 1024), max: MAX_EXTRA_BYTES / 1024 }
          setPhase({ kind: 'error', error: { message: t('profiles.import.tooBig', vars), fix: t('profiles.import.tooBig.fix', vars) } })
          return
        }
        files.push({ name: extra.name, content_base64: toBase64(new Uint8Array(await extra.arrayBuffer())) })
      }
    } catch {
      const vars = { file: current.name }
      setPhase({ kind: 'error', error: { message: t('profiles.import.readFail', vars), fix: t('profiles.import.readFail.fix', vars) } })
      return
    }
    save.mutate(
      { name: importName(profileName, file.name), type, content_base64: content, file_name: file.name, overwrite, ...(files.length > 0 ? { files } : {}) },
      {
        onSuccess: (result) => {
          setPhase({ kind: 'done', result, file: file.name })
          onImported(result.path)
        },
        onError: (err) => {
          const error = describeError(err, t)
          const taken = err instanceof ApiError && err.key === 'api.err.import_exists'
          setPhase(taken ? { kind: 'exists', file, extras, error } : { kind: 'error', error })
        },
      },
    )
  }

  // Root-only file: the daemon reads it after the OS permission window
  // (pkexec), with the same checks, and the copy replaces the path.
  const copyProtected = (path: string) => {
    const file = path.split('/').pop() ?? path
    save.mutate(
      { name: importName(profileName, file), type, path, elevate: true },
      {
        onSuccess: (result) => {
          setPhase({ kind: 'done', result })
          onImported(result.path)
        },
        onError: (err) => setPhase({ kind: 'error', error: describeError(err, t) }),
      },
    )
  }

  // The profile's own file (by extension) plus whatever came with it: its certificates.
  const pick = (list: FileList | null) => {
    const files = [...(list ?? [])]
    const file = files.find((f) => f.name.toLowerCase().endsWith(ext)) ?? files[0]
    if (file) void send(file, false, files.filter((f) => f !== file))
    if (input.current) input.current.value = ''
  }

  const onDrop = (e: DragEvent) => {
    e.preventDefault()
    dropZone.reset()
    pick(e.dataTransfer.files)
  }

  return (
    <div className="kp-import" data-drag={drag || undefined} {...dropZone.handlers} onDrop={onDrop}>
      <div className="kp-import__row">
        <input
          ref={input}
          id={id}
          type="file"
          accept={type === 'openvpn' ? `${ext},.crt,.pem,.key,.cer,.p12` : ext}
          multiple={type === 'openvpn'}
          className="kp-import__file"
          tabIndex={-1}
          aria-hidden="true"
          onChange={(e) => pick(e.target.files)}
        />
        <Button variant="outlined" icon={<UploadIcon />} loading={save.isPending} onClick={() => input.current?.click()}>
          {save.isPending ? t('profiles.import.busy') : t('profiles.import.button')}
        </Button>
        <p className="kp-import__what">{drag ? t('profiles.import.drop') : t('profiles.import.what', { ext })}</p>
      </div>

      {protectedPath && phase.kind !== 'done' && (
        <Banner
          tone="caution"
          title={t('profiles.import.protected', { path: protectedPath })}
          actions={
            <Button variant="outlined" size="sm" loading={save.isPending} onClick={() => copyProtected(protectedPath)}>
              {t('profiles.import.protectedCopy')}
            </Button>
          }
        >
          <span>{t('profiles.import.protectedWhat')}</span>
        </Banner>
      )}

      <div aria-live="polite">
        {phase.kind === 'done' && (
          <p className="kp-import__ok">
            {t(phase.result.overwritten ? 'profiles.import.replaced' : 'profiles.import.done')}{' '}
            <code className="mono">{phase.result.path}</code>
          </p>
        )}
        {phase.kind === 'done' && phase.result.found_in && (
          <p className="kp-import__ok">{t('add.file.found', { file: phase.file ?? '', dir: phase.result.found_in })}</p>
        )}
      </div>
      {phase.kind === 'exists' && (
        <Banner
          tone="caution"
          title={phase.error.message}
          actions={
            <>
              <Button variant="outlined" size="sm" onClick={() => setPhase({ kind: 'idle' })}>
                {t('profiles.import.keep')}
              </Button>
              <Button variant="danger" size="sm" loading={save.isPending} onClick={() => void send(phase.file, true, phase.extras)}>
                {t('profiles.import.replace')}
              </Button>
            </>
          }
        >
          <span>{t('profiles.import.replaceWhat', { file: importName(profileName, phase.file.name) + ext })}</span>
        </Banner>
      )}
      {phase.kind === 'error' && (
        <Banner tone="bad" title={phase.error.message} urgent>
          <span>{phase.error.fix}</span>
        </Banner>
      )}
    </div>
  )
}
