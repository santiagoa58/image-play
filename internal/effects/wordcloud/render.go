package wordcloud

import (
	"fmt"
	"image"
	"image/color"
	"math"

	"github.com/fogleman/gg"
	"golang.org/x/image/font"
)

// Render draws the accepted layout, sampling colors beneath the actual letters.
func Render(source image.Image, background color.NRGBA, fontPath string, placed []PlacedWord, outputPath string, debug bool, colorMode string) error {
	dc := gg.NewContext(source.Bounds().Dx(), source.Bounds().Dy())
	dc.SetColor(background)
	dc.Clear()
	for _, word := range placed {
		if err := drawPlacedWord(dc, source, background, fontPath, word, colorMode); err != nil {
			return err
		}
	}
	if err := dc.SavePNG(outputPath); err != nil {
		return fmt.Errorf("save word cloud: %w", err)
	}
	if err := writeWordcloudDebug(debug, outputPath, dc.Image()); err != nil {
		return fmt.Errorf("write word-cloud diagnostics: %w", err)
	}
	return nil
}

func drawPlacedWord(dc *gg.Context, source image.Image, background color.NRGBA, fontPath string, word PlacedWord, colorMode string) error {
	face, err := gg.LoadFontFace(fontPath, word.Word.FontSize)
	if err != nil {
		return fmt.Errorf("load font for word %q at %.1fpx: %w", word.Word.Text, word.Word.FontSize, err)
	}
	defer face.Close()
	glyph := wordGlyph(word, face)
	dc.SetFontFace(face)
	dc.SetColor(sampleGlyphColor(source, word, glyph, background, colorMode))
	dc.Push()
	defer dc.Pop()
	dc.Translate(word.X, word.Y)
	dc.Rotate(float64(word.Angle) * math.Pi / 180)
	dc.DrawString(word.Word.Text, word.Word.BaselineX, word.Word.BaselineY)
	return nil
}

// wordGlyph uses the same face and baseline offsets as the final drawing. The
// transparent margin prevents clipping at the sampled bitmap's edges.
func wordGlyph(word PlacedWord, face font.Face) image.Image {
	dc := gg.NewContext(max(1, int(math.Ceil(word.Word.Width))+4), max(1, int(math.Ceil(word.Word.Height))+4))
	dc.SetFontFace(face)
	dc.SetColor(color.White)
	dc.DrawString(word.Word.Text, float64(dc.Width())/2+word.Word.BaselineX, float64(dc.Height())/2+word.Word.BaselineY)
	return dc.Image()
}
