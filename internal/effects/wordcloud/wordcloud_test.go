package wordcloud

import (
	"image"
	"image/color"
	"log/slog"
	"math/rand"
	"testing"

	"github.com/disintegration/imaging"
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

func TestBuildMaskContrastKeepsDarkSaturatedPixelsOnBlack(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 12, 10))
	for y := 0; y < 10; y++ {
		for x := 0; x < 12; x++ {
			img.Set(x, y, color.RGBA{1, 1, 1, 255})
		}
	}
	for y := 2; y < 8; y++ {
		for x := 2; x < 6; x++ {
			img.Set(x, y, color.RGBA{88, 8, 10, 255})
		}
	}
	for y := 2; y < 8; y++ {
		for x := 7; x < 10; x++ {
			img.Set(x, y, color.RGBA{5, 5, 5, 255})
		}
	}

	threshold := 0.12
	mask, err := buildMask(img, MaskTypeContrast, threshold, 0.1, false)
	if err != nil {
		t.Fatalf("buildMask returned error: %v", err)
	}

	if !mask.playable(3, 4) {
		t.Fatal("expected saturated dark red pixels to be playable against black")
	}
	if mask.playable(8, 4) {
		t.Fatal("expected near-black pixels to remain negative space")
	}
	if mask.playable(0, 0) {
		t.Fatal("expected background pixels to be blocked")
	}
}

func TestBuildHierarchyOrdersLargeWordsBeforeDenseFillers(t *testing.T) {
	stats := []wordStat{
		{Word: "ALPHA", Count: 9},
		{Word: "BETA", Count: 4},
	}

	candidates := buildHierarchy(stats, 10, 50, 4, 0.5, defaultFillerMaxScale, rand.New(rand.NewSource(1)))
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

func TestGenerateResultWithRepositoryImage(t *testing.T) {
	img, err := imaging.Open("../../../testdata/images/couple_tour.jpg")
	if err != nil {
		t.Fatalf("open repository test image: %v", err)
	}
	threshold := 0.55
	result, err := GenerateResult(Config{
		Text:                 "travel couple tour memory photo road beach mountain city journey",
		InputImage:           img,
		TargetWidth:          160,
		FontPath:             testFontPath,
		MinFontSize:          5,
		MaxFontSize:          18,
		MaskType:             "dark",
		MaskThreshold:        &threshold,
		FillerWordCount:      30,
		MaxPlacementAttempts: 400,
		ColorMode:            ColorModeLuminancePalette,
		Seed:                 11,
	})
	if err != nil {
		t.Fatalf("GenerateResult returned error: %v", err)
	}
	if result.Image.Bounds().Dx() != 160 {
		t.Fatalf("expected resized output width 160, got %d", result.Image.Bounds().Dx())
	}
	if result.Stats.PlacedWords == 0 || result.Stats.PlacementChecks == 0 {
		t.Fatalf("expected real-image placement stats, got %#v", result.Stats)
	}
}

func TestGenerateResultWithForegroundOnWhiteImageHasDenseCoverage(t *testing.T) {
	img, err := imaging.Open("../../../testdata/images/gen-img-couple.png")
	if err != nil {
		t.Fatalf("open repository foreground image: %v", err)
	}
	threshold := 0.965
	result, err := GenerateResult(Config{
		Text: "love wedding couple bride groom forever vows embrace kiss family joy heart celebration " +
			"portrait devotion romance ceremony memory elegant together promise beloved",
		InputImage:                 img,
		TargetWidth:                256,
		FontPath:                   testFontPath,
		PackingProfile:             PackingProfileTonalDetail,
		QualityPreset:              QualityPresetDense,
		MinFontSize:                3,
		MaxFontSize:                24,
		MaskType:                   "dark",
		MaskThreshold:              &threshold,
		FillerWordCount:            1400,
		MaxPlacementAttempts:       3500,
		MaxFillerPlacementAttempts: 650,
		WordPadding:                0,
		FinalFillPasses:            6,
		FinalFillFontSize:          3,
		FinalFillAnchorSamples:     64,
		FillerMaxScale:             0.12,
		ColorMode:                  ColorModeSource,
		Seed:                       42,
		Background:                 color.White,
	})
	if err != nil {
		t.Fatalf("GenerateResult returned error: %v", err)
	}
	if result.Stats.MaskCoverage < 0.45 {
		t.Fatalf("expected foreground mask to cover the subject, got stats %#v", result.Stats)
	}
	if result.Stats.OccupiedCoverage < 0.45 {
		t.Fatalf("expected dense word occupancy, got stats %#v", result.Stats)
	}
}

func TestForegroundBackgroundProfilePacksForegroundMoreDensely(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 180, 140))
	mask := image.NewGray(image.Rect(0, 0, 180, 140))
	for y := range 140 {
		for x := range 180 {
			img.Set(x, y, color.RGBA{190, 210, 225, 255})
		}
	}
	for y := 35; y < 130; y++ {
		for x := 55; x < 130; x++ {
			img.Set(x, y, color.RGBA{45, 55, 65, 255})
			mask.SetGray(x, y, color.Gray{Y: 255})
		}
	}

	result, err := GenerateResult(Config{
		Text: "portrait couple travel memory together foreground background depth focus clarity " +
			"building sky jacket shirt people photo word cloud detail",
		InputImage:                 img,
		ForegroundMask:             mask,
		FontPath:                   testFontPath,
		PackingProfile:             PackingProfileForegroundBackground,
		QualityPreset:              QualityPresetBalanced,
		MinFontSize:                3,
		MaxFontSize:                22,
		FillerWordCount:            700,
		MaxPlacementAttempts:       2500,
		MaxFillerPlacementAttempts: 450,
		FinalFillPasses:            3,
		FinalFillFontSize:          3,
		ColorMode:                  ColorModeSource,
		Seed:                       21,
		Background:                 color.White,
	})
	if err != nil {
		t.Fatalf("GenerateResult returned error: %v", err)
	}
	if result.Stats.ForegroundMaskCoverage <= 0 {
		t.Fatalf("expected foreground mask coverage, got %#v", result.Stats)
	}
	if result.Stats.ForegroundOccupiedCoverage <= result.Stats.BackgroundOccupiedCoverage {
		t.Fatalf("expected foreground to be packed more densely than background, got %#v", result.Stats)
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

func TestPackingProfilesApplyTechnicalDefaults(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 10, 10))

	binary, err := normalizeConfig(Config{
		Text:           "alpha beta",
		InputImage:     img,
		FontPath:       testFontPath,
		PackingProfile: PackingProfileBinarySilhouette,
	})
	if err != nil {
		t.Fatalf("binary silhouette profile returned error: %v", err)
	}
	if binary.colorMode != ColorModePalette || binary.finalFillPasses != 1 || binary.fillerMaxScale != 0.22 {
		t.Fatalf("unexpected binary silhouette defaults: %#v", binary)
	}

	tonal, err := normalizeConfig(Config{
		Text:           "alpha beta",
		InputImage:     img,
		FontPath:       testFontPath,
		PackingProfile: PackingProfileTonalDetail,
	})
	if err != nil {
		t.Fatalf("tonal detail profile returned error: %v", err)
	}
	if tonal.colorMode != ColorModeLuminancePalette || tonal.finalFillPasses != 3 || tonal.wordPadding != 1 || tonal.fillerMaxScale != 0.16 {
		t.Fatalf("unexpected tonal detail defaults: %#v", tonal)
	}
}

func TestPackingProfilesRejectUnknownName(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 10, 10))
	_, err := normalizeConfig(Config{
		Text:           "alpha beta",
		InputImage:     img,
		FontPath:       testFontPath,
		PackingProfile: "example-specific",
	})
	if err == nil {
		t.Fatal("expected unknown packing profile to return an error")
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
	glyph := singlePixelGlyph(1, 1)

	for i := range 6 {
		if p1.chooseColor(glyph, 5, 5, i, 10) != p2.chooseColor(glyph, 5, 5, i, 10) {
			t.Fatal("expected seeded random palette mode to be deterministic")
		}
	}
}

func TestPaddingBroadPhaseRejectsNearNonOverlappingBoxes(t *testing.T) {
	mask := fullMask(30, 20)
	cfg := testSettings()
	cfg.wordPadding = 3
	p := newPacker(mask, image.NewRGBA(image.Rect(0, 0, 30, 20)), testFontPath, cfg)

	first := placedWord{
		glyph: singlePixelGlyph(1, 1),
		x:     10,
		y:     10,
		box:   rect{left: 10, top: 10, right: 11, bottom: 11},
	}
	p.stamp(first)
	p.hash.add(first.box.expand(p.wordPadding))

	nearby := singlePixelGlyph(1, 1)
	if p.canPlace(nearby, 13, 10) {
		t.Fatal("expected padding-expanded broad phase to trigger occupancy rejection")
	}
}

func TestNormalizeConfigNormalizesStopWords(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 10, 10))
	cfg, err := normalizeConfig(Config{
		Text:       "The alpha",
		InputImage: img,
		FontPath:   testFontPath,
		StopWords:  map[string]struct{}{"the": {}},
	})
	if err != nil {
		t.Fatalf("normalizeConfig returned error: %v", err)
	}
	stats, err := parseWordStats("the alpha", cfg)
	if err != nil {
		t.Fatalf("parseWordStats returned error: %v", err)
	}
	if len(stats) != 1 || stats[0].Word != "ALPHA" {
		t.Fatalf("expected lowercase stop word to be normalized, got %#v", stats)
	}
}

func TestLuminancePaletteCanInvertMapping(t *testing.T) {
	mask := fullMask(5, 5)
	for i := range mask.luminance {
		mask.luminance[i] = 0.9
	}
	palette := []color.Color{
		color.RGBA{1, 0, 0, 255},
		color.RGBA{2, 0, 0, 255},
	}
	cfg := testSettings()
	cfg.colorMode = ColorModeLuminancePalette
	cfg.palette = palette
	cfg.invertLuminancePalette = true
	p := newPacker(mask, image.NewRGBA(image.Rect(0, 0, 5, 5)), testFontPath, cfg)

	if p.chooseColor(singlePixelGlyph(1, 1), 2, 2, 0, 4) != palette[0] {
		t.Fatal("expected inverted luminance palette to map bright source to first palette entry")
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
	if result.Stats.WordsRejectedAfterAttempts == 0 && result.Stats.RejectedWords == 0 {
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

func singlePixelGlyph(width, height int) *glyphBitmap {
	x := width / 2
	y := height / 2
	return &glyphBitmap{
		width:  width,
		height: height,
		pixels: []glyphPixel{
			{x: x, y: y, alpha: 255},
		},
		spans: []glyphSpan{{y: y, x1: x, x2: x}},
	}
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
