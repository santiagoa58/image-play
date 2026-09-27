package wordcloud

import (
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"testing"

	"github.com/santiagoa58/image-play/internal/textutil"
)

func opaqueGlyph(width, height int) *image.Alpha {
	glyph := image.NewAlpha(image.Rect(0, 0, width, height))
	for i := range glyph.Pix {
		glyph.Pix[i] = 255
	}
	return glyph
}

func TestGlyphMeanFollowsPositionAndRotation(t *testing.T) {
	source := image.NewNRGBA(image.Rect(10, 20, 30, 40))
	red, blue := color.NRGBA{R: 255, A: 255}, color.NRGBA{B: 255, A: 255}
	for y := 20; y < 40; y++ {
		for x := 10; x < 30; x++ {
			c := red
			if x >= 20 {
				c = blue
			}
			source.SetNRGBA(x, y, c)
		}
	}
	for _, tc := range []struct {
		name  string
		x     float64
		angle int
		want  color.NRGBA
	}{
		{"left", 5, 0, red}, {"right", 15, 0, blue}, {"unrotated", 7, 0, red}, {"vertical", 7, 90, color.NRGBA{R: 223, B: 31, A: 255}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			word := PlacedWord{X: tc.x, Y: 10, Angle: tc.angle}
			if got := sampleGlyphColor(source, word, opaqueGlyph(4, 8), color.NRGBA{A: 255}, "mean"); got != tc.want {
				t.Fatalf("color = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestGlyphSamplingIgnoresEmptyLetterSpaceAndHiddenRGB(t *testing.T) {
	source := image.NewNRGBA(image.Rect(0, 0, 4, 1))
	source.SetNRGBA(0, 0, color.NRGBA{R: 255, A: 255})
	source.SetNRGBA(1, 0, color.NRGBA{B: 255, A: 85})
	source.SetNRGBA(2, 0, color.NRGBA{G: 255, A: 0})
	source.SetNRGBA(3, 0, color.NRGBA{G: 255, A: 255})
	glyph := opaqueGlyph(4, 1)
	glyph.Pix[3] = 0
	word := PlacedWord{X: 2, Y: 0.5}
	want := color.NRGBA{R: 191, B: 63, A: 255}
	if got := sampleGlyphColor(source, word, glyph, color.NRGBA{A: 255}, "mean"); got != want {
		t.Fatalf("color = %v, want %v", got, want)
	}
	if got := sampleGlyphColor(source, word, glyph, color.NRGBA{A: 255}, "representative"); got != (color.NRGBA{R: 255, A: 255}) {
		t.Fatalf("representative = %v, want actual red", got)
	}
	// Clipping must not dilute the surviving red pixel with out-of-bounds black.
	word.X = 0
	if got := sampleGlyphColor(source, word, glyph, color.NRGBA{A: 255}, "mean"); got != (color.NRGBA{R: 255, A: 255}) {
		t.Fatalf("clipped color = %v", got)
	}
}

func TestRepresentativeColorDoesNotInventMixedHue(t *testing.T) {
	source := image.NewNRGBA(image.Rect(0, 0, 3, 1))
	red, yellow := color.NRGBA{R: 255, A: 255}, color.NRGBA{R: 255, G: 255, A: 255}
	source.SetNRGBA(0, 0, red)
	source.SetNRGBA(1, 0, red)
	source.SetNRGBA(2, 0, yellow)
	word := PlacedWord{X: 1.5, Y: 0.5}
	glyph := opaqueGlyph(3, 1)
	if got := sampleGlyphColor(source, word, glyph, color.NRGBA{A: 255}, "representative"); got != red {
		t.Fatalf("representative = %v, want %v", got, red)
	}
	if got := sampleGlyphColor(source, word, glyph, color.NRGBA{A: 255}, "mean"); got != (color.NRGBA{R: 255, G: 85, A: 255}) {
		t.Fatalf("mean = %v", got)
	}
}

func TestRenderUsesSourceColorsAndBackground(t *testing.T) {
	source := image.NewNRGBA(image.Rect(0, 0, 100, 80))
	for y := 0; y < 80; y++ {
		for x := 0; x < 100; x++ {
			source.SetNRGBA(x, y, color.NRGBA{R: 255, A: 255})
		}
	}
	measured, err := textutil.MeasureWord("RED", 1, wordcloudTestFontPath(t), 20)
	if err != nil {
		t.Fatal(err)
	}
	words := []PlacedWord{{Word: measured, X: 50, Y: 40}}
	for _, background := range []color.NRGBA{{A: 255}, {R: 255, G: 255, B: 255, A: 255}} {
		output := filepath.Join(t.TempDir(), "cloud.png")
		if err := Render(source, background, wordcloudTestFontPath(t), words, output, false, "representative"); err != nil {
			t.Fatal(err)
		}
		file, err := os.Open(output)
		if err != nil {
			t.Fatal(err)
		}
		rendered, err := png.Decode(file)
		file.Close()
		if err != nil {
			t.Fatal(err)
		}
		if rendered.Bounds() != source.Bounds() {
			t.Fatalf("bounds = %v", rendered.Bounds())
		}
		if got := color.NRGBAModel.Convert(rendered.At(0, 0)); got != background {
			t.Fatalf("background = %v, want %v", got, background)
		}
		foundRed := false
		for y := 0; y < 80; y++ {
			for x := 0; x < 100; x++ {
				r, g, b, _ := rendered.At(x, y).RGBA()
				if r == 65535 && g == 0 && b == 0 {
					foundRed = true
				}
			}
		}
		if !foundRed {
			t.Fatal("no source-colored glyph pixels")
		}
	}
}
