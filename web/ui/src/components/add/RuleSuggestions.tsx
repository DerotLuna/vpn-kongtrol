import { usePreferences } from '../../app/preferences-context'
import { Button } from '../ui/Button'
import type { RulePreset } from './model'
import './suggestions.css'

const suggestions: { id: RulePreset; label: string }[] = [
  { id: 'claude', label: 'Claude' },
  { id: 'chatgpt', label: 'ChatGPT / OpenAI' },
  { id: 'oracle', label: 'Oracle / Java' },
]

export function RuleSuggestions({ onAdd }: { onAdd: (preset: RulePreset) => void }) {
  const { t } = usePreferences()
  return (
    <fieldset className="k-rule-suggestions">
      <legend className="k-field__label">{t('add.rule.presets')}</legend>
      <div className="k-rule-suggestions__buttons">
        {suggestions.map(({ id, label }) => (
          <Button key={id} type="button" variant="outlined" size="sm" onClick={() => onAdd(id)}><span aria-hidden="true">＋ </span>{label}</Button>
        ))}
      </div>
      <p>{t('add.rule.presets.hint')}</p>
    </fieldset>
  )
}
