package textmosaic

import (
	"image"
	"image/color"
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/disintegration/imaging"
)

func TestGenerateImage(t *testing.T) {
	fontPath := testFontPath(t)

	tests := []struct {
		name          string
		source        image.Image
		text          string
		config        Config
		wantWidth     int
		wantHeight    int
		wantNonEmpty  bool
		wantErrSubstr string
	}{
		{
			name:         "generates with original dimensions",
			source:       solidImage(120, 80, color.RGBA{R: 255, A: 255}),
			text:         "hello world",
			config:       NewConfig(WithFontPath(fontPath)),
			wantWidth:    120,
			wantHeight:   80,
			wantNonEmpty: true,
		},
		{
			name:   "resizes to target width",
			source: solidImage(200, 100, color.RGBA{G: 255, A: 255}),
			text:   "hello world",
			config: NewConfig(
				WithFontPath(fontPath),
				WithTargetWidth(100),
			),
			wantWidth:    100,
			wantHeight:   50,
			wantNonEmpty: true,
		},
		{
			name:         "supports basic unicode runes",
			source:       solidImage(160, 90, color.RGBA{B: 255, A: 255}),
			text:         "hello привет café",
			config:       NewConfig(WithFontPath(fontPath)),
			wantWidth:    160,
			wantHeight:   90,
			wantNonEmpty: true,
		},
		{
			name:         "transparent source stays transparent",
			source:       solidImage(120, 80, color.RGBA{R: 255, A: 0}),
			text:         "hello world",
			config:       NewConfig(WithFontPath(fontPath)),
			wantWidth:    120,
			wantHeight:   80,
			wantNonEmpty: false,
		},
		{
			name:          "rejects missing image",
			text:          "hello world",
			config:        NewConfig(WithFontPath(fontPath)),
			wantErrSubstr: "input image is required",
		},
		{
			name:          "rejects empty text",
			source:        solidImage(120, 80, color.RGBA{A: 255}),
			text:          " \n\t ",
			config:        NewConfig(WithFontPath(fontPath)),
			wantErrSubstr: "text cannot be empty",
		},
		{
			name:   "rejects oversized font",
			source: solidImage(20, 20, color.RGBA{A: 255}),
			text:   "hello world",
			config: NewConfig(
				WithFontPath(fontPath),
				WithBaseFontSize(100),
			),
			wantErrSubstr: "font size is too large",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := generateImage(tt.source, tt.text, tt.config)

			if tt.wantErrSubstr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErrSubstr) {
					t.Fatalf("generateImage() error = %v, want substring %q", err, tt.wantErrSubstr)
				}
				return
			}
			if err != nil {
				t.Fatalf("generateImage() error = %v", err)
			}

			if got.Bounds().Dx() != tt.wantWidth || got.Bounds().Dy() != tt.wantHeight {
				t.Fatalf(
					"generated size = %dx%d, want %dx%d",
					got.Bounds().Dx(),
					got.Bounds().Dy(),
					tt.wantWidth,
					tt.wantHeight,
				)
			}
			if visible := hasVisiblePixel(got); visible != tt.wantNonEmpty {
				t.Fatalf("hasVisiblePixel() = %v, want %v", visible, tt.wantNonEmpty)
			}
		})
	}
}

func TestGenerateWritesPNG(t *testing.T) {
	dir := t.TempDir()
	inputPath := filepath.Join(dir, "input.png")
	textPath := filepath.Join(dir, "text.txt")
	outputPath := filepath.Join(dir, "output.png")

	if err := imaging.Save(
		solidImage(160, 100, color.RGBA{R: 240, G: 180, B: 80, A: 255}),
		inputPath,
	); err != nil {
		t.Fatalf("save test input: %v", err)
	}
	if err := os.WriteFile(textPath, []byte("hello world from text mosaic"), 0o600); err != nil {
		t.Fatalf("write test text: %v", err)
	}

	cfg := NewConfig(
		WithInputPath(inputPath),
		WithOutputPath(outputPath),
		WithTextPath(textPath),
		WithFontPath(testFontPath(t)),
	)
	if err := Generate(cfg); err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	result, err := imaging.Open(outputPath)
	if err != nil {
		t.Fatalf("open generated output: %v", err)
	}
	if result.Bounds().Dx() != 160 || result.Bounds().Dy() != 100 {
		t.Fatalf(
			"generated size = %dx%d, want 160x100",
			result.Bounds().Dx(),
			result.Bounds().Dy(),
		)
	}
}

func TestGenerateImageHandlesNonZeroBounds(t *testing.T) {
	src := image.NewRGBA(image.Rect(50, 75, 170, 155))
	for y := src.Bounds().Min.Y; y < src.Bounds().Max.Y; y++ {
		for x := src.Bounds().Min.X; x < src.Bounds().Max.X; x++ {
			src.Set(x, y, color.RGBA{R: 255, A: 255})
		}
	}

	got, err := generateImage(
		src,
		"hello world",
		NewConfig(WithFontPath(testFontPath(t))),
	)
	if err != nil {
		t.Fatalf("generateImage() error = %v", err)
	}

	if got.Bounds().Min != (image.Point{}) {
		t.Fatalf("output min bounds = %v, want origin", got.Bounds().Min)
	}
	if got.Bounds().Dx() != src.Bounds().Dx() || got.Bounds().Dy() != src.Bounds().Dy() {
		t.Fatalf(
			"generated size = %dx%d, want %dx%d",
			got.Bounds().Dx(),
			got.Bounds().Dy(),
			src.Bounds().Dx(),
			src.Bounds().Dy(),
		)
	}
}

func TestRenderClipsSourceAtEachGlyphPixel(t *testing.T) {
	source := image.NewNRGBA(image.Rect(0, 0, 160, 100))
	for y := range 100 {
		for x := range 160 {
			c := color.NRGBA{R: 245, G: 30, B: 20, A: 255}
			if x%2 == 1 {
				c = color.NRGBA{R: 20, G: 80, B: 245, A: 255}
			}
			source.SetNRGBA(x, y, c)
		}
	}

	got, err := render(source, []rune("MMMM "), testFontPath(t), 14)
	if err != nil {
		t.Fatal(err)
	}
	var red, blue int
	var maxAlpha uint32
	for y := range 100 {
		for x := range 160 {
			_, _, _, a := got.At(x, y).RGBA()
			if a > maxAlpha {
				maxAlpha = a
			}
			if a < 50000 {
				continue
			}
			want := source.NRGBAAt(x, y)
			actual := color.NRGBAModel.Convert(got.At(x, y)).(color.NRGBA)
			if math.Abs(float64(actual.R)-float64(want.R)) > 2 || math.Abs(float64(actual.G)-float64(want.G)) > 2 || math.Abs(float64(actual.B)-float64(want.B)) > 2 {
				t.Fatalf("glyph pixel at (%d,%d) = %v, want source color %v", x, y, actual, want)
			}
			if x%2 == 0 {
				red++
			} else {
				blue++
			}
		}
	}
	if red == 0 || blue == 0 {
		t.Fatalf("expected both source colors inside glyphs; got red=%d blue=%d max alpha=%d", red, blue, maxAlpha)
	}
}

func TestRenderPreservesStraightColorWithPartialSourceAlpha(t *testing.T) {
	source := image.NewNRGBA(image.Rect(0, 0, 120, 80))
	want := color.NRGBA{R: 220, G: 90, B: 35, A: 128}
	for y := range 80 {
		for x := range 120 {
			source.SetNRGBA(x, y, want)
		}
	}
	got, err := render(source, []rune("MMMM "), testFontPath(t), 14)
	if err != nil {
		t.Fatal(err)
	}
	for y := range 80 {
		for x := range 120 {
			actual := color.NRGBAModel.Convert(got.At(x, y)).(color.NRGBA)
			if actual.A < 120 {
				continue
			}
			if actual.A > want.A || math.Abs(float64(actual.R)-float64(want.R)) > 3 || math.Abs(float64(actual.G)-float64(want.G)) > 3 || math.Abs(float64(actual.B)-float64(want.B)) > 3 {
				t.Fatalf("partially transparent glyph pixel at (%d,%d) = %v, want source color near %v", x, y, actual, want)
			}
			return
		}
	}
	t.Fatal("no glyph pixel with high coverage")
}

func TestGenerateImageUppercaseMatchesUppercaseText(t *testing.T) {
	source := solidImage(160, 100, color.RGBA{R: 255, G: 80, B: 20, A: 255})
	fontPath := testFontPath(t)
	got, err := generateImage(source, "hello world", NewConfig(WithFontPath(fontPath), WithUppercase(true)))
	if err != nil {
		t.Fatal(err)
	}
	want, err := generateImage(source, "HELLO WORLD", NewConfig(WithFontPath(fontPath)))
	if err != nil {
		t.Fatal(err)
	}
	for y := range 100 {
		for x := range 160 {
			if got.At(x, y) != want.At(x, y) {
				t.Fatalf("uppercase image differs at (%d,%d)", x, y)
			}
		}
	}
}

func TestNormalizeText(t *testing.T) {
	got, err := normalizeText(" hello\n\nworld\tпривет  café ")
	if err != nil {
		t.Fatalf("normalizeText() error = %v", err)
	}

	want := "hello world привет café "
	if string(got) != want {
		t.Fatalf("normalizeText() = %q, want %q", string(got), want)
	}
}

func TestCalculateScaledFontSize(t *testing.T) {
	tests := []struct {
		name     string
		baseSize float64
		width    float64
		want     float64
	}{
		{name: "small", baseSize: 14, width: 720, want: 10.5},
		{name: "default range", baseSize: 14, width: 900, want: 14},
		{name: "1080", baseSize: 14, width: 1080, want: 21},
		{name: "2160", baseSize: 14, width: 2160, want: 28},
		{name: "3600", baseSize: 14, width: 3600, want: 49},
		{name: "4800", baseSize: 14, width: 4800, want: 56},
		{name: "7200", baseSize: 14, width: 7200, want: 63},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := calculateScaledFontSize(tt.baseSize, tt.width); got != tt.want {
				t.Fatalf("calculateScaledFontSize() = %v, want %v", got, tt.want)
			}
		})
	}
}

func solidImage(width, height int, c color.RGBA) image.Image {
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	for y := range height {
		for x := range width {
			img.SetRGBA(x, y, c)
		}
	}
	return img
}

func hasVisiblePixel(img image.Image) bool {
	bounds := img.Bounds()
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			_, _, _, a := img.At(x, y).RGBA()
			if a != 0 {
				return true
			}
		}
	}
	return false
}

func testFontPath(t *testing.T) string {
	t.Helper()

	path := filepath.Join(
		"..",
		"..",
		"..",
		"fonts",
		"NotoSansMono-VariableFont_wdth,wght.ttf",
	)
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("test font missing at %q: %v", path, err)
	}

	return path
}
