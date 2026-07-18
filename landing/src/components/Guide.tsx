import { useEffect, useRef, useState } from 'react'
import { OS } from '../hooks/useOS'
import { Lang } from '../i18n'
import CodeBlock from './CodeBlock'
import GuideEN from './GuideEN'
import { GUIDE_SECTIONS_ES } from '../content/guideSections'

interface Props { os: OS; setOS: (os: OS) => void; lang: Lang }

function OsTabs({ os, setOS }: { os: OS; setOS: (o: OS) => void }) {
  return (
    <div className="os-tabs" style={{ marginBottom: 16 }}>
      {(['windows', 'macos', 'linux'] as OS[]).map(o => (
        <button key={o} className={`os-tab${os === o ? ' active' : ''}`} onClick={() => setOS(o)}>
          {o === 'windows' ? 'Windows' : o === 'macos' ? 'macOS' : 'Linux'}
        </button>
      ))}
    </div>
  )
}

const IC = ({ c }: { c: string }) => <code className="ic">{c}</code>

function GuideES({ os, setOS }: { os: OS; setOS: (os: OS) => void }) {
  const [activeSection, setActiveSection] = useState('prereqs')
  const contentRef = useRef<HTMLDivElement>(null)

  useEffect(() => {
    const observer = new IntersectionObserver(
      entries => {
        entries.forEach(e => {
          if (e.isIntersecting) setActiveSection(e.target.id)
        })
      },
      { rootMargin: '-20% 0px -70% 0px' }
    )
    GUIDE_SECTIONS_ES.forEach(s => {
      const el = document.getElementById(s.id)
      if (el) observer.observe(el)
    })
    return () => observer.disconnect()
  }, [])

  return (
    <section id="guia" style={{ borderTop: '1px solid var(--border)' }}>
      <div className="container">

        {/* ── Cómo funciona (contexto arquitectónico) ── */}
        <div className="guide-intro">
          <span className="guide-intro-badge">Cómo funciona</span>
          <p>
            Kongtrol no es un cliente VPN — es la capa que se sienta entre tus apps y los
            clientes VPN que ya tienes instalados (FortiClient, OpenVPN, ProtonVPN, etc.).
            El <strong>policy engine</strong> mira cada conexión saliente (dominio, IP o app) y
            decide por cuál túnel debe salir; el resto de comandos (<IC c="up" />, <IC c="doctor" />,
            <IC c="status" />) solo controlan ese enrutamiento y el ciclo de vida de los túneles.
          </p>
          <div className="guide-flow">
            <div className="guide-flow-col">
              <h4>tus apps</h4>
              <p>Chrome, Slack, terminal, código…</p>
            </div>
            <div className="guide-flow-arrow" aria-hidden="true">──▶</div>
            <div className="guide-flow-col">
              <h4>kongtrol</h4>
              <p>policy engine — dominios · IPs · apps</p>
            </div>
            <div className="guide-flow-arrow" aria-hidden="true">──▶</div>
            <div className="guide-flow-col guide-flow-vpns">
              <h4>tus VPNs</h4>
              <div className="guide-flow-vpn"><strong>FortiClient</strong><span>office</span></div>
              <div className="guide-flow-vpn"><strong>OpenVPN</strong><span>dev-server</span></div>
              <div className="guide-flow-vpn"><strong>ProtonVPN</strong><span>us-content</span></div>
            </div>
            <div className="guide-flow-footer">kill switch · dns guard · audit log firmado</div>
          </div>
        </div>

        <div className="guide-layout">

          {/* ── Sidebar ── */}
          <aside className="guide-sidebar">
            <div className="guide-nav-title">Contenido</div>
            <ul className="guide-nav-list">
              {GUIDE_SECTIONS_ES.map(s => (
                <li key={s.id}>
                  <a
                    href={`#${s.id}`}
                    className={activeSection === s.id ? 'active' : ''}
                  >
                    <span className="guide-num">{s.num}</span>{s.label}
                  </a>
                </li>
              ))}
            </ul>
          </aside>

          {/* ── Content ── */}
          <div className="guide-content" ref={contentRef}>

            {/* ── Mobile section nav (sidebar is hidden on small screens) ── */}
            <nav className="guide-mobile-nav">
              <span className="guide-mobile-nav-label">Ir a</span>
              <select
                value={activeSection}
                onChange={e => {
                  const id = e.target.value
                  setActiveSection(id)
                  document.getElementById(id)?.scrollIntoView({ behavior: 'smooth' })
                }}
              >
                {GUIDE_SECTIONS_ES.map(s => (
                  <option key={s.id} value={s.id}>{s.num} — {s.label}</option>
                ))}
              </select>
            </nav>

            {/* → Inicio rápido */}
            <div id="quickstart" className="guide-section guide-quickstart">
              <div className="guide-quickstart-head">
                <span className="guide-quickstart-badge">Inicio rápido</span>
                <p>Ya tienes tus VPNs instaladas y funcionando. Kongtrol solo las orquesta — tres comandos y estás operando. Las 10 secciones de abajo son referencia; vuelve a ellas solo si algo no calza.</p>
              </div>
              <ol className="qs-steps">
                <li>
                  <span className="qs-n">1</span>
                  <div>
                    <strong>Registra tus perfiles</strong>
                    <p>Detecta clientes, guarda credenciales en el keychain, arma políticas.</p>
                    <CodeBlock lang="bash">{`$ kongtrol init`}</CodeBlock>
                  </div>
                </li>
                <li>
                  <span className="qs-n">2</span>
                  <div>
                    <strong>Valida el stack</strong>
                    <p>Binarios, certificados, keychain y permisos — antes de conectar.</p>
                    <CodeBlock lang="bash">{`$ kongtrol doctor`}</CodeBlock>
                  </div>
                </li>
                <li>
                  <span className="qs-n">3</span>
                  <div>
                    <strong>Conecta y opera</strong>
                    <p>Levanta un grupo, mira el estado en vivo, abre el panel.</p>
                    <CodeBlock lang="bash">{`$ kongtrol up --group work
$ kongtrol status --watch
$ kongtrol dashboard`}</CodeBlock>
                  </div>
                </li>
              </ol>
            </div>

            {/* 01 — Prerequisitos */}
            <div id="prereqs" className="guide-section">
              <div className="guide-section-num">01</div>
              <h2 className="guide-section-title">Prerequisitos</h2>
              <p>
                Kongtrol <strong>orquesta</strong> los clientes VPN existentes — no los reemplaza.
                Deben estar instalados antes de correr <IC c="kongtrol init" />.
              </p>

              <div className="table-scroll">
              <table className="data-table">
                <thead>
                  <tr><th>VPN</th><th>Cliente requerido</th><th>Verificar</th></tr>
                </thead>
                <tbody>
                  <tr>
                    <td>FortiClient</td>
                    <td>FortiClient 6.4.x</td>
                    <td>
                      Abrir FortiClient — debe aparecer al menos una conexión guardada en la lista.
                      Si puedes conectarte manualmente desde la GUI, está listo.
                    </td>
                  </tr>
                  <tr>
                    <td>OpenVPN</td>
                    <td>OpenVPN Community</td>
                    <td>
                      {os === 'windows'
                        ? <>CLI o GUI según instalador. Community: <IC c="openvpn --version" /> en terminal (agrega al PATH). OpenVPN Connect: solo GUI — Kongtrol lo detecta automáticamente.</>
                        : <IC c="openvpn --version" />}
                    </td>
                  </tr>
                  <tr>
                    <td>WireGuard</td>
                    <td>wireguard-tools</td>
                    <td><IC c="wg --version" /></td>
                  </tr>
                  <tr>
                    <td>ProtonVPN</td>
                    <td>{os === 'windows' ? 'ProtonVPN app (GUI) + WireGuard' : 'protonvpn-cli o WireGuard'}</td>
                    <td>
                      {os === 'windows'
                        ? <>GUI manual: abrir ProtonVPN y comprobar conexión. Automático: WireGuard + <IC c="wg --version" />.</>
                        : <><IC c="protonvpn-cli --version" /> o <IC c="wg --version" /> si usarás perfil WireGuard.</>}
                    </td>
                  </tr>
                  <tr>
                    <td>Cloudflare WARP</td>
                    <td>warp-cli</td>
                    <td><IC c="warp-cli --version" /></td>
                  </tr>
                </tbody>
              </table>
              </div>

              {os === 'windows' && (
                <div className="callout">
                  <strong>Nota:</strong> si el wizard muestra ✓ para un cliente, está detectado y listo — no necesitas verificar manualmente en terminal.
                </div>
              )}

              <div className="callout" style={{ marginTop: 12 }}>
                <strong>Antes de conectar:</strong> cierra las GUIs de los clientes VPN.
                Kongtrol llama directamente al CLI de cada uno — dos instancias kongtrolando
                el mismo túnel entran en conflicto.
              </div>

              <div className="table-scroll" style={{ marginTop: 24 }}>
              <table className="data-table">
                <thead>
                  <tr><th>Comando</th><th>Estado requerido de los clientes</th></tr>
                </thead>
                <tbody>
                  <tr><td>kongtrol init</td><td>Cualquiera — solo lee archivos y keychain</td></tr>
                  <tr><td>kongtrol doctor</td><td>Cualquiera — solo valida, no conecta</td></tr>
                  <tr><td>kongtrol up &lt;perfil&gt;</td><td>GUI del cliente <strong>cerrada</strong></td></tr>
                </tbody>
              </table>
              </div>
            </div>

            {/* 02 — Instalación */}
            <div id="install" className="guide-section">
              <div className="guide-section-num">02</div>
              <h2 className="guide-section-title">Instalación</h2>
              <OsTabs os={os} setOS={setOS} />

              {os === 'windows' && (
                <div className="os-panel active">
                  <p>
                    1. Descarga <IC c="kongtrol_windows_amd64.zip" /> desde{' '}
                    <a href="/#install">la sección de descargas</a>.
                  </p>
                  <p>
                    2. Extrae el ZIP. Verás un archivo <IC c="kongtrol.exe" />.
                    Muévelo a una carpeta fija — por ejemplo crea{' '}
                    <IC c="C:\tools\" /> si no tienes una:
                  </p>
                  <CodeBlock lang="powershell">{`# Crea la carpeta (si no existe) y mueve el binario
New-Item -ItemType Directory -Force "C:\\tools"
Move-Item kongtrol.exe C:\\tools\\kongtrol.exe`}</CodeBlock>
                  <p>
                    3. Agrega <IC c="C:\tools" /> al PATH para poder correr{' '}
                    <IC c="kongtrol" /> desde cualquier terminal:
                  </p>
                  <CodeBlock lang="powershell">{`# Ejecuta esto en PowerShell como Administrador, una sola vez:
[Environment]::SetEnvironmentVariable(
  "Path",
  $env:Path + ";C:\\tools",
  "Machine"
)`}</CodeBlock>
                  <p>4. Abre una terminal <strong>nueva</strong> y verifica:</p>
                  <CodeBlock lang="powershell">{`kongtrol --help`}</CodeBlock>
                  <div className="callout">
                    <strong>¿Por qué una terminal nueva?</strong> Los cambios al PATH solo
                    aplican en terminales abiertas después del cambio — las que ya estaban
                    abiertas no lo ven.
                  </div>
                </div>
              )}

              {os === 'macos' && (
                <div className="os-panel active">
                  <CodeBlock lang="bash">{`# Apple Silicon (M1+)
$ curl -L https://github.com/vpn-kongtrol/kongtrol/releases/latest/download/kongtrol_darwin_arm64.tar.gz | tar xz
$ sudo mv kongtrol /usr/local/bin/

# Intel
$ curl -L https://github.com/vpn-kongtrol/kongtrol/releases/latest/download/kongtrol_darwin_amd64.tar.gz | tar xz
$ sudo mv kongtrol /usr/local/bin/

# Verificar
$ kongtrol --help`}</CodeBlock>
                  <div className="callout">
                    macOS puede pedir confirmación la primera vez que corres el binario.
                    Ve a <strong>Ajustes del sistema → Privacidad y seguridad</strong> y haz clic en "Permitir".
                  </div>
                </div>
              )}

              {os === 'linux' && (
                <div className="os-panel active">
                  <CodeBlock lang="bash">{`$ curl -L https://github.com/vpn-kongtrol/kongtrol/releases/latest/download/kongtrol_linux_amd64.tar.gz | tar xz
$ sudo mv kongtrol /usr/local/bin/

# Verificar
$ kongtrol --help`}</CodeBlock>
                  <p>
                    Kongtrol necesita permisos de red en Linux. Algunos comandos (kill switch,
                    modificar rutas) requieren <IC c="sudo" /> o capabilities.
                  </p>
                </div>
              )}
            </div>

            {/* 03 — Wizard */}
            <div id="wizard" className="guide-section">
              <div className="guide-section-num">03</div>
              <h2 className="guide-section-title">Registrar tus VPNs: kongtrol init</h2>
              <p>
                <IC c="kongtrol init" /> <strong>no configura tus VPNs</strong> — eso ya lo
                tienes hecho. Lo que hace es decirle a Kongtrol dónde están y cómo hablarles:
                qué cliente usar, qué tunnel name tiene, dónde está el <IC c=".ovpn" />,
                y guarda las credenciales en el <strong>keychain del OS</strong> (nunca en el YAML).
              </p>
              <p style={{ marginTop: 8 }}>
                Si ya tienes FortiClient con un túnel funcionando y OpenVPN con su{' '}
                <IC c=".ovpn" />, este paso solo recopila esa información.
              </p>

              <h3 style={{ marginBottom: 8, marginTop: 24 }}>Archivos de conexión</h3>
              <p>
                <strong>Si ya tienes tus VPNs configuradas, no necesitas hacer nada especial aquí.</strong>{' '}
                Kongtrol apunta a los archivos donde ya están — en el wizard simplemente
                escribes la ruta actual de cada archivo. Solo hace falta actuar si recibes
                archivos nuevos de un compañero o de IT y no tienes dónde guardarlos, o si
                quieres reorganizar todo en un directorio único.
              </p>

              <div className="table-scroll" style={{ marginTop: 12 }}>
              <table className="data-table">
                <thead>
                  <tr><th>Adaptador</th><th>Qué necesita el wizard</th><th>¿Tienes que hacer algo?</th></tr>
                </thead>
                <tbody>
                  <tr>
                    <td>FortiClient</td>
                    <td>{os === 'windows' ? 'Nombre del túnel + usuario/contraseña' : 'Ruta al cert, ruta a la key, credenciales'}</td>
                    <td>
                      {os === 'windows'
                        ? <>No. El túnel ya está en la GUI de FortiClient — Kongtrol lo activa por nombre. Sin certs.</>
                        : <>Tener el <IC c=".crt" /> y <IC c=".key" /> en alguna ruta accesible y anotarla.</>}
                    </td>
                  </tr>
                  <tr>
                    <td>OpenVPN</td>
                    <td>Ruta al <IC c=".ovpn" /></td>
                    <td>
                      {os === 'windows'
                        ? <>No. Usa el <IC c=".ovpn" /> donde ya está (ej. <IC c="C:\Users\TU\OpenVPN\config\" />).</>
                        : <>No, si ya tienes el <IC c=".ovpn" />. Solo anotar su ruta actual.</>}
                    </td>
                  </tr>
                  <tr>
                    <td>ProtonVPN</td>
                    <td>Modo GUI: credenciales Proton. Modo automático: archivo <IC c=".conf" /> de WireGuard.</td>
                    <td>GUI: no archivos. WireGuard: exportar config desde Proton y guardar la ruta.</td>
                  </tr>
                  <tr>
                    <td>WireGuard / otros</td>
                    <td>Ruta al archivo de config</td>
                    <td>No, si ya tienes el archivo. Solo anotar su ruta.</td>
                  </tr>
                </tbody>
              </table>
              </div>

              <div className="callout">
                <strong>Organización opcional:</strong> si quieres centralizar, puedes crear{' '}
                <IC c={os === 'windows' ? '%USERPROFILE%\\.kongtrol\\certs\\' : '~/.kongtrol/certs/'} />{' '}
                y copiar los archivos ahí. Útil al incorporar un compañero nuevo o al recibir
                certs frescos. No es requerido si ya tienes todo en su lugar.
              </div>

              <h3 style={{ marginBottom: 8, marginTop: 28 }}>Ejecutar el wizard</h3>

              <CodeBlock lang="bash">{`$ kongtrol init`}</CodeBlock>

              <p>
                Primero seleccionas el idioma, luego el wizard muestra los clientes detectados
                y te pregunta si quieres agregar un perfil:
              </p>

              <CodeBlock lang="text">{`▸  Clientes VPN detectados en este sistema:
    ✓  FortiClient             6.4.10.1821
    ✓  OpenVPN                 2.6.8
    ✓  ProtonVPN               4.3.14

¿Agregar un nuevo perfil VPN? [s/N]: s`}</CodeBlock>

              <p>
                El wizard usa <strong>menús numerados</strong> donde aplica — no tienes que
                recordar los valores válidos:
              </p>

              <CodeBlock lang="text">{`  Nombre del perfil: office

  Tipo de adaptador:
    1) forticlient         ✓ detectado  FortiClient SSL VPN
    2) openvpn             ✓ detectado  OpenVPN (archivo .ovpn)
    3) protonvpn           ✓ detectado  ProtonVPN (cuenta Proton)
    ───────────────────────────────
    4) ciscoanyconnect                  Cisco AnyConnect
    5) wireguard                        WireGuard (archivo .conf)
    6) globalprotect                    Palo Alto GlobalProtect
    7) tailscale                        Tailscale mesh / exit node
    8) cloudflarewarp                   Cloudflare WARP

  Elige [1]: 1`}</CodeBlock>

              <p>Cada campo tiene una pista de dónde encontrar el valor:</p>

              <details className="guide-more">
                <summary>Ver ejemplo del flujo por campos (FortiClient)</summary>
              <CodeBlock lang="text">{os === 'windows'
? `    Encuéntralo en FortiClient > Ajustes o pídelo a IT. Ej: vpn.empresa.com
  Host VPN: vpn.tuempresa.com

    443 para SSL VPN (default). Cambia solo si IT indica otro puerto.
  Puerto [443]:

    El nombre exacto de la conexión tal como aparece en la lista de FortiClient GUI.
  Nombre del túnel [Office]: Office

  Versión de FortiClient:
    1) 6  6.4.x — más común
    2) 7  7.x
    3) 5  5.x — legacy
  Elige [1]:

    En Windows FortiClient conecta por nombre de túnel — sin certificado. Se usará 'credentials'.
  Usuario: tu_usuario
  Contraseña para office (guardada en llavero): ****`
: `    Encuéntralo en FortiClient > Ajustes o pídelo a IT. Ej: vpn.empresa.com
  Host VPN: vpn.tuempresa.com

    443 para SSL VPN (default). Cambia solo si IT indica otro puerto.
  Puerto [443]:

    El nombre exacto de la conexión tal como aparece en la lista de FortiClient GUI.
  Nombre del túnel [Office]: Office

  Versión de FortiClient:
    1) 6  6.4.x — más común
    2) 7  7.x
    3) 5  5.x — legacy
  Elige [1]:

  Método de autenticación:
    1) credentials              usuario y contraseña
    2) certificate              solo certificado cliente
    3) certificate+credentials  certificado + usuario y contraseña (más seguro)
  Elige [3]:

  Ruta al certificado cliente: ~/.kongtrol/certs/office.crt
  Ruta a la clave privada: ~/.kongtrol/certs/office.key
  Usuario: tu_usuario
  Contraseña para office (guardada en llavero): ****`}</CodeBlock>
              </details>

              <p style={{ marginTop: 20 }}>
                Para OpenVPN, el wizard apunta a tu archivo <IC c=".ovpn" /> existente:
              </p>

              <details className="guide-more">
                <summary>Ver ejemplo del flujo por campos (OpenVPN)</summary>
              <CodeBlock lang="text">{os === 'windows'
? `  Nombre del perfil: dev-server

  Tipo de adaptador:
    ...
  Elige [2]: 2   ← openvpn

    Ruta completa al .ovpn. Usa la ubicación donde ya está, no hace falta copiar.
  Ruta al archivo .ovpn: C:\\Users\\TU\\OpenVPN\\config\\server.ovpn

    Si el .ovpn tiene <cert> y <key> embebidas, elige 'certificate' y deja los paths en blanco.
  Método de autenticación:
    1) credentials              usuario y contraseña
    2) certificate              solo certificado cliente
    3) certificate+credentials  certificado + usuario y contraseña (más seguro)
  Elige [2]:

  Ruta al certificado cliente (vacío si está dentro del .ovpn):
  Ruta a la clave privada (vacío si está dentro del .ovpn):`
: `  Nombre del perfil: dev-server

  Tipo de adaptador:
    ...
  Elige [2]: 2   ← openvpn

    Ruta completa al .ovpn. Usa la ubicación donde ya está, no hace falta copiar.
  Ruta al archivo .ovpn: ~/.config/openvpn/server.ovpn

    Si el .ovpn tiene <cert> y <key> embebidas, elige 'certificate' y deja los paths en blanco.
  Método de autenticación:
    1) credentials              usuario y contraseña
    2) certificate              solo certificado cliente
    3) certificate+credentials  certificado + usuario y contraseña (más seguro)
  Elige [2]:

  Ruta al certificado cliente (vacío si está dentro del .ovpn):
  Ruta a la clave privada (vacío si está dentro del .ovpn):`}</CodeBlock>
              </details>

              <h3 style={{ marginBottom: 8, marginTop: 24 }}>ProtonVPN: GUI manual vs WireGuard automático</h3>
              <p>
                Tienes dos caminos válidos con ProtonVPN según cuánto quieras automatizar.
              </p>
              <ul style={{ marginTop: 4, paddingLeft: 20, lineHeight: 1.8 }}>
                <li>
                  <strong>GUI manual (rápido):</strong> el usuario opera Proton desde su app y Kongtrol
                  enruta políticas sobre ese perfil.
                </li>
                <li>
                  <strong>WireGuard automático (recomendado):</strong> exporta un archivo <IC c=".conf" /> desde Proton,
                  crea un perfil <IC c="type: wireguard" /> y Kongtrol puede levantar/bajar la conexión por comando.
                </li>
              </ul>

              <CodeBlock lang="yaml">{`vpns:
  us-content:
    type: wireguard
    config: ~/.kongtrol/configs/proton-us.conf
    auth:
      method: certificate`}</CodeBlock>

              <div className="callout" style={{ marginTop: 12 }}>
                <strong>ProtonVPN junto a otros adapters (Forti/OpenVPN/etc.):</strong> úsalo en un
                perfil separado y enruta solo dominios/IPs objetivo (ej. streaming). Mantén
                tráfico corporativo en sus perfiles dedicados para evitar conflictos de ruta.
              </div>

              <div className="callout">
                <strong>¿Dónde encuentro el Host y el Tunnel Name de FortiClient?</strong><br />
                Abre FortiClient → pestaña <strong>Remote Access</strong>. Verás una lista de conexiones guardadas.
                El <strong>nombre de la conexión</strong> (ej. "Office VPN") es el <em>tunnel name</em>.
                Haz clic en el ícono de lápiz (editar) junto a esa conexión — ahí verás el campo{' '}
                <strong>Remote Gateway</strong> o <strong>Server</strong>: ese es el <em>host</em>.
              </div>

              <div className="callout" style={{ marginTop: 12 }}>
                Si un cliente no fue detectado automáticamente, el wizard te avisa y pide
                la ruta al binario. También puedes ponerla después en el YAML con <IC c="binary_path" />.
              </div>

              <p>Al final, opciones de seguridad, políticas de routing y confirmación:</p>

              <details className="guide-more">
                <summary>Ver flujo de seguridad y políticas</summary>
              <CodeBlock lang="text">{`¿Activar kill switch? [S/n]:
¿Activar DNS guard? [S/n]:
¿Activar log de auditoría firmado? [S/n]:
¿Activar panel web? (http://127.0.0.1:9741) [S/n]:

── Políticas de enrutamiento ──────────────────────────────────────────────

¿Agregar una política de enrutamiento? [S/n]: s

  Nombre de la política (ej: trabajo, streaming): Claude AI
  Perfil VPN para esta política:
    1) office
    2) us-content
  Elige [1]: 2

    Dominio o sufijo (ej: empresa.com, .internal) — deja en blanco para terminar
  Dominio: claude.ai
  Dominio: *.anthropic.com
  Dominio:

    Rango IP o IP (ej: 10.0.0.0/8) — deja en blanco para terminar
  IP / rango:

¿Agregar una política de enrutamiento? [S/n]: s

  Nombre de la política (ej: trabajo, streaming): Red interna
  Perfil VPN para esta política:
    1) office
    2) us-content
  Elige [1]: 1

  Dominio:

  IP / rango: 10.10.0.0/16
  IP / rango: 192.168.50.0/24
  IP / rango:

¿Agregar una política de enrutamiento? [S/n]:

    También puedes editar políticas directamente en kongtrol.yaml → sección 'policies'.

¿Escribir configuración en ~/.kongtrol/kongtrol.yaml? [S/n]: s

[✓] Configuración escrita en ~/.kongtrol/kongtrol.yaml
[✓] Configuración válida.`}</CodeBlock>
              </details>

              <div className="callout">
                <strong>Puedes escribir varios dominios separados por coma</strong> en un solo prompt —
                el wizard los divide automáticamente. <IC c="claude.ai, *.anthropic.com" /> → dos entradas.
              </div>

              <div className="callout" style={{ marginTop: 12 }}>
                Para agregar otro perfil o política después: vuelve a correr <IC c="kongtrol init" />.
                El wizard detecta el config existente y no sobreescribe nada sin confirmación.
              </div>
            </div>

            {/* 04 — Grupos */}
            <div id="groups" className="guide-section">
              <div className="guide-section-num">04</div>
              <h2 className="guide-section-title">Grupos</h2>
              <p>
                Agrupan perfiles para levantarlos juntos con un solo comando.
                Edita <IC c={os === 'windows' ? 'C:\\Users\\TU\\.kongtrol\\kongtrol.yaml' : '~/.kongtrol/kongtrol.yaml'} /> y añade al final:
              </p>

              <CodeBlock lang="yaml">{`groups:
  work:
    profiles: [office, dev-server, aws]

  travel:
    profiles: [us-content]

  full:
    profiles: [office, dev-server, aws, us-content]`}</CodeBlock>

              <CodeBlock lang="bash">{`kongtrol up --group work     # levanta office + dev-server + aws en paralelo
kongtrol down --group work`}</CodeBlock>

              <div className="callout">
                Un perfil puede pertenecer a varios grupos a la vez — <IC c="office" /> puede
                estar en <IC c="work" /> y también en <IC c="full" /> sin conflicto.
              </div>
            </div>

            {/* 05 — Políticas de routing */}
            <div id="policies" className="guide-section">
              <div className="guide-section-num">05</div>
              <h2 className="guide-section-title">Políticas de routing</h2>

              <p>
                Cuando tienes varios túneles activos al mismo tiempo, ¿cómo sabe tu computadora
                qué tráfico va por cuál VPN? Eso lo definen las <strong>políticas</strong>.
              </p>

              <CodeBlock lang="text">{`  petición                    política                          túnel
  ──────────────────────────────────────────────────────────────────────────
  claude.ai            ──▶   domains: claude.ai            ──▶  us-content (ProtonVPN)
  10.10.4.12           ──▶   ip_ranges: 10.10.0.0/16        ──▶  office     (FortiClient)
  api.tuempresa.com    ──▶   sin coincidencia               ──▶  conexión directa (sin VPN)`}</CodeBlock>

              <p>
                Sin políticas, cada VPN instala sus propias rutas al conectarse — el último
                en conectar suele ganar el tráfico general, y no puedes kongtrolar qué va por dónde.
                Con políticas, Kongtrol enruta de forma precisa:
              </p>

              <ul style={{ marginBottom: 16, paddingLeft: 20 }}>
                <li><IC c="claude.ai" /> y <IC c="*.anthropic.com" /> → van por <IC c="us-content" /> (ProtonVPN)</li>
                <li><IC c="10.10.0.0/16" /> (red de oficina) → van por <IC c="office" /> (FortiClient)</li>
                <li><IC c="*.tuempresa.com" /> → van por <IC c="office" /></li>
                <li>Todo lo demás → conexión directa sin VPN</li>
              </ul>

              <p>
                El wizard (<IC c="kongtrol init" />) te permite crear políticas al final del flujo.
                También puedes agregarlas o editarlas directamente en el YAML bajo la sección <IC c="policies:" />:
              </p>

              <CodeBlock lang="yaml">{`policies:

  # ── Contenido US / Claude AI ───────────────────────────────────────────
  - name: Claude AI
    match:
      domains:
        - claude.ai
        - "*.anthropic.com"
    via: us-content

  - name: Contenido geo-restringido
    match:
      domains:
        - "*.netflix.com"
        - "*.hulu.com"
        - "*.disneyplus.com"
    via: us-content

  # ── Oficina ────────────────────────────────────────────────────────────
  - name: Red interna de oficina
    match:
      ip_ranges:
        - 10.10.0.0/16          # red interna — ajusta a la tuya
        - 192.168.50.0/24
    via: office

  - name: Dominios internos
    match:
      domains:
        - "*.tuempresa.com"     # cambia al dominio de tu empresa
        - intranet.local
    via: office

  # ── Servidor de desarrollo ─────────────────────────────────────────────
  - name: Servidor dev
    match:
      ip_ranges:
        - 185.0.0.0/32          # reemplaza con la IP real de tu servidor
    via: dev-server

  # ── AWS ────────────────────────────────────────────────────────────────
  - name: AWS workloads
    match:
      ip_ranges:
        - 172.31.0.0/16         # VPC default de AWS — ajusta a tus CIDRs
      domains:
        - "*.amazonaws.com"
    via: aws

  # El tráfico que no coincide con ninguna regla va por tu
  # conexión normal (sin VPN). Para forzar TODO por una VPN:
  # - name: Default
  #   match:
  #     ip_ranges: [0.0.0.0/0]
  #   via: office`}</CodeBlock>

              <div className="callout">
                <strong>¿Cómo funciona el matching?</strong><br />
                Dominios: glob exacto o con prefijo <IC c="*." /> — <IC c="*.empresa.com" /> cubre todos los subdominios pero no <IC c="empresa.com" /> solo (agrega ambos si los necesitas).<br />
                IPs: longest-prefix match — gana la regla más específica. <IC c="10.10.1.0/24" /> tiene prioridad sobre <IC c="10.0.0.0/8" />.
              </div>

              <div className="callout" style={{ marginTop: 12 }}>
                <strong>Flow-aware policy:</strong> puedes combinar app + dominio/IP en la misma regla.
                Si una regla define <IC c="apps" /> y también <IC c="domains" /> / <IC c="ip_ranges" />,
                ambas condiciones deben coincidir para enrutar por ese perfil.
              </div>

              <div className="callout" style={{ marginTop: 12 }}>
                <strong>Split-DNS transparente:</strong> con <IC c="monitor.split_dns.enabled: true" />,
                Kongtrol inyecta dominios de policy en resolución del sistema (hosts) para que apps
                normales resuelvan por el túnel correcto sin llamar la API.
              </div>

              <CodeBlock lang="bash">{`# Validar después de editar
$ kongtrol config validate
# [OK] Config is valid.`}</CodeBlock>

              <div className="callout">
                Las políticas se leen al conectar — no necesitas reiniciar nada para que los cambios tomen efecto.
              </div>
            </div>

            {/* 06 — Doctor */}
            <div id="doctor" className="guide-section">
              <div className="guide-section-num">06</div>
              <h2 className="guide-section-title">Verificar con doctor</h2>
              <p>
                Antes de conectar por primera vez, deja que Kongtrol valide que todo esté en orden:
              </p>

              <CodeBlock lang="bash">{`$ kongtrol doctor`}</CodeBlock>

              <CodeBlock lang="text">{os === 'windows'
? `Kongtrol Doctor
────────────────────────────────────────────────────

  Configuration
  ✓  config file         C:\\Users\\tu\\.kongtrol\\kongtrol.yaml
  ✓  config valid        4 profile(s) defined

  VPN Binaries
  ✓  forticlient         FortiClient 6.4.10.1821
  ✓  openvpn             OpenVPN 2.6.8
  ✓  protonvpn           ProtonVPN 4.3.14

  Certificates & Keys
  ✓  office: cert        C:\\Users\\tu\\.kongtrol\\certs\\office.crt
  ✓  office: key         C:\\Users\\tu\\.kongtrol\\certs\\office.key
  ✓  dev-server: config  C:\\Users\\tu\\.kongtrol\\configs\\server.ovpn

  Keychain Credentials
  ✓  office: password    found in OS keychain (Credential Manager)
  ✓  us-content          found in OS keychain (Credential Manager)

  Permissions
  ✓  kill switch         disponible
  ✓  dns guard           disponible

All checks passed. You're good to go.`
: `Kongtrol Doctor
────────────────────────────────────────────────────

  Configuration
  ✓  config file         ~/.kongtrol/kongtrol.yaml
  ✓  config valid        4 profile(s) defined

  VPN Binaries
  ✓  forticlient         FortiClient 6.4.10.1821
  ✓  openvpn             OpenVPN 2.6.8
  ✓  protonvpn           protonvpn-cli 4.3.14

  Certificates & Keys
  ✓  office: cert        ~/.kongtrol/certs/office.crt
  ✓  office: key         ~/.kongtrol/certs/office.key
  ✓  dev-server: config  ~/.kongtrol/configs/server.ovpn

  Keychain Credentials
  ✓  office: password    found in OS keychain
  ✓  us-content          found in OS keychain

  Permissions
  ✓  kill switch         disponible
  ✓  dns guard           disponible

All checks passed. You're good to go.`}</CodeBlock>

              <p>Si hay ✗ en alguna línea, el mensaje dice exactamente qué falta.</p>
            </div>

            {/* 07 — Primera conexión */}
            <div id="connect" className="guide-section">
              <div className="guide-section-num">07</div>
              <h2 className="guide-section-title">Primera conexión</h2>

              <p><strong>Un grupo completo:</strong></p>
              <CodeBlock lang="bash">{`$ kongtrol up --group work
[+] office      conectado
[+] dev-server  conectado
[+] aws         conectado`}</CodeBlock>

              <p><strong>Varios perfiles sin grupo:</strong></p>
              <CodeBlock lang="bash">{`$ kongtrol up office dev-server aws`}</CodeBlock>

              <p>Los tres túneles se levantan en paralelo. El routing policy determina qué tráfico va por cuál.</p>

              <p><strong>Un solo perfil:</strong></p>
              <CodeBlock lang="bash">{`$ kongtrol up office
$ kongtrol up us-content`}</CodeBlock>

              <p>Kongtrol se queda corriendo en primer plano. Cuando termines:</p>
              <CodeBlock lang="bash">{`# Ctrl+C  →  desconecta todo limpiamente

# O desde otra terminal:
$ kongtrol down --group work
$ kongtrol down --all`}</CodeBlock>

              <p><strong>Ver estado:</strong></p>
              <CodeBlock lang="bash">{`$ kongtrol status

PROFILE      STATUS        IP           UPTIME
office       connected     10.10.0.5    1h 23m
dev-server   connected     185.x.x.x    1h 23m
aws          connected     172.31.4.7   1h 23m
us-content   disconnected  —            —

Kill Switch: ON

# Vista en tiempo real (refresca cada 2s):
$ kongtrol status --watch`}</CodeBlock>
            </div>

            {/* 08 — Dashboard */}
            <div id="dashboard" className="guide-section">
              <div className="guide-section-num">08</div>
              <h2 className="guide-section-title">Dashboard web</h2>

              <CodeBlock lang="bash">{`$ kongtrol dashboard
# Dashboard running at http://127.0.0.1:9741`}</CodeBlock>

              <p>
                Abre <a href="http://localhost:9741" target="_blank" rel="noreferrer">http://localhost:9741</a> en
                tu navegador. Es una consola de gestión completa, con navegación lateral:
              </p>

              <div className="table-scroll">
              <table className="data-table">
                <thead><tr><th>Página</th><th>Qué hace</th></tr></thead>
                <tbody>
                  <tr><td>Overview</td><td>Túneles en vivo, gráficos de tráfico por túnel, rutas activas, resolución de policy, conectar/desconectar todo</td></tr>
                  <tr><td>Policy Studio</td><td>CRUD de políticas de routing + probador de reglas antes de guardar</td></tr>
                  <tr><td>Security</td><td>Activar/desactivar Kill Switch y DNS Guard en vivo, override de kill switch por perfil, estado de leak check</td></tr>
                  <tr><td>VPN Profiles</td><td>CRUD de perfiles VPN y grupos (config + keychain; requiere reiniciar el daemon para activarse)</td></tr>
                  <tr><td>Audit Log</td><td>Eventos de auditoría filtrables por perfil/nivel, con indicador de firma HMAC</td></tr>
                  <tr><td>Settings</td><td>Health check, scheduler + reglas, split DNS, ajustes de kill switch/DNS guard, audit log</td></tr>
                </tbody>
              </table>
              </div>

              <CodeBlock lang="bash">{`# Endpoints útiles
GET  /api/v1/metrics/history
GET  /api/v1/dns/resolve?domain=claude.ai&via=us-content
GET  /api/v1/resolve?target=portal.empresa.com&app=chrome.exe
POST /api/v1/security/killswitch   {"enabled": true}
POST /api/v1/security/dnsguard     {"enabled": true}
GET  /api/v1/vpns                  # perfiles · POST/PUT/DELETE para CRUD
GET  /api/v1/groups                # grupos · POST/PUT/DELETE + /connect /disconnect
GET  /api/v1/settings              # PUT para guardar
GET  /api/v1/audit                 # ?profile=&level=&limit=`}</CodeBlock>

              <div className="callout">
                El dashboard está compilado dentro del binario — no necesitas instalar nada
                extra. Es gestión completa (crear/editar perfiles, políticas, grupos, activar
                kill switch en vivo), no solo lectura. El puerto/bind del propio dashboard
                sigue siendo CLI-only (<code>kongtrol config dashboard set-port &lt;puerto&gt;</code>)
                — cambiarlo desde la página que sirve la petición rompería la conexión.
              </div>
            </div>

            {/* 09 — Uso diario */}
            <div id="daily" className="guide-section">
              <div className="guide-section-num">09</div>
              <h2 className="guide-section-title">Uso diario</h2>

              <div className="table-scroll">
              <table className="data-table">
                <thead><tr><th>Comando</th><th>Qué hace</th></tr></thead>
                <tbody>
                  <tr><td>kongtrol up --group work</td><td>Empezar el día (office + dev-server + aws)</td></tr>
                  <tr><td>kongtrol down --group work</td><td>Terminar el día</td></tr>
                  <tr><td>kongtrol up us-content</td><td>Netflix, Hulu, Claude AI...</td></tr>
                  <tr><td>kongtrol down --all</td><td>Apagar todo</td></tr>
                  <tr><td>kongtrol status</td><td>Ver qué está conectado</td></tr>
                  <tr><td>kongtrol status --watch</td><td>Monitoreo en vivo (2s)</td></tr>
                  <tr><td>kongtrol status --watch --dashboard</td><td>Monitoreo en vivo + levanta el dashboard embebido a la vez</td></tr>
                  <tr><td>kongtrol check</td><td>Test de leaks ahora mismo</td></tr>
                  <tr><td>kongtrol dashboard</td><td>Abrir UI web</td></tr>
                  <tr><td>kongtrol doctor</td><td>Diagnóstico completo</td></tr>
                  <tr><td>kongtrol export</td><td>Exportar config sin contraseñas (para compartir)</td></tr>
                </tbody>
              </table>
              </div>

              <p style={{ marginTop: 20 }}><strong>Scheduler opcional por horario:</strong></p>
              <CodeBlock lang="yaml">{`monitor:
  scheduler:
    enabled: true
    interval: "1m"
    rules:
      - name: "work-hours"
        profiles: ["office", "aws"]
        weekdays: ["mon","tue","wed","thu","fri"]
        start: "09:00"
        end: "18:00"`}</CodeBlock>

              <p style={{ marginTop: 16 }}><strong>Kill switch por perfil (override):</strong></p>
              <CodeBlock lang="yaml">{`vpns:
  office:
    kill_switch: true
  us-content:
    kill_switch: false`}</CodeBlock>

              <p style={{ marginTop: 24 }}><strong>Actualizar una contraseña:</strong></p>
              <CodeBlock lang="bash">{`$ kongtrol init
# → elige el perfil → "¿Actualizar credenciales?" → ingresa la nueva`}</CodeBlock>
            </div>

            {/* 10 — Troubleshooting */}
            <div id="trouble" className="guide-section">
              <div className="guide-section-num">10</div>
              <h2 className="guide-section-title">Solución de problemas</h2>

              <p><strong>DNS no restaurado después de un crash:</strong></p>
              <OsTabs os={os} setOS={setOS} />

              {os === 'windows' && (
                <CodeBlock lang="bash">{`$ netsh interface ip set dns "Ethernet" dhcp`}</CodeBlock>
              )}
              {os === 'macos' && (
                <CodeBlock lang="bash">{`$ networksetup -setdnsservers Wi-Fi empty`}</CodeBlock>
              )}
              {os === 'linux' && (
                <CodeBlock lang="bash">{`$ sudo cp /etc/resolv.conf.kongtrol.bak /etc/resolv.conf`}</CodeBlock>
              )}

              <p style={{ marginTop: 32 }}><strong>Kill switch activo sin internet después de desconectar:</strong></p>
              <CodeBlock lang="bash">{`$ kongtrol down --all    # desactiva kill switch automáticamente`}</CodeBlock>
              <p>Si Kongtrol se cerró abruptamente:</p>
              <OsTabs os={os} setOS={setOS} />
              {os === 'windows' && (
                <CodeBlock lang="bash">{`# Como Administrador:
$ netsh advfirewall reset`}</CodeBlock>
              )}
              {os === 'macos' && (
                <CodeBlock lang="bash">{`$ sudo pfctl -d`}</CodeBlock>
              )}
              {os === 'linux' && (
                <CodeBlock lang="bash">{`$ sudo iptables -F OUTPUT`}</CodeBlock>
              )}

              <p style={{ marginTop: 32 }}>
                <strong>El cliente VPN no aparece en la detección automática:</strong>
              </p>
              <CodeBlock lang="bash">{os === 'windows'
? `# El wizard pregunta la ruta al binario si no lo detecta:
> kongtrol init
#   !  Este cliente VPN no fue detectado automáticamente.
#   Ruta al binario (vacío = autodetectar): C:\\ruta\\personalizada\\FortiClient.exe

# O agrégalo manualmente en %USERPROFILE%\\.kongtrol\\kongtrol.yaml:
# office:
#   type: forticlient
#   binary_path: "C:\\\\ruta\\\\personalizada\\\\FortiClient.exe"`
: `# El wizard pregunta la ruta al binario si no lo detecta:
$ kongtrol init
#   !  Este cliente VPN no fue detectado automáticamente.
#   Ruta al binario (vacío = autodetectar): /ruta/al/binario

# O agrégalo manualmente en ~/.kongtrol/kongtrol.yaml:
# office:
#   type: forticlient
#   binary_path: "/ruta/personalizada/FortiClient"`}</CodeBlock>

              <p style={{ marginTop: 32 }}><strong>Cloudflare WARP — "not registered":</strong></p>
              <CodeBlock lang="bash">{`$ warp-cli register`}</CodeBlock>

              <p style={{ marginTop: 32 }}><strong>Contraseña incorrecta / expirada:</strong></p>
              <CodeBlock lang="bash">{`$ kongtrol init
# Selecciona el perfil → "Refresh credentials" → ingresa la nueva`}</CodeBlock>

              <div className="callout" style={{ marginTop: 12 }}>
                Si detecta fallo de sesión/token/credenciales durante connect/reconnect,
                Kongtrol emite alerta de <strong>reauth requerida</strong> con hint específico por adapter.
              </div>

              <p style={{ marginTop: 32 }}><strong>Estructura de archivos:</strong></p>
              <CodeBlock lang="text">{`~/.kongtrol/
├── kongtrol.yaml     ← config principal (sin contraseñas)
├── certs/
│   ├── office.crt
│   └── office.key
├── configs/
│   ├── server.ovpn
│   └── aws.ovpn
└── audit.log         ← log firmado de todos los eventos

Contraseñas guardadas en:
  Windows → Windows Credential Manager
  macOS   → Keychain
  Linux   → libsecret / GNOME Keyring`}</CodeBlock>
            </div>

          </div>
        </div>
      </div>
    </section>
  )
}

export default function Guide({ os, setOS, lang }: Props) {
  if (lang === 'en') {
    return <GuideEN os={os} setOS={setOS} />
  }
  return <GuideES os={os} setOS={setOS} />
}
