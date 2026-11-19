// Terms & conditions — version 1.4.0 · 2026-11-19.
// 1.4.0: the product name is just Kongtrol (formerly VPN Kongtrol, ADR-0045).
// 1.3.0: the public IP check is optional and off by default.
// 1.2.0: states it is a free tool by one individual (no company); liability rests on
// Apache-2.0 sections 7-8; contact is GitHub only, no governing-law clause.
// 1.1.0: privacy section now points to /privacidad and no longer claims
// "no external calls" (Kongtrol checks the public IP for leak detection);
// vulnerability reports go to /seguridad.
import type { Lang } from '../../i18n'
import type { LegalDocContent } from '../../components/sections/LegalDoc'
import { pathFor } from '../../site'
import { GITHUB_REPO } from '../../links'

const VERSION = '1.4.0'
const UPDATED = '2026-11-19'
const issues = <a href={`${GITHUB_REPO}/issues`} target="_blank" rel="noreferrer">github.com/DerotLuna/vpn-kongtrol/issues</a>
const license = <a href={`${GITHUB_REPO}/blob/main/LICENSE`} target="_blank" rel="noreferrer">Apache 2.0</a>

const es: LegalDocContent = {
  label: '$ cat TERMS.md',
  title: 'Términos y condiciones',
  sub: 'Software de código abierto, sin garantía, uso bajo tu propia responsabilidad.',
  version: VERSION,
  updated: UPDATED,
  session: 'kongtrol:terminos',
  summaryBadge: 'En resumen',
  summary: <>Kongtrol es código abierto ({license}), se entrega “tal cual” sin garantía, y está en beta activa —
    úsalo con criterio en escenarios donde una fuga de red tenga consecuencias serias.</>,
  toc: 'Contenido',
  goto: 'Ir a',
  versionLabel: 'Versión',
  contactTitle: 'Preguntas o problemas',
  contact: <>abre un issue en el repositorio de GitHub: {issues}. Para vulnerabilidades, sigue la{' '}
    <a href={pathFor('seguridad', 'es')}>política de seguridad</a> (no abras un issue público).</>,
  sections: [
    {
      id: 'que-es', num: '1', label: 'Qué es Kongtrol', tmux: 'que-es', title: 'Qué es Kongtrol',
      body: [<>Kongtrol (antes llamado “VPN Kongtrol”) es software de código abierto, publicado bajo la licencia {license},
        que actúa como una capa de orquestación sobre clientes VPN que ya tienes instalados en tu equipo (OpenVPN,
        WireGuard, ProtonVPN, etc.). No es un proveedor de VPN, no opera servidores, y no tiene acceso a tu tráfico
        salvo el que corre localmente en tu propia máquina. Es una herramienta gratuita mantenida por una
        persona, sin empresa detrás.</>],
    },
    {
      id: 'garantia', num: '2', label: 'Sin garantía', tmux: 'garantia', title: 'Software “tal cual”, sin garantía',
      callout: 'warn',
      body: [<>Kongtrol se distribuye “AS IS” (tal cual), sin garantías de ningún tipo, expresas o implícitas,
        incluyendo — sin limitarse a — garantías de comerciabilidad, idoneidad para un propósito particular, o no
        infracción. En ningún caso los mantenedores o colaboradores del proyecto serán responsables por reclamos,
        daños u otras responsabilidades derivadas del uso del software, según los términos de la licencia Apache 2.0
        (secciones 7 y 8).</>],
    },
    {
      id: 'beta', num: '3', label: 'Beta activa', tmux: 'beta', title: 'Software en beta activa',
      body: [<>El proyecto está en pruebas activas: no todos los adaptadores, escenarios de red, ni sistemas
        operativos están completamente validados. Puede fallar, comportarse de forma inesperada, o no bloquear una
        fuga de tráfico correctamente en ciertos escenarios (kill switch, DNS guard).</>,
      <>No uses Kongtrol como única capa de protección en escenarios donde una fuga de IP o DNS tenga consecuencias
        legales o de seguridad graves para ti.</>],
    },
    {
      id: 'responsabilidad', num: '4', label: 'Tu responsabilidad', tmux: 'responsabilidad', title: 'Tu responsabilidad',
      body: ['Eres el único responsable de:'],
      list: [
        'Cumplir los términos de servicio de cada proveedor VPN que conectes a través de Kongtrol.',
        'Cumplir las leyes aplicables en tu jurisdicción respecto al uso de VPNs y herramientas de enrutamiento de red.',
        'Verificar que el comportamiento de tus políticas de enrutamiento coincide con lo que esperas antes de depender de ellas.',
      ],
      after: ['Kongtrol solo enruta tráfico según la configuración que tú defines — no valida legalidad ni idoneidad de esa configuración.'],
    },
    {
      id: 'binarios', num: '5', label: 'Binarios no firmados', tmux: 'binarios', title: 'Binarios no firmados',
      body: [<>Los binarios publicados en GitHub Releases no están firmados con un certificado de firma de código
        pagado. Es normal que Windows SmartScreen o macOS Gatekeeper muestren una advertencia al ejecutarlos — eso
        no implica que contengan malware.</>,
      <>Cada release incluye un checksums.txt (SHA256); verifica el hash antes de ejecutar un binario, o compila
        desde el código fuente si prefieres no confiar en un binario precompilado.</>],
    },
    {
      id: 'privacidad', num: '6', label: 'Privacidad', tmux: 'privacidad', title: 'Privacidad',
      body: [<>Kongtrol no envía telemetría ni analíticas: tu configuración y tus credenciales se quedan en tu
        equipo. Sí hace algunas conexiones propias (por ejemplo, comprobar que tu servidor VPN responde; consultar tu
        IP pública es opcional y viene apagado). El
        detalle completo — qué sale de tu equipo, hacia quién y cómo desactivarlo — está en la{' '}
        <a href={pathFor('privacidad', 'es')}>política de privacidad</a>.</>],
    },
    {
      id: 'cambios', num: '7', label: 'Cambios', tmux: 'cambios', title: 'Cambios a estos términos',
      body: [<>Estos términos pueden actualizarse a medida que el proyecto evoluciona. La versión vigente, con su
        número y fecha, siempre vive en esta página y en el repositorio de GitHub.</>],
    },
  ],
}

const en: LegalDocContent = {
  label: '$ cat TERMS.md',
  title: 'Terms & conditions',
  sub: 'Open-source software, no warranty, use at your own responsibility.',
  version: VERSION,
  updated: UPDATED,
  session: 'kongtrol:terms',
  summaryBadge: 'TL;DR',
  summary: <>Kongtrol is open source ({license}), shipped “as is” with no warranty, and under active beta — use
    good judgment in scenarios where a network leak would have serious consequences.</>,
  toc: 'Contents',
  goto: 'Jump to',
  versionLabel: 'Version',
  contactTitle: 'Questions or issues',
  contact: <>open an issue on the GitHub repository: {issues}. For vulnerabilities, follow the{' '}
    <a href={pathFor('seguridad', 'en')}>security policy</a> (do not open a public issue).</>,
  sections: [
    {
      id: 'que-es', num: '1', label: 'What Kongtrol is', tmux: 'what-is', title: 'What Kongtrol is',
      body: [<>Kongtrol (formerly “VPN Kongtrol”) is open-source software, released under the {license} license,
        that acts as an orchestration layer on top of VPN clients already installed on your machine (OpenVPN,
        WireGuard, ProtonVPN, etc.). It is not a VPN provider, does not operate any servers, and has no access to
        your traffic beyond what runs locally on your own machine. It is a free tool maintained by one person, with
        no company behind it.</>],
    },
    {
      id: 'garantia', num: '2', label: 'No warranty', tmux: 'warranty', title: 'Software provided “as is”',
      callout: 'warn',
      body: [<>Kongtrol is distributed “AS IS”, without warranty of any kind, express or implied, including but
        not limited to warranties of merchantability, fitness for a particular purpose, or non-infringement. In no
        event shall the project maintainers or contributors be liable for any claim, damages, or other liability
        arising from the use of the software, as set out in the Apache 2.0 license terms (sections 7 and 8).</>],
    },
    {
      id: 'beta', num: '3', label: 'Active beta', tmux: 'beta', title: 'Active beta software',
      body: [<>The project is under active testing: not every adapter, network scenario, or operating system is
        fully validated yet. It can fail, behave unexpectedly, or not correctly block a traffic leak in certain
        scenarios (kill switch, DNS guard).</>,
      <>Do not rely on Kongtrol as your only layer of protection in scenarios where an IP or DNS leak would have
        serious legal or security consequences for you.</>],
    },
    {
      id: 'responsabilidad', num: '4', label: 'Your responsibility', tmux: 'liability', title: 'Your responsibility',
      body: ['You are solely responsible for:'],
      list: [
        'Complying with the terms of service of every VPN provider you connect through Kongtrol.',
        'Complying with applicable laws in your jurisdiction regarding VPN use and network routing tools.',
        'Verifying that your routing policies behave as you expect before relying on them.',
      ],
      after: ['Kongtrol only routes traffic according to the configuration you define — it does not validate the legality or suitability of that configuration.'],
    },
    {
      id: 'binarios', num: '5', label: 'Unsigned binaries', tmux: 'binaries', title: 'Unsigned binaries',
      body: [<>Binaries published on GitHub Releases are not signed with a paid code-signing certificate. It is
        normal for Windows SmartScreen or macOS Gatekeeper to show a warning when running them — that does not mean
        they contain malware.</>,
      <>Every release ships a checksums.txt (SHA256); verify the hash before running a binary, or build from source
        if you would rather not trust a prebuilt one.</>],
    },
    {
      id: 'privacidad', num: '6', label: 'Privacy', tmux: 'privacy', title: 'Privacy',
      body: [<>Kongtrol sends no telemetry and no analytics: your configuration and credentials stay on your
        machine. It does make a few connections of its own (for example, checking that your VPN server answers; looking up your public IP is optional and off by default).
        The full detail — what leaves your machine, to whom, and how to turn it off — is in the{' '}
        <a href={pathFor('privacidad', 'en')}>privacy policy</a>.</>],
    },
    {
      id: 'cambios', num: '7', label: 'Changes', tmux: 'changes', title: 'Changes to these terms',
      body: [<>These terms may be updated as the project evolves. The current version, with its number and date,
        always lives on this page and in the GitHub repository.</>],
    },
  ],
}

const doc: Record<Lang, LegalDocContent> = { es, en }
export default doc
