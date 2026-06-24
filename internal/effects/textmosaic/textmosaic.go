package textmosaic

import (
	"errors"
	"fmt"
	"image"
	"log/slog"
	"math"
	"strings"

	"github.com/disintegration/imaging"
	"github.com/fogleman/gg"
)

// Constants for visual tuning
const (
	// verticalSpacingMultiplier adds breathing room between rows of text
	verticalSpacingMultiplier = 1.4

	// gridOffsetFraction starts the grid slightly inset for better balance
	gridOffsetFraction = 0.5

	maxColorChannelValue = 65535.0
)

// Config defines all parameters needed to generate a text mosaic.
type Config struct {
	// Logger receives debug logs. If nil, slog.Default() is used.
	Logger *slog.Logger

	// Text is repeated across the image.
	//
	// Text supports ASCII and basic Unicode letters such as Cyrillic, Greek,
	// accented Latin characters, etc. Emoji and complex grapheme
	// clusters are not supported. For best alignment, use text that renders well
	// in the selected monospace font.
	Text string

	// InputImage is the source image to base the mosaic on.
	InputImage image.Image

	// TargetWidth is the desired output width. Use 0 to keep the original width.
	TargetWidth int

	// MonoFontPath is the path to a monospace TTF/OTF font.
	MonoFontPath string

	// BaseFontSize is the base font size before automatic scaling.
	// Use 0 for the default of 14px.
	BaseFontSize float64

	// IsBlackAndWhite converts the processed image to grayscale before sampling.
	IsBlackAndWhite bool

	// ContrastPercent adjusts image contrast before sampling.
	// 0 means no change. Positive values increase contrast; negative values decrease it.
	ContrastPercent float64
}

// fontMetrics holds only the measurements needed for the grid.
type fontMetrics struct {
	charWidth  int
	charHeight int
}

// textCanvas holds the drawing context and canvas dimensions.
type textCanvas struct {
	context *gg.Context
	width   int
	height  int
}

// Generate creates a text mosaic where repeated text forms the visual structure of the input image.
// The text color is sampled from the source image at each position.
func Generate(conf Config) (image.Image, error) {
	conf, err := normalizeConfig(conf)
	if err != nil {
		return nil, err
	}
	txt, err := normalizeText(conf.Text)
	if err != nil {
		return nil, err
	}
	conf.Logger.Debug("starting text mosaic generation")
	// Pre-process the source image (resize, contrast, optional B&W)
	processed := applyImageProcessing(conf)
	canvas := createTextCanvas(processed)
	// Load font and create drawing context
	font, err := calculateFontMetrics(conf, canvas.context, processed)
	if err != nil {
		return nil, err
	}
	if font.charWidth > canvas.width || font.charHeight > canvas.height {
		return nil, fmt.Errorf(
			"font size is too large for the image: char cell is %dx%d, image is %dx%d",
			font.charWidth,
			font.charHeight,
			canvas.width,
			canvas.height,
		)
	}

	conf.Logger.Debug("drawing text mosaic", "text_runes", len(txt))

	// Draw the colored text on a transparent canvas
	result := drawTextMosaic(font, canvas, processed, txt)

	conf.Logger.Debug("text mosaic generation completed")
	return result, nil
}

// createTextCanvas creates a new transparent canvas for the final image output
func createTextCanvas(img image.Image) *textCanvas {
	w := img.Bounds().Dx()
	h := img.Bounds().Dy()
	canvas := image.NewRGBA(image.Rect(0, 0, w, h))
	ctx := gg.NewContextForRGBA(canvas)

	return &textCanvas{
		context: ctx,
		width:   w,
		height:  h,
	}
}

// normalizeConfig validates the config and applies default values.
func normalizeConfig(conf Config) (Config, error) {
	if conf.Logger == nil {
		conf.Logger = slog.Default()
	}
	if conf.InputImage == nil {
		return conf, errors.New("input image is required")
	}
	bounds := conf.InputImage.Bounds()
	if bounds.Dx() <= 0 || bounds.Dy() <= 0 {
		return conf, errors.New("input image must have positive dimensions")
	}
	if conf.MonoFontPath == "" {
		return conf, errors.New("monospace font path is required")
	}
	if conf.BaseFontSize < 0 {
		return conf, errors.New("base font size cannot be negative")
	}
	if conf.ContrastPercent < -100 || conf.ContrastPercent > 100 {
		return conf, errors.New("contrast percent must be between -100.0 and 100.0")
	}
	if conf.TargetWidth < 0 {
		return conf, errors.New("target width cannot be negative")
	}
	if conf.BaseFontSize == 0 {
		conf.BaseFontSize = 14.0
	}
	return conf, nil
}

// applyImageProcessing resizes and enhances the source image for better mosaic quality.
func applyImageProcessing(conf Config) image.Image {
	img := conf.InputImage

	if conf.TargetWidth > 0 {
		// only resizes for non-default target width (preserving original aspect-ratio)
		img = imaging.Resize(img, conf.TargetWidth, 0, imaging.Lanczos)
	}
	if conf.ContrastPercent != 0 {
		img = imaging.AdjustContrast(img, conf.ContrastPercent)
	}
	if conf.IsBlackAndWhite {
		img = imaging.Grayscale(img)
	}

	return img
}

// calculateFontMetrics loads the font and calculates character dimensions for the grid.
func calculateFontMetrics(conf Config, ctx *gg.Context, img image.Image) (*fontMetrics, error) {
	fontSize := calculateScaledFontSize(conf.BaseFontSize, float64(img.Bounds().Dx()))

	if err := ctx.LoadFontFace(conf.MonoFontPath, fontSize); err != nil {
		return nil, fmt.Errorf("load font face %q: %w", conf.MonoFontPath, err)
	}

	// Measures 10 'M' chars for consistent grid sizing (more accurate than measuring only 1 char)
	w, h := ctx.MeasureString("MMMMMMMMMM")
	// get the average char width
	w = math.Ceil(w / 10)
	// adds generous vertical spacing
	h = math.Ceil(h * verticalSpacingMultiplier)

	return &fontMetrics{
		charWidth:  max(1, int(w)),
		charHeight: max(1, int(h)),
	}, nil
}

// calculateScaledFontSize scales font size based on image resolution.
// These multipliers were tuned to maintain good visual density across resolutions.
func calculateScaledFontSize(baseSize float64, imgWidth float64) float64 {
	switch {
	case imgWidth >= 7200:
		return baseSize * 4.5
	case imgWidth >= 4800:
		return baseSize * 4.0
	case imgWidth >= 3600:
		return baseSize * 3.5
	case imgWidth >= 2160:
		return baseSize * 2.0
	case imgWidth >= 1080:
		return baseSize * 1.5
	case imgWidth <= 720:
		return baseSize * 0.75
	default:
		return baseSize
	}
}

// normalizeText normalizes whitespace for consistent text flow.
func normalizeText(text string) ([]rune, error) {
	s := strings.NewReplacer(
		"\n", " ",
		"\r", " ",
		"\t", " ",
	).Replace(text)
	// collapses repeated spaces into a single one
	s = strings.Join(strings.Fields(s), " ")
	if s == "" {
		return nil, errors.New("text cannot be empty")
	}
	s = s + " "
	return []rune(s), nil
}

// drawTextMosaic draws the repeated colored text to form the mosaic effect on a transparent canvas.
func drawTextMosaic(font *fontMetrics, canvas *textCanvas, sample image.Image, txt []rune) image.Image {
	var (
		ctx   = canvas.context
		charW = font.charWidth
		charH = font.charHeight
		imgH  = canvas.height
		imgW  = canvas.width
	)

	txtIdx := 0
	txtLen := len(txt)

	for y := charH / 2; y < imgH; y += charH {
		for x := charW / 2; x < imgW; x += charW {
			r, g, b, a := getImgRGBA(sample, x, y)

			if a == 0 {
				// avoids drawing if pixel is transparent
				continue
			}
			ctx.SetRGBA(r, g, b, a)
			char := txt[txtIdx]
			ctx.DrawStringAnchored(string(char), float64(x), float64(y), gridOffsetFraction, gridOffsetFraction)

			txtIdx++
			if txtIdx >= txtLen {
				txtIdx = 0
			}

		}
	}

	return ctx.Image()
}

// getImgRGBA gets the pixel color of an image at pixel position (x,y) and returns it as RGBA with values between 0 and 1
func getImgRGBA(img image.Image, x, y int) (float64, float64, float64, float64) {
	bounds := img.Bounds()
	pixelColor := img.At(bounds.Min.X+x, bounds.Min.Y+y)
	r, g, b, a := pixelColor.RGBA()
	return float64(r) / maxColorChannelValue,
		float64(g) / maxColorChannelValue,
		float64(b) / maxColorChannelValue,
		float64(a) / maxColorChannelValue
}
