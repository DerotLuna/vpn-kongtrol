//go:build ignore

// Brand asset generator — version 1.0.0 · 2026-11-17.
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
package main

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"golang.org/x/image/draw"
)

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
