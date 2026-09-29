package textmosaic

import (
	"image"

	"github.com/disintegration/imaging"
)

// prepareSource applies the image operations that determine what values are
// sampled while rendering the mosaic.
func prepareSource(source image.Image, cfg Config) image.Image {
	processed := source

	if cfg.TargetWidth > 0 {
		processed = imaging.Resize(processed, cfg.TargetWidth, 0, imaging.Lanczos)
	}
	if cfg.ContrastPercent != 0 {
		processed = imaging.AdjustContrast(processed, cfg.ContrastPercent)
	}

	// Keep the source colors: render clips this image through the text glyphs.
	return processed
}
