// Kongtrol UI — "Nuevo grupo" / "Editar grupo" · v1.0.0 · 2026-11-18
//
// Name + the profiles it contains, picked from the existing ones (legacy
// took free text, one per line: docs/dashboard-parity.md §3.2 Groups,
// **fix**), so an unknown name cannot be typed. Checks on save, then live.
import { useId, useState, type FormEvent } from 'react'
import { usePreferences } from '../../app/preferences-context'
import { useUnsavedChanges } from '../../app/unsaved'
import type { Group } from '../../api/types'
import type { VPNProfile } from '../../data/profiles'
import { Button } from '../ui/Button'
import { Dialog } from '../ui/Dialog'
import { TextField } from '../ui/Field'
import { Monogram } from './Bits'
import { specOf } from './catalog'

interface Props {
  open: boolean
  onClose: () => void
  editing?: Group | undefined
  groups: readonly Group[]
  profiles: readonly VPNProfile[]
  onSave: (vars: { original?: string | undefined; name: string; profiles: string[] }) => void
}

export function GroupEditor({ open, onClose, editing, groups, profiles, onSave }: Props) {
  const { t } = usePreferences()
  const id = useId()
  const [name, setName] = useState(editing?.name ?? '')
  const [members, setMembers] = useState<string[]>(editing?.profiles ?? [])
  const [tried, setTried] = useState(false)
  useUnsavedChanges(name !== (editing?.name ?? '') || members.join('\n') !== (editing?.profiles ?? []).join('\n'))

  const trimmed = name.trim()
  const nameError = !tried
    ? null
    : !trimmed
      ? { message: t('groups.check.nameRequired'), fix: t('groups.check.nameRequired.fix') }
      : !editing && groups.some((g) => g.name === trimmed)
        ? { message: t('groups.err.exists'), fix: t('groups.err.exists.fix') }
        : null
  const membersError = tried && members.length === 0

  const toggle = (p: string, on: boolean) =>
    setMembers((m) => (on ? [...m, p] : m.filter((x) => x !== p)))

  const submit = (e: FormEvent) => {
    e.preventDefault()
    setTried(true)
    const nameBad = !trimmed || (!editing && groups.some((g) => g.name === trimmed))
    if (nameBad || members.length === 0) return
    // Keep the profile order of the list (stable, readable YAML).
    const ordered = profiles.map((p) => p.name).filter((n) => members.includes(n))
    onSave({ original: editing?.name, name: trimmed, profiles: ordered })
    onClose()
  }

  return (
    <Dialog
      open={open}
      onClose={onClose}
      variant="sheet"
      title={editing ? t('groups.editor.editTitle', { name: editing.name }) : t('groups.editor.newTitle')}
      description={t('groups.editor.lead')}
    >
      <form className="kp-editor" onSubmit={submit} noValidate>
        <TextField
          label={t('groups.field.name')}
          required
          autoComplete="off"
          spellCheck={false}
          placeholder={t('groups.placeholder.name')}
          readOnly={!!editing}
          hint={editing ? t('groups.hint.nameLocked') : undefined}
          value={name}
          onChange={(e) => setName(e.target.value)}
          error={nameError}
        />
        <fieldset className="kp-members" aria-describedby={membersError ? `${id}-err` : undefined} aria-invalid={membersError || undefined}>
          <legend className="k-field__label">{t('groups.field.profiles')}</legend>
          <ul>
            {profiles.map((p) => (
              <li key={p.name}>
                <label className="kp-check">
                  <input type="checkbox" checked={members.includes(p.name)} onChange={(e) => toggle(p.name, e.target.checked)} />
                  <Monogram spec={specOf(p.type)} />
                  <span className="mono">{p.name}</span>
                  <span className="kp-muted kp-small">{specOf(p.type).label}</span>
                </label>
              </li>
            ))}
          </ul>
          {membersError && (
            <p id={`${id}-err`} className="k-field__error" role="alert">
              <strong>{t('groups.check.profilesRequired')}</strong> <span>{t('groups.check.profilesRequired.fix')}</span>
            </p>
          )}
        </fieldset>
        <footer className="k-dialog__actions">
          <Button variant="outlined" onClick={onClose}>
            {t('profiles.action.cancel')}
          </Button>
          <Button type="submit">{editing ? t('profiles.action.save') : t('groups.action.create')}</Button>
        </footer>
      </form>
    </Dialog>
  )
}
