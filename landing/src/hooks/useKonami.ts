import { useEffect } from 'react'

const SEQUENCE = ['ArrowUp', 'ArrowUp', 'ArrowDown', 'ArrowDown', 'ArrowLeft', 'ArrowRight', 'ArrowLeft', 'ArrowRight', 'b', 'a']

// ↑↑↓↓←→←→BA — flips the whole site into green-phosphor CRT mode.
export function useKonami() {
  useEffect(() => {
    let pos = 0
    const onKey = (e: KeyboardEvent) => {
      const key = e.key.length === 1 ? e.key.toLowerCase() : e.key
      pos = key === SEQUENCE[pos] ? pos + 1 : (key === SEQUENCE[0] ? 1 : 0)
      if (pos === SEQUENCE.length) {
        pos = 0
        const el = document.documentElement
        el.toggleAttribute('data-crt')
      }
    }
    window.addEventListener('keydown', onKey)
    return () => window.removeEventListener('keydown', onKey)
  }, [])
}
