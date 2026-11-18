// Kongtrol UI strings · v0.3.0 · 2026-11-18
//
// Spanish is the default and the source of the key set; English must have
// exactly the same keys (enforced by the `Messages` type below and by
// messages.test.ts). Tone and terms follow internal/i18n/i18n.go: "tú",
// "ayudante" for the privileged helper, "Instalar ayudante" as the button the
// daemon's hints refer to.
//
// Phase 2 plan: keys shared with the CLI/daemon (helper.*, attach.*, …) are
// exported from internal/i18n to JSON at build time instead of being copied.

export type Lang = 'es' | 'en'

const es = {
  'app.name': 'Kongtrol',
  'app.skip': 'Saltar al contenido',
  'app.footer.version': 'Kongtrol {version}',

  'header.legacy': 'Panel clásico',
  'header.lang.switch': 'English',
  'header.lang.switchLabel': 'Cambiar idioma a inglés',
  'header.theme.toDark': 'Usar tema oscuro',
  'header.theme.toLight': 'Usar tema claro',

  'welcome.eyebrow': 'Primeros pasos',
  'welcome.title': 'Bienvenido a Kongtrol',
  'welcome.lead': 'Controla todas tus VPN desde un solo lugar. Sigue estos pasos para empezar.',

  'welcome.actions.title': '¿Qué quieres hacer?',
  'welcome.actions.add.title': 'Agregar VPN',
  'welcome.actions.add.body': 'Suma la VPN de tu trabajo o de tu proveedor.',
  'welcome.actions.connect.title': 'Conectar',
  'welcome.actions.connect.body': 'Elige una VPN y conéctala con un clic.',
  'welcome.actions.status.title': 'Ver estado',
  'welcome.actions.status.body': 'Mira qué está conectado y cuánto tráfico pasa.',
  'welcome.actions.opensClassic': 'Se abre en el panel clásico.',

  'helper.title': 'Ayudante del sistema',
  'helper.what': 'Hace los cambios de red que necesitan permiso de administrador. Se instala una sola vez.',
  'helper.loading': 'Revisando el ayudante…',
  'helper.ready': 'Instalado y funcionando.',
  'helper.elevated': 'No hace falta: Kongtrol ya tiene permisos de administrador.',
  'helper.needsInstall': 'Falta instalarlo. Sin él, Kongtrol no puede conectar tus VPN.',
  'helper.install': 'Instalar ayudante',
  'helper.installing': 'Instalando… Acepta el permiso en la ventana que aparece.',
  'helper.installed': 'Listo: el ayudante quedó instalado.',
  'helper.version': 'Versión {version}',
  'helper.pill.ready': 'Listo',
  'helper.pill.missing': 'Falta',
  'helper.pill.notNeeded': 'No hace falta',

  'tunnels.title': 'Tus VPN',
  'tunnels.loading': 'Cargando tus VPN…',
  'tunnels.none': 'Todavía no tienes VPN. Empieza con «Agregar VPN».',
  'tunnels.summary': '{connected} de {total} conectadas',
  'tunnels.allOff': 'Ninguna conectada ahora.',
  'tunnels.status.connected': 'Conectada',
  'tunnels.status.connecting': 'Conectando',
  'tunnels.status.disconnected': 'Desconectada',
  'tunnels.status.error': 'Con error',
  'tunnels.live': 'Se actualiza solo.',
  'tunnels.status.noResponse': 'Sin respuesta',
  'tunnels.health.noResponse':
    'Sin respuesta del servidor · Kongtrol reintenta solo; si sigue, revisa tu internet o avisa a quien administra esa VPN',
  'tunnels.health.unknown':
    'No se puede comprobar si el servidor responde · si algo no carga, desconéctala y vuelve a conectarla',
  'tunnels.health.outdated':
    'No se puede comprobar si el servidor responde · el ayudante está desactualizado: pulsa «Instalar ayudante»',

  'error.offline': 'No se pudo hablar con Kongtrol.',
  'error.offline.fix': 'Qué hacer: abre Kongtrol desde el ícono de la bandeja o ejecuta kongtrol up.',
  'error.unauthorized': 'Esta página no tiene permiso para hablar con Kongtrol.',
  'error.unauthorized.fix': 'Qué hacer: ábrela desde el ícono de Kongtrol o con kongtrol dashboard.',
  'error.generic.fix': 'Qué hacer: recarga la página; si sigue igual, reinicia Kongtrol.',
  'error.retry': 'Reintentar',
  'publicip.title': 'Salida a internet',
  'publicip.label': 'IP pública:',
  'publicip.pill.on': 'Activada',
  'publicip.pill.off': 'Apagada',
  'publicip.hint': 'Actívalo para ver tu IP pública.',
  'publicip.notice': 'Usa un servicio externo ({host}) que ve tu IP pública',
  'publicip.check': 'Revisar',
  'publicip.checking': 'Revisando…',
  'publicip.turnOn': 'Activar',
  'publicip.turnOff': 'Apagar',
  'publicip.enabled': 'Listo: Kongtrol consultará tu IP al conectar o desconectar y cuando pulses Revisar.',

  'help.guide': 'Guía de uso',
  'help.newTab': '(se abre en otra pestaña)',
  'help.quickstart': 'Ver la guía de inicio rápido',
  'help.helper': '¿Qué es el ayudante?',
  'help.connect': 'Cómo hacer la primera conexión',
  'help.trouble': 'Solución de problemas',

  // Ajustes: aviso de versión nueva (opt-in, ADR-0032) y borrar mis datos (ADR-0033)
  'settings.title': 'Ajustes',
  'updates.title': 'Aviso de versión nueva',
  'updates.what': 'Te avisa cuando sale una versión nueva. Nunca descarga ni instala nada.',
  'updates.privacy': 'Encendido, Kongtrol consulta {host} una vez al día como máximo; GitHub ve tu IP y la versión de Kongtrol, nada más.',
  'updates.pill.on': 'Activado',
  'updates.pill.off': 'Apagado',
  'updates.turnOn': 'Activar aviso',
  'updates.turnOff': 'Apagar aviso',
  'updates.newer': 'Hay una versión nueva: {latest} (tienes {current}). Actualiza con tu instalador o tu gestor de paquetes, o descárgala desde kongtrol.app.',
  'updates.upToDate': 'Tienes la última versión ({current}).',
  'updates.notChecked': 'Todavía no se ha revisado.',
  'updates.releaseNotes': 'Ver novedades',
  'data.title': 'Borrar todos mis datos',
  'data.what': 'Borra tus VPN, ajustes y contraseñas guardadas de este equipo. La app queda instalada.',
  'data.button': 'Borrar todos mis datos',
  'data.dialog.title': '¿Borrar todos tus datos?',
  'data.dialog.body': 'Se desconectan tus VPN y se borra:',
  'data.dialog.item.profiles': 'tus perfiles de VPN y sus ajustes (kongtrol.yaml)',
  'data.dialog.item.passwords': 'las contraseñas guardadas en el llavero del sistema',
  'data.dialog.item.prefs': 'tus preferencias, historial y registro de auditoría',
  'data.dialog.keep': 'Kongtrol sigue instalado y después te guía para crear tu primera VPN. No se puede deshacer.',
  'data.dialog.confirm': 'Sí, borrar todo',
  'data.dialog.cancel': 'Cancelar',
  'data.busy': 'Borrando…',
  'data.done': 'Listo: tus datos se borraron. Kongtrol se reinicia para empezar de cero.',
  'data.uninstallHint': 'Para quitar la app, desinstálala desde tu sistema (en Windows: Configuración › Aplicaciones) o con kongtrol uninstall.',
} as const

export type MessageKey = keyof typeof es
export type Messages = Record<MessageKey, string>

const en: Messages = {
  'app.name': 'Kongtrol',
  'app.skip': 'Skip to content',
  'app.footer.version': 'Kongtrol {version}',

  'header.legacy': 'Classic dashboard',
  'header.lang.switch': 'Español',
  'header.lang.switchLabel': 'Switch language to Spanish',
  'header.theme.toDark': 'Use dark theme',
  'header.theme.toLight': 'Use light theme',

  'welcome.eyebrow': 'Getting started',
  'welcome.title': 'Welcome to Kongtrol',
  'welcome.lead': 'Control all your VPNs from one place. Follow these steps to get started.',

  'welcome.actions.title': 'What do you want to do?',
  'welcome.actions.add.title': 'Add VPN',
  'welcome.actions.add.body': 'Add your work VPN or your provider’s.',
  'welcome.actions.connect.title': 'Connect',
  'welcome.actions.connect.body': 'Pick a VPN and connect it in one click.',
  'welcome.actions.status.title': 'View status',
  'welcome.actions.status.body': 'See what is connected and how much traffic flows.',
  'welcome.actions.opensClassic': 'Opens in the classic dashboard.',

  'helper.title': 'System helper',
  'helper.what': 'Makes the network changes that need administrator permission. You install it once.',
  'helper.loading': 'Checking the helper…',
  'helper.ready': 'Installed and running.',
  'helper.elevated': 'Not needed: Kongtrol already has administrator permission.',
  'helper.needsInstall': 'Not installed yet. Without it, Kongtrol cannot connect your VPNs.',
  'helper.install': 'Install helper',
  'helper.installing': 'Installing… Accept the permission in the window that appears.',
  'helper.installed': 'Done: the helper is installed.',
  'helper.version': 'Version {version}',
  'helper.pill.ready': 'Ready',
  'helper.pill.missing': 'Missing',
  'helper.pill.notNeeded': 'Not needed',

  'tunnels.title': 'Your VPNs',
  'tunnels.loading': 'Loading your VPNs…',
  'tunnels.none': 'You have no VPNs yet. Start with “Add VPN”.',
  'tunnels.summary': '{connected} of {total} connected',
  'tunnels.allOff': 'None connected right now.',
  'tunnels.status.connected': 'Connected',
  'tunnels.status.connecting': 'Connecting',
  'tunnels.status.disconnected': 'Disconnected',
  'tunnels.status.error': 'Error',
  'tunnels.live': 'Updates on its own.',
  'tunnels.status.noResponse': 'No answer',
  'tunnels.health.noResponse':
    'No answer from the server · Kongtrol keeps retrying; if it persists, check your internet or tell whoever runs that VPN',
  'tunnels.health.unknown':
    "Can't check whether the server answers · if something doesn't load, disconnect and connect it again",
  'tunnels.health.outdated':
    "Can't check whether the server answers · the helper is out of date: click “Install helper”",

  'error.offline': 'Could not reach Kongtrol.',
  'error.offline.fix': 'What to do: open Kongtrol from the tray icon or run kongtrol up.',
  'error.unauthorized': 'This page is not allowed to talk to Kongtrol.',
  'error.unauthorized.fix': 'What to do: open it from the Kongtrol icon or with kongtrol dashboard.',
  'error.generic.fix': 'What to do: reload the page; if nothing changes, restart Kongtrol.',
  'error.retry': 'Retry',
  'publicip.title': 'Internet egress',
  'publicip.label': 'Public IP:',
  'publicip.pill.on': 'On',
  'publicip.pill.off': 'Off',
  'publicip.hint': 'Turn it on to see your public IP.',
  'publicip.notice': 'Uses an external service ({host}) that sees your public IP',
  'publicip.check': 'Check',
  'publicip.checking': 'Checking…',
  'publicip.turnOn': 'Turn on',
  'publicip.turnOff': 'Turn off',
  'publicip.enabled': 'Done: Kongtrol will look up your IP when you connect or disconnect and when you press Check.',

  'help.guide': 'User guide',
  'help.newTab': '(opens in a new tab)',
  'help.quickstart': 'Read the quickstart guide',
  'help.helper': 'What is the helper?',
  'help.connect': 'How to make your first connection',
  'help.trouble': 'Troubleshooting',

  'settings.title': 'Settings',
  'updates.title': 'New version notice',
  'updates.what': 'Tells you when a new version is out. It never downloads or installs anything.',
  'updates.privacy': 'When on, Kongtrol asks {host} at most once a day; GitHub sees your IP and the Kongtrol version, nothing else.',
  'updates.pill.on': 'On',
  'updates.pill.off': 'Off',
  'updates.turnOn': 'Turn notice on',
  'updates.turnOff': 'Turn notice off',
  'updates.newer': 'A new version is out: {latest} (you have {current}). Update with your installer or package manager, or download it from kongtrol.app.',
  'updates.upToDate': 'You have the latest version ({current}).',
  'updates.notChecked': 'Not checked yet.',
  'updates.releaseNotes': 'See what’s new',
  'data.title': 'Delete all my data',
  'data.what': 'Deletes your VPNs, settings and saved passwords from this computer. The app stays installed.',
  'data.button': 'Delete all my data',
  'data.dialog.title': 'Delete all your data?',
  'data.dialog.body': 'Your VPNs are disconnected and this is deleted:',
  'data.dialog.item.profiles': 'your VPN profiles and their settings (kongtrol.yaml)',
  'data.dialog.item.passwords': 'the passwords saved in the system keychain',
  'data.dialog.item.prefs': 'your preferences, history and audit log',
  'data.dialog.keep': 'Kongtrol stays installed and then guides you to create your first VPN. This cannot be undone.',
  'data.dialog.confirm': 'Yes, delete everything',
  'data.dialog.cancel': 'Cancel',
  'data.busy': 'Deleting…',
  'data.done': 'Done: your data was deleted. Kongtrol restarts to start from scratch.',
  'data.uninstallHint': 'To remove the app, uninstall it from your system (on Windows: Settings › Apps) or with kongtrol uninstall.',
}

export const messages: Record<Lang, Messages> = { es, en }

/** Replaces {name} placeholders; unknown placeholders are left as-is. */
export function format(template: string, vars: Record<string, string | number> = {}): string {
  return template.replace(/\{(\w+)\}/g, (match, name: string) =>
    name in vars ? String(vars[name]) : match,
  )
}
