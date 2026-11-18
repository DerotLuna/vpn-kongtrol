package assets

import (
	"bytes"
	"encoding/binary"
	"image/png"
	"testing"
)

func TestStateDotPNG(t *testing.T) {
	img, err := png.Decode(bytes.NewReader(StateDotPNG(ColorLive, 16, false)))
	if err != nil {
		t.Fatal(err)
	}
	r, g, b, a := img.At(8, 8).RGBA()
	if a>>8 != 0xff || r>>8 != 0x3e || g>>8 != 0xcf || b>>8 != 0x72 {
		t.Fatalf("center = %x %x %x %x, want live green", r>>8, g>>8, b>>8, a>>8)
	}
	if _, _, _, a := img.At(0, 0).RGBA(); a != 0 {
		t.Fatalf("corner alpha = %d, want transparent (no background disc)", a)
	}
	ring, err := png.Decode(bytes.NewReader(StateDotPNG(ColorDim, 16, true)))
	if err != nil {
		t.Fatal(err)
	}
	if _, _, _, a := ring.At(8, 8).RGBA(); a != 0 {
		t.Fatalf("hollow center alpha = %d, want transparent", a)
	}
}

func TestTrayMarkTransparentAndTinted(t *testing.T) {
	for _, s := range MaskSizes {
		img, err := TrayMark(s, InkOnDark, BadgeNone)
		if err != nil {
			t.Fatal(err)
		}
		// Corners are background: transparent, never a filled disc/square.
		for _, p := range [][2]int{{0, 0}, {s - 1, 0}, {0, s - 1}} {
			if a := img.RGBAAt(p[0], p[1]).A; a != 0 {
				t.Fatalf("%dpx corner %v alpha = %d, want 0", s, p, a)
			}
		}
		// Premultiplied: every visible pixel is the ink scaled by its alpha.
		inked := 0
		for i := 3; i < len(img.Pix); i += 4 {
			a := int(img.Pix[i])
			if a < 0x40 {
				continue
			}
			inked++
			if want := int(InkOnDark.R) * a / 0xff; abs(int(img.Pix[i-3])-want) > 2 {
				t.Fatalf("%dpx: pixel R=%d A=%d, want ink R≈%d", s, img.Pix[i-3], a, want)
			}
		}
		if inked == 0 {
			t.Fatalf("%dpx: no visible pixel", s)
		}
	}
}

func abs(n int) int {
	if n < 0 {
		return -n
	}
	return n
}

func TestTrayBadges(t *testing.T) {
	cases := map[TrayBadge][3]uint8{
		BadgeLive:    {ColorLive.R, ColorLive.G, ColorLive.B},
		BadgeCaution: {ColorCaution.R, ColorCaution.G, ColorCaution.B},
	}
	for badge, want := range cases {
		img, err := TrayMark(32, InkOnLight, badge)
		if err != nil {
			t.Fatal(err)
		}
		c := img.RGBAAt(25, 25) // badge center: 32 - 0.2*32 - 0.5 ≈ 25
		if [3]uint8{c.R, c.G, c.B} != want || c.A != 0xff {
			t.Fatalf("badge %d center = %v, want %v", badge, c, want)
		}
	}
	stopped, _ := TrayMark(32, InkOnLight, BadgeStopped)
	if a := stopped.RGBAAt(25, 25).A; a != 0 {
		t.Fatalf("stopped badge center alpha = %d, want a hollow ring", a)
	}
}

func TestTrayICO(t *testing.T) {
	ico, err := TrayICO(InkOnDark, BadgeLive)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.HasPrefix(ico, []byte{0, 0, 1, 0}) || int(binary.LittleEndian.Uint16(ico[4:])) != len(MaskSizes) {
		t.Fatalf("bad ICONDIR header % x", ico[:6])
	}
	for i, s := range MaskSizes {
		e := ico[6+16*i:]
		if int(e[0]) != s {
			t.Fatalf("entry %d width = %d, want %d", i, e[0], s)
		}
		off := binary.LittleEndian.Uint32(e[12:])
		if !bytes.HasPrefix(ico[off:], []byte("\x89PNG")) {
			t.Fatalf("entry %d is not a PNG frame", i)
		}
	}
}
