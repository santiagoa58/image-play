package wordcloud

import (
	"fmt"
	"math"

	"github.com/fogleman/gg"
)

// Render draws the accepted word layout to a PNG.
func Render(
	width, height int,
	fontPath string,
	placed []PlacedWord,
	outputPath string,
	debug bool,
) error {
	dc := gg.NewContext(width, height)
	dc.SetRGB(1, 1, 1)
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

		dc.SetRGB(0, 0, 0)
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
