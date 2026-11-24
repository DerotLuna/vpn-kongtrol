import { describe, expect, it } from 'vitest'
import { format, messages, type MessageKey } from '../../i18n/messages'
import { conflictText, originSummary, profileSummary, routeRows, rowLabel, ruleTarget, type ProfileOwnRoutes } from './vpnRoutesModel'

const tES = (k: MessageKey, v?: Record<string, string | number>) => format(messages.es[k], v)
const tEN = (k: MessageKey, v?: Record<string, string | number>) => format(messages.en[k], v)

function profile(over: Partial<ProfileOwnRoutes>): ProfileOwnRoutes {
  return {
    name: 'ekiipago',
    type: 'openvpn',
    connected: true,
    source: 'config',
    all_traffic: false,
    rules_only: false,
    server_pending: false,
    routes: [],
    ...over,
  }
}

describe('Rutas que trae cada VPN: grouping', () => {
  it('summarises count and origin per profile', () => {
    const p = profile({
      routes: [
        { destination: '10.0.0.0/8', origin: 'server' },
        { destination: '172.16.0.0/12', origin: 'server' },
        { destination: '192.168.10.0/24', origin: 'server' },
      ],
    })
    expect(profileSummary(p, tES)).toBe('3 rutas · del servidor')
    expect(profileSummary(p, tEN)).toBe('3 routes · from the server')
    const mixed = profile({ routes: [{ destination: '10.0.0.0/8', origin: 'file' }, { destination: '10.1.0.0/16', origin: 'server' }] })
    expect(originSummary(routeRows(mixed))).toBe('mixed')
    expect(profileSummary(profile({ routes: [{ destination: '10.0.0.0/8', origin: 'file' }] }), tES)).toBe('1 ruta · del archivo')
  })

  it('says when there is nothing, when the app decides and when Kongtrol does not use them', () => {
    expect(profileSummary(profile({ routes: null }), tES)).toBe('sin rutas propias')
    expect(profileSummary(profile({ type: 'protonvpn', source: 'app', all_traffic: true }), tES)).toBe('la app decide')
    const wg = profile({ type: 'wireguard', rules_only: true, routes: [{ destination: '0.0.0.0/0', origin: 'file' }] })
    expect(profileSummary(wg, tES)).toBe('1 ruta · del archivo · Kongtrol no las usa')
  })
})

describe('Rutas que trae cada VPN: full tunnel in words', () => {
  it('folds 0.0.0.0/0 into one "Todo el tráfico" row, first', () => {
    const rows = routeRows(profile({ routes: [{ destination: '10.0.0.0/8', origin: 'file' }, { destination: '0.0.0.0/0', origin: 'server' }] }))
    expect(rows.map((r) => r.kind)).toEqual(['all', 'route'])
    expect(rowLabel(rows[0]!, tES)).toBe('Todo el tráfico')
    expect(rowLabel(rows[0]!, tEN)).toBe('All traffic')
    expect(rows[0]!.origin).toBe('server')
  })

  it('folds both /1 halves, keeps a lone half and names ::/0', () => {
    const halves = routeRows(profile({ routes: [{ destination: '0.0.0.0/1', origin: 'server' }, { destination: '128.0.0.0/1', origin: 'server' }] }))
    expect(halves).toHaveLength(1)
    expect(rowLabel(halves[0]!, tES)).toBe('Todo el tráfico')
    const lone = routeRows(profile({ routes: [{ destination: '0.0.0.0/1', origin: 'file' }] }))
    expect(rowLabel(lone[0]!, tES)).toBe('0.0.0.0/1')
    const v6 = routeRows(profile({ routes: [{ destination: '::/0', origin: 'file' }] }))
    expect(rowLabel(v6[0]!, tES)).toBe('Todo el tráfico IPv6')
  })

  it('gives the rule sheet a host without /32', () => {
    expect(ruleTarget('10.1.2.3/32')).toBe('10.1.2.3')
    expect(ruleTarget('2001:db8::1/128')).toBe('2001:db8::1')
    expect(ruleTarget('10.0.0.0/8')).toBe('10.0.0.0/8')
  })
})

describe('Rutas que trae cada VPN: conflict note', () => {
  it('words the winner the daemon decided', () => {
    expect(conflictText({ rule: 'db', via: 'casa', destination: '1.2.3.4/32', winner: 'rule' }, '1.0.0.0/8', tES)).toBe(
      'Tu regla «db» gana para 1.2.3.4: es más específica. Sale por «casa» cuando está conectada.',
    )
    expect(conflictText({ rule: 'lan', via: 'casa', destination: '172.16.0.0/16', winner: 'vpn' }, '172.16.5.0/24', tES)).toBe(
      'Esta ruta gana para 172.16.5.0/24 sobre tu regla «lan»: es más específica.',
    )
    const same = conflictText({ rule: 'net', via: 'casa', destination: '10.0.0.0/8', winner: 'system' }, '10.0.0.0/8', tEN)
    expect(same).toContain('The system picks one')
    expect(same).toContain('“net”')
  })
})
