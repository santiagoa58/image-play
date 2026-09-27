package textmosaic

import (
	"image"
	"image/color"
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

func TestSampleRGBAHandlesNonZeroBounds(t *testing.T) {
	img := image.NewRGBA(image.Rect(10, 20, 11, 21))
	img.Set(10, 20, color.RGBA{R: 255, G: 128, B: 64, A: 255})

	r, g, b, a := sampleRGBA(img, 0, 0)
	if r <= 0.99 || g <= 0.49 || g >= 0.51 || b <= 0.24 || b >= 0.26 || a <= 0.99 {
		t.Fatalf("sampleRGBA() = (%v,%v,%v,%v), want approximately (1,.5,.25,1)", r, g, b, a)
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
