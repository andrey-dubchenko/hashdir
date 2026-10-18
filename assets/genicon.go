//go:build ignore

// genicon renders assets/icon.ico (PNG-compressed entries, 16-256 px) and the PNG icons in the
// freedesktop hicolor layout under assets/icons. Run from the repository root: go run assets/genicon.go
package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"log"
	"math"
	"os"
	"path/filepath"
)

var sizes = []int{16, 24, 32, 48, 64, 128, 256}

// inRoundedRect reports whether the point (x, y) in the unit square lies inside a rounded square.
func inRoundedRect(x, y, r float64) bool {
	cx := math.Min(math.Max(x, r), 1-r)
	cy := math.Min(math.Max(y, r), 1-r)
	return math.Hypot(x-cx, y-cy) <= r
}

// inHash reports whether the point lies on the "#" glyph (two slanted vertical and two horizontal bars).
func inHash(x, y float64) bool {
	const w = 0.055
	// horizontal bars
	if x > 0.2 && x < 0.8 && (math.Abs(y-0.40) < w || math.Abs(y-0.60) < w) {
		return true
	}
	// vertical bars, slanted slightly to the right towards the top
	shift := (0.5 - y) * 0.12
	return y > 0.2 && y < 0.8 && (math.Abs(x-0.38-shift) < w || math.Abs(x-0.62-shift) < w)
}

func render(size int) *image.NRGBA {
	const ss = 4 // supersampling per axis
	bg := color.NRGBA{0x1f, 0x4e, 0x9c, 0xff}
	img := image.NewNRGBA(image.Rect(0, 0, size, size))
	for py := 0; py < size; py++ {
		for px := 0; px < size; px++ {
			var rr, gg, bb, aa float64
			for sy := 0; sy < ss; sy++ {
				for sx := 0; sx < ss; sx++ {
					x := (float64(px) + (float64(sx)+0.5)/ss) / float64(size)
					y := (float64(py) + (float64(sy)+0.5)/ss) / float64(size)
					if !inRoundedRect(x, y, 0.2) {
						continue
					}
					c := bg
					if inHash(x, y) {
						c = color.NRGBA{0xff, 0xff, 0xff, 0xff}
					}
					rr += float64(c.R)
					gg += float64(c.G)
					bb += float64(c.B)
					aa++
				}
			}
			if aa == 0 {
				continue
			}
			img.SetNRGBA(px, py, color.NRGBA{uint8(rr / aa), uint8(gg / aa), uint8(bb / aa), uint8(255 * aa / (ss * ss))})
		}
	}
	return img
}

func main() {
	var pngs [][]byte
	for _, s := range sizes {
		var buf bytes.Buffer
		if err := png.Encode(&buf, render(s)); err != nil {
			log.Fatal(err)
		}
		pngs = append(pngs, buf.Bytes())
	}

	var out bytes.Buffer
	binary.Write(&out, binary.LittleEndian, [3]uint16{0, 1, uint16(len(sizes))})
	offset := 6 + 16*len(sizes)
	for i, s := range sizes {
		dim := byte(s)
		if s >= 256 {
			dim = 0
		}
		out.Write([]byte{dim, dim, 0, 0})
		binary.Write(&out, binary.LittleEndian, [2]uint16{1, 32})
		binary.Write(&out, binary.LittleEndian, [2]uint32{uint32(len(pngs[i])), uint32(offset)})
		offset += len(pngs[i])
	}
	for _, p := range pngs {
		out.Write(p)
	}

	if err := os.WriteFile("assets/icon.ico", out.Bytes(), 0o644); err != nil {
		log.Fatal(err)
	}
	for i, s := range sizes {
		dir := filepath.Join("assets", "icons", "hicolor", fmt.Sprintf("%dx%d", s, s), "apps")
		if err := os.MkdirAll(dir, 0o755); err != nil {
			log.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "hashdir.png"), pngs[i], 0o644); err != nil {
			log.Fatal(err)
		}
	}
}
