import { Dispatch, SetStateAction, useEffect, useState } from 'react'

// Tracks which of a list of section ids is currently in view, via
// IntersectionObserver. Shared by every "jump to section" nav — the top
// tmux bar, the guide/terms sidebar, and their mobile jump menus. Returns
// a setter too so a click on a nav item can optimistically highlight its
// target before the smooth-scroll finishes and the observer catches up.
export function useActiveSection(
  ids: string[],
  rootMargin = '-20% 0px -70% 0px',
  initial = ids[0] ?? '',
): [string, Dispatch<SetStateAction<string>>] {
  const [active, setActive] = useState(initial)

  useEffect(() => {
    const observer = new IntersectionObserver(
      entries => entries.forEach(e => { if (e.isIntersecting) setActive(e.target.id) }),
      { rootMargin },
    )
    ids.forEach(id => {
      const el = document.getElementById(id)
      if (el) observer.observe(el)
    })
    return () => observer.disconnect()
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [ids.join(','), rootMargin])

  return [active, setActive]
}
