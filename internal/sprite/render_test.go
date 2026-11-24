package sprite

import (
	"image"
	"image/color"
	"strings"
	"testing"
)

// testSprite is a 10×10 transparent image with a 4×6 red block (black top row) at (3,2).
func testSprite() *image.NRGBA {
	img := image.NewNRGBA(image.Rect(0, 0, 10, 10))
	for y := 2; y < 8; y++ {
		for x := 3; x < 7; x++ {
			c := color.NRGBA{255, 0, 0, 255}
			if y == 2 {
				c = color.NRGBA{0, 0, 0, 255}
			}
			img.Set(x, y, c)
		}
	}
	return img
}

func TestOpaqueBounds(t *testing.T) {
	if got, want := opaqueBounds(testSprite()), image.Rect(3, 2, 7, 8); got != want {
		t.Errorf("opaqueBounds = %v, want %v", got, want)
	}
	if got := opaqueBounds(image.NewNRGBA(image.Rect(0, 0, 4, 4))); !got.Empty() {
		t.Errorf("transparent image should have empty bounds, got %v", got)
	}
}

func TestDownscaleKeepsAspect(t *testing.T) {
	img := testSprite()
	px := downscale(img, opaqueBounds(img), 2, 3)
	if len(px) != 3 || len(px[0]) != 2 {
		t.Fatalf("got %dx%d pixels, want 2x3", len(px[0]), len(px))
	}
	if px[2][0] != (color.NRGBA{255, 0, 0, 255}) {
		t.Errorf("bottom pixel = %v, want red", px[2][0])
	}
}

func TestRenderSpriteUsesHalfBlocks(t *testing.T) {
	out := Render(testSprite(), 10, 10)
	if !strings.Contains(out, "▀") {
		t.Errorf("expected half-block characters, got %q", out)
	}
	if lines := strings.Count(out, "\n") + 1; lines != 3 { // 6 pixel rows -> 3 text rows
		t.Errorf("got %d rows, want 3", lines)
	}
}
