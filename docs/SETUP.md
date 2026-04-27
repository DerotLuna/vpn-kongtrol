# Guía de configuración — Kongtrol

Esta guía te lleva desde cero hasta tener todas tus VPNs corriendo bajo Kongtrol.  
Tiempo estimado: **15–20 minutos**.

---

## Lo que vas a lograr

```
kongtrol up --group work
# [+] office connected     → tráfico 10.10.x.x va por FortiClient
# [+] dev-server connected → tráfico del servidor va por OpenVPN
# [+] aws connected        → *.amazonaws.com va por OpenVPN (AWS)

kongtrol up us-content
# [+] us-content connected → Netflix, Hulu por ProtonVPN
```

Un solo comando. Sin tocar cada cliente VPN por separado.

---

## Índice

1. [Prerequisitos](#1-prerequisitos)
2. [Instalar Kongtrol](#2-instalar-kongtrol)
3. [Organizar tus certificados](#3-organizar-tus-certificados)
4. [Ejecutar el wizard](#4-ejecutar-el-wizard-kongtrol-init)
5. [Configurar grupos](#5-configurar-grupos)
6. [Verificar con doctor](#6-verificar-con-doctor)
7. [Primera conexión](#7-primera-conexión)
8. [Dashboard web](#8-dashboard-web)
9. [Uso diario](#9-uso-diario)
10. [Para compañeros de equipo](#10-para-compañeros-de-equipo)
11. [Solución de problemas](#11-solución-de-problemas)

---

## 1. Prerequisitos

### Estado de las VPNs antes de empezar

> **TL;DR:** Para configurar (`init`, `doctor`) el estado no importa. Para conectar (`up`), cierra la GUI del cliente primero.

| Acción | Estado requerido de los clientes VPN |
|---|---|
| `kongtrol init` | Cualquiera — solo lee archivos y keychain |
| `kongtrol doctor` | Cualquiera — solo valida, no conecta |
| `kongtrol up office` | FortiClient GUI **cerrado** |
| `kongtrol up dev-server` | OpenVPN GUI **cerrado** |
| `kongtrol up us-content` | ProtonVPN GUI **cerrado** |

**¿Por qué?** Kongtrol llama directamente al CLI del cliente. Si la GUI ya tiene una sesión abierta, dos instancias intentan controlar el mismo túnel y chocan. A partir del momento en que uses Kongtrol, las GUIs quedan cerradas — Kongtrol es quien las maneja.

---

### Clientes VPN requeridos

Asegúrate de tener instalados los clientes VPN que vayas a usar:

| VPN | Cliente requerido | Verificar |
|---|---|---|
| FortiClient (oficina) | [FortiClient 6.4.x](https://www.fortinet.com/support/product-downloads) | Abre FortiClient, debe tener el túnel configurado |
| OpenVPN (servidor / AWS) | [OpenVPN Community](https://openvpn.net/community-downloads/) | `openvpn --version` en terminal |
| ProtonVPN (contenido US) | [protonvpn-cli](https://protonvpn.com/support/linux-vpn-tool/) | `protonvpn-cli --version` |

> **Importante:** Kongtrol no reemplaza estos clientes — los *orquesta*. Deben estar instalados.

---

## 2. Instalar Kongtrol

### Windows

1. Descarga `kongtrol_windows_amd64.zip` de [Releases](../../releases)
2. Extrae y mueve `kongtrol.exe` a una carpeta en tu PATH, por ejemplo:
   ```
   C:\Users\TuUsuario\bin\kongtrol.exe
   ```
3. Agrega esa carpeta a la variable de entorno `PATH` si no está
4. Abre una terminal nueva y verifica:
   ```
   kongtrol --help
   ```

### macOS

```bash
# Descarga y extrae
curl -L https://github.com/yourorg/vpn-kongtrol/releases/latest/download/kongtrol_darwin_arm64.tar.gz | tar xz
sudo mv kongtrol /usr/local/bin/

# Verifica
kongtrol --help
```

### Linux

```bash
curl -L https://github.com/yourorg/vpn-kongtrol/releases/latest/download/kongtrol_linux_amd64.tar.gz | tar xz
sudo mv kongtrol /usr/local/bin/

kongtrol --help
```

### Desde el código fuente (si tienes Go 1.22+)

```bash
git clone https://github.com/yourorg/vpn-kongtrol
cd vpn-kongtrol
make build
# Binario en: build/dist/kongtrol (o kongtrol.exe en Windows)
```

---

## 3. Organizar tus certificados

Kongtrol espera tus certs en `~/.kongtrol/certs/`. Crea la estructura una sola vez:

```bash
mkdir -p ~/.kongtrol/certs
mkdir -p ~/.kongtrol/configs
```

### Copia tus archivos

| Archivo | Destino sugerido |
|---|---|
| Cert de FortiClient (`.crt`) | `~/.kongtrol/certs/office.crt` |
| Key de FortiClient (`.key`) | `~/.kongtrol/certs/office.key` |
| Config OpenVPN servidor (`.ovpn`) | `~/.kongtrol/configs/server.ovpn` |
| Config OpenVPN AWS (`.ovpn`) | `~/.kongtrol/configs/aws.ovpn` |

```bash
# Ejemplo (ajusta las rutas a donde están tus archivos actuales)
cp /ruta/actual/office.crt   ~/.kongtrol/certs/office.crt
cp /ruta/actual/office.key   ~/.kongtrol/certs/office.key
cp /ruta/actual/server.ovpn  ~/.kongtrol/configs/server.ovpn
cp /ruta/actual/aws.ovpn     ~/.kongtrol/configs/aws.ovpn
```

> **Seguridad:** Estos archivos nunca van al repositorio (están en `.gitignore`).  
> Solo viven en tu máquina local.

---

## 4. Ejecutar el wizard: `kongtrol init`

El wizard detecta tus clientes VPN instalados, te guía por cada perfil, y guarda las contraseñas en el **keychain del OS** (nunca en el archivo YAML).

```bash
kongtrol init
```

Lo primero que verás es la selección de idioma — presiona Enter para español o `n` para inglés:

```
¿Continuar en español? [S/n]  (Press n for English):
```

Luego el wizard continúa en español:

```
╔═══════════════════════════════════════════════╗
║   Kongtrol — Asistente de Configuración   ║
╚═══════════════════════════════════════════════╝

Este asistente crea o actualiza ~/.kongtrol/kongtrol.yaml.
Las contraseñas se guardan en el llavero del sistema — nunca en el YAML.

Clientes VPN detectados en este sistema:
  ✓ FortiClient             FortiClient 6.4.10.1821
  ✓ OpenVPN                 OpenVPN 2.6.x
  ✓ ProtonVPN               protonvpn-cli 3.x

¿Agregar un nuevo perfil VPN? [s/N]: s
```

### Perfil 1: FortiClient (oficina)

```
  Nombre del perfil (ej. oficina, aws, wg-casa): office
  Tipos de adaptador: forticlient | openvpn | protonvpn | ciscoanyconnect |
                      wireguard | globalprotect | tailscale | cloudflarewarp
  Tipo [openvpn]: forticlient
  Host VPN (ej. vpn.empresa.com): vpn.tuempresa.com
  Puerto [443]:
  Nombre del túnel (como aparece en la UI de FortiClient) [Office]:
  Versión mayor de FortiClient [6]:
  Método de autenticación (certificate | credentials | certificate+credentials) [certificate+credentials]:
  Ruta al certificado cliente (ej. ~/.kongtrol/certs/oficina.crt): ~/.kongtrol/certs/office.crt
  Ruta a la clave privada (ej. ~/.kongtrol/certs/oficina.key): ~/.kongtrol/certs/office.key
  Usuario: tu_usuario
  Prioridad (menor = preferido, 1–100) [10]:
  Contraseña para office (guardada en llavero del sistema, no en YAML): ****
```

### Perfil 2: OpenVPN servidor

```
¿Agregar un nuevo perfil VPN? [s/N]: s
  Nombre del perfil (ej. oficina, aws, wg-casa): dev-server
  Tipo [openvpn]:
  Ruta al archivo .ovpn (ej. ~/.kongtrol/configs/servidor.ovpn): ~/.kongtrol/configs/server.ovpn
  Método de autenticación (certificate | credentials | certificate+credentials) [certificate]:
  Ruta al certificado cliente (vacío si está dentro del .ovpn):
  Ruta a la clave privada (vacío si está dentro del .ovpn):
  Prioridad (menor = preferido, 1–100) [10]: 20
```

> Si el `.ovpn` ya tiene `<cert>` y `<key>` embebidas, deja los paths en blanco.

### Perfil 3: OpenVPN AWS

```
¿Agregar un nuevo perfil VPN? [s/N]: s
  Nombre del perfil (ej. oficina, aws, wg-casa): aws
  Tipo [openvpn]:
  Ruta al archivo .ovpn (ej. ~/.kongtrol/configs/servidor.ovpn): ~/.kongtrol/configs/aws.ovpn
  Método de autenticación (certificate | credentials | certificate+credentials) [certificate]:
  Prioridad (menor = preferido, 1–100) [10]: 20
```

### Perfil 4: ProtonVPN

```
¿Agregar un nuevo perfil VPN? [s/N]: s
  Nombre del perfil (ej. oficina, aws, wg-casa): us-content
  Tipo [openvpn]: protonvpn
  Servidor / código de país (ej. US, NL, fastest) [fastest]: US
  Protocolo (wireguard | openvpn) [wireguard]:
  Usuario de ProtonVPN: tu_usuario_proton
  Prioridad (menor = preferido, 1–100) [10]: 5
  Contraseña para us-content (guardada en llavero del sistema, no en YAML): ****

¿Agregar un nuevo perfil VPN? [s/N]:
```

### Opciones de seguridad

```
¿Activar kill switch? (bloquea todo el tráfico si cae la VPN) [S/n]:    ← Enter = sí
¿Activar DNS guard? (previene fugas de DNS) [S/n]:
¿Activar log de auditoría firmado? [S/n]:
¿Activar panel web? (http://127.0.0.1:9741) [S/n]:

¿Escribir configuración en ~/.kongtrol/kongtrol.yaml? [S/n]: s

[✓] Configuración escrita en ~/.kongtrol/kongtrol.yaml
[✓] Configuración válida.

Próximos pasos:
  kongtrol status                  — ver estado de los túneles
  kongtrol up <perfil>             — conectar un perfil
  kongtrol dashboard               — abrir el panel web
```

---

## 5. Configurar grupos

Los grupos te permiten conectar varios perfiles con un solo comando. Edita `~/.kongtrol/kongtrol.yaml` y agrega al final:

```yaml
groups:
  work:
    profiles: [office, dev-server, aws]

  travel:
    profiles: [us-content]

  full:
    profiles: [office, dev-server, aws, us-content]
```

Guarda el archivo. Verifica que sigue siendo válido:

```bash
kongtrol config validate
# [OK] Config is valid.
```

---

## 5b. Configurar políticas de routing

Las políticas definen qué tráfico va por qué VPN. El wizard no las configura — las agregas manualmente en `~/.kongtrol/kongtrol.yaml` bajo la sección `policies:`.

### Ejemplo completo para tu setup

```yaml
policies:

  # ── Oficina ────────────────────────────────────────────────────────
  - name: "Servidores de oficina"
    match:
      ip_ranges:
        - "10.10.0.0/16"        # red interna de oficina
        - "192.168.50.0/24"     # subnets adicionales — ajusta a las tuyas
    via: office

  - name: "Dominios internos"
    match:
      domains:
        - "*.tuempresa.com"     # cambia al dominio de tu empresa
        - "intranet.local"
    via: office

  # ── Servidor de desarrollo ─────────────────────────────────────────
  - name: "Servidor dev"
    match:
      ip_ranges:
        - "185.0.0.0/32"        # reemplaza con la IP real de tu servidor
    via: dev-server

  # ── AWS ────────────────────────────────────────────────────────────
  - name: "AWS workloads"
    match:
      ip_ranges:
        - "172.31.0.0/16"       # VPC default de AWS
        - "10.200.0.0/16"       # ajusta a tus CIDRs
      domains:
        - "*.amazonaws.com"
        - "*.aws.tuempresa.com"
    via: aws

  # ── Contenido US / Claude ──────────────────────────────────────────
  - name: "Claude AI"
    match:
      domains:
        - "claude.ai"
        - "*.claude.ai"
        - "*.anthropic.com"     # API de Anthropic también
    via: us-content

  - name: "Contenido geo-restringido US"
    match:
      domains:
        - "netflix.com"
        - "*.netflix.com"
        - "hulu.com"
        - "*.hulu.com"
        - "disneyplus.com"
        - "*.disneyplus.com"
    via: us-content

  # El tráfico que no coincide con ninguna regla va por tu conexión
  # normal (sin VPN). Si quieres forzar TODO el tráfico por una VPN:
  # - name: "Default"
  #   match:
  #     ip_ranges: ["0.0.0.0/0"]
  #   via: office
```

### Agregar cualquier otro dominio a us-content

El patrón es siempre el mismo — agrega una entrada en `policies:`:

```yaml
  - name: "Mi servicio"
    match:
      domains:
        - "ejemplo.com"
        - "*.ejemplo.com"
    via: us-content
```

Guarda y valida:

```bash
kongtrol config validate
```

No necesitas reiniciar nada — las políticas se leen al conectar.

---

## 6. Verificar con doctor

Antes de conectar por primera vez, deja que Kongtrol valide todo:

```bash
kongtrol doctor
```

Salida esperada:

```
Kongtrol Doctor
────────────────────────────────────────────────────

  Configuration
  ✓  config file                       /Users/tu/.kongtrol/kongtrol.yaml
  ✓  config valid                      4 profile(s) defined

  VPN Binaries
  ✓  forticlient binary                FortiClient 6.4.10.1821
  ✓  openvpn binary                    OpenVPN 2.6.8
  ✓  protonvpn binary                  protonvpn-cli 3.10.0

  Certificates & Keys
  ✓  office: cert                      /Users/tu/.kongtrol/certs/office.crt
  ✓  office: key                       /Users/tu/.kongtrol/certs/office.key
  ✓  dev-server: config file           /Users/tu/.kongtrol/configs/server.ovpn
  ✓  aws: config file                  /Users/tu/.kongtrol/configs/aws.ovpn

  Keychain Credentials
  ✓  office: password                  found in OS keychain
  ✓  us-content: password              found in OS keychain

  Permissions
  ✓  kill switch                       platform implementation available
  ✓  dns guard                         available

  Registered Adapters
  ✓  registered adapters               ciscoanyconnect, cloudflarewarp, forticlient, ...

All checks passed. You're good to go.
```

Si hay ✗ en alguna línea, el mensaje te dice exactamente qué falta.

---

## 7. Primera conexión

### Modo trabajo (oficina + servidores)

```bash
kongtrol up --group work
```

```
[+] office connected
[+] dev-server connected
[+] aws connected
```

Kongtrol se queda corriendo en primer plano. Cuando termines:

```
Ctrl+C   →  desconecta todo limpiamente
```

O en otra terminal:

```bash
kongtrol down --group work
```

### Solo una VPN

```bash
kongtrol up office        # solo FortiClient
kongtrol up us-content    # solo ProtonVPN
```

### Ver estado

```bash
kongtrol status

PROFILE          STATUS         IP                 UPTIME
-------          ------         --                 ------
office           connected      10.10.0.5          1h 23m
dev-server       connected      185.x.x.x          1h 23m
aws              connected      172.31.4.7         1h 23m
us-content       disconnected   —                  —

Kill Switch: ON
```

Vista en tiempo real (se refresca cada 2 segundos):

```bash
kongtrol status --watch
```

---

## 8. Dashboard web

```bash
kongtrol dashboard
# Dashboard running at http://127.0.0.1:9741
```

Abre `http://localhost:9741` en tu navegador. Verás:

- Estado de todos los túneles en tiempo real
- Tráfico por túnel (upload / download)
- Rutas activas
- Estado del kill switch y DNS guard
- Últimos eventos de conexión

> El dashboard está compilado dentro del binario. No necesitas instalar nada extra.

---

## 9. Uso diario

### Comandos más usados

```bash
kongtrol up --group work          # empezar el día (office + dev-server + aws)
kongtrol down --group work        # terminar el día
kongtrol up us-content            # Netflix, Hulu, Claude AI...
kongtrol down --all               # apagar todo

kongtrol status                   # ver qué está conectado
kongtrol status --watch           # monitoreo en vivo
kongtrol check                    # test de leaks ahora mismo
kongtrol dashboard                # abrir UI web
```

### Si algo falla

```bash
kongtrol doctor                   # diagnóstico completo
```

### Actualizar una contraseña en el keychain

```bash
# Si cambias tu contraseña de FortiClient:
kongtrol init
# → elige el perfil "office" → "¿Actualizar credenciales de este perfil?" → ingresa la nueva
```

---

## 10. Para compañeros de equipo

### Generar una plantilla compartible

```bash
kongtrol export > kongtrol-template.yaml
```

Esto genera el YAML **sin contraseñas** — seguro para compartir. Ejemplo de output:

```yaml
# Kongtrol config template — generated by 'kongtrol export'
# Passwords and keys are redacted. Run 'kongtrol init' on the target machine

vpns:
  office:
    type: forticlient
    host: vpn.tuempresa.com
    port: 443
    tunnel_name: "Office"
    auth:
      method: certificate+credentials
      cert: /Users/tu/.kongtrol/certs/office.crt
      key:  /Users/tu/.kongtrol/certs/office.key
      username: tu_usuario
      password_keychain: office.password  # store via: kongtrol init
  ...
```

### Pasos para un compañero nuevo

1. Instala los clientes VPN (FortiClient, OpenVPN, etc.)
2. Descarga `kongtrol` para su plataforma
3. Recibe la plantilla (`kongtrol-template.yaml`) y sus propios certs
4. Copia el template a `~/.kongtrol/kongtrol.yaml`
5. Ajusta los paths de certs
6. Ejecuta `kongtrol init` para guardar sus credenciales
7. Ejecuta `kongtrol doctor` para verificar
8. `kongtrol up --group work` ✓

---

## 11. Solución de problemas

### "warp-cli not found" / "openvpn not found"

El cliente VPN no está en el PATH. Instálalo o verifica la instalación:

```bash
which openvpn        # Linux/macOS
where openvpn        # Windows
```

### "not registered — run 'warp-cli register' first"

Para Cloudflare WARP, debes registrar el dispositivo una vez:

```bash
warp-cli register
```

### DNS no restaurado después de un crash

Si Kongtrol se cierra inesperadamente con DNS guard activo:

```bash
# Windows
netsh interface ip set dns "Ethernet" dhcp

# Linux
sudo cp /etc/resolv.conf.kongtrol.bak /etc/resolv.conf

# macOS
networksetup -setdnsservers Wi-Fi empty
```

### "unknown profile" al usar --group

Verifica que el nombre del grupo esté en `~/.kongtrol/kongtrol.yaml` bajo `groups:` y que el config sea válido:

```bash
kongtrol config validate
```

### Kill switch activo y sin internet después de desconectar

```bash
kongtrol down --all    # desactiva kill switch automáticamente
```

Si Kongtrol se cerró abruptamente:

```bash
# Windows (como Administrador)
netsh advfirewall reset

# Linux
sudo iptables -F OUTPUT

# macOS
sudo pfctl -d
```

### Contraseña incorrecta / expirada

```bash
kongtrol init
# Selecciona el perfil → "Refresh credentials" → ingresa la nueva contraseña
```

---

## Estructura de archivos

```
~/.kongtrol/
├── kongtrol.yaml          ← config principal (NO contiene contraseñas)
├── certs/
│   ├── office.crt
│   ├── office.key
│   └── ...
├── configs/
│   ├── server.ovpn
│   ├── aws.ovpn
│   └── ...
└── audit.log              ← log firmado de todos los eventos
```

Las contraseñas viven en:
- **Windows:** Windows Credential Manager
- **macOS:** Keychain
- **Linux:** libsecret / GNOME Keyring

---

*¿Problemas o preguntas? Abre un issue en el repositorio.*
