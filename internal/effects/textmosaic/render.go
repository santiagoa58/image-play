package textmosaic

import (
	"errors"
	"fmt"
	"image"
	"math"

	"github.com/fogleman/gg"
)

const (
	verticalSpacingMultiplier = 1.4
	gridOffsetFraction        = 0.5
	maxColorChannelValue      = 65535.0
)

type fontMetrics struct {
	charWidth  int
	charHeight int
}

func generateImage(source image.Image, text string, cfg Config) (image.Image, error) {
	if source == nil {
		return nil, errors.New("input image is required")
	}
	if source.Bounds().Dx() <= 0 || source.Bounds().Dy() <= 0 {
		return nil, errors.New("input image must have positive dimensions")
	}
	if err := validateSourcePalette(source); err != nil {
		return nil, err
	}

	runes, err := normalizeText(text)
	if err != nil {
		return nil, err
	}

	processed := prepareSource(source, cfg)
	return render(processed, runes, cfg.FontPath, cfg.BaseFontSize)
}

// validateSourcePalette rejects invalid indexes before imaging's concurrent
// scanner can panic while resizing, adjusting contrast, or converting to gray.
// See https://github.com/disintegration/imaging/issues/165 (CVE-2023-36308).
func validateSourcePalette(source image.Image) error {
	paletted, ok := source.(*image.Paletted)
	if !ok {
		return nil
	}
	bounds := paletted.Bounds()
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		start := paletted.PixOffset(bounds.Min.X, y)
		for _, index := range paletted.Pix[start : start+bounds.Dx()] {
			if int(index) >= len(paletted.Palette) {
				return fmt.Errorf("input image has palette index %d outside palette of %d colors", index, len(paletted.Palette))
			}
		}
	}
	return nil
}

func render(
	source image.Image,
	text []rune,
	fontPath string,
	baseFontSize float64,
) (image.Image, error) {
	width := source.Bounds().Dx()
	height := source.Bounds().Dy()

	canvas := image.NewRGBA(image.Rect(0, 0, width, height))
	ctx := gg.NewContextForRGBA(canvas)

	metrics, err := measureFontGrid(ctx, fontPath, baseFontSize, width)
	if err != nil {
		return nil, err
	}
	if metrics.charWidth > width || metrics.charHeight > height {
		return nil, fmt.Errorf(
			"font size is too large for the image: char cell is %dx%d, image is %dx%d",
			metrics.charWidth,
			metrics.charHeight,
			width,
			height,
		)
	}

	drawText(ctx, source, text, metrics)
	return ctx.Image(), nil
}

func measureFontGrid(
	ctx *gg.Context,
	fontPath string,
	baseFontSize float64,
	imageWidth int,
) (fontMetrics, error) {
	fontSize := calculateScaledFontSize(baseFontSize, float64(imageWidth))
	if err := ctx.LoadFontFace(fontPath, fontSize); err != nil {
		return fontMetrics{}, fmt.Errorf("load font face %q: %w", fontPath, err)
	}

	// Measure several monospace glyphs and average them to reduce rounding noise.
	width, height := ctx.MeasureString("MMMMMMMMMM")
	return fontMetrics{
		charWidth:  max(1, int(math.Ceil(width/10))),
		charHeight: max(1, int(math.Ceil(height*verticalSpacingMultiplier))),
	}, nil
}

func drawText(
	ctx *gg.Context,
	source image.Image,
	text []rune,
	metrics fontMetrics,
) {
	textIndex := 0

	for y := metrics.charHeight / 2; y < source.Bounds().Dy(); y += metrics.charHeight {
		for x := metrics.charWidth / 2; x < source.Bounds().Dx(); x += metrics.charWidth {
			r, g, b, a := sampleRGBA(source, x, y)
			if a == 0 {
				continue
			}

			ctx.SetRGBA(r, g, b, a)
			ctx.DrawStringAnchored(
				string(text[textIndex]),
				float64(x),
				float64(y),
				gridOffsetFraction,
				gridOffsetFraction,
			)

			textIndex = (textIndex + 1) % len(text)
		}
	}
}

// calculateScaledFontSize preserves the established text density across common
// image resolutions.
func calculateScaledFontSize(baseSize, imageWidth float64) float64 {
	switch {
	case imageWidth >= 7200:
		return baseSize * 4.5
	case imageWidth >= 4800:
		return baseSize * 4.0
	case imageWidth >= 3600:
		return baseSize * 3.5
	case imageWidth >= 2160:
		return baseSize * 2.0
	case imageWidth >= 1080:
		return baseSize * 1.5
	case imageWidth <= 720:
		return baseSize * 0.75
	default:
		return baseSize
	}
}

func sampleRGBA(img image.Image, x, y int) (float64, float64, float64, float64) {
	bounds := img.Bounds()
	r, g, b, a := img.At(bounds.Min.X+x, bounds.Min.Y+y).RGBA()
	return float64(r) / maxColorChannelValue,
		float64(g) / maxColorChannelValue,
		float64(b) / maxColorChannelValue,
		float64(a) / maxColorChannelValue
}
