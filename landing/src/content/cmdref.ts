// Command reference rows for the guide — version 1.0.0 · 2026-11-17.
//
// Data comes from docs/cli.json, generated from the CLI's own command tree
// and translations (`go generate ./cmd/kongtrol`, see internal/docgen), so
// the guide always lists exactly what `kongtrol --help` shows. Never write
// commands here by hand.
import cli from '../../../docs/cli.json'
import type { Lang } from '../i18n'

export interface CmdrefRow { cmd: string; desc: string }
export interface CmdrefGroup { title: string; rows: CmdrefRow[] }

export function cmdrefGroups(lang: Lang): CmdrefGroup[] {
  const ref = cli.langs[lang]
  const groups: CmdrefGroup[] = ref.groups.map((g) => ({
    title: g.title,
    rows: g.commands.map((c) => ({
      cmd: [c.usage, ...(c.flags ?? []).map((f) => f.flag)].join(' '),
      desc: c.short,
    })),
  }))
  groups.push({
    title: ref.labels.globalFlags,
    rows: ref.globalFlags.map((f) => ({ cmd: f.flag, desc: f.usage })),
  })
  return groups
}
