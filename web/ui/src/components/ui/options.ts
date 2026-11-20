// Kongtrol UI kit — shared option types and helpers for Select / MultiSelect · v1.0.0 · 2026-11-19
// (Kept apart from the components so fast refresh keeps working.)

export interface SelectOption<V extends string = string> {
  value: V
  label: string
  /** A second, quieter line (e.g. what the choice does). */
  hint?: string
  disabled?: boolean | undefined
  /** Heading of a group of options (like <optgroup>); consecutive options share it. */
  group?: string | undefined
}

export interface ControlA11y {
  id?: string | undefined
  'aria-labelledby'?: string | undefined
  'aria-label'?: string | undefined
  'aria-describedby'?: string | undefined
  'aria-invalid'?: boolean | undefined
}

type Defined<T> = { [K in keyof T]?: Exclude<T[K], undefined> }

/** Drops undefined props (React Aria's prop types are strict optionals under exactOptionalPropertyTypes). */
export function defined<T extends object>(props: T): Defined<T> {
  return Object.fromEntries(Object.entries(props).filter(([, v]) => v !== undefined)) as Defined<T>
}

/** Consecutive options with the same `group` become one section. */
export function groupOptions<V extends string>(options: readonly SelectOption<V>[]): Array<{ group: string | undefined; items: SelectOption<V>[] }> {
  const out: Array<{ group: string | undefined; items: SelectOption<V>[] }> = []
  for (const o of options) {
    const last = out.at(-1)
    if (last && last.group === o.group) last.items.push(o)
    else out.push({ group: o.group, items: [o] })
  }
  return out
}
