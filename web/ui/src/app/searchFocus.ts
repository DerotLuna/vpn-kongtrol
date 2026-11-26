import { useEffect } from 'react'
import { useSearchParams } from 'react-router'

export function useSearchFocus(ready: boolean) {
  const [params, setParams] = useSearchParams()
  const name = params.get('focus')
  useEffect(() => {
    if (!name || !ready) return
    const row = Array.from(document.querySelectorAll<HTMLElement>('#main tr[data-key]'))
      .find((element) => element.dataset.key === name)
    if (row) {
      row.tabIndex = -1
      row.scrollIntoView?.({ block: 'center' })
      row.focus({ preventScroll: true })
    }
    setParams((previous) => {
      const next = new URLSearchParams(previous)
      next.delete('focus')
      return next
    }, { replace: true })
  }, [name, ready, setParams])
}
