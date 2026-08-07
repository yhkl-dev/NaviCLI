package ui

import (
	"bytes"
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/yhkl-dev/NaviCLI/domain"
)

// fetchCover downloads and decodes the cover art at url into an image. The
// response body is capped at 4 MB so a malformed server cannot exhaust memory.
func fetchCover(url string) (image.Image, error) {
	client := &http.Client{Timeout: 8 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("cover: status %d", resp.StatusCode)
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if err != nil {
		return nil, err
	}
	img, _, err := image.Decode(bytes.NewReader(data))
	return img, err
}

// brailleBits maps dot index 0..7 to the Unicode braille code-point bits.
// Dot layout in a 2-wide x 4-tall cell:
//
//	0 3
//	1 4
//	2 5
//	6 7
var brailleBits = [8]uint{0x01, 0x02, 0x04, 0x40, 0x08, 0x10, 0x20, 0x80}

func dotIndex(c, r int) int {
	switch {
	case c == 0 && r == 0:
		return 0
	case c == 0 && r == 1:
		return 1
	case c == 0 && r == 2:
		return 2
	case c == 1 && r == 0:
		return 3
	case c == 1 && r == 1:
		return 4
	case c == 1 && r == 2:
		return 5
	case c == 0 && r == 3:
		return 6
	default:
		return 7
	}
}

// renderCoverImage renders img as a truecolor braille string cols cells wide.
// Each cell packs 8 sub-pixels in a 2x4 grid, 4x the resolution of a
// half-block, so a small thumbnail still reads as the album cover instead of
// a blur of blocks. Within a cell, sub-pixels at or above the cell's average
// luminance light up in their average color (foreground); the rest show the
// darker average (background). Uniform areas collapse to a solid cell, varied
// areas dither. The source is center-cropped to a square, then box-averaged
// down. Returns "" if img is nil.
func renderCoverImage(img image.Image, cols int) string {
	if img == nil {
		return ""
	}
	if cols < 8 {
		cols = 8
	}
	if cols > 24 {
		cols = 24
	}
	b := img.Bounds()
	side := b.Dx()
	if b.Dy() < side {
		side = b.Dy()
	}
	ox := b.Min.X + (b.Dx()-side)/2
	oy := b.Min.Y + (b.Dy()-side)/2

	rows := cols / 2
	pxW := cols * 2 // 2 sub-pixels wide per cell
	pxH := rows * 4 // 4 sub-pixels tall per cell

	var sb strings.Builder
	for cy := 0; cy < rows; cy++ {
		sb.WriteString("  ")
		for cx := 0; cx < cols; cx++ {
			var pr, pg, pb [8]uint32
			var lum [8]float64
			var sumR, sumG, sumB, sumL float64
			for r := 0; r < 4; r++ {
				for c := 0; c < 2; c++ {
					i := dotIndex(c, r)
					sx := ox + (cx*2+c)*side/pxW
					sy := oy + (cy*4+r)*side/pxH
					if sx >= ox+side {
						sx = ox + side - 1
					}
					if sy >= oy+side {
						sy = oy + side - 1
					}
					cr, cg, cb, _ := img.At(sx, sy).RGBA()
					pr[i], pg[i], pb[i] = cr, cg, cb
					l := 0.299*float64(cr) + 0.587*float64(cg) + 0.114*float64(cb)
					lum[i] = l
					sumR += float64(cr)
					sumG += float64(cg)
					sumB += float64(cb)
					sumL += l
				}
			}
			avgL := sumL / 8
			var onR, onG, onB, offR, offG, offB float64
			onN, offN := 0, 0
			var bits uint
			for i := 0; i < 8; i++ {
				if lum[i] >= avgL {
					bits |= brailleBits[i]
					onR += float64(pr[i])
					onG += float64(pg[i])
					onB += float64(pb[i])
					onN++
				} else {
					offR += float64(pr[i])
					offG += float64(pg[i])
					offB += float64(pb[i])
					offN++
				}
			}
			cellR, cellG, cellB := sumR/8, sumG/8, sumB/8
			fgR, fgG, fgB := cellR, cellG, cellB
			if onN > 0 {
				fgR, fgG, fgB = onR/float64(onN), onG/float64(onN), onB/float64(onN)
			}
			bgR, bgG, bgB := cellR, cellG, cellB
			if offN > 0 {
				bgR, bgG, bgB = offR/float64(offN), offG/float64(offN), offB/float64(offN)
			}
			ch := rune(0x2800 + bits)
			fmt.Fprintf(&sb, "[#%02x%02x%02x:#%02x%02x%02x]%c",
				uint8(fgR/256), uint8(fgG/256), uint8(fgB/256),
				uint8(bgR/256), uint8(bgG/256), uint8(bgB/256), ch)
		}
		sb.WriteString("[-:-:-]\n")
	}
	return sb.String()
}

// coverCols returns the cell width for the album cover. It is clamped to a
// readable range by both the panel WIDTH and the terminal HEIGHT: on a short
// terminal the cover shrinks so the title, artist, and specs below it stay
// visible instead of being pushed off-screen. Rounded down to even so the
// pixel height stays even and the thumbnail stays square.
func (a *App) coverCols() int {
	cw := a.getTerminalWidth()/4 - 4
	if cw < 8 {
		cw = 8
	}
	if cw > 24 {
		cw = 24
	}
	// Height budget: title bar + bottom bar + the per-field metadata block
	// (title + up to 8 label:value lines + breathing room). Cap cover rows so
	// the metadata stays visible instead of being pushed off-screen.
	maxRows := 12
	if h := a.getTerminalHeight(); h > 0 {
		avail := h - 15
		if avail < 4 {
			avail = 4
		}
		if avail < maxRows {
			maxRows = avail
		}
	}
	cols := cw
	if cols/2 > maxRows {
		cols = maxRows * 2
	}
	if cols%2 != 0 {
		cols--
	}
	if cols < 8 {
		cols = 8
	}
	return cols
}

// coverText returns the rendered braille cover string for the currently
// cached cover image, or "" if none is loaded. The rendered string is cached
// per (coverArtID, cols) so the per-second tick re-render is a cheap lookup;
// re-render happens only when the song or the terminal size changes.
func (a *App) coverText() string {
	cols := a.coverCols()
	a.coverMu.RLock()
	img := a.coverImg
	id := a.coverArtID
	if a.coverRenderedID == id && a.coverRenderedCols == cols && a.coverRendered != "" && img != nil {
		out := a.coverRendered
		a.coverMu.RUnlock()
		return out
	}
	a.coverMu.RUnlock()
	if img == nil {
		return ""
	}
	fresh := renderCoverImage(img, cols)
	a.coverMu.Lock()
	a.coverRendered = fresh
	a.coverRenderedID = id
	a.coverRenderedCols = cols
	a.coverMu.Unlock()
	return fresh
}

// nowPlaying builds the left "Now Playing" panel text: cover (if loaded),
// then track metadata. Live state lives in the bottom strip.
func (a *App) nowPlaying(track domain.Song, extra string) string {
	return FormatSongInfo(track, a.leftPanelTextWidth(), a.coverText(), extra)
}

// nowPlayingText rebuilds the now-playing panel for the current song. Called
// off the UI thread (after a cover fetch) so terminal-width probing does not
// block drawing.
func (a *App) nowPlayingText() string {
	currentSong, _, _, _ := a.state.GetState()
	if currentSong == nil {
		return ""
	}
	return a.nowPlaying(*currentSong, CreateTrackExtras(*currentSong))
}

// loadCoverArt fetches the cover for song asynchronously. The previous cover
// is cleared immediately so a new song never shows the old art; the new cover
// pops in when the fetch completes. Stale fetches (the song changed again
// mid-flight) are discarded via coverReq so a slow earlier request cannot
// overwrite a newer one.
func (a *App) loadCoverArt(song *domain.Song) {
	if song == nil || song.CoverArt == "" {
		a.coverMu.Lock()
		a.coverImg = nil
		a.coverArtID = ""
		a.coverRendered = ""
		a.coverRenderedID = ""
		a.coverMu.Unlock()
		return
	}
	id := song.CoverArt
	a.coverMu.RLock()
	cached := a.coverArtID == id && a.coverImg != nil
	a.coverMu.RUnlock()
	if cached {
		return
	}

	// Clear the stale cover immediately; the new one loads asynchronously.
	a.coverMu.Lock()
	a.coverImg = nil
	a.coverArtID = ""
	a.coverRendered = ""
	a.coverRenderedID = ""
	a.coverReq++
	req := a.coverReq
	a.coverMu.Unlock()

	go func() {
		img, err := fetchCover(a.library.GetCoverArtURL(id))
		if err != nil || img == nil {
			return
		}
		a.coverMu.Lock()
		if a.coverReq != req {
			a.coverMu.Unlock()
			return // superseded by a newer request
		}
		a.coverImg = img
		a.coverArtID = id
		a.coverRendered = ""
		a.coverRenderedID = ""
		a.coverMu.Unlock()

		text := a.nowPlayingText()
		a.tviewApp.QueueUpdateDraw(func() {
			if a.statusBar != nil {
				a.statusBar.SetText(text)
			}
		})
	}()
}
