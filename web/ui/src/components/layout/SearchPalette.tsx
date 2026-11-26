import { useEffect, useMemo, useRef } from 'react'
import { useNavigate } from 'react-router'
import { Autocomplete, Input, ListBox, ListBoxItem, SearchField, useFilter } from 'react-aria-components'
import { useAddSheet } from '../../app/addLink'
import { usePreferences } from '../../app/preferences-context'
import { SECTIONS } from '../../app/sections'
import { useGroupsList, useProfilesQuery } from '../../data/profiles'
import { usePoliciesQuery } from '../../data/rules'
import { profileDescription } from '../../data/profileList'
import { Dialog } from '../ui/Dialog'
import './search.css'

interface Item {
  id: string
  label: string
  detail: string
  text: string
  path?: string
  add?: 'vpn' | 'file' | 'rule'
}

function SearchContents({ close }: { close: () => void }) {
  const { t } = usePreferences()
  const navigate = useNavigate()
  const { open } = useAddSheet()
  const profiles = useProfilesQuery()
  const groups = useGroupsList()
  const policies = usePoliciesQuery()
  const { contains } = useFilter({ sensitivity: 'base' })
  const input = useRef<HTMLInputElement>(null)
  useEffect(() => {
    const timer = window.setTimeout(() => input.current?.focus(), 0)
    return () => window.clearTimeout(timer)
  }, [])
  const items = useMemo<Item[]>(() => {
    const sections = SECTIONS.map((section) => ({
      id: `section:${section.id}`, label: t(section.label), detail: t('search.section'),
      text: t(section.label), path: section.path,
    }))
    const subpages = [
      { id: 'groups', label: t('page.profiles.tab.groups'), path: '/profiles/groups' },
      { id: 'schedule', label: t('page.rules.tab.schedule'), path: '/rules/schedule' },
      { id: 'diagnostics', label: t('diag.open'), path: '/settings?section=diagnostics' },
    ].map((page) => ({ ...page, id: `page:${page.id}`, detail: t('search.section'), text: page.label }))
    const actions = (['vpn', 'file', 'rule'] as const).map((add) => ({
      id: `add:${add}`, label: t(`search.add.${add}`), detail: t('search.action'), text: t(`search.add.${add}`), add,
    }))
    const vpns = (profiles.data ?? []).map((p) => ({
      id: `vpn:${p.name}`, label: p.name, detail: profileDescription(p, t('profile.identity.app')) || t('search.profile'),
      text: `${p.name} ${p.type} ${profileDescription(p, t('profile.identity.app'))}`,
      path: `/profiles?edit=${encodeURIComponent(p.name)}`,
    }))
    const groupItems = (groups.data ?? []).map((g) => ({
      id: `group:${g.name}`, label: g.name, detail: t('search.group'), text: `${g.name} ${g.profiles.join(' ')}`,
      path: `/profiles/groups?focus=${encodeURIComponent(g.name)}`,
    }))
    const ruleItems = (policies.data ?? []).map((p) => ({
      id: `rule:${p.name}`, label: p.name, detail: t('search.rule'),
      text: [p.name, p.via, ...(p.domains ?? []), ...(p.ip_ranges ?? []), ...(p.apps ?? [])].join(' '),
      path: `/rules?focus=${encodeURIComponent(p.name)}`,
    }))
    return [...sections, ...subpages, ...actions, ...vpns, ...groupItems, ...ruleItems]
  }, [groups.data, policies.data, profiles.data, t])

  return (
    <div onKeyDownCapture={(event) => {
      if (event.key === 'Escape' && !event.nativeEvent.isComposing) {
        event.preventDefault()
        event.stopPropagation()
        close()
      }
    }}>
    <Autocomplete filter={contains}>
      <SearchField aria-label={t('search.label')} className="k-search__field">
        <Input ref={input} className="k-input" placeholder={t('search.placeholder')} />
      </SearchField>
      {(profiles.isError || groups.isError || policies.isError) && <p role="status">{t('search.partial')}</p>}
      <ListBox
        aria-label={t('search.results')}
        className="k-search__results"
        items={items}
        renderEmptyState={() => <p className="k-listbox__empty">{t('kit.select.empty')}</p>}
        onAction={(key) => {
          const item = items.find((entry) => entry.id === key)
          if (!item) return
          close()
          if (item.add) open({ mode: item.add })
          else if (item.path) void navigate(item.path)
        }}
      >
        {(item) => (
          <ListBoxItem id={item.id} textValue={item.text} className="k-search__item">
            <span>{item.label}</span><small>{item.detail}</small>
          </ListBoxItem>
        )}
      </ListBox>
    </Autocomplete>
    </div>
  )
}

export function SearchPalette({ open, onClose }: { open: boolean; onClose: () => void }) {
  const { t } = usePreferences()
  return (
    <Dialog open={open} onClose={onClose} title={t('search.title')} className="k-search">
      {open && <SearchContents close={onClose} />}
    </Dialog>
  )
}
