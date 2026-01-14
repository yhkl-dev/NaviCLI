package coverart

import (
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"net/http"
	"time"

	"github.com/qeesung/image2ascii/convert"
)

// Converter handles album cover art conversion to ASCII
type Converter struct {
	httpClient *http.Client
	converter  *convert.ImageConverter
}

// NewConverter creates a new cover art converter
func NewConverter() *Converter {
	return &Converter{
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
		converter: convert.NewImageConverter(),
	}
}

// ConvertFromURL downloads and converts an image URL to ASCII art
func (c *Converter) ConvertFromURL(url string) (string, error) {
	if url == "" {
		return c.getPlaceholder(), nil
	}

	resp, err := c.httpClient.Get(url)
	if err != nil {
		return c.getPlaceholder(), fmt.Errorf("failed to download: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return c.getPlaceholder(), fmt.Errorf("status %d", resp.StatusCode)
	}

	img, _, err := image.Decode(resp.Body)
	if err != nil {
		return c.getPlaceholder(), fmt.Errorf("failed to decode: %w", err)
	}

	convertOptions := convert.DefaultOptions
	convertOptions.FixedWidth = 25
	convertOptions.FixedHeight = 12
	convertOptions.Colored = false // Disable ANSI colors for tview compatibility

	ascii := c.converter.Image2ASCIIString(img, &convertOptions)
	return ascii, nil
}

// getPlaceholder returns a placeholder when cover art is not available
func (c *Converter) getPlaceholder() string {
	return `[darkgray]┌────────────────────────────────────────────────┐
[darkgray]│                                                │
[darkgray]│                                                │
[darkgray]│                                                │
[darkgray]│                   ♫  ♪  ♫                     │
[darkgray]│              No Cover Art Available           │
[darkgray]│                   ♫  ♪  ♫                     │
[darkgray]│                                                │
[darkgray]│                                                │
[darkgray]│                                                │
[darkgray]└────────────────────────────────────────────────┘`
}
