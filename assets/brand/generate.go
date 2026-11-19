//go:build brandgen

// Brand asset generator — version 1.5.0 · 2026-11-19.
//
// Regenerates the raster fallbacks in assets/brand/ from the masters in
// assets/brand/source/. The SVGs (logo.svg, logo-kong.svg, favicon.svg, plus
// logo-kong-on-dark.svg, the white-ink mark the README shows on GitHub's dark
// theme) are the masters for anything that can render vectors; the rasters exist only
// for consumers that cannot (OS tray icons, Open Graph cards, app manifests,
// README images).
//
// Size rule (DESIGN.md, ADR-0047): below ~32 px the solid mark is used
// (favicon.svg, logo-kong-32.png, tray-<px>.png, all derived from
// source/logo-kong-solid.svg below); visible logos from ~40 px up use the
// line-art original. logo.svg is the "Kongtrol" lockup (the name is just
// Kongtrol, ADR-0047); logo-512.png is rasterized from that vector.
//
// App icon (ADR-0052, the only logo with a background): Ink squircle plate,
// solid mark in Bone, from source/logo-kong-solid.svg — assets/logo.ico (both
// Windows binaries and Setup.exe), app-icon-1024.png (macOS .icns),
// app-icon-512/256.png (Linux), app-icon-180.png (web touch icon),
// source/app-icon.svg (vector reference). Installer images:
// installer/wizard-<w>x<h>.png (lockup in Bone on Ink, no words) and
// installer/wizard-small-<px>.png (the app icon).
//
// Uses only the standard library plus golang.org/x/image (in go.mod; the
// build tag is "brandgen", not "ignore", because go mod tidy skips files
// tagged ignore and would drop the dependency). Run from the repo root:
//
//	go run -tags brandgen assets/brand/generate.go
//
// Encoding choices:
//   - The logo is black line art on white, so logo rasters are converted to
//     luma and stored as a 16-level grayscale palette (4-bit PNG). 16 levels
//     are enough for anti-aliased edges and cut the size several-fold.
//   - The Open Graph card is a photo-like gradient, so it is emitted as JPEG,
//     which every OG/Twitter consumer accepts (https://ogp.me/#structured).
//   - WebP is not emitted: golang.org/x/image/webp only decodes.
//   - source/logo-kong-solid.svg (the same head, solid instead of lines, the
//     face as negative space) is derived from source/logo-kong.svg here; see
//     deriveSolid. It is the tray / menu bar glyph (ADR-0044): at 16–24 px
//     the concentric lines of the full mark blur into grey.
//   - Tray / menu bar masks (tray-<px>.png) are rasterized straight from that
//     vector (golang.org/x/image/vector), cropped to the mark, as black on a
//     TRANSPARENT background, with an edge-contrast hint at ≤32 px (see
//     hint): the tray tints them for the taskbar theme (Windows, Linux) or
//     hands them to macOS as a template image, and draws the state badge on
//     top at run time (assets.BadgeGeometry). One file per tray size, so
//     small sizes are rendered, not downscaled.
package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"log"
	"math"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"golang.org/x/image/draw"
	"golang.org/x/image/vector"
)

// traySizes are the tray / menu bar icon sizes: Windows small icons at
// 100–200 % scaling (16, 20, 24, 32), its large icon (32, 48), macOS
// menu bar @1x/@2x (18/36 fit in 16..48 after scaling) and Linux panels.
var traySizes = []int{16, 20, 24, 32, 48, 64}

const dir = "assets/brand"

type format int

const (
	lineArtPNG format = iota // 16-level grayscale PNG
	photoJPEG                // JPEG, quality 88
)

type job struct {
	src  string
	out  string
	w, h int
	fmt  format
}

func main() {
	jobs := []job{
		{"source/logo-kong.png", "logo-kong-64.png", 64, 64, lineArtPNG},
		{"source/logo-kong.png", "logo-kong-512.png", 512, 512, lineArtPNG},
		{"source/social-preview.png", "og-1200x630.jpg", 1200, 630, photoJPEG},
	}
	for _, name := range []string{"logo.svg", "logo-kong.svg"} {
		n, err := minifySVG(filepath.Join(dir, "source", name), filepath.Join(dir, name), false)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Printf("%-20s %9s %7d bytes\n", name, "vector", n)
	}
	// White-ink copy of the mark for dark backgrounds where the page picks the
	// image (README <picture> with prefers-color-scheme sources), since an SVG
	// in <img> follows the OS theme, not the site's theme.
	n, err := minifySVG(filepath.Join(dir, "source", "logo-kong.svg"), filepath.Join(dir, "logo-kong-on-dark.svg"), true)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("%-20s %9s %7d bytes\n", "logo-kong-on-dark.svg", "vector", n)
	for _, j := range jobs {
		src, err := load(filepath.Join(dir, j.src))
		if err != nil {
			log.Fatal(err)
		}
		n, err := save(filepath.Join(dir, j.out), cover(src, j.w, j.h), j.fmt)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Printf("%-20s %4dx%-4d %7d bytes\n", j.out, j.w, j.h, n)
	}
	if err := renderLockup(); err != nil {
		log.Fatal(err)
	}
	if err := deriveSolid(); err != nil {
		log.Fatal(err)
	}
	if err := renderFavicons(); err != nil {
		log.Fatal(err)
	}
	if err := renderTrayMasks(); err != nil {
		log.Fatal(err)
	}
	if err := renderAppIcons(); err != nil {
		log.Fatal(err)
	}
	if err := renderInstallerImages(); err != nil {
		log.Fatal(err)
	}
}

// renderLockup rasterizes the vector lockup source/logo.svg to logo-512.png
// (black on white, 16-level gray), so the raster never drifts from the vector.
func renderLockup() error {
	src := filepath.Join(dir, "source", "logo.svg")
	paths, err := loadSVGPaths(src)
	if err != nil {
		return err
	}
	vb, err := svgViewBox(src)
	if err != nil {
		return err
	}
	const px = 512
	k := px / math.Max(vb[2], vb[3])
	img := fill(paths, px, func(p pt) (float32, float32) {
		return float32((p.x - vb[0]) * k), float32((p.y - vb[1]) * k)
	})
	n, err := save(filepath.Join(dir, "logo-512.png"), onWhite(img), lineArtPNG)
	if err != nil {
		return err
	}
	fmt.Printf("%-20s %4dx%-4d %7d bytes\n", "logo-512.png", px, px, n)
	return nil
}

// renderFavicons writes the favicons from the solid mark (line art blurs
// into grey at tab size): favicon.svg, cropped to the mark, transparent,
// black ink that turns white under prefers-color-scheme: dark (it follows the
// browser chrome, not the page theme); and logo-kong-32.png, the PNG
// fallback for browsers without SVG favicons, black on white like before
// (a PNG cannot switch with the theme; the name stays for old links).
func renderFavicons() error {
	src := filepath.Join(dir, solidOut)
	b, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	m := svgPathD.FindStringSubmatch(string(b))
	if m == nil {
		return fmt.Errorf("%s: no path", src)
	}
	paths, err := loadSVGPaths(src)
	if err != nil {
		return err
	}
	minX, minY, maxX, maxY := math.Inf(1), math.Inf(1), math.Inf(-1), math.Inf(-1)
	for _, s := range paths {
		for _, p := range s.p {
			minX, minY = math.Min(minX, p.x), math.Min(minY, p.y)
			maxX, maxY = math.Max(maxX, p.x), math.Max(maxY, p.y)
		}
	}
	// Square viewBox around the mark with a 2 % margin, rounded outward.
	side := math.Max(maxX-minX, maxY-minY) * 1.02
	x0 := math.Floor((minX + maxX - side) / 2)
	y0 := math.Floor((minY + maxY - side) / 2)
	side = math.Ceil(side) + 1
	svg := fmt.Sprintf(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="%s %s %s %s">`+
		`<style>@media (prefers-color-scheme: dark){path{fill:#ffffff}}</style>`+
		`<path fill="#000000" fill-rule="evenodd" d="%s"/></svg>`+"\n",
		num(x0), num(y0), num(side), num(side), m[1])
	if err := os.WriteFile(filepath.Join(dir, "favicon.svg"), []byte(svg), 0o644); err != nil {
		return err
	}
	fmt.Printf("%-20s %9s %7d bytes\n", "favicon.svg", "vector", len(svg))
	n, err := save(filepath.Join(dir, "logo-kong-32.png"), onWhite(hint(rasterMask(paths, 32), 32)), lineArtPNG)
	if err != nil {
		return err
	}
	fmt.Printf("%-20s %4dx%-4d %7d bytes\n", "logo-kong-32.png", 32, 32, n)
	return nil
}

// onWhite composites black ink with alpha coverage onto white.
func onWhite(img *image.NRGBA) *image.Gray {
	out := image.NewGray(img.Bounds())
	for i := range out.Pix {
		out.Pix[i] = 255 - img.Pix[i*4+3]
	}
	return out
}

// renderTrayMasks rasterizes source/logo-kong-solid.svg to tray-<px>.png for
// every traySizes entry: black + alpha, cropped to the mark, hinted.
func renderTrayMasks() error {
	paths, err := loadSVGPaths(filepath.Join(dir, solidOut))
	if err != nil {
		return err
	}
	for _, px := range traySizes {
		out := fmt.Sprintf("tray-%d.png", px)
		var buf bytes.Buffer
		if err := (&png.Encoder{CompressionLevel: png.BestCompression}).Encode(&buf, hint(rasterMask(paths, px), px)); err != nil {
			return err
		}
		if err := os.WriteFile(filepath.Join(dir, out), buf.Bytes(), 0o644); err != nil {
			return err
		}
		fmt.Printf("%-20s %4dx%-4d %7d bytes\n", out, px, px, buf.Len())
	}
	return nil
}

// ── Solid mark: derived from the line-art master ────────────────────────────
//
// source/logo-kong-solid.svg is the same head as source/logo-kong.svg with
// the concentric contour lines merged into one solid shape and the face plate
// (with eyes, nose and mouth) kept as negative space, for sizes where the
// lines blur into grey (tray, menu bar). It is derived, not redrawn:
//
//  1. rasterize the master at solidRes px (golang.org/x/image/vector);
//  2. morphological closing (dilate, then erode, by solidClose px) on exact
//     Euclidean distance transforms merges the lines across the gaps
//     (Gonzalez & Woods, Digital Image Processing, ch. 9; distance transform:
//     Felzenszwalb & Huttenlocher, "Distance Transforms of Sampled
//     Functions", Theory of Computing 8, 2012);
//  3. flood fill from the canvas edge: everything not reached is the head;
//  4. white regions of the master enclosed by ink whose inscribed circle is
//     wider than solidHole px (only the face plate; the gaps between lines
//     are much thinner) are cut back out, so the face keeps its features;
//  5. blur slightly, trace the 50 % iso-contour (marching squares), simplify
//     (Douglas–Peucker) and fit smooth cubic Béziers, corners kept.
//
// Every outline keeps the ink on the same side, so outer edges and holes
// alternate winding and the SVG renders the same with nonzero or evenodd.
const (
	solidSrc   = "source/logo-kong.svg"
	solidOut   = "source/logo-kong-solid.svg"
	solidRes   = 2048 // px across the master's viewBox
	solidPad   = 96   // px of empty margin so the dilation never hits the canvas edge
	solidClose = 30.0 // closing radius, px (> half the widest gap between lines)
	solidSeal  = 10.0 // light closing radius, px, that seals slits in the face plate rim
	solidHole  = 40.0 // min inscribed radius, px, of a white region kept as a hole
	solidBlur  = 1.5  // Gaussian sigma, px, before tracing (removes the pixel staircase)
	solidTol   = 0.8  // Douglas–Peucker tolerance, px
	solidAngle = 50.0 // degrees of turn at a vertex above which it stays a corner
)

func deriveSolid() error {
	src := filepath.Join(dir, solidSrc)
	paths, err := loadSVGPaths(src)
	if err != nil {
		return err
	}
	vb, err := svgViewBox(src)
	if err != nil {
		return err
	}
	k := solidRes / math.Max(vb[2], vb[3])
	n := solidRes + 2*solidPad
	img := fill(paths, n, func(p pt) (float32, float32) {
		return float32(solidPad + (p.x-vb[0])*k), float32(solidPad + (p.y-vb[1])*k)
	})
	ink := make([]bool, n*n)
	for i := range ink {
		ink[i] = img.Pix[i*4+3] >= 128
	}
	solid, st := solidify(ink, n)
	g := make([]float64, n*n)
	for i, on := range solid {
		if on {
			g[i] = 1
		}
	}
	g = blur(g, n, solidBlur)
	var d strings.Builder
	points := 0
	for _, loop := range traceContours(g, n, 0.5) {
		poly := simplifyClosed(loop, solidTol)
		if len(poly) < 3 {
			continue
		}
		points += len(poly)
		d.WriteString(bezierPath(poly, func(p pt) pt {
			return pt{vb[0] + (p.x-solidPad)/k, vb[1] + (p.y-solidPad)/k}
		}))
	}
	svg := fmt.Sprintf(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="%s %s %s %s">`+
		`<!-- Derived from logo-kong.svg by assets/brand/generate.go; do not edit. -->`+
		`<path fill="#000000" fill-rule="evenodd" d="%s"/></svg>`+"\n",
		num(vb[0]), num(vb[1]), num(vb[2]), num(vb[3]), strings.TrimSpace(d.String()))
	out := filepath.Join(dir, solidOut)
	if err := os.WriteFile(out, []byte(svg), 0o644); err != nil {
		return err
	}
	fmt.Printf("%-20s %9s %7d bytes (%d points; closing %gpx, holes kept %v px inscribed, gaps up to %.1f px filled, outline within %.1f px of the master's ink)\n",
		filepath.Base(out), "vector", len(svg), points, solidClose, st.holes, st.filled, st.outline)
	return nil
}

type solidStats struct {
	holes   []string // inscribed radius, px, of each white region kept as a hole
	filled  float64  // widest gap filled in: max distance, px, from a filled pixel to the master's ink
	outline float64  // max distance, px, from the solid's outer edge to the master's ink
}

// closing returns the morphological closing of set by a disc of radius r
// (dilate, then erode), using exact distance transforms.
func closing(dSet []float64, n int, r float64) []bool {
	notDil := make([]bool, n*n)
	for i, d := range dSet {
		notDil[i] = d > r*r
	}
	out := make([]bool, n*n)
	for i, d := range edt(notDil, n) {
		out[i] = d > r*r
	}
	return out
}

// solidify returns the closed, flood-filled head minus the face plate.
func solidify(ink []bool, n int) ([]bool, solidStats) {
	var st solidStats
	dInk := edt(ink, n)
	closed := closing(dInk, n, solidClose)
	outside := floodFrom(0, n, func(i int) bool { return !closed[i] })
	head := make([]bool, n*n)
	for i := range head {
		head[i] = !outside[i]
	}
	// Face plate: an enclosed white region much wider than any gap. Found on
	// a light closing (solidSeal) that only seals slits in its rim, then
	// grown back over the master's own white within solidSeal px, so eyes,
	// nose and mouth keep the master's exact shapes.
	sealed := closing(dInk, n, solidSeal)
	dSealed := edt(sealed, n)
	seen := make([]bool, n*n)
	flood(0, n, func(i int) bool { return !sealed[i] }, seen)
	for i := range sealed {
		if sealed[i] || seen[i] {
			continue
		}
		region := flood(i, n, func(j int) bool { return !sealed[j] }, seen)
		inscribed := 0.0
		for _, j := range region {
			inscribed = math.Max(inscribed, math.Sqrt(dSealed[j]))
		}
		if inscribed <= solidHole {
			continue
		}
		st.holes = append(st.holes, fmt.Sprintf("%.0f", inscribed))
		core := make([]bool, n*n)
		for _, j := range region {
			core[j] = true
		}
		dCore := edt(core, n)
		grown := make([]bool, n*n)
		for _, j := range flood(region[0], n, func(j int) bool {
			return !ink[j] && dCore[j] <= solidSeal*solidSeal
		}, grown) {
			head[j] = false
		}
	}
	for i, on := range head {
		if on && !ink[i] {
			st.filled = math.Max(st.filled, math.Sqrt(dInk[i]))
		}
	}
	// Outline fidelity: how far the solid's outer edge lies from real ink.
	for y := 1; y < n-1; y++ {
		for x := 1; x < n-1; x++ {
			i := y*n + x
			if head[i] && (outside[i-1] || outside[i+1] || outside[i-n] || outside[i+n]) {
				st.outline = math.Max(st.outline, math.Sqrt(dInk[i]))
			}
		}
	}
	return head, st
}

// floodFrom returns the 4-connected region around seed where pass is true.
func floodFrom(seed, n int, pass func(int) bool) []bool {
	seen := make([]bool, n*n)
	flood(seed, n, pass, seen)
	return seen
}

// flood marks in seen, and returns, the 4-connected pixels around seed where
// pass is true and seen is still false.
func flood(seed, n int, pass func(int) bool, seen []bool) []int {
	if seen[seed] || !pass(seed) {
		return nil
	}
	seen[seed] = true
	stack, region := []int{seed}, []int{seed}
	for len(stack) > 0 {
		i := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		x, y := i%n, i/n
		for _, j := range [4]int{i - 1, i + 1, i - n, i + n} {
			if (j == i-1 && x == 0) || (j == i+1 && x == n-1) || (j == i-n && y == 0) || (j == i+n && y == n-1) {
				continue
			}
			if !seen[j] && pass(j) {
				seen[j] = true
				stack = append(stack, j)
				region = append(region, j)
			}
		}
	}
	return region
}

// edt returns, per pixel, the squared Euclidean distance to the nearest
// pixel of set (Felzenszwalb & Huttenlocher 2012, two 1-D lower-envelope
// passes; exact, O(pixels)).
func edt(set []bool, n int) []float64 {
	const far = 1e20
	d := make([]float64, n*n)
	for i, on := range set {
		if !on {
			d[i] = far
		}
	}
	f, out := make([]float64, n), make([]float64, n)
	v, z := make([]int, n), make([]float64, n+1)
	pass := func(base, step int) {
		for q := 0; q < n; q++ {
			f[q] = d[base+q*step]
		}
		k := 0
		v[0], z[0], z[1] = 0, math.Inf(-1), math.Inf(1)
		for q := 1; q < n; q++ {
			s := ((f[q] + float64(q*q)) - (f[v[k]] + float64(v[k]*v[k]))) / float64(2*(q-v[k]))
			for s <= z[k] {
				k--
				s = ((f[q] + float64(q*q)) - (f[v[k]] + float64(v[k]*v[k]))) / float64(2*(q-v[k]))
			}
			k++
			v[k], z[k], z[k+1] = q, s, math.Inf(1)
		}
		k = 0
		for q := 0; q < n; q++ {
			for z[k+1] < float64(q) {
				k++
			}
			dq := float64(q - v[k])
			out[q] = dq*dq + f[v[k]]
		}
		for q := 0; q < n; q++ {
			d[base+q*step] = out[q]
		}
	}
	for x := 0; x < n; x++ {
		pass(x, n)
	}
	for y := 0; y < n; y++ {
		pass(y*n, 1)
	}
	return d
}

// blur is a separable Gaussian blur (sigma in px) of an n×n field.
func blur(g []float64, n int, sigma float64) []float64 {
	rad := int(math.Ceil(3 * sigma))
	w := make([]float64, 2*rad+1)
	sum := 0.0
	for i := range w {
		x := float64(i - rad)
		w[i] = math.Exp(-x * x / (2 * sigma * sigma))
		sum += w[i]
	}
	for i := range w {
		w[i] /= sum
	}
	tmp, out := make([]float64, n*n), make([]float64, n*n)
	for y := 0; y < n; y++ {
		for x := 0; x < n; x++ {
			a := 0.0
			for i, wi := range w {
				if xx := x + i - rad; xx >= 0 && xx < n {
					a += wi * g[y*n+xx]
				}
			}
			tmp[y*n+x] = a
		}
	}
	for y := 0; y < n; y++ {
		for x := 0; x < n; x++ {
			a := 0.0
			for i, wi := range w {
				if yy := y + i - rad; yy >= 0 && yy < n {
					a += wi * tmp[yy*n+x]
				}
			}
			out[y*n+x] = a
		}
	}
	return out
}

// traceContours returns the closed iso-lines of g at thr (marching squares
// with linear interpolation; saddles resolved by the cell's mean). Each
// segment runs from the edge where the walk enters the ink to the edge where
// it leaves, so every loop keeps the ink on the same side. g must be below
// thr along the canvas border.
func traceContours(g []float64, n int, thr float64) [][]pt {
	in := func(x, y int) bool { return g[y*n+x] > thr }
	next := map[int]int{}
	at := map[int]pt{}
	for y := 0; y < n-1; y++ {
		for x := 0; x < n-1; x++ {
			c := [4][2]int{{x, y}, {x + 1, y}, {x + 1, y + 1}, {x, y + 1}} // clockwise
			keys := [4]int{2 * (y*n + x), 2*(y*n+x+1) + 1, 2 * ((y+1)*n + x), 2*(y*n+x) + 1}
			type cross struct {
				key   int
				enter bool
			}
			var cs []cross
			for e := 0; e < 4; e++ {
				a, b := c[e], c[(e+1)%4]
				ia, ib := in(a[0], a[1]), in(b[0], b[1])
				if ia == ib {
					continue
				}
				if _, ok := at[keys[e]]; !ok {
					ga, gb := g[a[1]*n+a[0]], g[b[1]*n+b[0]]
					t := (thr - ga) / (gb - ga)
					at[keys[e]] = pt{float64(a[0]) + t*float64(b[0]-a[0]), float64(a[1]) + t*float64(b[1]-a[1])}
				}
				cs = append(cs, cross{keys[e], ib})
			}
			if len(cs) == 0 {
				continue
			}
			for !cs[0].enter {
				cs = append(cs[1:], cs[0])
			}
			if len(cs) == 2 {
				next[cs[0].key] = cs[1].key
				continue
			}
			mean := (g[y*n+x] + g[y*n+x+1] + g[(y+1)*n+x] + g[(y+1)*n+x+1]) / 4
			if mean > thr { // ink joins across the cell
				next[cs[0].key], next[cs[2].key] = cs[3].key, cs[1].key
			} else {
				next[cs[0].key], next[cs[2].key] = cs[1].key, cs[3].key
			}
		}
	}
	starts := make([]int, 0, len(next))
	for k := range next {
		starts = append(starts, k)
	}
	sort.Ints(starts) // deterministic output
	var loops [][]pt
	for _, s := range starts {
		if _, ok := next[s]; !ok {
			continue
		}
		var loop []pt
		for k := s; ; {
			loop = append(loop, at[k])
			nk := next[k]
			delete(next, k)
			k = nk
			if k == s {
				break
			}
		}
		loops = append(loops, loop)
	}
	return loops
}

// simplifyClosed runs Douglas–Peucker on a closed polyline, split at its
// first point and the point farthest from it.
func simplifyClosed(p []pt, tol float64) []pt {
	far, best := 0, -1.0
	for i, q := range p {
		if d := math.Hypot(q.x-p[0].x, q.y-p[0].y); d > best {
			far, best = i, d
		}
	}
	a := dp(p[:far+1], tol)
	b := dp(append(append([]pt{}, p[far:]...), p[0]), tol)
	return append(a[:len(a)-1], b[:len(b)-1]...)
}

func dp(p []pt, tol float64) []pt {
	if len(p) < 3 {
		return p
	}
	a, b := p[0], p[len(p)-1]
	dx, dy := b.x-a.x, b.y-a.y
	l := math.Hypot(dx, dy)
	idx, worst := 0, -1.0
	for i := 1; i < len(p)-1; i++ {
		var d float64
		if l == 0 {
			d = math.Hypot(p[i].x-a.x, p[i].y-a.y)
		} else {
			d = math.Abs(dy*(p[i].x-a.x)-dx*(p[i].y-a.y)) / l
		}
		if d > worst {
			idx, worst = i, d
		}
	}
	if worst <= tol {
		return []pt{a, b}
	}
	left := dp(p[:idx+1], tol)
	return append(left[:len(left)-1], dp(p[idx:], tol)...)
}

// bezierPath turns a closed polygon into a smooth closed cubic path through
// its vertices (tangent along the neighbours' chord, handles a third of the
// segment), except at vertices that turn more than solidAngle, which stay
// corners. xf maps tracing px to viewBox units.
func bezierPath(p []pt, xf func(pt) pt) string {
	m := len(p)
	unit := func(dx, dy float64) pt {
		l := math.Hypot(dx, dy)
		if l == 0 {
			return pt{}
		}
		return pt{dx / l, dy / l}
	}
	in := make([]pt, m)  // tangent arriving at vertex i
	out := make([]pt, m) // tangent leaving vertex i
	for i := range p {
		prev, cur, nxt := p[(i+m-1)%m], p[i], p[(i+1)%m]
		a := unit(cur.x-prev.x, cur.y-prev.y)
		b := unit(nxt.x-cur.x, nxt.y-cur.y)
		turn := math.Acos(math.Max(-1, math.Min(1, a.x*b.x+a.y*b.y))) * 180 / math.Pi
		if turn > solidAngle {
			in[i], out[i] = a, b
			continue
		}
		t := unit(nxt.x-prev.x, nxt.y-prev.y)
		in[i], out[i] = t, t
	}
	var s strings.Builder
	q := xf(p[0])
	fmt.Fprintf(&s, "M%s %s", num(q.x), num(q.y))
	for i := 0; i < m; i++ {
		a, b := p[i], p[(i+1)%m]
		h := math.Hypot(b.x-a.x, b.y-a.y) / 3
		c1 := xf(pt{a.x + out[i].x*h, a.y + out[i].y*h})
		c2 := xf(pt{b.x - in[(i+1)%m].x*h, b.y - in[(i+1)%m].y*h})
		e := xf(b)
		fmt.Fprintf(&s, "C%s %s %s %s %s %s", num(c1.x), num(c1.y), num(c2.x), num(c2.y), num(e.x), num(e.y))
	}
	s.WriteString("Z")
	return s.String()
}

// num formats a viewBox coordinate with one decimal (0.1 unit is about
// 1/5000 of the mark), dropping a trailing ".0".
func num(v float64) string {
	return strconv.FormatFloat(math.Round(v*10)/10, 'f', -1, 64)
}

var svgViewBoxRe = regexp.MustCompile(`viewBox="([-\d.]+)[ ,]+([-\d.]+)[ ,]+([-\d.]+)[ ,]+([-\d.]+)"`)

// svgViewBox returns minX, minY, width, height of the root viewBox.
func svgViewBox(path string) ([4]float64, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return [4]float64{}, err
	}
	m := svgViewBoxRe.FindStringSubmatch(string(b))
	if m == nil {
		return [4]float64{}, fmt.Errorf("%s: no viewBox", path)
	}
	return [4]float64{atof(m[1]), atof(m[2]), atof(m[3]), atof(m[4])}, nil
}

// hint sharpens anti-aliased edges at small sizes (a contrast curve on the
// coverage around 50 %), the raster equivalent of hinting a glyph to the
// pixel grid: half-covered edge pixels become clearly on or off, so bold
// shapes stay crisp at 16–24 px instead of turning grey. Not true
// grid-fitting (no outline moves); larger sizes are left untouched.
func hint(img *image.NRGBA, px int) *image.NRGBA {
	gain := 1.0
	switch {
	case px <= 20:
		gain = 1.8
	case px <= 24:
		gain = 1.5
	case px <= 32:
		gain = 1.2
	}
	if gain == 1 {
		return img
	}
	for i := 3; i < len(img.Pix); i += 4 {
		a := (float64(img.Pix[i])/255-0.5)*gain + 0.5
		img.Pix[i] = uint8(math.Round(math.Max(0, math.Min(1, a)) * 255))
	}
	return img
}

// ── SVG master → tray masks ─────────────────────────────────────────────────

type pt struct{ x, y float64 }

// seg is one path command in viewBox units: 'M', 'L', 'C' (3 points) or 'Z'.
type seg struct {
	op byte
	p  []pt
}

var (
	svgPathD     = regexp.MustCompile(`<path[^>]*\sd="([^"]+)"`)
	svgTransform = regexp.MustCompile(`translate\(([-\d.]+),([-\d.]+)\)\s*scale\(([-\d.]+),([-\d.]+)\)`)
	svgToken     = regexp.MustCompile(`[MmLlCcZz]|-?(?:\d+\.?\d*|\.\d+)(?:[eE][-+]?\d+)?`)
)

// loadSVGPaths reads a potrace SVG (one <g transform="translate() scale()">
// with M/m, L/l, C/c, Z/z paths, https://www.w3.org/TR/SVG2/paths.html) and
// returns its outlines in viewBox units. Anything else is rejected rather
// than silently drawn wrong.
func loadSVGPaths(path string) ([]seg, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	s := string(b)
	tx, ty, sx, sy := 0.0, 0.0, 1.0, 1.0
	if m := svgTransform.FindStringSubmatch(s); m != nil {
		tx, ty, sx, sy = atof(m[1]), atof(m[2]), atof(m[3]), atof(m[4])
	}
	xf := func(p pt) pt { return pt{tx + sx*p.x, ty + sy*p.y} }
	var out []seg
	for _, m := range svgPathD.FindAllStringSubmatch(s, -1) {
		toks := svgToken.FindAllString(m[1], -1)
		var cur, start pt
		var cmd byte
		i := 0
		num := func() float64 { i++; return atof(toks[i-1]) }
		for i < len(toks) {
			if c := toks[i][0]; strings.ContainsRune("MmLlCcZz", rune(c)) {
				cmd = c
				i++
				if cmd == 'Z' || cmd == 'z' {
					out = append(out, seg{op: 'Z'})
					cur = start
					continue
				}
			}
			rel := cmd >= 'a'
			at := func(x, y float64) pt {
				if rel {
					return pt{cur.x + x, cur.y + y}
				}
				return pt{x, y}
			}
			switch cmd {
			case 'M', 'm':
				cur = at(num(), num())
				start = cur
				out = append(out, seg{op: 'M', p: []pt{xf(cur)}})
				// Further pairs after a moveto are implicit linetos.
				if cmd == 'm' {
					cmd = 'l'
				} else {
					cmd = 'L'
				}
			case 'L', 'l':
				cur = at(num(), num())
				out = append(out, seg{op: 'L', p: []pt{xf(cur)}})
			case 'C', 'c':
				c1 := at(num(), num())
				c2 := at(num(), num())
				end := at(num(), num())
				cur = end
				out = append(out, seg{op: 'C', p: []pt{xf(c1), xf(c2), xf(end)}})
			default:
				return nil, fmt.Errorf("%s: unsupported path command %q", path, cmd)
			}
		}
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("%s: no paths", path)
	}
	return out, nil
}

func atof(s string) float64 {
	f, err := strconv.ParseFloat(s, 64)
	if err != nil {
		log.Fatalf("svg number %q: %v", s, err)
	}
	return f
}

// rasterMask fills the outlines at px×px, cropped to the mark's bounding
// box (square, centered, 1px margin), as black with anti-aliased alpha on
// a transparent background. potrace emits holes with opposite winding, so
// the rasterizer's non-zero accumulation keeps them open.
func rasterMask(paths []seg, px int) *image.NRGBA {
	minX, minY, maxX, maxY := math.Inf(1), math.Inf(1), math.Inf(-1), math.Inf(-1)
	for _, s := range paths {
		for _, p := range s.p {
			minX, minY = math.Min(minX, p.x), math.Min(minY, p.y)
			maxX, maxY = math.Max(maxX, p.x), math.Max(maxY, p.y)
		}
	}
	side := math.Max(maxX-minX, maxY-minY)
	inner := float64(px) - 2
	k := inner / side
	ox := 1 + (inner-(maxX-minX)*k)/2 - minX*k
	oy := 1 + (inner-(maxY-minY)*k)/2 - minY*k
	return fill(paths, px, func(p pt) (float32, float32) { return float32(ox + p.x*k), float32(oy + p.y*k) })
}

// fill rasterizes the outlines mapped by f into a px×px black + alpha image.
func fill(paths []seg, px int, f func(pt) (float32, float32)) *image.NRGBA {
	return fillWH(paths, px, px, f)
}

// fillWH is fill on a w×h canvas.
func fillWH(paths []seg, w, h int, f func(pt) (float32, float32)) *image.NRGBA {
	r := vector.NewRasterizer(w, h)
	for _, s := range paths {
		switch s.op {
		case 'M':
			r.MoveTo(f(s.p[0]))
		case 'L':
			r.LineTo(f(s.p[0]))
		case 'C':
			x1, y1 := f(s.p[0])
			x2, y2 := f(s.p[1])
			x3, y3 := f(s.p[2])
			r.CubeTo(x1, y1, x2, y2, x3, y3)
		case 'Z':
			r.ClosePath()
		}
	}
	mask := image.NewAlpha(image.Rect(0, 0, w, h))
	r.Draw(mask, mask.Bounds(), image.Opaque, image.Point{})
	out := image.NewNRGBA(mask.Bounds())
	for i, a := range mask.Pix {
		out.Pix[i*4+3] = a // black ink, alpha = coverage
	}
	return out
}

func load(path string) (image.Image, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	img, _, err := image.Decode(f)
	return img, err
}

// cover scales src to fill w×h keeping its aspect ratio, then center-crops the
// overflow (like CSS object-fit: cover). CatmullRom gives crisp downscaling.
func cover(src image.Image, w, h int) image.Image {
	sb := src.Bounds()
	scale := max(float64(w)/float64(sb.Dx()), float64(h)/float64(sb.Dy()))
	sw, sh := int(float64(sb.Dx())*scale+0.5), int(float64(sb.Dy())*scale+0.5)
	scaled := image.NewNRGBA(image.Rect(0, 0, sw, sh))
	draw.CatmullRom.Scale(scaled, scaled.Bounds(), src, sb, draw.Src, nil)
	ox, oy := (sw-w)/2, (sh-h)/2
	dst := image.NewNRGBA(image.Rect(0, 0, w, h))
	draw.Copy(dst, image.Point{}, scaled, image.Rect(ox, oy, ox+w, oy+h), draw.Src, nil)
	return dst
}

func save(path string, img image.Image, f format) (int, error) {
	var buf bytes.Buffer
	var err error
	switch f {
	case photoJPEG:
		err = jpeg.Encode(&buf, img, &jpeg.Options{Quality: 88})
	default:
		enc := png.Encoder{CompressionLevel: png.BestCompression}
		err = enc.Encode(&buf, grayPalette16(img))
	}
	if err != nil {
		return 0, err
	}
	return buf.Len(), os.WriteFile(path, buf.Bytes(), 0o644)
}

// grayPalette16 maps every pixel to its nearest of 16 evenly spaced gray
// levels (0x00, 0x11, ... 0xff), producing a 4-bit paletted PNG.
func grayPalette16(img image.Image) *image.Paletted {
	pal := make(color.Palette, 16)
	for i := range pal {
		v := uint8(i * 0x11)
		pal[i] = color.Gray{Y: v}
	}
	b := img.Bounds()
	out := image.NewPaletted(b, pal)
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			g := color.GrayModel.Convert(img.At(x, y)).(color.Gray)
			out.SetColorIndex(x, y, uint8((int(g.Y)+8)/0x11))
		}
	}
	return out
}

var (
	svgProlog   = regexp.MustCompile(`(?s)<\?xml.*?\?>|<!DOCTYPE.*?>|<!--.*?-->|/\*.*?\*/`)
	svgSpace    = regexp.MustCompile(`\s+`)
	svgTagGap   = regexp.MustCompile(`>\s+<`)
	svgZeroFrac = regexp.MustCompile(`(\d)\.0+(\D|$)`)        // 527.000000pt -> 527pt
	svgTrailing = regexp.MustCompile(`(\.\d*?[1-9])0+(\D|$)`) // 0.100000 -> 0.1
)

// minifySVG applies safe, lossless text reductions to a potrace-style SVG:
// drops the XML prolog, DOCTYPE and comments, collapses whitespace, removes
// the space before negative path numbers ("c-38 -5" -> "c-38-5", valid per
// the SVG path grammar: https://www.w3.org/TR/SVG2/paths.html#PathDataBNF)
// and redundant zero fractions. Geometry is unchanged. inverse turns the
// black ink (potrace fill="#000000") white.
func minifySVG(src, dst string, inverse bool) (int, error) {
	b, err := os.ReadFile(src)
	if err != nil {
		return 0, err
	}
	s := svgProlog.ReplaceAllString(string(b), "")
	s = svgSpace.ReplaceAllString(s, " ")
	s = svgTagGap.ReplaceAllString(s, "><")
	s = strings.ReplaceAll(s, " -", "-")
	s = svgZeroFrac.ReplaceAllString(s, "$1$2")
	s = svgTrailing.ReplaceAllString(s, "$1$2")
	if inverse {
		if !strings.Contains(s, `fill="#000000"`) {
			return 0, fmt.Errorf("%s: no fill=\"#000000\" to invert", src)
		}
		s = strings.ReplaceAll(s, `fill="#000000"`, `fill="#ffffff"`)
	}
	s = strings.TrimSpace(s) + "\n"
	return len(s), os.WriteFile(dst, []byte(s), 0o644)
}

// ── App icon and installer images (ADR-0052) ────────────────────────────────
//
// The app icon is the one exception to "logos have no background"
// (DESIGN.md "Logo"): a squircle plate in Ink with the solid mark in Bone
// Paper, both DESIGN.md tokens. It is drawn here from
// source/logo-kong-solid.svg; source/app-icon.svg is its vector form (macOS
// geometry), written for reference and never edited by hand.
//
// Geometry (fractions of the square canvas; sources and what is unverified
// in docs/research/2026-11-19-app-icon.md):
//   - macOS: plate 824/1024 centered (Apple's Big Sur icon grid; the
//     transparent margin is part of the icon); the .icns is built from
//     app-icon-1024.png by packaging/macos/build-app.sh with Apple's iconutil.
//   - Windows and Linux: plate 44/48 (Fluent icons are drawn on a 48 px grid
//     and fill it); assets/logo.ico, app-icon-256/512.png.
//   - Web touch icon: full-bleed square, no transparency (iOS masks it).
//   - .ico frames up to icoPlainMax px carry the solid mark without a plate,
//     in a mid-tone that keeps ≥ 3:1 on light and dark taskbars (Fluent asks
//     at least half of the icon to pass 3:1 on both themes); on a 16–24 px
//     plate the head is 10–15 px and the face no longer reads. From 32 px
//     (Start pins at 100 %, taskbar from 125 %) the plate reads and is used.

var (
	appInk  = color.NRGBA{0x1b, 0x18, 0x13, 0xff} // DESIGN.md text-light, "Ink"
	appBone = color.NRGBA{0xf1, 0xed, 0xe4, 0xff} // DESIGN.md bg-light, "Bone Paper"
	// appMid is the plateless small-size ink: relative luminance ≈ 0.20, so
	// ≈ 4:1 on white and ≈ 3.9:1 on #202020 (Windows dark taskbar).
	appMid = color.NRGBA{0x85, 0x7c, 0x6b, 0xff}
)

// iconGeom places the plate and the mark on a square canvas.
type iconGeom struct {
	plate  float64 // plate side / canvas side
	mark   float64 // mark's larger side / plate side
	square bool    // plain full-bleed square (the OS applies its own mask)
}

var (
	geomMac   = iconGeom{plate: 824.0 / 1024, mark: 0.66}
	geomWin   = iconGeom{plate: 44.0 / 48, mark: 0.68}
	geomTouch = iconGeom{plate: 1, mark: 0.62, square: true}
)

// icoSizes are the frames of assets/logo.ico (Explorer, taskbar, Start,
// Alt+Tab at 100–200 % scaling); up to icoPlainMax px they are plateless.
var icoSizes = []int{16, 20, 24, 32, 40, 48, 64, 256}

const icoPlainMax = 24

func renderAppIcons() error {
	solid, err := loadSVGPaths(filepath.Join(dir, solidOut))
	if err != nil {
		return err
	}
	if err := writeAppIconSVG(solid); err != nil {
		return err
	}
	frames := make([][]byte, 0, len(icoSizes))
	for _, px := range icoSizes {
		var img *image.NRGBA
		if px <= icoPlainMax {
			img = tint(hint(rasterMask(solid, px), px), appMid)
		} else {
			img = appIcon(solid, px, geomWin)
		}
		frames = append(frames, encodePNG(img))
	}
	ico := wrapICO(frames, icoSizes)
	if err := os.WriteFile(filepath.Join("assets", "logo.ico"), ico, 0o644); err != nil {
		return err
	}
	fmt.Printf("%-20s %9s %7d bytes (%v px, plateless up to %d)\n", "../logo.ico", "ico", len(ico), icoSizes, icoPlainMax)
	for _, o := range []struct {
		name string
		px   int
		g    iconGeom
	}{
		{"app-icon-1024.png", 1024, geomMac},
		{"app-icon-512.png", 512, geomWin},
		{"app-icon-256.png", 256, geomWin},
		{"app-icon-180.png", 180, geomTouch},
	} {
		if err := writePNG(filepath.Join(dir, o.name), appIcon(solid, o.px, o.g)); err != nil {
			return err
		}
	}
	return nil
}

// appIcon draws the plate in Ink with the solid mark centered in Bone.
func appIcon(solid []seg, px int, g iconGeom) *image.NRGBA {
	side := float64(px) * g.plate
	off := (float64(px) - side) / 2
	var plate []uint8
	if g.square {
		plate = bytes.Repeat([]byte{0xff}, px*px)
	} else {
		plate = alphaOf(fillWH(squircle(off, off, side), px, px, func(p pt) (float32, float32) {
			return float32(p.x), float32(p.y)
		}))
	}
	mark := alphaOf(fitPaths(solid, px, px, float64(px)/2, float64(px)/2, side*g.mark, side*g.mark))
	out := image.NewNRGBA(image.Rect(0, 0, px, px))
	for i := range plate {
		c := mix(appInk, appBone, float64(mark[i])/255)
		out.Pix[i*4], out.Pix[i*4+1], out.Pix[i*4+2], out.Pix[i*4+3] = c.R, c.G, c.B, plate[i]
	}
	return out
}

// writeAppIconSVG writes source/app-icon.svg: the macOS-geometry icon as a
// vector (the plate, then the solid mark's own path under a transform).
func writeAppIconSVG(solid []seg) error {
	b, err := os.ReadFile(filepath.Join(dir, solidOut))
	if err != nil {
		return err
	}
	m := svgPathD.FindStringSubmatch(string(b))
	if m == nil {
		return fmt.Errorf("%s: no path", solidOut)
	}
	const px = 1024
	side := px * geomMac.plate
	off := (px - side) / 2
	var d strings.Builder
	for _, s := range squircle(off, off, side) {
		if s.op == 'Z' {
			d.WriteString("Z")
			continue
		}
		fmt.Fprintf(&d, "%c%s %s", s.op, num(s.p[0].x), num(s.p[0].y))
	}
	k, ox, oy := fitTransform(solid, px/2, px/2, side*geomMac.mark, side*geomMac.mark)
	svg := fmt.Sprintf(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 %d %d">`+
		`<!-- Derived from logo-kong-solid.svg by assets/brand/generate.go (ADR-0052); do not edit. -->`+
		`<path fill="#%02x%02x%02x" d="%s"/>`+
		`<path fill="#%02x%02x%02x" fill-rule="evenodd" transform="translate(%s %s) scale(%s)" d="%s"/></svg>`+"\n",
		px, px, appInk.R, appInk.G, appInk.B, d.String(),
		appBone.R, appBone.G, appBone.B, num(ox), num(oy), strconv.FormatFloat(k, 'f', 5, 64), m[1])
	if err := os.WriteFile(filepath.Join(dir, "source", "app-icon.svg"), []byte(svg), 0o644); err != nil {
		return err
	}
	fmt.Printf("%-20s %9s %7d bytes\n", "source/app-icon.svg", "vector", len(svg))
	return nil
}

// squircle returns the superellipse |x|^5 + |y|^5 = 1 scaled to side: the
// usual approximation of Apple's continuous-corner icon plate (Apple does
// not publish the exact curve).
func squircle(x0, y0, side float64) []seg {
	const n, steps = 5.0, 720
	a := side / 2
	cx, cy := x0+a, y0+a
	pow := func(v float64) float64 { return math.Copysign(math.Pow(math.Abs(v), 2/n), v) }
	out := make([]seg, 0, steps+1)
	for i := 0; i < steps; i++ {
		t := 2 * math.Pi * float64(i) / steps
		op := byte('L')
		if i == 0 {
			op = 'M'
		}
		out = append(out, seg{op: op, p: []pt{{cx + a*pow(math.Cos(t)), cy + a*pow(math.Sin(t))}}})
	}
	return append(out, seg{op: 'Z'})
}

func bounds(paths []seg) (minX, minY, maxX, maxY float64) {
	minX, minY, maxX, maxY = math.Inf(1), math.Inf(1), math.Inf(-1), math.Inf(-1)
	for _, s := range paths {
		for _, p := range s.p {
			minX, minY = math.Min(minX, p.x), math.Min(minY, p.y)
			maxX, maxY = math.Max(maxX, p.x), math.Max(maxY, p.y)
		}
	}
	return
}

// fitTransform scales paths to fit a boxW×boxH box centered on (cx, cy):
// canvas point = p·k + (ox, oy).
func fitTransform(paths []seg, cx, cy, boxW, boxH float64) (k, ox, oy float64) {
	minX, minY, maxX, maxY := bounds(paths)
	k = math.Min(boxW/(maxX-minX), boxH/(maxY-minY))
	return k, cx - (minX+maxX)/2*k, cy - (minY+maxY)/2*k
}

// fitPaths rasterizes paths into a w×h canvas, placed as in fitTransform.
func fitPaths(paths []seg, w, h int, cx, cy, boxW, boxH float64) *image.NRGBA {
	k, ox, oy := fitTransform(paths, cx, cy, boxW, boxH)
	return fillWH(paths, w, h, func(p pt) (float32, float32) { return float32(ox + p.x*k), float32(oy + p.y*k) })
}

func alphaOf(img *image.NRGBA) []uint8 {
	a := make([]uint8, len(img.Pix)/4)
	for i := range a {
		a[i] = img.Pix[i*4+3]
	}
	return a
}

// tint colors a black + alpha mask.
func tint(img *image.NRGBA, c color.NRGBA) *image.NRGBA {
	for i := 0; i < len(img.Pix); i += 4 {
		img.Pix[i], img.Pix[i+1], img.Pix[i+2] = c.R, c.G, c.B
	}
	return img
}

// mix blends a toward b by t (coverage; only edge pixels are blended).
func mix(a, b color.NRGBA, t float64) color.NRGBA {
	l := func(x, y uint8) uint8 { return uint8(math.Round(float64(x) + (float64(y)-float64(x))*t)) }
	return color.NRGBA{l(a.R, b.R), l(a.G, b.G), l(a.B, b.B), 0xff}
}

func encodePNG(img image.Image) []byte {
	var buf bytes.Buffer
	if err := (&png.Encoder{CompressionLevel: png.BestCompression}).Encode(&buf, img); err != nil {
		log.Fatal(err)
	}
	return buf.Bytes()
}

func writePNG(path string, img image.Image) error {
	b := encodePNG(img)
	if err := os.WriteFile(path, b, 0o644); err != nil {
		return err
	}
	r := img.Bounds()
	fmt.Printf("%-20s %4dx%-4d %7d bytes\n", strings.TrimPrefix(filepath.ToSlash(path), dir+"/"), r.Dx(), r.Dy(), len(b))
	return nil
}

// wrapICO wraps square PNG frames in an ICONDIR (PNG frames are valid since
// Windows Vista; https://en.wikipedia.org/wiki/ICO_(file_format)). Same
// layout as assets.wrapICO, which builds the tray icon at run time.
func wrapICO(frames [][]byte, sizes []int) []byte {
	head := 6 + 16*len(frames)
	buf := []byte{0, 0, 1, 0}
	buf = binary.LittleEndian.AppendUint16(buf, uint16(len(frames)))
	off := head
	for i, f := range frames {
		dim := byte(sizes[i])
		if sizes[i] >= 256 {
			dim = 0 // 0 means 256
		}
		buf = append(buf, dim, dim, 0, 0, 1, 0, 32, 0)
		buf = binary.LittleEndian.AppendUint32(buf, uint32(len(f)))
		buf = binary.LittleEndian.AppendUint32(buf, uint32(off))
		off += len(f)
	}
	for _, f := range frames {
		buf = append(buf, f...)
	}
	return buf
}

// ── Installer images (Inno Setup 6.6+) ─────────────────────────────────────
//
// WizardImageFile (left panel of the welcome and finished pages, aspect
// 164:314) and WizardSmallImageFile (top-right square): one file per image
// area size Inno lists for 100/125/150/200/250 % scaling, so Setup picks a
// rendered one instead of stretching
// (https://jrsoftware.org/ishelp/topic_setup_wizardimagefile.htm,
// https://jrsoftware.org/ishelp/topic_setup_wizardsmallimagefile.htm).
// The wizard image carries no words: the installer speaks Spanish and
// English and one image cannot switch language, so the tagline is the
// welcome page's text (packaging/windows/kongtrol.iss, [Messages]).

var (
	wizardSizes      = [][2]int{{202, 386}, {269, 515}, {336, 643}, {430, 824}, {534, 1022}}
	wizardSmallSizes = []int{58, 77, 97, 124, 159}
)

func renderInstallerImages() error {
	out := filepath.Join(dir, "installer")
	if err := os.MkdirAll(out, 0o755); err != nil {
		return err
	}
	lockup, err := loadSVGPaths(filepath.Join(dir, "source", "logo.svg"))
	if err != nil {
		return err
	}
	mark, word := splitLockup(lockup)
	for _, s := range wizardSizes {
		if err := writePNG(filepath.Join(out, fmt.Sprintf("wizard-%dx%d.png", s[0], s[1])), wizardImage(mark, word, s[0], s[1])); err != nil {
			return err
		}
	}
	solid, err := loadSVGPaths(filepath.Join(dir, solidOut))
	if err != nil {
		return err
	}
	for _, px := range wizardSmallSizes {
		if err := writePNG(filepath.Join(out, fmt.Sprintf("wizard-small-%d.png", px)), appIcon(solid, px, geomWin)); err != nil {
			return err
		}
	}
	return nil
}

// wizardImage: an Ink panel with the line-art mark large and the wordmark
// under it, both in Bone, the group a little above the middle.
func wizardImage(mark, word []seg, w, h int) *image.NRGBA {
	fw := float64(w)
	mx0, my0, mx1, my1 := bounds(mark)
	wx0, wy0, wx1, wy1 := bounds(word)
	markW := 0.78 * fw
	markH := markW * (my1 - my0) / (mx1 - mx0)
	wordW := 0.60 * fw
	wordH := wordW * (wy1 - wy0) / (wx1 - wx0)
	gap := 0.10 * fw
	top := (float64(h)-(markH+gap+wordH))/2 - 0.06*float64(h)
	a := alphaOf(fitPaths(mark, w, h, fw/2, top+markH/2, markW, markH))
	b := alphaOf(fitPaths(word, w, h, fw/2, top+markH+gap+wordH/2, wordW, wordH))
	img := image.NewNRGBA(image.Rect(0, 0, w, h))
	for i := range a {
		c := mix(appInk, appBone, float64(max(a[i], b[i]))/255)
		img.Pix[i*4], img.Pix[i*4+1], img.Pix[i*4+2], img.Pix[i*4+3] = c.R, c.G, c.B, 0xff
	}
	return img
}

// splitLockup separates the lockup's subpaths into the mark (above) and the
// wordmark (below) at the widest empty horizontal band between them.
func splitLockup(paths []seg) (mark, word []seg) {
	type sub struct {
		s          []seg
		minY, maxY float64
	}
	var subs []sub
	for _, s := range paths {
		if s.op == 'M' {
			subs = append(subs, sub{minY: math.Inf(1), maxY: math.Inf(-1)})
		}
		c := &subs[len(subs)-1]
		c.s = append(c.s, s)
		for _, p := range s.p {
			c.minY, c.maxY = math.Min(c.minY, p.y), math.Max(c.maxY, p.y)
		}
	}
	sort.SliceStable(subs, func(i, j int) bool { return subs[i].minY < subs[j].minY })
	cut, best, reach := 0.0, -1.0, math.Inf(-1)
	for i, s := range subs {
		if i > 0 && s.minY-reach > best {
			best, cut = s.minY-reach, (s.minY+reach)/2
		}
		reach = math.Max(reach, s.maxY)
	}
	for _, s := range subs {
		if s.maxY < cut {
			mark = append(mark, s.s...)
		} else {
			word = append(word, s.s...)
		}
	}
	return mark, word
}
