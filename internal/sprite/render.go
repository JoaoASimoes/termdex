// Package sprite draws Pokémon sprites in the terminal as half-block pixel art.
package sprite

import (
	"fmt"
	"image"
	"image/color"
	"strings"

	"charm.land/lipgloss/v2"
)

// opaqueBounds returns the smallest rectangle containing every visible pixel.
func opaqueBounds(img image.Image) image.Rectangle {
	b := img.Bounds()
	minX, minY, maxX, maxY := b.Max.X, b.Max.Y, b.Min.X-1, b.Min.Y-1
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			if _, _, _, a := img.At(x, y).RGBA(); a > 0x8000 {
				minX, minY = min(minX, x), min(minY, y)
				maxX, maxY = max(maxX, x), max(maxY, y)
			}
		}
	}
	if maxX < minX {
		return image.Rectangle{}
	}
	return image.Rect(minX, minY, maxX+1, maxY+1)
}

// downscale shrinks the cropped sprite so it fits in w×h pixels, keeping its aspect ratio.
// Each target pixel takes the most common visible color of its source block, which keeps
// the hard edges of pixel art (averaging would blur outlines).
func downscale(img image.Image, crop image.Rectangle, w, h int) [][]color.NRGBA {
	cw, ch := crop.Dx(), crop.Dy()
	scale := max(1.0, max(float64(cw)/float64(w), float64(ch)/float64(h)))
	tw, th := int(float64(cw)/scale+0.5), int(float64(ch)/scale+0.5)
	tw, th = max(1, tw), max(1, th)

	out := make([][]color.NRGBA, th)
	for ty := 0; ty < th; ty++ {
		out[ty] = make([]color.NRGBA, tw)
		y0, y1 := crop.Min.Y+int(float64(ty)*scale), crop.Min.Y+int(float64(ty+1)*scale)
		for tx := 0; tx < tw; tx++ {
			x0, x1 := crop.Min.X+int(float64(tx)*scale), crop.Min.X+int(float64(tx+1)*scale)
			counts := map[color.NRGBA]int{}
			total, visible := 0, 0
			for y := y0; y < max(y1, y0+1); y++ {
				for x := x0; x < max(x1, x0+1); x++ {
					total++
					c := color.NRGBAModel.Convert(img.At(x, y)).(color.NRGBA)
					if c.A < 128 {
						continue
					}
					visible++
					c.A = 255
					counts[c]++
				}
			}
			if visible*2 < total { // mostly transparent
				continue
			}
			var best color.NRGBA
			bestN := -1
			for c, n := range counts {
				if n > bestN || (n == bestN && lum(c) < lum(best)) { // ties: darker wins (outlines)
					best, bestN = c, n
				}
			}
			out[ty][tx] = best
		}
	}
	return out
}

func lum(c color.NRGBA) int { return int(c.R)*299 + int(c.G)*587 + int(c.B)*114 }

func hex(c color.NRGBA) string { return fmt.Sprintf("#%02X%02X%02X", c.R, c.G, c.B) }

// Render draws img with half-block characters: every terminal cell shows two
// vertical pixels (foreground = top, background = bottom). maxCols×maxRows is in cells.
func Render(img image.Image, maxCols, maxRows int) string {
	crop := opaqueBounds(img)
	if crop.Empty() {
		return ""
	}
	px := downscale(img, crop, maxCols, maxRows*2)
	if len(px)%2 == 1 {
		px = append(px, make([]color.NRGBA, len(px[0])))
	}
	var b strings.Builder
	for y := 0; y < len(px); y += 2 {
		for x := range px[y] {
			top, bot := px[y][x], px[y+1][x]
			switch {
			case top.A == 0 && bot.A == 0:
				b.WriteByte(' ')
			case bot.A == 0:
				b.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color(hex(top))).Render("▀"))
			case top.A == 0:
				b.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color(hex(bot))).Render("▄"))
			default:
				b.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color(hex(top))).
					Background(lipgloss.Color(hex(bot))).Render("▀"))
			}
		}
		if y+2 < len(px) {
			b.WriteByte('\n')
		}
	}
	return b.String()
}
