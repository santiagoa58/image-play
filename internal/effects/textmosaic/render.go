package textmosaic

import (
	"errors"
	"fmt"
	"image"
	"image/draw"
	"math"
	"strings"

	"github.com/disintegration/imaging"
	"github.com/fogleman/gg"
)

const (
	verticalSpacingMultiplier = 1.08
	maxUpscaledDimension      = 4096
)

type fontMetrics struct {
	charWidth  int
	lineHeight int
	fontSize   float64
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
	return render(processed, runes, cfg)
}

// validateSourcePalette rejects invalid indexes before imaging's concurrent
// scanner can panic while resizing or adjusting contrast.
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
	cfg Config,
) (image.Image, error) {
	width := source.Bounds().Dx()
	height := source.Bounds().Dy()
	scale := outputScaleForSize(width, height, cfg.OutputScale)
	if scale > 1 {
		source = imaging.Resize(source, int(math.Round(float64(width)*scale)), 0, imaging.Lanczos)
	}
	outputWidth := source.Bounds().Dx()
	outputHeight := source.Bounds().Dy()
	fontScale := float64(outputWidth) / float64(width)

	mask := image.NewRGBA(image.Rect(0, 0, outputWidth, outputHeight))
	ctx := gg.NewContextForRGBA(mask)

	metrics, err := measureFontGrid(ctx, cfg.FontPath, cfg.BaseFontSize, width, fontScale)
	if err != nil {
		return nil, err
	}
	if metrics.charWidth > outputWidth || metrics.lineHeight > outputHeight {
		return nil, fmt.Errorf(
			"font size is too large for the image: minimum cell is %dx%d, image is %dx%d",
			metrics.charWidth,
			metrics.lineHeight,
			outputWidth,
			outputHeight,
		)
	}

	ctx.SetRGB(1, 1, 1)
	drawTextMask(ctx, text, metrics, cfg.LetterSpacing, cfg.WordSpacing, outputWidth, outputHeight)

	// DrawMask multiplies the source alpha by the antialiased glyph coverage.
	// Sampling the image here, rather than once per character, preserves sharp
	// boundaries and color changes inside every glyph.
	canvas := image.NewRGBA(mask.Bounds())
	draw.DrawMask(canvas, canvas.Bounds(), source, source.Bounds().Min, mask, image.Point{}, draw.Src)
	return canvas, nil
}

func outputScaleForSize(width, height int, requested float64) float64 {
	return min(requested, max(1, float64(maxUpscaledDimension)/float64(max(width, height))))
}

func measureFontGrid(
	ctx *gg.Context,
	fontPath string,
	baseFontSize float64,
	imageWidth int,
	outputScale float64,
) (fontMetrics, error) {
	fontSize := calculateScaledFontSize(baseFontSize, float64(imageWidth)) * outputScale
	if err := ctx.LoadFontFace(fontPath, fontSize); err != nil {
		return fontMetrics{}, fmt.Errorf("load font face %q: %w", fontPath, err)
	}

	width, _ := ctx.MeasureString("M")
	return fontMetrics{
		charWidth:  max(1, int(math.Ceil(width))),
		lineHeight: max(1, int(math.Ceil(ctx.FontHeight()*verticalSpacingMultiplier))),
		fontSize:   fontSize,
	}, nil
}

func drawTextMask(
	ctx *gg.Context,
	text []rune,
	metrics fontMetrics,
	letterSpacing, wordSpacing float64,
	width, height int,
) {
	textIndex := 0
	for y := metrics.lineHeight / 2; y < height; y += metrics.lineHeight {
		x := 0.0
		for x < float64(width) {
			glyph := string(text[textIndex])
			advance, _ := ctx.MeasureString(glyph)
			advance += letterSpacing * metrics.fontSize
			if text[textIndex] == ' ' {
				advance += wordSpacing * metrics.fontSize
			}
			ctx.DrawStringAnchored(glyph, x, float64(y), 0, 0.5)
			x += max(1, advance)
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
