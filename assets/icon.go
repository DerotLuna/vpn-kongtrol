package assets

import (
	"bytes"
	"embed"
	"encoding/binary"
	"fmt"
	"image"
	"image/color"
	"image/png"
)

// trayMaskFS holds the tray / menu bar masks rasterized from the vector
// master by brand/generate.go: black line art, transparent background, one
// file per size (MaskSizes).
//
//go:embed brand/tray-*.png
var trayMaskFS embed.FS

// MaskSizes are the sizes generate.go renders (traySizes there).
var MaskSizes = []int{16, 20, 24, 32, 48, 64}

// Ink colors for the tray mark (DESIGN.md text tokens): light ink on dark
// taskbars / panels, dark ink on light ones. InkTemplate is plain black for
// macOS template images, where only the alpha channel counts.
var (
	InkOnDark   = color.RGBA{0xee, 0xf1, 0xf5, 0xff} // text
	InkOnLight  = color.RGBA{0x1b, 0x18, 0x13, 0xff} // text-light
	InkTemplate = color.RGBA{0, 0, 0, 0xff}
)

// TrayMark returns the mark at size×size (size must be one of MaskSizes)
// tinted with ink, with the state badge drawn in the bottom-right corner.
func TrayMark(size int, ink color.RGBA, badge TrayBadge) (*image.RGBA, error) {
	b, err := trayMaskFS.ReadFile(fmt.Sprintf("brand/tray-%d.png", size))
	if err != nil {
		return nil, fmt.Errorf("assets: no tray mask for %dpx: %w", size, err)
	}
	mask, err := png.Decode(bytes.NewReader(b))
	if err != nil {
		return nil, err
	}
	dst := image.NewRGBA(image.Rect(0, 0, size, size))
	for y := 0; y < size; y++ {
		for x := 0; x < size; x++ {
			_, _, _, a := mask.At(x, y).RGBA()
			if a == 0 {
				continue
			}
			blend(dst, x, y, ink, float64(a)/0xffff)
		}
	}
	drawBadge(dst, badge)
	return dst, nil
}

// TrayPNG is TrayMark encoded as PNG (macOS, Linux and menu items).
func TrayPNG(size int, ink color.RGBA, badge TrayBadge) ([]byte, error) {
	img, err := TrayMark(size, ink, badge)
	if err != nil {
		return nil, err
	}
	return encodePNG(img), nil
}

// TrayICO renders every size in MaskSizes into one .ico, so Windows picks
// a rendered frame for the current DPI instead of scaling one.
//
// The .ico container matters on Windows: systray's Windows backend loads
// icons with LoadImage(IMAGE_ICON, LR_LOADFROMFILE|LR_DEFAULTSIZE), which
// needs a real ICO; a bare PNG fails there (LoadImage returns 0 and the
// tray icon stays blank). PNG-compressed ICO frames are valid since Vista.
func TrayICO(ink color.RGBA, badge TrayBadge) ([]byte, error) {
	frames := make([][]byte, 0, len(MaskSizes))
	for _, s := range MaskSizes {
		p, err := TrayPNG(s, ink, badge)
		if err != nil {
			return nil, err
		}
		frames = append(frames, p)
	}
	return wrapICO(frames, MaskSizes), nil
}

// wrapICO wraps square PNG images in an ICONDIR with one ICONDIRENTRY per
// frame. See https://en.wikipedia.org/wiki/ICO_(file_format).
func wrapICO(frames [][]byte, sizes []int) []byte {
	headerLen := 6 + 16*len(frames)
	total := headerLen
	for _, f := range frames {
		total += len(f)
	}
	buf := make([]byte, 0, total)
	// ICONDIR: reserved(2)=0, type(2)=1 (icon), count(2)
	buf = append(buf, 0, 0, 1, 0)
	buf = binary.LittleEndian.AppendUint16(buf, uint16(len(frames)))
	offset := headerLen
	for i, f := range frames {
		dim := byte(sizes[i])
		if sizes[i] >= 256 {
			dim = 0 // 0 means 256 in the 1-byte width/height fields
		}
		// ICONDIRENTRY: width, height, colorCount, reserved, planes(2), bitCount(2)
		buf = append(buf, dim, dim, 0, 0, 1, 0, 32, 0)
		buf = binary.LittleEndian.AppendUint32(buf, uint32(len(f)))
		buf = binary.LittleEndian.AppendUint32(buf, uint32(offset))
		offset += len(f)
	}
	for _, f := range frames {
		buf = append(buf, f...)
	}
	return buf
}
