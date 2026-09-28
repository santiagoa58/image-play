package wordcloud

import (
	"image"
	"image/color"
	"image/png"
	"math"
	"os"
	"path/filepath"
	"testing"

	"github.com/santiagoa58/image-play/internal/textutil"
)

func TestSampleWordColorFollowsPositionAndRotation(t *testing.T) {
	source := image.NewNRGBA(image.Rect(10, 20, 30, 40))
	red := color.NRGBA{R: 255, A: 255}
	blue := color.NRGBA{B: 255, A: 255}
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
		{"left", 5, 0, red}, {"right", 15, 0, blue},
		// The rotated footprint crosses both halves only after the dimensions swap.
		{"unrotated", 7, 0, red},
		{"vertical", 7, 90, red},
	} {
		t.Run(tc.name, func(t *testing.T) {
			word := PlacedWord{Word: textutil.Word{Width: 4, Height: 8}, X: tc.x, Y: 10, Angle: tc.angle}
			if got := sampleWordColor(source, word); got != tc.want {
				t.Fatalf("color = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestSampleWordColorIgnoresHiddenRGBAndWeightsAlpha(t *testing.T) {
	source := image.NewNRGBA(image.Rect(0, 0, 3, 1))
	source.SetNRGBA(0, 0, color.NRGBA{R: 255, A: 255})
	source.SetNRGBA(1, 0, color.NRGBA{B: 255, A: 85})
	source.SetNRGBA(2, 0, color.NRGBA{G: 255, A: 0})
	word := PlacedWord{Word: textutil.Word{Width: 3, Height: 1}, X: 1.5, Y: 0.5}
	want := color.NRGBA{R: 255, A: 255}
	if got := sampleWordColor(source, word); got != want {
		t.Fatalf("color = %v, want %v", got, want)
	}
	// Clipping an out-of-bounds footprint must not dilute the surviving pixels.
	word.X = -1
	word.Word.Width = 4
	if got := sampleWordColor(source, word); got != (color.NRGBA{R: 255, A: 255}) {
		t.Fatalf("clipped color = %v", got)
	}
}

func TestSampleWordColorKeepsDominantSourceHue(t *testing.T) {
	source := image.NewNRGBA(image.Rect(0, 0, 3, 1))
	red := color.NRGBA{R: 255, A: 255}
	yellow := color.NRGBA{R: 255, G: 255, A: 255}
	source.SetNRGBA(0, 0, red)
	source.SetNRGBA(1, 0, red)
	source.SetNRGBA(2, 0, yellow)
	word := PlacedWord{Word: textutil.Word{Width: 3, Height: 1}, X: 1.5, Y: 0.5}
	if got := sampleWordColor(source, word); got != red {
		t.Fatalf("color = %v, want actual dominant source color %v", got, red)
	}
}

func TestContrastingWordColorMovesAwayFromEitherCanvas(t *testing.T) {
	gray := color.NRGBA{R: 48, G: 48, B: 48, A: 255}
	got := contrastingWordColor(gray, color.Black)
	if got.R <= gray.R || got.R != got.G || got.G != got.B {
		t.Fatalf("dark gray = %v, want a brighter neutral gray", got)
	}
	light := color.NRGBA{R: 230, G: 220, B: 210, A: 255}
	got = contrastingWordColor(light, color.White)
	if got.R >= light.R || got.G >= light.G || got.B >= light.B {
		t.Fatalf("light color = %v, want darker color against white", got)
	}
	for _, tc := range []struct {
		name       string
		foreground color.NRGBA
		background color.Color
	}{
		{"green on white", color.NRGBA{G: 255, A: 255}, color.White},
		{"black on black", color.NRGBA{A: 255}, color.Black},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := contrastingWordColor(tc.foreground, tc.background)
			bg := color.NRGBAModel.Convert(tc.background).(color.NRGBA)
			light, dark := math.Max(luminance(got), luminance(bg)), math.Min(luminance(got), luminance(bg))
			if ratio := (light + 0.05) / (dark + 0.05); ratio < 4.5 {
				t.Fatalf("color = %v, contrast ratio = %.2f, want at least 4.5", got, ratio)
			}
		})
	}
}

func TestRenderUsesSourceColorsAndBackground(t *testing.T) {
	source := image.NewNRGBA(image.Rect(0, 0, 100, 80))
	for y := 0; y < 80; y++ {
		for x := 0; x < 100; x++ {
			source.SetNRGBA(x, y, color.NRGBA{R: 255, A: 255})
		}
	}
	words := []PlacedWord{{Word: textutil.Word{Text: "RED", FontSize: 20, Width: 36, Height: 24}, X: 50, Y: 40}}
	for _, dark := range []bool{false, true} {
		output := filepath.Join(t.TempDir(), "cloud.png")
		if err := Render(source, dark, "../../../fonts/NotoSansMono-VariableFont_wdth,wght.ttf", words, output, false); err != nil {
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
		if got, want := color.RGBAModel.Convert(rendered.At(0, 0)), color.RGBAModel.Convert(canvasColor(dark)); got != want {
			t.Fatalf("background = %v, want %v", got, want)
		}
		foundRed := false
		for y := 0; y < 80; y++ {
			for x := 0; x < 100; x++ {
				r, g, b, _ := rendered.At(x, y).RGBA()
				if r > 0 && g == 0 && b == 0 {
					foundRed = true
				}
			}
		}
		if !foundRed {
			t.Fatal("no source-colored glyph pixels")
		}
	}
}
