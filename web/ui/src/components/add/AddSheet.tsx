// Kongtrol UI — "＋ Agregar": the shell's host for the add sheet · v1.0.0 · 2026-11-19
//
// Mounted once by AppShell (ADR-0055). It reads the deep link
// (?add=menu|vpn|file|rule…, app/addLink.ts) and loads the sheet's code
// only when it opens, so the main bundle stays lean. It also turns the
// whole window into a drop target for .conf / .ovpn files while no other
// dialog is open: an overlay says where to drop (announced through a polite
// live region) and the drop opens the sheet on "Importar un archivo". The
// "Elegir archivo" button inside the sheet stays the accessible path.
import { lazy, Suspense, useEffect, useRef, useState } from 'react'
import { useSearchParams } from 'react-router'
import { useAddSheet, parseAddParams } from '../../app/addLink'
import { usePreferences } from '../../app/preferences-context'
import { droppedFile } from '../../data/add'
import { UploadIcon } from '../profiles/icons'
import './add.css'

const AddFlow = lazy(() => import('./AddFlow').then((m) => ({ default: m.AddFlow })))

const hasFiles = (e: DragEvent) => !!e.dataTransfer && [...e.dataTransfer.types].includes('Files')

/** Another modal (an editor, a confirm) owns drops while it is open. */
const otherDialogOpen = () => document.querySelector('dialog[open]:not(.k-add-sheet)') !== null

export function AddSheet() {
  const { t } = usePreferences()
  const [params] = useSearchParams()
  const req = parseAddParams(params)
  const { open, close } = useAddSheet()
  const [over, setOver] = useState(false)
  const depth = useRef(0)
  const isOpen = req !== null
  // Latest values for the window listeners, which are added once.
  const latest = useRef({ open, isOpen })
  useEffect(() => {
    latest.current = { open, isOpen }
  })

  useEffect(() => {
    const reset = () => {
      depth.current = 0
      setOver(false)
    }
    const enter = (e: DragEvent) => {
      if (!hasFiles(e) || otherDialogOpen()) return
      depth.current++
      if (!latest.current.isOpen) setOver(true)
    }
    const leave = (e: DragEvent) => {
      if (!hasFiles(e)) return
      depth.current = Math.max(0, depth.current - 1)
      if (depth.current === 0) setOver(false)
    }
    const overHandler = (e: DragEvent) => {
      // Without this the browser would open the file instead of handing it to us.
      if (hasFiles(e) && !e.defaultPrevented && !otherDialogOpen()) e.preventDefault()
    }
    const drop = (e: DragEvent) => {
      reset()
      // A drop zone inside the page (the profile editor's import) handled it already.
      if (!hasFiles(e) || e.defaultPrevented || otherDialogOpen()) return
      e.preventDefault()
      const file = e.dataTransfer?.files[0]
      if (!file) return
      droppedFile.set(file)
      if (!latest.current.isOpen) latest.current.open({ mode: 'file' })
    }
    window.addEventListener('dragenter', enter)
    window.addEventListener('dragleave', leave)
    window.addEventListener('dragover', overHandler)
    window.addEventListener('drop', drop)
    return () => {
      window.removeEventListener('dragenter', enter)
      window.removeEventListener('dragleave', leave)
      window.removeEventListener('dragover', overHandler)
      window.removeEventListener('drop', drop)
    }
  }, [])

  return (
    <>
      {req && (
        <Suspense fallback={null}>
          <AddFlow req={req} onClose={close} />
        </Suspense>
      )}
      <div className="k-drop" data-show={over || undefined} aria-hidden="true">
        <div className="k-drop__box">
          <UploadIcon />
          <p>{t('add.file.overlay')}</p>
        </div>
      </div>
      <p className="visually-hidden" aria-live="polite">
        {over ? t('add.file.overlay') : ''}
      </p>
    </>
  )
}
