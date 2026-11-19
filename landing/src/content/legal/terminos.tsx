// Terms & conditions — version 1.5.0 · 2026-11-19.
// 1.5.0: plain-language summary in points and shorter sentences (legal meaning unchanged: Apache-2.0
//   "AS IS" and sections 7-8 kept verbatim in substance). "Unsigned binaries" now says what IS
//   signed: from v0.6.0 checksums.txt carries a keyless Sigstore signature and every file a GitHub
//   build provenance attestation, and install.sh/install.ps1 check them (ADR-0035, ADR-0057);
//   Windows has no paid code-signing certificate yet (SignPath pending, ADR-0025) and the macOS app
//   is ad-hoc signed (packaging/macos/build-app.sh), so SmartScreen/Gatekeeper may still warn.
// 1.4.0: the product name is just Kongtrol (formerly VPN Kongtrol, ADR-0047).
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
import IC from '../../components/ui/InlineCode'

const VERSION = '1.5.0'
const UPDATED = '2026-11-19'
const issues = <a href={`${GITHUB_REPO}/issues`} target="_blank" rel="noreferrer">github.com/DerotLuna/vpn-kongtrol/issues</a>
const license = <a href={`${GITHUB_REPO}/blob/main/LICENSE`} target="_blank" rel="noreferrer">Apache 2.0</a>
const verify = (text: string) => <a href={`${GITHUB_REPO}/blob/main/docs/SECURITY.md#verifying-releases`} target="_blank" rel="noreferrer">{text}</a>

const es: LegalDocContent = {
  label: '$ cat TERMS.md',
  title: 'Términos y condiciones',
  sub: 'Software de código abierto, sin garantía, uso bajo tu propia responsabilidad.',
  version: VERSION,
  updated: UPDATED,
  session: 'kongtrol:terminos',
  summaryBadge: 'En resumen',
  summary: [
    <>Kongtrol es gratis y de código abierto, bajo la licencia {license}.</>,
    <>Se entrega «tal cual»: sin garantía de ningún tipo.</>,
    <>Está en beta. Puede fallar, también al bloquear una fuga.</>,
    <>Tú respondes por cómo lo usas y por cumplir las reglas de tus VPN y las leyes de tu país.</>,
    <>No es un proveedor de VPN: organiza las VPN que ya tienes.</>,
  ],
  toc: 'Contenido',
  goto: 'Ir a',
  versionLabel: 'Versión',
  updatedLabel: 'actualizados el',
  contactTitle: 'Preguntas o problemas',
  contact: <>abre un issue en el repositorio de GitHub: {issues}. Para vulnerabilidades, sigue la{' '}
    <a href={pathFor('seguridad', 'es')}>política de seguridad</a> (no abras un issue público).</>,
  sections: [
    {
      id: 'que-es', num: '1', label: 'Qué es Kongtrol', tmux: 'que-es', title: 'Qué es Kongtrol',
      body: [<>Kongtrol (antes «VPN Kongtrol») es software de código abierto, publicado bajo la licencia {license}.
        Organiza los clientes VPN que ya tienes en tu equipo (OpenVPN, WireGuard, ProtonVPN, etc.).</>,
      <>No es un proveedor de VPN y no opera servidores. Solo ve el tráfico que pasa por tu propio equipo. Es una
        herramienta gratuita que mantiene una persona, sin empresa detrás.</>],
    },
    {
      id: 'garantia', num: '2', label: 'Sin garantía', tmux: 'garantia', title: 'Software «tal cual», sin garantía',
      callout: 'warn',
      body: [<>Kongtrol se distribuye «AS IS» (tal cual), sin garantías de ningún tipo, expresas o implícitas,
        incluyendo — sin limitarse a — garantías de comerciabilidad, idoneidad para un propósito particular, o no
        infracción. En ningún caso los mantenedores o colaboradores del proyecto serán responsables por reclamos,
        daños u otras responsabilidades derivadas del uso del software, según los términos de la licencia Apache 2.0
        (secciones 7 y 8).</>,
      <>Dicho de forma simple: si algo sale mal al usarlo, nadie del proyecto lo garantiza ni responde por los
        daños. El texto que vale es el de la licencia.</>],
    },
    {
      id: 'beta', num: '3', label: 'Beta activa', tmux: 'beta', title: 'Software en beta activa',
      body: [<>El proyecto está en pruebas. No todos los adaptadores, redes ni sistemas operativos están
        validados del todo. Puede fallar o comportarse de forma inesperada. En algunos casos, el kill switch o el
        DNS guard pueden no bloquear una fuga.</>,
      <>No uses Kongtrol como tu única protección si una fuga de IP o DNS puede tener consecuencias legales o de
        seguridad graves para ti.</>],
    },
    {
      id: 'responsabilidad', num: '4', label: 'Tu responsabilidad', tmux: 'responsabilidad', title: 'Tu responsabilidad',
      body: ['Eres el único responsable de:'],
      list: [
        'Cumplir los términos de servicio de cada proveedor VPN que uses con Kongtrol.',
        'Cumplir las leyes de tu país sobre el uso de VPN y de herramientas de red.',
        'Comprobar que tus reglas de enrutamiento hacen lo que esperas antes de depender de ellas.',
      ],
      after: ['Kongtrol enruta el tráfico según la configuración que tú defines. No valida si esa configuración es legal o adecuada.'],
    },
    {
      id: 'binarios', num: '5', label: 'Firma de los programas', tmux: 'binarios', title: 'Firma de los programas',
      body: [<>Los programas no llevan todavía un certificado de firma de código de pago en Windows, ni la firma de
        desarrollador de Apple en macOS. Por eso Windows SmartScreen o macOS Gatekeeper pueden mostrar una
        advertencia. Eso no significa que contengan malware.</>,
      <>Cada versión publica <IC c="checksums.txt" /> con la huella SHA-256 de cada archivo. Desde la v0.6.0 ese
        archivo va firmado (Sigstore) y cada archivo lleva una atestación de origen de GitHub. Los scripts de
        instalación comprueban la huella siempre, y la firma si tienes <IC c="cosign" />.{' '}
        {verify('Cómo verificarlo a mano')} (en inglés). También puedes compilarlo desde el código fuente.</>],
    },
    {
      id: 'privacidad', num: '6', label: 'Privacidad', tmux: 'privacidad', title: 'Privacidad',
      body: [<>Kongtrol no envía telemetría ni analíticas. Tu configuración y tus contraseñas se quedan en tu equipo.
        Sí hace algunas conexiones propias, como comprobar que tu servidor VPN responde. Consultar tu IP pública es
        opcional y viene apagado. El detalle completo (qué sale de tu equipo, hacia quién y cómo apagarlo) está en
        la <a href={pathFor('privacidad', 'es')}>política de privacidad</a>.</>],
    },
    {
      id: 'cambios', num: '7', label: 'Cambios', tmux: 'cambios', title: 'Cambios a estos términos',
      body: [<>Estos términos pueden cambiar a medida que el proyecto avanza. La versión vigente, con su número y
        fecha, siempre está en esta página y en el repositorio de GitHub.</>],
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
  summaryBadge: 'In short',
  summary: [
    <>Kongtrol is free and open source, under the {license} license.</>,
    <>It comes “as is”: no warranty of any kind.</>,
    <>It is in beta. It can fail, including when blocking a leak.</>,
    <>You are responsible for how you use it and for following your VPNs' rules and your country's laws.</>,
    <>It is not a VPN provider: it organizes the VPNs you already have.</>,
  ],
  toc: 'Contents',
  goto: 'Jump to',
  versionLabel: 'Version',
  updatedLabel: 'updated',
  contactTitle: 'Questions or issues',
  contact: <>open an issue on the GitHub repository: {issues}. For vulnerabilities, follow the{' '}
    <a href={pathFor('seguridad', 'en')}>security policy</a> (do not open a public issue).</>,
  sections: [
    {
      id: 'que-es', num: '1', label: 'What Kongtrol is', tmux: 'what-is', title: 'What Kongtrol is',
      body: [<>Kongtrol (formerly “VPN Kongtrol”) is open-source software, released under the {license} license.
        It organizes the VPN clients you already have on your computer (OpenVPN, WireGuard, ProtonVPN, etc.).</>,
      <>It is not a VPN provider and runs no servers. It only sees the traffic that goes through your own computer.
        It is a free tool maintained by one person, with no company behind it.</>],
    },
    {
      id: 'garantia', num: '2', label: 'No warranty', tmux: 'warranty', title: 'Software provided “as is”',
      callout: 'warn',
      body: [<>Kongtrol is distributed “AS IS”, without warranty of any kind, express or implied, including but
        not limited to warranties of merchantability, fitness for a particular purpose, or non-infringement. In no
        event shall the project maintainers or contributors be liable for any claim, damages, or other liability
        arising from the use of the software, as set out in the Apache 2.0 license terms (sections 7 and 8).</>,
      <>Put simply: if something goes wrong while you use it, nobody in the project guarantees it or is liable for
        the damage. The license text is what counts.</>],
    },
    {
      id: 'beta', num: '3', label: 'Active beta', tmux: 'beta', title: 'Active beta software',
      body: [<>The project is under testing. Not every adapter, network or operating system is fully validated. It
        can fail or behave unexpectedly. In some cases, the kill switch or DNS guard may not block a leak.</>,
      <>Do not rely on Kongtrol as your only protection if an IP or DNS leak could have serious legal or security
        consequences for you.</>],
    },
    {
      id: 'responsabilidad', num: '4', label: 'Your responsibility', tmux: 'liability', title: 'Your responsibility',
      body: ['You are solely responsible for:'],
      list: [
        'Following the terms of service of every VPN provider you use with Kongtrol.',
        'Following the laws of your country on the use of VPNs and network tools.',
        'Checking that your routing rules do what you expect before relying on them.',
      ],
      after: ['Kongtrol routes traffic according to the configuration you define. It does not check whether that configuration is legal or suitable.'],
    },
    {
      id: 'binarios', num: '5', label: 'Program signing', tmux: 'binaries', title: 'Program signing',
      body: [<>The programs do not carry a paid code-signing certificate on Windows or Apple's developer signature
        on macOS yet. So Windows SmartScreen or macOS Gatekeeper may show a warning. That does not mean they
        contain malware.</>,
      <>Every release publishes <IC c="checksums.txt" /> with the SHA-256 fingerprint of every file. From v0.6.0
        that file is signed (Sigstore) and every file carries a GitHub provenance attestation. The install scripts
        always check the fingerprint, and the signature if you have <IC c="cosign" />.{' '}
        {verify('How to check it by hand')}. You can also build it from source.</>],
    },
    {
      id: 'privacidad', num: '6', label: 'Privacy', tmux: 'privacy', title: 'Privacy',
      body: [<>Kongtrol sends no telemetry and no analytics. Your configuration and passwords stay on your computer.
        It does make a few connections of its own, such as checking that your VPN server answers. Looking up your
        public IP is optional and off by default. The full detail (what leaves your computer, to whom, and how to
        turn it off) is in the <a href={pathFor('privacidad', 'en')}>privacy policy</a>.</>],
    },
    {
      id: 'cambios', num: '7', label: 'Changes', tmux: 'changes', title: 'Changes to these terms',
      body: [<>These terms may change as the project evolves. The current version, with its number and date, always
        lives on this page and in the GitHub repository.</>],
    },
  ],
}

const doc: Record<Lang, LegalDocContent> = { es, en }
export default doc
