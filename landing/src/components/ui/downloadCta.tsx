import { OS } from '../../hooks/useOS'
import { Lang } from '../../i18n'
import { PageId, pathFor } from '../../site'

// Shared by every "download" entry point (hero, header, Para todos, the
// download section). Until a release ships the Windows installer (issue #32)
// these buttons scroll to the download section instead of starting a file,
// so the visitor sees the recommended file, the other systems and the
// unsigned-binary note before downloading (ADR-0027).
export const DOWNLOAD_SECTION_ID = 'install'

export function downloadHref(page: PageId, lang: Lang): string {
  return page === 'home' ? `#${DOWNLOAD_SECTION_ID}` : `${pathFor('home', lang)}#${DOWNLOAD_SECTION_ID}`
}

export const CTA_COPY: Record<Lang, {
  generic: string
  forOS: Record<OS, string>
  github: string
  githubLabel: string
  navDownload: string
  mobileNotice: string
  send: string
  copy: string
  copied: string
  copyFailed: string
  shareTitle: string
}> = {
  es: {
    generic: 'Descargar gratis',
    forOS: { windows: 'Descargar para Windows', macos: 'Descargar para Mac', linux: 'Descargar para Linux' },
    github: 'Ver en GitHub',
    githubLabel: 'Kongtrol en GitHub',
    navDownload: 'Descargar',
    mobileNotice: 'Estás en un celular: Kongtrol se instala en tu computador.',
    send: 'Enviar a mi computador',
    copy: 'Copiar enlace',
    copied: 'Enlace copiado. Ábrelo en tu computador.',
    copyFailed: 'No se pudo copiar. Copia la dirección de esta página.',
    shareTitle: 'Kongtrol — todas tus VPN en un solo lugar',
  },
  en: {
    generic: 'Download free',
    forOS: { windows: 'Download for Windows', macos: 'Download for Mac', linux: 'Download for Linux' },
    github: 'View on GitHub',
    githubLabel: 'Kongtrol on GitHub',
    navDownload: 'Download',
    mobileNotice: "You're on a phone: Kongtrol installs on your computer.",
    send: 'Send to my computer',
    copy: 'Copy link',
    copied: 'Link copied. Open it on your computer.',
    copyFailed: "Couldn't copy. Copy this page's address.",
    shareTitle: 'Kongtrol — all your VPNs in one place',
  },
}

/** "Descargar para Windows", or the generic label while the OS is unknown (prerender). */
export const downloadLabel = (lang: Lang, os: OS | null) =>
  os ? CTA_COPY[lang].forOS[os] : CTA_COPY[lang].generic

export function DownloadIcon({ size = 16 }: { size?: number }) {
  return (
    <svg width={size} height={size} viewBox="0 0 16 16" fill="currentColor" aria-hidden="true">
      <path d="M8 1a.5.5 0 0 1 .5.5v7.793l2.646-2.647a.5.5 0 0 1 .708.708l-3.5 3.5a.5.5 0 0 1-.708 0l-3.5-3.5a.5.5 0 1 1 .708-.708L7.5 9.293V1.5A.5.5 0 0 1 8 1ZM2.5 13a.5.5 0 0 0 0 1h11a.5.5 0 0 0 0-1h-11Z"/>
    </svg>
  )
}

export function GitHubIcon({ size = 16 }: { size?: number }) {
  return (
    <svg width={size} height={size} viewBox="0 0 16 16" fill="currentColor" aria-hidden="true">
      <path d="M8 0C3.58 0 0 3.58 0 8c0 3.54 2.29 6.53 5.47 7.59.4.07.55-.17.55-.38 0-.19-.01-.82-.01-1.49-2.01.37-2.53-.49-2.69-.94-.09-.23-.48-.94-.82-1.13-.28-.15-.68-.52-.01-.53.63-.01 1.08.58 1.23.82.72 1.21 1.87.87 2.33.66.07-.52.28-.87.51-1.07-1.78-.2-3.64-.89-3.64-3.95 0-.87.31-1.59.82-2.15-.08-.2-.36-1.02.08-2.12 0 0 .67-.21 2.2.82.64-.18 1.32-.27 2-.27s1.36.09 2 .27c1.53-1.04 2.2-.82 2.2-.82.44 1.1.16 1.92.08 2.12.51.56.82 1.27.82 2.15 0 3.07-1.87 3.75-3.65 3.95.29.25.54.73.54 1.48 0 1.07-.01 1.93-.01 2.2 0 .21.15.46.55.38A8.01 8.01 0 0 0 16 8c0-4.42-3.58-8-8-8Z"/>
    </svg>
  )
}

export function SendIcon({ size = 16 }: { size?: number }) {
  // phone → screen: a laptop outline with an arrow coming in
  return (
    <svg width={size} height={size} viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
      <path d="M4 16V6a1 1 0 0 1 1-1h14a1 1 0 0 1 1 1v10" />
      <path d="M2 19h20" />
      <path d="M12 8v5m-2.5-2.5L12 13l2.5-2.5" />
    </svg>
  )
}
