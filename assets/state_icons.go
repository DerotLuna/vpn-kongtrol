package assets

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"math"
)

// State colors from DESIGN.md (dark-theme tokens): green means up, steel
// is connecting, caution orange is "no answer" / degraded, red is error,
// dim grey is off. Amber is the brand, never a state, so it is not here.
var (
	ColorLive    = color.RGBA{0x3e, 0xcf, 0x72, 0xff} // live
	ColorSteel   = color.RGBA{0x7f, 0xb4, 0xd4, 0xff} // steel
	ColorCaution = color.RGBA{0xff, 0x8a, 0x3d, 0xff} // caution
	ColorDanger  = color.RGBA{0xff, 0x5a, 0x52, 0xff} // danger
	ColorDim     = color.RGBA{0x8e, 0x98, 0xa8, 0xff} // text-dim
)

// StateDotPNG draws a size×size status dot on a transparent background
// (the dashboards' status-dot, as a raster for native menus). hollow draws
// a ring instead of a disc, so "off" keeps a shape and color is never the
// only signal.
func StateDotPNG(c color.RGBA, size int, hollow bool) []byte {
	img := image.NewRGBA(image.Rect(0, 0, size, size))
	r := float64(size) * 0.3 // a dot, not a full square: it sits in the icon column
	ring := math.Max(1.25, float64(size)/9)
	paintCircle(img, float64(size)/2, float64(size)/2, r, func(d float64) float64 {
		if hollow {
			return coverage(r-d) * coverage(d-(r-ring))
		}
		return coverage(r - d)
	}, c)
	return encodePNG(img)
}

// MenuIconICO wraps one PNG (StateDotPNG) for Windows menu items, which
// load icons through LoadImage(IMAGE_ICON) like the tray icon.
func MenuIconICO(pngBytes []byte, size int) []byte {
	return wrapICO([][]byte{pngBytes}, []int{size})
}

// TrayBadge is the aggregate state drawn on the tray icon as a small
// corner badge; the mark itself is never recolored.
type TrayBadge int

const (
	BadgeNone    TrayBadge = iota // Kongtrol running, nothing connected
	BadgeLive                     // at least one tunnel up, none in trouble
	BadgeCaution                  // a tunnel dropped / not answering / failed
	BadgeStopped                  // Kongtrol (the daemon) is not running: hollow ring
)

// BadgeGeometry is the status dot on a size×size tray icon: center (cx, cy)
// tucked into the bottom-right corner, radius r = 17 % of the icon (at least
// 2.5 px) and a transparent gap ring of size/16 (at least 1 px) cut out of
// the glyph around it. Sized so the dot and its gap stay off the face of
// the solid mark at every MaskSizes size (ADR-0044; TestBadgeOffFace).
func BadgeGeometry(size int) (cx, cy, r, gap float64) {
	s := float64(size)
	r = math.Max(2.5, s*0.17)
	gap = math.Max(1, s/16)
	return s - r - 0.5, s - r - 0.5, r, gap
}

// drawBadge clears a transparent gap around the badge (so it reads on any
// taskbar without a background disc) and draws the dot or ring.
func drawBadge(dst *image.RGBA, badge TrayBadge) {
	if badge == BadgeNone {
		return
	}
	size := float64(dst.Bounds().Dx())
	cx, cy, r, gap := BadgeGeometry(dst.Bounds().Dx())
	// Cut the gap: scale existing pixels down by the gap coverage.
	paintCircle(dst, cx, cy, r+gap, func(d float64) float64 { return coverage(r + gap - d) }, color.RGBA{})
	switch badge {
	case BadgeLive:
		paintCircle(dst, cx, cy, r, func(d float64) float64 { return coverage(r - d) }, ColorLive)
	case BadgeCaution:
		paintCircle(dst, cx, cy, r, func(d float64) float64 { return coverage(r - d) }, ColorCaution)
	case BadgeStopped:
		ring := math.Max(1, size/14)
		paintCircle(dst, cx, cy, r, func(d float64) float64 {
			return coverage(r-d) * coverage(d-(r-ring))
		}, ColorDim)
	}
}

// coverage maps a signed distance (px inside the edge) to 0..1 alpha,
// a one-pixel anti-aliased edge.
func coverage(inside float64) float64 {
	return math.Max(0, math.Min(1, inside+0.5))
}

// paintCircle blends c over img inside radius r+1 of (cx, cy), weighted by
// alpha(distance from the center). A transparent c erases (cuts a gap).
func paintCircle(img *image.RGBA, cx, cy, r float64, alpha func(d float64) float64, c color.RGBA) {
	b := img.Bounds()
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			d := math.Hypot(float64(x)+0.5-cx, float64(y)+0.5-cy)
			if d > r+1 {
				continue
			}
			if a := alpha(d); a > 0 {
				blend(img, x, y, c, a)
			}
		}
	}
}

// blend draws c with opacity a over the pixel (premultiplied source-over;
// a transparent c with a=1 clears the pixel).
func blend(img *image.RGBA, x, y int, c color.RGBA, a float64) {
	i := img.PixOffset(x, y)
	p := img.Pix[i : i+4 : i+4]
	ca := float64(c.A) / 255 * a
	inv := 1 - a
	p[0] = uint8(float64(c.R)*ca + float64(p[0])*inv)
	p[1] = uint8(float64(c.G)*ca + float64(p[1])*inv)
	p[2] = uint8(float64(c.B)*ca + float64(p[2])*inv)
	p[3] = uint8(255*ca + float64(p[3])*inv)
}

func encodePNG(img image.Image) []byte {
	var buf bytes.Buffer
	_ = png.Encode(&buf, img) // in-memory RGBA: cannot fail
	return buf.Bytes()
}
