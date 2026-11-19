// Security policy & responsible disclosure — version 1.2.0 · 2026-11-19.
// 1.3.0: scope adds another user account on the same computer (port squatting, cookies sent to
//   every loopback port; ADR-0062).
// 1.2.0: plain-language summary in points; scope says "without signing in" (control token for
// CLI/tray, one-time code + session cookie for browsers, ADR-0039) and adds the install scripts'
// checksum/signature check (ADR-0057); new section linking the technical security doc.
// 1.1.0: best-effort response (no fixed 7/30/90-day targets): free, open-source
// project maintained by one person; first person singular throughout.
//
// Mirrors SECURITY.md at the repo root and /.well-known/security.txt (RFC 9116,
// https://www.rfc-editor.org/rfc/rfc9116). Reporting channel: GitHub private
// vulnerability reporting (https://docs.github.com/code-security/security-advisories/guidance-on-reporting-and-writing-information-about-vulnerabilities/privately-reporting-a-security-vulnerability).
import type { Lang } from '../../i18n'
import type { LegalDocContent } from '../../components/sections/LegalDoc'
import { GITHUB_REPO } from '../../links'
import IC from '../../components/ui/InlineCode'

const VERSION = '1.3.0'
const UPDATED = '2026-11-19'
export const SECURITY_REPORT_URL = `${GITHUB_REPO}/security/advisories/new`
const report = <a href={SECURITY_REPORT_URL} target="_blank" rel="noreferrer">github.com/DerotLuna/vpn-kongtrol/security/advisories/new</a>
const techDoc = <a href={`${GITHUB_REPO}/blob/main/docs/SECURITY.md`} target="_blank" rel="noreferrer">docs/SECURITY.md</a>
const releases = <a href={`${GITHUB_REPO}/releases/latest`} target="_blank" rel="noreferrer">GitHub Releases</a>

const es: LegalDocContent = {
  label: '$ cat SECURITY.md',
  title: 'Política de seguridad',
  sub: 'Cómo reportar una vulnerabilidad de forma privada y qué esperar después.',
  version: VERSION,
  updated: UPDATED,
  session: 'kongtrol:seguridad',
  summaryBadge: 'En resumen',
  summary: [
    <>¿Encontraste una vulnerabilidad? Repórtala en privado en GitHub: {report}.</>,
    <>No abras un issue público ni publiques detalles hasta que haya una corrección.</>,
    <>Lo mantiene una persona: respondo lo antes posible, sin plazos garantizados.</>,
    <>Solo la última versión recibe correcciones de seguridad.</>,
    <>Si investigas de buena fe, no tomaré acciones contra ti.</>,
  ],
  toc: 'Contenido',
  goto: 'Ir a',
  versionLabel: 'Versión',
  updatedLabel: 'actualizada el',
  contactTitle: 'Reportar',
  contact: <>{report} (reporte privado de vulnerabilidades de GitHub). También publicado en{' '}
    <a href="/.well-known/security.txt">/.well-known/security.txt</a>.</>,
  sections: [
    {
      id: 'reportar', num: '1', label: 'Cómo reportar', tmux: 'reportar', title: 'Cómo reportar',
      body: [<>Usa el reporte privado de vulnerabilidades de GitHub: {report}. Solo lo ve el mantenedor. No
        publiques detalles en issues, discusiones ni redes sociales hasta que haya una corrección.</>,
      'Incluye, si puedes:'],
      list: [
        'Versión de Kongtrol (kongtrol version) y sistema operativo.',
        'Qué componente: CLI, servicio en segundo plano, app de bandeja, panel web, ayudante con privilegios o este sitio.',
        'Pasos para reproducirlo y qué impacto tiene (por ejemplo, una fuga de tráfico fuera del túnel).',
        <>Nunca incluyas contraseñas reales, tu <IC c="kongtrol.yaml" /> completo ni el token de{' '}
          <IC c="~/.kongtrol/api-token" />; usa datos de prueba.</>,
      ],
    },
    {
      id: 'alcance', num: '2', label: 'Alcance', tmux: 'alcance', title: 'Qué está en alcance',
      body: ['En alcance: el código de este repositorio y los binarios publicados en la última versión.'],
      list: [
        'Fugas de tráfico o DNS que el kill switch, el DNS guard o las reglas deberían impedir.',
        'Escalada de privilegios a través del ayudante del sistema o del servicio en segundo plano.',
        'Entrar al panel local o a su API sin iniciar sesión, desde otro equipo o desde otra cuenta de usuario de este equipo (por ejemplo, ocupando el puerto de Kongtrol).',
        'Exposición de credenciales fuera del almacén de credenciales del sistema.',
        'Manipulación del registro de auditoría sin que se detecte.',
        'Saltarse la comprobación de suma o de firma de los scripts de instalación.',
        'Vulnerabilidades en kongtrol.app (por ejemplo, XSS).',
      ],
      after: ['Fuera de alcance: los clientes y servicios de los proveedores VPN, ataques que ya requieren control de administrador del equipo, denegación de servicio por volumen, ingeniería social y cabeceras faltantes sin impacto demostrable.'],
    },
    {
      id: 'respuesta', num: '3', label: 'Respuesta', tmux: 'respuesta', title: 'Qué esperar',
      body: ['Intentaré responder lo antes posible; es un proyecto sin fines de lucro mantenido por una persona, así que no hay plazos garantizados.'],
      list: [
        'Te confirmaré la recepción y, después, si el problema se confirma y su gravedad.',
        'La corrección y su publicación se coordinan contigo; los detalles se publican cuando la corrección está disponible.',
        'Crédito en el aviso de seguridad, si lo deseas.',
      ],
    },
    {
      id: 'versiones', num: '4', label: 'Versiones', tmux: 'versiones', title: 'Versiones con soporte',
      body: [<>Kongtrol está en beta: solo la última versión publicada en {releases} recibe correcciones de
        seguridad. Actualiza antes de reportar si puedes.</>],
    },
    {
      id: 'buena-fe', num: '5', label: 'Buena fe', tmux: 'buena-fe', title: 'Investigación de buena fe',
      body: [<>Si investigas de buena fe, en tus propios equipos, sin acceder a datos de otras personas y siguiendo
        esta política, no tomaré acciones contra ti y trabajaré contigo para entender y corregir el problema.</>],
    },
    {
      id: 'detalles', num: '6', label: 'Detalles técnicos', tmux: 'detalles', title: 'Cómo se protege Kongtrol',
      body: [<>Qué conexiones hace, cómo funcionan el ayudante del sistema, el kill switch y el DNS guard, y cómo
        verificar una descarga: {techDoc} (en inglés).</>],
    },
  ],
}

const en: LegalDocContent = {
  label: '$ cat SECURITY.md',
  title: 'Security policy',
  sub: 'How to report a vulnerability privately and what to expect next.',
  version: VERSION,
  updated: UPDATED,
  session: 'kongtrol:security',
  summaryBadge: 'In short',
  summary: [
    <>Found a vulnerability? Report it privately on GitHub: {report}.</>,
    <>Do not open a public issue or post details until a fix is out.</>,
    <>One person maintains it: I answer as soon as I can, with no guaranteed timelines.</>,
    <>Only the latest release gets security fixes.</>,
    <>If you research in good faith, I will not take action against you.</>,
  ],
  toc: 'Contents',
  goto: 'Jump to',
  versionLabel: 'Version',
  updatedLabel: 'updated',
  contactTitle: 'Report',
  contact: <>{report} (GitHub private vulnerability reporting). Also published at{' '}
    <a href="/.well-known/security.txt">/.well-known/security.txt</a>.</>,
  sections: [
    {
      id: 'reportar', num: '1', label: 'How to report', tmux: 'report', title: 'How to report',
      body: [<>Use GitHub private vulnerability reporting: {report}. Only the maintainer can see it. Do not post
        details in issues, discussions or social media until a fix is out.</>,
      'If you can, include:'],
      list: [
        'Kongtrol version (kongtrol version) and operating system.',
        'Which component: CLI, background service, tray app, web dashboard, privileged helper or this website.',
        'Steps to reproduce and the impact (for example, traffic leaking outside the tunnel).',
        <>Never include real passwords, your full <IC c="kongtrol.yaml" /> or the{' '}
          <IC c="~/.kongtrol/api-token" /> token; use test data.</>,
      ],
    },
    {
      id: 'alcance', num: '2', label: 'Scope', tmux: 'scope', title: 'What is in scope',
      body: ['In scope: the code in this repository and the binaries of the latest release.'],
      list: [
        'Traffic or DNS leaks that the kill switch, DNS guard or rules should prevent.',
        'Privilege escalation through the system helper or the background service.',
        'Access to the local dashboard or its API without signing in, from another machine, or from another user account on this computer (for example, by taking the port Kongtrol uses).',
        'Credentials exposed outside the operating system credential store.',
        'Undetected tampering with the audit log.',
        "Getting past the install scripts' checksum or signature check.",
        'Vulnerabilities in kongtrol.app (for example, XSS).',
      ],
      after: ['Out of scope: VPN providers\' own clients and services, attacks that already require administrator control of the machine, volumetric denial of service, social engineering, and missing headers with no demonstrable impact.'],
    },
    {
      id: 'respuesta', num: '3', label: 'Response', tmux: 'response', title: 'What to expect',
      body: ['I will try to answer as soon as possible; this is a non-profit project maintained by one person, so there are no guaranteed timelines.'],
      list: [
        'I will confirm receipt and, later, whether the issue is confirmed and how severe it is.',
        'The fix and its release are coordinated with you; details are published once the fix is available.',
        'Credit in the security advisory, if you want it.',
      ],
    },
    {
      id: 'versiones', num: '4', label: 'Versions', tmux: 'versions', title: 'Supported versions',
      body: [<>Kongtrol is in beta: only the latest release on {releases} gets security fixes. Update before
        reporting if you can.</>],
    },
    {
      id: 'buena-fe', num: '5', label: 'Good faith', tmux: 'good-faith', title: 'Good-faith research',
      body: [<>If you research in good faith, on your own machines, without accessing other people's data and
        following this policy, I will not take action against you and will work with you to understand and fix
        the issue.</>],
    },
    {
      id: 'detalles', num: '6', label: 'Technical details', tmux: 'details', title: 'How Kongtrol is secured',
      body: [<>Which connections it makes, how the system helper, the kill switch and the DNS guard work, and how to
        verify a download: {techDoc}.</>],
    },
  ],
}

const doc: Record<Lang, LegalDocContent> = { es, en }
export default doc
