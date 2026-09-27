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
		{"vertical", 7, 90, color.NRGBA{R: 223, B: 31, A: 255}},
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
	want := color.NRGBA{R: 191, B: 63, A: 255}
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
