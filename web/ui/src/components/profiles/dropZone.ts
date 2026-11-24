// Kongtrol UI — drop zone highlight without flicker · v1.0.0 · 2026-11-24
//
// Moving the pointer onto a child of the zone (its button, its text) fires
// dragenter on the child and dragleave on the zone, in that order; a plain
// "dragleave → off" turns the highlight off and dragover turns it on again
// (the flicker). A depth counter (enter +1, leave −1, off at 0) is the
// common fix and does not depend on dragleave's relatedTarget, which some
// browsers leave null. Used by the Add sheet's file step and the profile
// editor's "Importar archivo".
import { useRef, useState, type DragEvent } from 'react'

export function useDropZone() {
  const [drag, setDrag] = useState(false)
  const depth = useRef(0)
  const reset = () => {
    depth.current = 0
    setDrag(false)
  }
  const handlers = {
    onDragEnter: (e: DragEvent) => {
      e.preventDefault()
      depth.current++
      setDrag(true)
    },
    onDragOver: (e: DragEvent) => {
      // Without this the browser opens the file instead of dropping it here.
      e.preventDefault()
    },
    onDragLeave: () => {
      depth.current = Math.max(0, depth.current - 1)
      if (depth.current === 0) setDrag(false)
    },
  }
  return { drag, handlers, reset }
}
