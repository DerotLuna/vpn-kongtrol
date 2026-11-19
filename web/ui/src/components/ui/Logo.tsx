// Kongtrol UI — the monkey mark · v0.2.0 · 2026-11-19
//
// The brand mark (single source assets/brand, ADR-0013) used as a CSS mask
// filled with currentColor: light on dark, dark on light, no background disc,
// vector-crisp at any size. Size rule (DESIGN.md, ADR-0045): up to 32 px the
// solid silhouette (the concentric lines blur into grey that small), from
// 36 px up the line-art original.
import lineUrl from '@brand/logo-kong.svg'
import solidUrl from '@brand/source/logo-kong-solid.svg'

export function Logo({ size = 32 }: { size?: number }) {
  const url = size <= 32 ? solidUrl : lineUrl
  return (
    <span
      className="k-logo"
      aria-hidden="true"
      style={{ width: size, height: size, maskImage: `url("${url}")`, WebkitMaskImage: `url("${url}")` }}
    />
  )
}
