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

		dc.SetColor(contrastingWordColor(sampleWordColor(source, p), canvasColor(darkBackground)))
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

// sampleWordColor selects a source color from the most prevalent color family
// beneath the word's measured rectangle. This avoids making
// a muted, artificial blend across differently colored source pixels.
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
	type bucket struct{ weight, red, green, blue uint64 }
	buckets := make(map[uint16]bucket)
	for y := area.Min.Y; y < area.Max.Y; y++ {
		for x := area.Min.X; x < area.Max.X; x++ {
			c := color.NRGBAModel.Convert(source.At(x, y)).(color.NRGBA)
			if c.A == 0 {
				continue
			}
			key := uint16(c.R/16)<<8 | uint16(c.G/16)<<4 | uint16(c.B/16)
			b := buckets[key]
			weight := uint64(c.A)
			b.weight += weight
			b.red += uint64(c.R) * weight
			b.green += uint64(c.G) * weight
			b.blue += uint64(c.B) * weight
			buckets[key] = b
		}
	}
	var chosen bucket
	var chosenKey uint16
	for key, b := range buckets {
		if b.weight > chosen.weight || b.weight == chosen.weight && key < chosenKey {
			chosen, chosenKey = b, key
		}
	}
	if chosen.weight == 0 {
		return color.NRGBA{A: 255}
	}
	meanRed := int(chosen.red / chosen.weight)
	meanGreen := int(chosen.green / chosen.weight)
	meanBlue := int(chosen.blue / chosen.weight)
	bestDistance := int(^uint(0) >> 1)
	best := color.NRGBA{A: 255}
	for y := area.Min.Y; y < area.Max.Y; y++ {
		for x := area.Min.X; x < area.Max.X; x++ {
			c := color.NRGBAModel.Convert(source.At(x, y)).(color.NRGBA)
			key := uint16(c.R/16)<<8 | uint16(c.G/16)<<4 | uint16(c.B/16)
			if c.A == 0 || key != chosenKey {
				continue
			}
			dr, dg, db := int(c.R)-meanRed, int(c.G)-meanGreen, int(c.B)-meanBlue
			distance := dr*dr + dg*dg + db*db
			if distance < bestDistance {
				bestDistance, best = distance, c
			}
		}
	}
	best.A = 255
	return best
}

// contrastingWordColor adjusts a sampled color until it is legible on the canvas.
func contrastingWordColor(c color.NRGBA, background color.Color) color.NRGBA {
	bg := color.NRGBAModel.Convert(background).(color.NRGBA)
	backgroundLuminance := luminance(bg)
	contrast := func(candidate color.NRGBA) float64 {
		light, dark := math.Max(luminance(candidate), backgroundLuminance), math.Min(luminance(candidate), backgroundLuminance)
		return (light + 0.05) / (dark + 0.05)
	}
	if contrast(c) >= 4.5 {
		return c
	}
	target := color.NRGBA{A: 255}
	if backgroundLuminance < 0.5 {
		target.R, target.G, target.B = 255, 255, 255
	}
	blend := func(t float64) color.NRGBA {
		return color.NRGBA{
			R: uint8(math.Round(float64(c.R)*(1-t) + float64(target.R)*t)),
			G: uint8(math.Round(float64(c.G)*(1-t) + float64(target.G)*t)),
			B: uint8(math.Round(float64(c.B)*(1-t) + float64(target.B)*t)),
			A: 255,
		}
	}
	low, high := 0.0, 1.0
	for range 12 {
		mid := (low + high) / 2
		if contrast(blend(mid)) >= 4.5 {
			high = mid
		} else {
			low = mid
		}
	}
	return blend(high)
}

func luminance(c color.NRGBA) float64 {
	linear := func(channel uint8) float64 {
		value := float64(channel) / 255
		if value <= 0.04045 {
			return value / 12.92
		}
		return math.Pow((value+0.055)/1.055, 2.4)
	}
	return 0.2126*linear(c.R) + 0.7152*linear(c.G) + 0.0722*linear(c.B)
}
