package textmosaic

import (
	"errors"
	"fmt"
	"image"
	"image/draw"
	"math"
	"strings"

	"github.com/fogleman/gg"
)

const (
	verticalSpacingMultiplier = 1.08
)

type fontMetrics struct {
	charWidth  int
	lineHeight int
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

	if cfg.Uppercase {
		text = strings.ToUpper(text)
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

	mask := image.NewRGBA(image.Rect(0, 0, width, height))
	ctx := gg.NewContextForRGBA(mask)

	metrics, err := measureFontGrid(ctx, fontPath, baseFontSize, width)
	if err != nil {
		return nil, err
	}
	if metrics.charWidth > width || metrics.lineHeight > height {
		return nil, fmt.Errorf(
			"font size is too large for the image: minimum cell is %dx%d, image is %dx%d",
			metrics.charWidth,
			metrics.lineHeight,
			width,
			height,
		)
	}

	ctx.SetRGB(1, 1, 1)
	drawTextMask(ctx, text, metrics, width, height)

	// DrawMask multiplies the source alpha by the antialiased glyph coverage.
	// Sampling the image here, rather than once per character, preserves sharp
	// boundaries and color changes inside every glyph.
	canvas := image.NewRGBA(mask.Bounds())
	draw.DrawMask(canvas, canvas.Bounds(), source, source.Bounds().Min, mask, image.Point{}, draw.Src)
	return canvas, nil
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

	width, _ := ctx.MeasureString("M")
	return fontMetrics{
		charWidth:  max(1, int(math.Ceil(width))),
		lineHeight: max(1, int(math.Ceil(ctx.FontHeight()*verticalSpacingMultiplier))),
	}, nil
}

func drawTextMask(
	ctx *gg.Context,
	text []rune,
	metrics fontMetrics,
	width, height int,
) {
	textIndex := 0
	for y := metrics.lineHeight / 2; y < height; y += metrics.lineHeight {
		x := 0.0
		for x < float64(width) {
			glyph := string(text[textIndex])
			advance, _ := ctx.MeasureString(glyph)
			if advance <= 0 {
				advance = 1
			}
			ctx.DrawStringAnchored(glyph, x, float64(y), 0, 0.5)
			x += advance
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
