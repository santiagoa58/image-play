package wordcloud

import (
	"fmt"
	"image"
	"image/color"
	"math"

	"github.com/fogleman/gg"
)

// Render draws the accepted word layout to a PNG.
func Render(
	source image.Image,
	darkBackground bool,
	fontPath string,
	placed []PlacedWord,
	outputPath string,
	debug bool,
) error {
	bounds := source.Bounds()
	dc := gg.NewContext(bounds.Dx(), bounds.Dy())
	dc.SetColor(canvasColor(darkBackground))
	dc.Clear()

	for _, p := range placed {
		if err := dc.LoadFontFace(fontPath, p.Word.FontSize); err != nil {
			return fmt.Errorf(
				"load font for word %q at %.1fpx: %w",
				p.Word.Text,
				p.Word.FontSize,
				err,
			)
		}

		dc.SetColor(sampleWordColor(source, p))
		dc.Push()
		dc.Translate(p.X, p.Y)
		dc.Rotate(float64(p.Angle) * math.Pi / 180)
		dc.DrawStringAnchored(p.Word.Text, 0, 0, 0.5, 0.5)
		dc.Pop()
	}

	if err := dc.SavePNG(outputPath); err != nil {
		return fmt.Errorf("save word cloud: %w", err)
	}
	if err := writeWordcloudDebug(debug, outputPath, dc.Image()); err != nil {
		return fmt.Errorf("write word-cloud diagnostics: %w", err)
	}

	return nil
}

// canvasColor keeps unoccupied space on the same brightness side as the border.
func canvasColor(darkBackground bool) color.Color {
	if darkBackground {
		return color.Black
	}
	return color.White
}

// sampleWordColor averages the source under a word's measured rectangle. Each
// word has one color, so letters remain legible across small source variations.
// Alpha weights visible contributions; hidden RGB cannot tint the result.
func sampleWordColor(source image.Image, word PlacedWord) color.NRGBA {
	width, height := word.Word.Width, word.Word.Height
	if word.Angle == 90 {
		width, height = height, width
	}
	bounds := source.Bounds()
	area := image.Rect(
		int(math.Floor(word.X-width/2))+bounds.Min.X,
		int(math.Floor(word.Y-height/2))+bounds.Min.Y,
		int(math.Ceil(word.X+width/2))+bounds.Min.X,
		int(math.Ceil(word.Y+height/2))+bounds.Min.Y,
	).Intersect(bounds)
	var red, green, blue, alpha uint64
	for y := area.Min.Y; y < area.Max.Y; y++ {
		for x := area.Min.X; x < area.Max.X; x++ {
			r, g, b, a := source.At(x, y).RGBA()
			red += uint64(r)
			green += uint64(g)
			blue += uint64(b)
			alpha += uint64(a)
		}
	}
	if alpha == 0 {
		return color.NRGBA{A: 255}
	}
	return color.NRGBA{
		R: uint8(red * 255 / alpha),
		G: uint8(green * 255 / alpha),
		B: uint8(blue * 255 / alpha),
		A: 255,
	}
}
