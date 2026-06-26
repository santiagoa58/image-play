package wordcloud

import (
	"image"
	"image/color"
	"log/slog"
	"math/rand"
	"testing"
)

const testFontPath = "../../../fonts/NotoSansMono-VariableFont_wdth,wght.ttf"

func TestBuildMaskFindsLightShapeAndBackground(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 10, 10))
	for y := 0; y < 10; y++ {
		for x := 0; x < 10; x++ {
			img.Set(x, y, color.RGBA{0, 0, 0, 255})
		}
	}
	for y := 3; y < 7; y++ {
		for x := 2; x < 8; x++ {
			img.Set(x, y, color.RGBA{240, 240, 240, 255})
		}
	}

	mask, err := buildMask(img, "light", 0.5, 0.1, true)
	if err != nil {
		t.Fatalf("buildMask returned error: %v", err)
	}

	if !mask.playable(2, 3) || !mask.playable(7, 6) {
		t.Fatal("expected light rectangle to be playable")
	}
	if mask.playable(1, 3) || mask.playable(8, 6) {
		t.Fatal("expected dark background to be blocked")
	}
	if mask.minX != 2 || mask.maxX != 7 || mask.minY != 3 || mask.maxY != 6 {
		t.Fatalf("unexpected mask bounds: (%d,%d)-(%d,%d)", mask.minX, mask.minY, mask.maxX, mask.maxY)
	}
}

func TestBuildHierarchyOrdersLargeWordsBeforeDenseFillers(t *testing.T) {
	stats := []wordStat{
		{Word: "ALPHA", Count: 9},
		{Word: "BETA", Count: 4},
	}

	candidates := buildHierarchy(stats, 10, 50, 4, 0.5, rand.New(rand.NewSource(1)))
	if len(candidates) != 6 {
		t.Fatalf("expected 6 candidates, got %d", len(candidates))
	}
	if candidates[0].Word != "ALPHA" || candidates[1].Word != "BETA" {
		t.Fatalf("primary candidates not first: %#v", candidates[:2])
	}
	if candidates[0].Size <= candidates[1].Size {
		t.Fatalf("expected higher-frequency word to be larger: %.2f <= %.2f", candidates[0].Size, candidates[1].Size)
	}
	if candidates[len(candidates)-1].Size >= candidates[2].Size {
		t.Fatalf("expected filler sizes to descend: last %.2f first filler %.2f", candidates[len(candidates)-1].Size, candidates[2].Size)
	}
}

func TestSpatialHashDetectsOnlyOverlappingBoxes(t *testing.T) {
	hash := newSpatialHash(10)
	hash.add(rect{left: 10, top: 10, right: 30, bottom: 30})

	if !hash.hasOverlap(rect{left: 25, top: 25, right: 40, bottom: 40}) {
		t.Fatal("expected overlapping box to collide")
	}
	if hash.hasOverlap(rect{left: 31, top: 31, right: 40, bottom: 40}) {
		t.Fatal("expected separated box not to collide")
	}
}

func TestGlyphOccupancyAllowsOverlappingBoxesWithDisjointPixels(t *testing.T) {
	mask := fullMask(20, 20)
	p := newPacker(mask, image.NewRGBA(image.Rect(0, 0, 20, 20)), testFontPath, testSettings())

	first := placedWord{
		glyph: &glyphBitmap{
			width:  10,
			height: 10,
			pixels: []glyphPixel{
				{x: 0, y: 0, alpha: 255},
			},
			spans: []glyphSpan{{y: 0, x1: 0, x2: 0}},
		},
		x:   5,
		y:   5,
		box: rect{left: 5, top: 5, right: 15, bottom: 15},
	}
	p.stamp(first)
	p.hash.add(first.box)

	disjoint := &glyphBitmap{
		width:  10,
		height: 10,
		pixels: []glyphPixel{
			{x: 9, y: 9, alpha: 255},
		},
		spans: []glyphSpan{{y: 9, x1: 9, x2: 9}},
	}
	if !p.canPlace(disjoint, 5, 5) {
		t.Fatal("expected disjoint visible pixels to place even when bounding boxes overlap")
	}

	overlapping := &glyphBitmap{
		width:  10,
		height: 10,
		pixels: []glyphPixel{
			{x: 0, y: 0, alpha: 255},
		},
		spans: []glyphSpan{{y: 0, x1: 0, x2: 0}},
	}
	if p.canPlace(overlapping, 5, 5) {
		t.Fatal("expected overlapping visible pixels to be rejected")
	}
}

func TestRenderGlyphBitmapProducesTightVisiblePixels(t *testing.T) {
	glyph, err := renderGlyphBitmap(testFontPath, "SKYWALKER", 18, defaultGlyphAlphaThreshold)
	if err != nil {
		t.Fatalf("renderGlyphBitmap returned error: %v", err)
	}
	if glyph.width <= 0 || glyph.height <= 0 || len(glyph.pixels) == 0 {
		t.Fatalf("expected visible glyph pixels, got width=%d height=%d pixels=%d", glyph.width, glyph.height, len(glyph.pixels))
	}

	rotated := rotateGlyph(glyph)
	if rotated.width != glyph.height || rotated.height != glyph.width {
		t.Fatalf("unexpected rotated dimensions: got %dx%d from %dx%d", rotated.width, rotated.height, glyph.width, glyph.height)
	}
	if len(rotated.pixels) != len(glyph.pixels) {
		t.Fatalf("rotation changed visible pixel count: got %d want %d", len(rotated.pixels), len(glyph.pixels))
	}
}

func TestGenerateRendersWordCloudInsideMask(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 180, 120))
	for y := range 120 {
		for x := range 180 {
			img.Set(x, y, color.RGBA{255, 255, 255, 255})
		}
	}
	for y := 20; y < 100; y++ {
		for x := 30; x < 150; x++ {
			img.Set(x, y, color.RGBA{40, 90, 150, 255})
		}
	}

	threshold := 0.8
	result, err := GenerateResult(Config{
		Text:            "alpha beta beta gamma gamma gamma delta epsilon",
		InputImage:      img,
		FontPath:        testFontPath,
		MinFontSize:     7,
		MaxFontSize:     22,
		MaskType:        "dark",
		MaskThreshold:   &threshold,
		FillerWordCount: 80,
		Seed:            7,
	})
	if err != nil {
		t.Fatalf("GenerateResult returned error: %v", err)
	}
	out := result.Image

	bounds := out.Bounds()
	if bounds.Dx() != 180 || bounds.Dy() != 120 {
		t.Fatalf("unexpected output size: %dx%d", bounds.Dx(), bounds.Dy())
	}

	var nonBackground int
	bg := out.At(0, 0)
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			if out.At(x, y) != bg {
				nonBackground++
			}
		}
	}
	if nonBackground == 0 {
		t.Fatal("expected generated image to contain rendered words")
	}
	if result.Stats.PlacedWords == 0 || result.Stats.AttemptedWords == 0 {
		t.Fatalf("expected placement stats, got %#v", result.Stats)
	}
	if result.Stats.MaskCoverage <= 0 || result.Stats.OccupiedCoverage <= 0 {
		t.Fatalf("expected positive coverage stats, got %#v", result.Stats)
	}
}

func TestNormalizeConfigAllowsExplicitZeroThreshold(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 4, 4))
	threshold := 0.0
	cfg, err := normalizeConfig(Config{
		Text:          "alpha beta",
		InputImage:    img,
		FontPath:      testFontPath,
		MaskThreshold: &threshold,
	})
	if err != nil {
		t.Fatalf("normalizeConfig returned error: %v", err)
	}
	if cfg.maskThreshold != 0 {
		t.Fatalf("expected explicit zero threshold to be preserved, got %.2f", cfg.maskThreshold)
	}
}

func TestGenerateResultRejectsImagesOverMaxPixels(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 20, 20))
	_, err := GenerateResult(Config{
		Text:       "alpha beta",
		InputImage: img,
		FontPath:   testFontPath,
		MaxPixels:  100,
	})
	if err == nil {
		t.Fatal("expected max pixels error")
	}
}

func TestWordPaddingRejectsNearbyGlyphPixels(t *testing.T) {
	mask := fullMask(20, 20)
	cfg := testSettings()
	cfg.wordPadding = 1
	p := newPacker(mask, image.NewRGBA(image.Rect(0, 0, 20, 20)), testFontPath, cfg)

	first := placedWord{
		glyph: &glyphBitmap{
			width:  3,
			height: 3,
			pixels: []glyphPixel{
				{x: 1, y: 1, alpha: 255},
			},
			spans: []glyphSpan{{y: 1, x1: 1, x2: 1}},
		},
		x:   5,
		y:   5,
		box: rect{left: 5, top: 5, right: 8, bottom: 8},
	}
	p.stamp(first)
	p.hash.add(first.box)

	nearby := &glyphBitmap{
		width:  3,
		height: 3,
		pixels: []glyphPixel{
			{x: 1, y: 1, alpha: 255},
		},
		spans: []glyphSpan{{y: 1, x1: 1, x2: 1}},
	}
	if p.canPlace(nearby, 6, 5) {
		t.Fatal("expected word padding to reject adjacent glyph")
	}
}

func TestPaletteColorModesAreSeededAndDeterministic(t *testing.T) {
	mask := fullMask(10, 10)
	palette := []color.Color{
		color.RGBA{255, 0, 0, 255},
		color.RGBA{0, 255, 0, 255},
	}
	cfg := testSettings()
	cfg.colorMode = ColorModeRandomPalette
	cfg.palette = palette
	cfg.seed = 99

	p1 := newPacker(mask, image.NewRGBA(image.Rect(0, 0, 10, 10)), testFontPath, cfg)
	p2 := newPacker(mask, image.NewRGBA(image.Rect(0, 0, 10, 10)), testFontPath, cfg)

	for i := range 6 {
		if p1.chooseColor(5, 5, i, 10) != p2.chooseColor(5, 5, i, 10) {
			t.Fatal("expected seeded random palette mode to be deterministic")
		}
	}
}

func TestMaxPlacementAttemptsReportsRejectedWords(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 80, 60))
	for y := range 60 {
		for x := range 80 {
			img.Set(x, y, color.RGBA{0, 0, 0, 255})
		}
	}
	threshold := 0.5
	result, err := GenerateResult(Config{
		Text:                 "extraordinarilylongword anotherlongword",
		InputImage:           img,
		FontPath:             testFontPath,
		MinFontSize:          18,
		MaxFontSize:          28,
		MaskType:             "dark",
		MaskThreshold:        &threshold,
		FillerWordCount:      1,
		MaxPlacementAttempts: 1,
	})
	if result.Stats.MaxAttemptRejects == 0 && result.Stats.RejectedWords == 0 {
		t.Fatalf("expected bounded placement attempts to be reflected in stats, err=%v stats=%#v", err, result.Stats)
	}
}

func TestFinalFillPassesIncreaseAttemptedWords(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 120, 80))
	for y := range 80 {
		for x := range 120 {
			img.Set(x, y, color.RGBA{0, 0, 0, 255})
		}
	}
	threshold := 0.5
	result, err := GenerateResult(Config{
		Text:                 "alpha beta gamma delta",
		InputImage:           img,
		FontPath:             testFontPath,
		MinFontSize:          6,
		MaxFontSize:          18,
		MaskType:             "dark",
		MaskThreshold:        &threshold,
		FillerWordCount:      0,
		MaxPlacementAttempts: 300,
		FinalFillPasses:      2,
		FinalFillFontSize:    6,
	})
	if err != nil {
		t.Fatalf("GenerateResult returned error: %v", err)
	}
	if result.Stats.AttemptedWords <= 4 {
		t.Fatalf("expected final fill to add attempts beyond primary words, got %#v", result.Stats)
	}
}

func fullMask(width, height int) *shapeMask {
	mask := &shapeMask{
		width:      width,
		height:     height,
		bits:       make([]bool, width*height),
		luminance:  make([]float64, width*height),
		playablePx: width * height,
		minX:       0,
		minY:       0,
		maxX:       width - 1,
		maxY:       height - 1,
		centerX:    float64(width) / 2,
		centerY:    float64(height) / 2,
		bgColor:    color.White,
	}
	for i := range mask.bits {
		mask.bits[i] = true
	}
	return mask
}

func testSettings() settings {
	return settings{
		logger:               slogDiscard(),
		minFontSize:          4,
		maxFontSize:          24,
		sizeExponent:         defaultSizeExponent,
		maskThreshold:        defaultMaskThreshold,
		alphaThreshold:       defaultAlphaThreshold,
		glyphAlphaThreshold:  defaultGlyphAlphaThreshold,
		maskType:             "dark",
		fillerWordCount:      10,
		maxPlacementAttempts: 100,
		wordPadding:          0,
		maxPixels:            defaultMaxPixels,
		colorMode:            ColorModePalette,
		palette:              defaultPalette,
		seed:                 1,
		heroWordCount:        2,
		finalFillFontSize:    4,
		minWordLength:        2,
	}
}

func slogDiscard() *slog.Logger {
	return slog.New(slog.NewTextHandler(testingWriter{}, nil))
}

type testingWriter struct{}

func (testingWriter) Write(p []byte) (int, error) {
	return len(p), nil
}
