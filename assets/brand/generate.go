//go:build ignore

// Brand asset generator — version 1.1.0 · 2026-11-18.
//
// Regenerates the raster fallbacks in assets/brand/ from the masters in
// assets/brand/source/. The SVGs (logo.svg, logo-kong.svg, favicon.svg) are
// the masters for anything that can render vectors; the rasters exist only
// for consumers that cannot (OS tray icons, Open Graph cards, app manifests,
// README images).
//
// Uses only the standard library plus golang.org/x/image (already in go.mod),
// so it needs no extra install. Run from the repo root:
//
//	go run assets/brand/generate.go
//
// Encoding choices:
//   - The logo is black line art on white, so logo rasters are converted to
//     luma and stored as a 16-level grayscale palette (4-bit PNG). 16 levels
//     are enough for anti-aliased edges and cut the size several-fold.
//   - The Open Graph card is a photo-like gradient, so it is emitted as JPEG,
//     which every OG/Twitter consumer accepts (https://ogp.me/#structured).
//   - WebP is not emitted: golang.org/x/image/webp only decodes.
//   - Tray / menu bar masks (tray-<px>.png) are rasterized straight from the
//     vector master source/logo-kong.svg (golang.org/x/image/vector), cropped
//     to the mark, as black line art on a TRANSPARENT background: the tray
//     tints them for the taskbar theme (Windows, Linux) or hands them to
//     macOS as a template image, and draws state badges on top at run time.
//     One file per tray size, so small sizes are rendered, not downscaled.
package main

import (
	"bytes"
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
		{"source/logo-kong.png", "logo-kong-32.png", 32, 32, lineArtPNG},
		{"source/logo-kong.png", "logo-kong-64.png", 64, 64, lineArtPNG},
		{"source/logo-kong.png", "logo-kong-192.png", 192, 192, lineArtPNG},
		{"source/logo-kong.png", "logo-kong-512.png", 512, 512, lineArtPNG},
		{"source/logo.png", "logo-512.png", 512, 512, lineArtPNG},
		{"source/social-preview.png", "og-1200x630.jpg", 1200, 630, photoJPEG},
	}
	for _, name := range []string{"logo.svg", "logo-kong.svg", "favicon.svg"} {
		n, err := minifySVG(filepath.Join(dir, "source", name), filepath.Join(dir, name))
		if err != nil {
			log.Fatal(err)
		}
		fmt.Printf("%-20s %9s %7d bytes\n", name, "vector", n)
	}
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
	paths, err := loadSVGPaths(filepath.Join(dir, "source", "logo-kong.svg"))
	if err != nil {
		log.Fatal(err)
	}
	for _, px := range traySizes {
		out := fmt.Sprintf("tray-%d.png", px)
		var buf bytes.Buffer
		if err := (&png.Encoder{CompressionLevel: png.BestCompression}).Encode(&buf, rasterMask(paths, px)); err != nil {
			log.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, out), buf.Bytes(), 0o644); err != nil {
			log.Fatal(err)
		}
		fmt.Printf("%-20s %4dx%-4d %7d bytes\n", out, px, px, buf.Len())
	}
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
	f := func(p pt) (float32, float32) { return float32(ox + p.x*k), float32(oy + p.y*k) }

	r := vector.NewRasterizer(px, px)
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
	mask := image.NewAlpha(image.Rect(0, 0, px, px))
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
// and redundant zero fractions. Geometry is unchanged.
func minifySVG(src, dst string) (int, error) {
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
	s = strings.TrimSpace(s) + "\n"
	return len(s), os.WriteFile(dst, []byte(s), 0o644)
}
