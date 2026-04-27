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

Antes de empezar, asegúrate de tener instalados los clientes VPN que vayas a usar:

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

Verás algo así:

```
╔══════════════════════════════════════════╗
║        Kongtrol — Setup Wizard           ║
╚══════════════════════════════════════════╝

Detected VPN clients on this system:
  ✓ FortiClient             FortiClient 6.4.10.1821
  ✓ OpenVPN                 OpenVPN 2.6.x
  ✓ ProtonVPN               protonvpn-cli 3.x

Add a new VPN profile? [Y/n]: y
```

### Perfil 1: FortiClient (oficina)

```
Profile name: office
Type: forticlient
VPN host: vpn.tuempresa.com
Port [443]:
Tunnel name: Office          ← exactamente como aparece en FortiClient
FortiClient major version [6]: 6
Auth method [certificate+credentials]:
Client cert path: ~/.kongtrol/certs/office.crt
Private key path: ~/.kongtrol/certs/office.key
Username: tu_usuario
Password: ****              ← se guarda en keychain, no en el YAML
Priority [10]:
```

### Perfil 2: OpenVPN servidor

```
Add a new VPN profile? [Y/n]: y
Profile name: dev-server
Type: openvpn
.ovpn config path: ~/.kongtrol/configs/server.ovpn
Auth method [certificate]:
Client cert path (leave blank if embedded in .ovpn):
Private key path (leave blank if embedded in .ovpn):
Priority [10]: 20
```

> Si el `.ovpn` ya tiene `<cert>` y `<key>` embebidas, deja los paths en blanco.

### Perfil 3: OpenVPN AWS

```
Add a new VPN profile? [Y/n]: y
Profile name: aws
Type: openvpn
.ovpn config path: ~/.kongtrol/configs/aws.ovpn
Auth method [certificate]:
Priority [10]: 20
```

### Perfil 4: ProtonVPN

```
Add a new VPN profile? [Y/n]: y
Profile name: us-content
Type: protonvpn
Server / country code [fastest]: US
Protocol [wireguard]:
Username: tu_usuario_proton
Password: ****
Priority [10]: 5
```

### Opciones de seguridad

```
Enable kill switch? [Y/n]: y     ← bloquea tráfico si cae una VPN
Enable DNS guard? [Y/n]: y       ← previene DNS leaks
Enable signed audit log? [Y/n]: y
Enable web dashboard? [Y/n]: y

Write config to ~/.kongtrol/kongtrol.yaml? [Y/n]: y

[✓] Config written
[✓] Config is valid.
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
kongtrol up --group work          # empezar el día
kongtrol down --group work        # terminar el día
kongtrol up us-content            # ver Netflix / Hulu
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
# → elige el perfil "office" → "Refresh credentials" → ingresa la nueva
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
