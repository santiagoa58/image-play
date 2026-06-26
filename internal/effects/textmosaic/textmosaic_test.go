package textmosaic

import (
	"image"
	"image/color"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGenerate(t *testing.T) {
	fontPath := testFontPath(t)

	tests := []struct {
		name          string
		config        Config
		wantWidth     int
		wantHeight    int
		wantNonEmpty  bool
		wantErrSubstr string
	}{
		{
			name: "generates mosaic with original dimensions",
			config: Config{
				Logger:       testLogger(),
				Text:         "hello world",
				InputImage:   solidImage(120, 80, color.RGBA{R: 255, G: 0, B: 0, A: 255}),
				MonoFontPath: fontPath,
			},
			wantWidth:    120,
			wantHeight:   80,
			wantNonEmpty: true,
		},
		{
			name: "resizes to target width",
			config: Config{
				Logger:       testLogger(),
				Text:         "hello world",
				InputImage:   solidImage(200, 100, color.RGBA{R: 0, G: 255, B: 0, A: 255}),
				MonoFontPath: fontPath,
				TargetWidth:  100,
			},
			wantWidth:    100,
			wantHeight:   50,
			wantNonEmpty: true,
		},
		{
			name: "supports basic unicode runes",
			config: Config{
				Logger:       testLogger(),
				Text:         "hello привет café",
				InputImage:   solidImage(160, 90, color.RGBA{R: 0, G: 0, B: 255, A: 255}),
				MonoFontPath: fontPath,
			},
			wantWidth:    160,
			wantHeight:   90,
			wantNonEmpty: true,
		},
		{
			name: "transparent source stays transparent",
			config: Config{
				Logger:       testLogger(),
				Text:         "hello world",
				InputImage:   solidImage(120, 80, color.RGBA{R: 255, G: 0, B: 0, A: 0}),
				MonoFontPath: fontPath,
			},
			wantWidth:    120,
			wantHeight:   80,
			wantNonEmpty: false,
		},
		{
			name: "errors when input image is missing",
			config: Config{
				Logger:       testLogger(),
				Text:         "hello world",
				MonoFontPath: fontPath,
			},
			wantErrSubstr: "input image is required",
		},
		{
			name: "errors when font path is missing",
			config: Config{
				Logger:     testLogger(),
				Text:       "hello world",
				InputImage: solidImage(120, 80, color.RGBA{A: 255}),
			},
			wantErrSubstr: "monospace font path is required",
		},
		{
			name: "errors when text is empty",
			config: Config{
				Logger:       testLogger(),
				Text:         " \n\t ",
				InputImage:   solidImage(120, 80, color.RGBA{A: 255}),
				MonoFontPath: fontPath,
			},
			wantErrSubstr: "text cannot be empty",
		},
		{
			name: "errors when target width is negative",
			config: Config{
				Logger:       testLogger(),
				Text:         "hello world",
				InputImage:   solidImage(120, 80, color.RGBA{A: 255}),
				MonoFontPath: fontPath,
				TargetWidth:  -1,
			},
			wantErrSubstr: "target width cannot be negative",
		},
		{
			name: "errors when base font size is negative",
			config: Config{
				Logger:       testLogger(),
				Text:         "hello world",
				InputImage:   solidImage(120, 80, color.RGBA{A: 255}),
				MonoFontPath: fontPath,
				BaseFontSize: -1,
			},
			wantErrSubstr: "base font size cannot be negative",
		},
		{
			name: "errors when contrast is out of range",
			config: Config{
				Logger:          testLogger(),
				Text:            "hello world",
				InputImage:      solidImage(120, 80, color.RGBA{A: 255}),
				MonoFontPath:    fontPath,
				ContrastPercent: 101,
			},
			wantErrSubstr: "contrast percent must be between",
		},
		{
			name: "errors when text weight is out of range",
			config: Config{
				Logger:       testLogger(),
				Text:         "hello world",
				InputImage:   solidImage(120, 80, color.RGBA{A: 255}),
				MonoFontPath: fontPath,
				TextWeight:   5,
			},
			wantErrSubstr: "text weight cannot be greater than",
		},
		{
			name: "errors when font is too large",
			config: Config{
				Logger:       testLogger(),
				Text:         "hello world",
				InputImage:   solidImage(20, 20, color.RGBA{A: 255}),
				MonoFontPath: fontPath,
				BaseFontSize: 100,
			},
			wantErrSubstr: "font size is too large",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Generate(tt.config)

			if tt.wantErrSubstr != "" {
				if err == nil {
					t.Fatalf("Generate() expected error containing %q, got nil", tt.wantErrSubstr)
				}
				if !strings.Contains(err.Error(), tt.wantErrSubstr) {
					t.Fatalf("Generate() error = %q; want substring %q", err.Error(), tt.wantErrSubstr)
				}
				return
			}

			if err != nil {
				t.Fatalf("Generate() returned error: %v", err)
			}

			if got == nil {
				t.Fatal("Generate() returned nil image")
			}

			if got.Bounds().Dx() != tt.wantWidth {
				t.Errorf("width = %d; want %d", got.Bounds().Dx(), tt.wantWidth)
			}

			if got.Bounds().Dy() != tt.wantHeight {
				t.Errorf("height = %d; want %d", got.Bounds().Dy(), tt.wantHeight)
			}

			if hasVisiblePixel(got) != tt.wantNonEmpty {
				t.Errorf("hasVisiblePixel = %v; want %v", hasVisiblePixel(got), tt.wantNonEmpty)
			}
		})
	}
}

func TestGenerateHandlesNonZeroImageBounds(t *testing.T) {
	fontPath := testFontPath(t)

	src := image.NewRGBA(image.Rect(50, 75, 170, 155))
	for y := src.Bounds().Min.Y; y < src.Bounds().Max.Y; y++ {
		for x := src.Bounds().Min.X; x < src.Bounds().Max.X; x++ {
			src.Set(x, y, color.RGBA{R: 255, A: 255})
		}
	}

	got, err := Generate(Config{
		Logger:       testLogger(),
		Text:         "hello world",
		InputImage:   src,
		MonoFontPath: fontPath,
	})
	if err != nil {
		t.Fatalf("Generate() returned error: %v", err)
	}

	if got.Bounds().Min != (image.Point{}) {
		t.Errorf("output min bounds = %v; want %v", got.Bounds().Min, image.Point{})
	}

	if got.Bounds().Dx() != src.Bounds().Dx() {
		t.Errorf("output width = %d; want %d", got.Bounds().Dx(), src.Bounds().Dx())
	}

	if got.Bounds().Dy() != src.Bounds().Dy() {
		t.Errorf("output height = %d; want %d", got.Bounds().Dy(), src.Bounds().Dy())
	}

	if !hasVisiblePixel(got) {
		t.Fatal("expected output to contain visible text pixels")
	}
}

func TestNormalizeText(t *testing.T) {
	got, err := normalizeText(" hello\n\nworld\tпривет  café ")
	if err != nil {
		t.Fatalf("normalizeText() returned error: %v", err)
	}

	want := []rune("hello world привет café ")
	if string(got) != string(want) {
		t.Errorf("normalizeText() = %q; want %q", string(got), string(want))
	}
}

func TestNormalizeTextEmpty(t *testing.T) {
	_, err := normalizeText(" \n\t ")
	if err == nil {
		t.Fatal("normalizeText() expected error, got nil")
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
			got := calculateScaledFontSize(tt.baseSize, tt.width)
			if got != tt.want {
				t.Errorf("calculateScaledFontSize(%v, %v) = %v; want %v", tt.baseSize, tt.width, got, tt.want)
			}
		})
	}
}

func TestGetImgRGBAHandlesNonZeroBounds(t *testing.T) {
	img := image.NewRGBA(image.Rect(10, 20, 11, 21))
	img.Set(10, 20, color.RGBA{R: 255, G: 128, B: 64, A: 255})

	r, g, b, a := getImgRGBA(img, 0, 0)

	if r <= 0.99 {
		t.Errorf("r = %v; want close to 1", r)
	}
	if g <= 0.49 || g >= 0.51 {
		t.Errorf("g = %v; want close to 0.5", g)
	}
	if b <= 0.24 || b >= 0.26 {
		t.Errorf("b = %v; want close to 0.25", b)
	}
	if a <= 0.99 {
		t.Errorf("a = %v; want close to 1", a)
	}
}

func solidImage(width, height int, c color.RGBA) image.Image {
	img := image.NewRGBA(image.Rect(0, 0, width, height))

	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
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

	path := filepath.Join("..", "..", "..", "fonts", "NotoSansMono-VariableFont_wdth,wght.ttf")

	if _, err := os.Stat(path); err != nil {
		t.Fatalf("test font missing at %q: %v", path, err)
	}

	return path
}

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelError,
	}))
}
