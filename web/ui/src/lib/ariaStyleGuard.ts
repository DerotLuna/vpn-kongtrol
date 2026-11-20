// React Aria under the panel's CSP · v1.0.0 · 2026-11-19
//
// React Aria's usePress injects one <style> element (`touch-action` for
// pressable elements) unless an element with its id already exists. The
// page's CSP is `style-src 'self'` with no nonce, so that <style> would be
// blocked and reported as a violation. The same rule lives in
// kit-controls.css; this marks it as present so React Aria skips the
// injection (ADR-0068). A <meta> carries no style, so the CSP is unaffected.
// kit.test.tsx checks the id still matches the installed react-aria.
export const ARIA_PRESS_STYLE_ID = 'react-aria-pressable-style'

export function guardAriaStyle(doc: Document = document): void {
  if (doc.getElementById(ARIA_PRESS_STYLE_ID)) return
  const marker = doc.createElement('meta')
  marker.id = ARIA_PRESS_STYLE_ID
  marker.name = 'kongtrol-aria-press-style'
  marker.content = 'in kit-controls.css'
  doc.head.append(marker)
}
