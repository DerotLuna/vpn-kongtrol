// Kongtrol UI — the monkey mark · v0.1.0 · 2026-11-18
//
// The line-art master assets/brand/logo-kong.svg (single source, ADR-0013)
// used as a CSS mask filled with currentColor: light lines on dark, dark
// lines on light, no background disc, vector-crisp at any size.
import logoUrl from '@brand/logo-kong.svg'

export function Logo({ size = 32 }: { size?: number }) {
  return (
    <span
      className="k-logo"
      aria-hidden="true"
      style={{ width: size, height: size, maskImage: `url("${logoUrl}")`, WebkitMaskImage: `url("${logoUrl}")` }}
    />
  )
}
