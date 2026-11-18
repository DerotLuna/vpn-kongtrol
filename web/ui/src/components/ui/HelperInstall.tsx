// Kongtrol UI — helper install button + progress, shared by every surface · v0.1.0 · 2026-11-18
//
// One install flow for the Home card, the diagnostics "Reparar" and the
// app-wide notice: the same mutation key, so all of them show "Instalando…"
// while any one runs, and all disappear when the cache says the helper is
// ready (data/resources.ts). The progress line names the OS prompt to look
// for; a failure arrives as one toast with the "what to do" line.
import { useMutationState } from '@tanstack/react-query'
import { describeError } from '../../api/errors'
import type { HelperState } from '../../api/hooks'
import { mutationKeys } from '../../data/keys'
import { usePreferences } from '../../app/preferences-context'
import type { MessageKey } from '../../i18n/messages'
import { detectPlatform } from '../../lib/platform'
import { Button } from './Button'

const INSTALLING: Record<ReturnType<typeof detectPlatform>, MessageKey> = {
  windows: 'helper.installing.windows',
  mac: 'helper.installing.mac',
  linux: 'helper.installing.linux',
  other: 'helper.installing',
}

export function HelperInstallButton({ helper, outdated }: { helper: HelperState; outdated: boolean }) {
  const { t } = usePreferences()
  return (
    <Button variant="filled" loading={helper.installing} onClick={helper.install}>
      {outdated ? t('helper.update') : t('helper.install')}
    </Button>
  )
}

/**
 * Polite live line: progress while installing, confirmation after, and the
 * last failure with its "what to do" line (the toast is never the only copy
 * of an error), whichever button started the install.
 */
export function HelperInstallProgress({ helper }: { helper: HelperState }) {
  const { t } = usePreferences()
  const states = useMutationState({ filters: { mutationKey: mutationKeys.helperInstall }, select: (m) => m.state })
  const last = states.at(-1)
  const failure = !helper.installing && last?.status === 'error' ? describeError(last.error, t) : null
  return (
    <div aria-live="polite">
      {helper.installing && <p className="notice">{t(INSTALLING[detectPlatform()])}</p>}
      {helper.installed && !helper.installing && <p className="notice ok">{t('helper.installed')}</p>}
      {failure && (
        <p className="notice bad">
          <strong>{failure.message}</strong>
          <span>{failure.fix}</span>
        </p>
      )}
    </div>
  )
}
