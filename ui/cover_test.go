package ui

import (
	"image"
	"image/color"
	"strings"
	"testing"
)

// solidImage builds a WxH image filled with c, for deterministic renderer tests.
func solidImage(w, h int, c color.Color) image.Image {
	img := image.NewNRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.Set(x, y, c)
		}
	}
	return img
}

func TestRenderCoverImage(t *testing.T) {
	// nil image renders nothing.
	if got := renderCoverImage(nil, 8); got != "" {
		t.Errorf("renderCoverImage(nil) = %q, want empty", got)
	}

	// A solid red 16x16 image at cols=8 yields 8 cells x 4 rows. Every
	// sub-pixel equals the average, so all 8 dots light and each cell is the
	// full braille block in red on red.
	red := solidImage(16, 16, color.RGBA{R: 255, A: 255})
	got := renderCoverImage(red, 8)
	if cells := strings.Count(got, "[#ff0000:#ff0000]⣿"); cells != 8*4 {
		t.Errorf("renderCoverImage(red,8): want %d full red cells, got %d", 8*4, cells)
	}
	if lines := strings.Count(got, "\n"); lines != 4 {
		t.Errorf("renderCoverImage(red,8): want 4 rows, got %d", lines)
	}
	// Each row resets color so the cover never bleeds into the metadata below.
	if !strings.Contains(got, "[-:-:-]") {
		t.Error("renderCoverImage: row should reset colors with [-:-:-]")
	}

	// Cols clamp to [8, 24].
	if got := renderCoverImage(red, 4); strings.Count(got, "⣿") != 8*4 {
		t.Errorf("renderCoverImage(,4): want 8x4 cells (floor), got %d", strings.Count(got, "⣿"))
	}
	if got := renderCoverImage(red, 100); strings.Count(got, "⣿") != 24*12 {
		t.Errorf("renderCoverImage(,100): want 24x12 cells (cap), got %d", strings.Count(got, "⣿"))
	}

	// A non-square source is center-cropped to square, no panic.
	cropped := renderCoverImage(solidImage(32, 16, color.RGBA{G: 255, A: 255}), 8)
	if strings.Count(cropped, "⣿") != 8*4 {
		t.Errorf("renderCoverImage(non-square): want 8x4 cells, got %d", strings.Count(cropped, "⣿"))
	}

	// A gradient image dithers: sub-pixels vary within a cell, so at least one
	// cell is neither the empty nor the full braille block (some dots on/off).
	grad := image.NewNRGBA(image.Rect(0, 0, 16, 16))
	for y := 0; y < 16; y++ {
		for x := 0; x < 16; x++ {
			v := uint8(x * 16) // 0..240 across the width
			grad.Set(x, y, color.RGBA{R: v, G: v, B: v, A: 255})
		}
	}
	dithered := renderCoverImage(grad, 8)
	full := strings.Count(dithered, "⣿")
	empty := strings.Count(dithered, "⠀")
	if full+empty == 8*4 {
		t.Error("renderCoverImage(grad): expected at least one partial dither cell, all cells were full or empty")
	}
}
