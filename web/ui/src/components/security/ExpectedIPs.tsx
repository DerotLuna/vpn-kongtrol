// Kongtrol UI — Seguridad: expected public IP per VPN · v1.0.0 · 2026-11-18
//
// security.integrity_check.expected_ips (PUT /api/v1/settings
// `expected_ips`, ADR-0042): the public IP each VPN should give while it is
// connected. A check that sees another IP for a connected VPN warns of a
// leak; VPNs without a row are not compared. Rows are added, edited and
// removed here and saved together (the daemon replaces the whole map);
// checks run on Save (VPN picked, once each, a valid IPv4/IPv6) and land
// on the field with what to do; a daemon refusal shows under the list.
import { useId, useState, type FormEvent } from 'react'
import { describeError } from '../../api/errors'
import { usePreferences } from '../../app/preferences-context'
import { useUnsavedChanges } from '../../app/unsaved'
import {
  expectedMap,
  expectedProblems,
  expectedRows,
  useProfiles,
  useSettingsSave,
  type ExpectedProblem,
  type ExpectedRow,
} from '../../data/security'
import { Button, IconButton } from '../ui/Button'
import { CloseIcon, PlusIcon } from '../ui/icons'
import { Select } from '../ui/Select'
import { isIP } from './model'

const PROBLEM = {
  pick: ['security.expected.pick', 'security.expected.pick.fix'],
  duplicate: ['security.expected.duplicate', 'security.expected.duplicate.fix'],
  invalid: ['security.expected.invalid', 'security.expected.invalid.fix'],
} as const

export function ExpectedIPs({ stored }: { stored: Record<string, string> | undefined }) {
  const { t } = usePreferences()
  const profiles = useProfiles()
  const save = useSettingsSave()
  const id = useId()
  const initial = expectedRows(stored)
  const [rows, setRows] = useState<ExpectedRow[]>(initial)
  const [problems, setProblems] = useState<ExpectedProblem[]>([])
  const dirty = JSON.stringify(expectedMap(rows)) !== JSON.stringify(expectedMap(initial)) || rows.length !== initial.length
  useUnsavedChanges(dirty)

  const names = (profiles.data ?? []).map((p) => p.name)
  const remote = save.error ? describeError(save.error, t) : null

  const edit = (i: number, patch: Partial<ExpectedRow>) => {
    setRows((rs) => rs.map((r, j) => (j === i ? { ...r, ...patch } : r)))
    setProblems((ps) => ps.filter((p) => p.row !== i))
  }

  const submit = (e: FormEvent<HTMLFormElement>) => {
    e.preventDefault()
    const found = expectedProblems(rows, isIP)
    setProblems(found)
    const first = found[0]
    if (first) {
      e.currentTarget.querySelector<HTMLElement>(`#${CSS.escape(`${id}-${first.field}-${first.row}`)}`)?.focus()
      return
    }
    save.mutate({ expected_ips: expectedMap(rows) })
  }

  const problemOf = (row: number, field: 'profile' | 'ip') => problems.find((p) => p.row === row && p.field === field)

  return (
    <form className="sec-form sec-expected" onSubmit={submit} noValidate aria-labelledby={`${id}-title`}>
      <h3 id={`${id}-title`} className="sec-expected__title">
        {t('security.expected.title')}
      </h3>
      <p className="sec-meta">{t('security.expected.what')}</p>

      {rows.length === 0 ? (
        <p className="sec-meta">{t('security.expected.empty')}</p>
      ) : (
        <ul className="sec-expected__rows">
          {rows.map((r, i) => {
            const pp = problemOf(i, 'profile')
            const ip = problemOf(i, 'ip')
            const err = pp ?? ip
            const vars = { name: r.profile, value: r.ip.trim() }
            const errId = `${id}-err-${i}`
            // A row may name a VPN that no longer exists: keep it selectable so it can be removed or changed.
            const options = r.profile && !names.includes(r.profile) ? [r.profile, ...names] : names
            return (
              <li key={i} className="sec-expected__row">
                <Select
                  id={`${id}-profile-${i}`}
                  className="sec-select"
                  aria-label={t('security.expected.profile')}
                  mono
                  value={r.profile}
                  placeholder={t('security.expected.choose')}
                  aria-invalid={pp ? true : undefined}
                  aria-describedby={pp ? errId : undefined}
                  onChange={(profile) => edit(i, { profile })}
                  options={options.map((n) => ({ value: n, label: n }))}
                />
                <label className="visually-hidden" htmlFor={`${id}-ip-${i}`}>
                  {t('security.expected.ip', { name: r.profile || '—' })}
                </label>
                <input
                  id={`${id}-ip-${i}`}
                  className="k-input mono sec-expected__ip"
                  value={r.ip}
                  inputMode="text"
                  autoComplete="off"
                  spellCheck={false}
                  placeholder="203.0.113.7"
                  aria-invalid={ip ? true : undefined}
                  aria-describedby={ip ? errId : undefined}
                  onChange={(e) => edit(i, { ip: e.target.value })}
                />
                <IconButton
                  label={t('security.expected.remove', { name: r.profile || '—' })}
                  icon={<CloseIcon />}
                  onClick={() => {
                    setRows((rs) => rs.filter((_, j) => j !== i))
                    setProblems([])
                  }}
                />
                {err && (
                  <p id={errId} className="k-field__error sec-expected__err" role="alert">
                    <strong>{t(PROBLEM[err.kind][0], vars)}</strong> <span>{t(PROBLEM[err.kind][1], vars)}</span>
                  </p>
                )}
              </li>
            )
          })}
        </ul>
      )}

      {remote && (
        <p className="sec-alert" role="alert">
          <strong>{remote.message}</strong> {remote.fix}
        </p>
      )}

      <div className="sec-form__actions">
        <Button
          variant="text"
          size="sm"
          icon={<PlusIcon />}
          onClick={() => setRows((rs) => [...rs, { profile: names.find((n) => !rs.some((r) => r.profile === n)) ?? '', ip: '' }])}
        >
          {t('security.expected.add')}
        </Button>
        <Button type="submit" variant="outlined" size="sm" loading={save.isPending} disabled={!dirty}>
          {t('security.save')}
        </Button>
        <span className="sec-form__done" aria-live="polite">
          {save.isSuccess && !dirty ? t('security.saved') : ''}
        </span>
      </div>
    </form>
  )
}
