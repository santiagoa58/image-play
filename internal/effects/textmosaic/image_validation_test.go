package textmosaic

import (
	"image"
	"image/color"
	"strings"
	"testing"
)

func TestGenerateImageRejectsInvalidPaletteBeforeProcessing(t *testing.T) {
	// CVE-2023-36308: imaging's scanner indexes the palette without bounds
	// checks. Its worker goroutines can panic before rendering even begins.
	for _, test := range []struct {
		name string
		cfg  Config
	}{
		{name: "grayscale", cfg: NewConfig()},
		{name: "resize", cfg: NewConfig(WithTargetWidth(60))},
		{name: "contrast", cfg: NewConfig(WithContrastPercent(15))},
	} {
		t.Run(test.name, func(t *testing.T) {
			source := image.NewPaletted(image.Rect(0, 0, 120, 80), color.Palette{color.Black})
			source.Pix[len(source.Pix)-1] = 1

			_, err := generateImage(source, "hello", test.cfg)
			if err == nil || !strings.Contains(err.Error(), "palette index 1 outside palette of 1 colors") {
				t.Fatalf("generateImage() error = %v, want invalid palette index", err)
			}
		})
	}
}

func TestGenerateImageAcceptsValidPalettedSubimage(t *testing.T) {
	source := image.NewPaletted(image.Rect(10, 20, 170, 120), color.Palette{color.White})
	// An invalid pixel outside the subimage must not reject its valid contents.
	source.Pix[0] = 1
	subimage := source.SubImage(image.Rect(30, 40, 150, 100))

	got, err := generateImage(subimage, "hello", NewConfig(WithFontPath(testFontPath(t))))
	if err != nil {
		t.Fatal(err)
	}
	if got.Bounds() != image.Rect(0, 0, 120, 60) || !hasVisiblePixel(got) {
		t.Fatalf("generated paletted subimage bounds = %v, want visible 120x60 image", got.Bounds())
	}
}
