package main

import (
	"errors"
	"flag"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"log/slog"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/disintegration/imaging"
	"github.com/santiagoa58/image-play/internal/effects/wordcloud"
	"github.com/santiagoa58/image-play/internal/imageutil"
)

type wordCloudOptions struct {
	packingProfile             string
	qualityPreset              string
	maskType                   string
	maskThreshold              float64
	foregroundThreshold        float64
	foregroundMaskPath         string
	saveForegroundMaskPath     string
	foregroundRect             string
	grabCutIterations          int
	includeProbableForeground  bool
	alphaThreshold             float64
	glyphAlphaThreshold        float64
	minFontSize                int
	maxFontSize                int
	sizeExponent               float64
	fillerWordCount            int
	densityAlias               int
	wordPadding                int
	maxPlacementAttempts       int
	maxHeroPlacementAttempts   int
	maxFillerPlacementAttempts int
	heroWordCount              int
	finalFillPasses            int
	finalFillFontSize          int
	fillerMaxScale             float64
	detailPlacementBias        float64
	sourceColorMinContrast     float64
	colorMode                  string
	palette                    string
	derivePaletteFromSource    bool
	background                 string
	inferBackground            bool
	seed                       int64
	minWordLength              int
	maxWordLength              int
	stopWords                  string
	maxPixels                  int
	workWidth                  int
}

func registerWordCloudFlags(fs *flag.FlagSet, opts *wordCloudOptions) {
	fs.StringVar(&opts.packingProfile, "packing-profile", "", `Word cloud: packing profile: "", "binary-silhouette", "tonal-detail", or "foreground-background"`)
	fs.StringVar(&opts.qualityPreset, "quality", "", `Word cloud: quality preset: "", "fast", "balanced", "dense", or "poster". Empty = auto`)
	fs.StringVar(&opts.maskType, "mask-type", "", `Word cloud: mask type: "", "dark", "light", "contrast", or "all". Empty = profile/default`)
	fs.Float64Var(&opts.maskThreshold, "mask-threshold", defaultOptionalFloatFlag, "Word cloud: mask luminance threshold from 0..1. Negative = default")
	fs.Float64Var(&opts.foregroundThreshold, "foreground-threshold", defaultOptionalFloatFlag, "Word cloud: foreground mask threshold from 0..1. Negative = default")
	fs.StringVar(&opts.foregroundMaskPath, "foreground-mask", "", "Word cloud: optional foreground mask path. White = foreground, black = background")
	fs.StringVar(&opts.saveForegroundMaskPath, "save-foreground-mask", "", "Word cloud: optional path to save the OpenCV foreground mask")
	fs.StringVar(&opts.foregroundRect, "foreground-rect", "", "Word cloud: OpenCV GrabCut rectangle as x,y,width,height. Empty = default central foreground")
	fs.IntVar(&opts.grabCutIterations, "grabcut-iterations", 5, "Word cloud: OpenCV GrabCut iterations for automatic foreground masks")
	fs.BoolVar(&opts.includeProbableForeground, "include-probable-foreground", true, "Word cloud: include OpenCV probable-foreground pixels in the mask")
	fs.Float64Var(&opts.alphaThreshold, "alpha-threshold", defaultOptionalFloatFlag, "Word cloud: source alpha threshold from 0..1. Negative = default")
	fs.Float64Var(&opts.glyphAlphaThreshold, "glyph-alpha-threshold", defaultOptionalFloatFlag, "Word cloud: rendered glyph alpha threshold from 0..1. Negative = default")
	fs.IntVar(&opts.minFontSize, "min-font-size", 0, "Word cloud: minimum font size. 0 = default")
	fs.IntVar(&opts.maxFontSize, "max-font-size", 0, "Word cloud: maximum font size. 0 = default")
	fs.Float64Var(&opts.sizeExponent, "size-exponent", 0, "Word cloud: frequency-to-font-size exponent. 0 = default")
	fs.IntVar(&opts.fillerWordCount, "filler-word-count", 0, "Word cloud: number of extra filler word candidates. 0 = quality preset/default")
	fs.IntVar(&opts.densityAlias, "density", 0, "Word cloud: alias for -filler-word-count")
	fs.IntVar(&opts.wordPadding, "word-padding", defaultOptionalIntegerFlag, "Word cloud: padding in pixels around glyphs. Negative = profile/default")
	fs.IntVar(&opts.maxPlacementAttempts, "max-placement-attempts", 0, "Word cloud: max placement checks per word. 0 = quality preset/default")
	fs.IntVar(&opts.maxHeroPlacementAttempts, "max-hero-placement-attempts", 0, "Word cloud: max placement checks for hero words. 0 = default")
	fs.IntVar(&opts.maxFillerPlacementAttempts, "max-filler-placement-attempts", 0, "Word cloud: max placement checks for filler words. 0 = default")
	fs.IntVar(&opts.heroWordCount, "hero-word-count", 0, "Word cloud: number of largest words to treat as hero words. 0 = default")
	fs.IntVar(&opts.finalFillPasses, "final-fill-passes", defaultOptionalIntegerFlag, "Word cloud: final tiny-word fill passes. Negative = preset/default")
	fs.IntVar(&opts.finalFillFontSize, "final-fill-font-size", 0, "Word cloud: final fill font size. 0 = min font size")
	fs.Float64Var(&opts.fillerMaxScale, "filler-max-scale", 0, "Word cloud: max filler size as fraction of max font size. 0 = default")
	fs.Float64Var(&opts.detailPlacementBias, "detail-placement-bias", 0, "Word cloud: prefer source-detail regions during tonal final fill. 0 = profile/default")
	fs.Float64Var(&opts.sourceColorMinContrast, "source-color-min-contrast", 0, "Word cloud: minimum contrast for source-sampled colors. 0 = profile/default")
	fs.StringVar(&opts.colorMode, "color-mode", "", `Word cloud: color mode: "", "source", "palette", "luminance-palette", "random-palette", or "sequential-palette". Empty = auto`)
	fs.StringVar(&opts.palette, "palette", "", "Word cloud: comma-separated hex colors, e.g. '#111111,#777777,#eeeeee'")
	fs.StringVar(&opts.background, "background", "#ffffff", "Word cloud: canvas background hex color. Empty = effect default")
	fs.BoolVar(&opts.inferBackground, "infer-background", false, "Word cloud: infer background from non-playable pixels")
	fs.Int64Var(&opts.seed, "seed", 1, "Word cloud: deterministic random seed")
	fs.IntVar(&opts.minWordLength, "min-word-length", 0, "Word cloud: minimum parsed word length. 0 = default")
	fs.IntVar(&opts.maxWordLength, "max-word-length", 0, "Word cloud: maximum parsed word length. 0 = no limit")
	fs.StringVar(&opts.stopWords, "stop-words", "", "Word cloud: comma-separated words to exclude")
	fs.IntVar(&opts.maxPixels, "max-pixels", 0, "Word cloud: max pixels for direct internal packing. 0 = 4K-safe default")
	fs.IntVar(&opts.workWidth, "work-width", 0, "Word cloud: internal packing width. 0 = auto; lower is faster, higher is sharper")
}

func generateWordCloud(logger *slog.Logger, img image.Image, text string, fontPath string, targetWidth int, opts wordCloudOptions) (wordcloud.Result, error) {
	opts = resolveWordCloudAutoOptions(img, opts)
	renderWidth, err := resolveWordCloudWorkWidth(targetWidth, opts.workWidth)
	if err != nil {
		return wordcloud.Result{}, err
	}
	maskThreshold, err := optionalUnitFloat(opts.maskThreshold, "mask threshold")
	if err != nil {
		return wordcloud.Result{}, err
	}
	foregroundThreshold, err := optionalUnitFloat(opts.foregroundThreshold, "foreground threshold")
	if err != nil {
		return wordcloud.Result{}, err
	}
	alphaThreshold, err := optionalUnitFloat(opts.alphaThreshold, "alpha threshold")
	if err != nil {
		return wordcloud.Result{}, err
	}
	glyphAlphaThreshold, err := optionalUnitFloat(opts.glyphAlphaThreshold, "glyph alpha threshold")
	if err != nil {
		return wordcloud.Result{}, err
	}
	palette, err := parsePalette(opts.palette)
	if err != nil {
		return wordcloud.Result{}, err
	}
	if len(palette) == 0 && opts.derivePaletteFromSource {
		palette = deriveSourcePalette(img, 6)
	}
	background, err := optionalColor(opts.background)
	if err != nil {
		return wordcloud.Result{}, err
	}
	fillerWordCount, err := resolveFillerWordCount(opts.fillerWordCount, opts.densityAlias)
	if err != nil {
		return wordcloud.Result{}, err
	}
	stopWords := parseStopWords(opts.stopWords)
	foregroundMask, err := resolveForegroundMask(img, opts)
	if err != nil {
		return wordcloud.Result{}, err
	}

	conf := wordcloud.Config{
		Logger:                     logger,
		Text:                       text,
		InputImage:                 img,
		TargetWidth:                renderWidth,
		FontPath:                   fontPath,
		ForegroundMask:             foregroundMask,
		PackingProfile:             opts.packingProfile,
		QualityPreset:              opts.qualityPreset,
		MaskType:                   opts.maskType,
		MaskThreshold:              maskThreshold,
		ForegroundThreshold:        foregroundThreshold,
		AlphaThreshold:             alphaThreshold,
		GlyphAlphaThreshold:        glyphAlphaThreshold,
		MinFontSize:                opts.minFontSize,
		MaxFontSize:                opts.maxFontSize,
		SizeExponent:               opts.sizeExponent,
		FillerWordCount:            fillerWordCount,
		MaxPlacementAttempts:       opts.maxPlacementAttempts,
		MaxHeroPlacementAttempts:   opts.maxHeroPlacementAttempts,
		MaxFillerPlacementAttempts: opts.maxFillerPlacementAttempts,
		HeroWordCount:              opts.heroWordCount,
		FinalFillFontSize:          opts.finalFillFontSize,
		FillerMaxScale:             opts.fillerMaxScale,
		DetailPlacementBias:        opts.detailPlacementBias,
		SourceColorMinContrast:     opts.sourceColorMinContrast,
		ColorMode:                  opts.colorMode,
		Palette:                    palette,
		Background:                 background,
		InferBackground:            opts.inferBackground,
		Seed:                       opts.seed,
		MinWordLength:              opts.minWordLength,
		MaxWordLength:              opts.maxWordLength,
		StopWords:                  stopWords,
		MaxPixels:                  opts.maxPixels,
	}
	if opts.wordPadding >= 0 {
		conf.WordPadding = opts.wordPadding
		conf.WordPaddingSet = true
	}
	if opts.finalFillPasses >= 0 {
		conf.FinalFillPasses = opts.finalFillPasses
		conf.FinalFillPassesSet = true
	}

	result, err := wordcloud.GenerateResult(conf)
	if err != nil {
		return wordcloud.Result{}, err
	}
	if targetWidth > 0 && renderWidth > 0 && targetWidth != renderWidth {
		result.Image = imaging.Resize(result.Image, targetWidth, 0, imaging.Lanczos)
	}
	return result, nil
}

func resolveWordCloudWorkWidth(targetWidth, workWidth int) (int, error) {
	if workWidth < 0 {
		return 0, errors.New("word cloud work width cannot be negative")
	}
	if targetWidth < 0 {
		return 0, errors.New("target width cannot be negative")
	}
	if targetWidth == 0 {
		return workWidth, nil
	}
	if workWidth > 0 {
		return workWidth, nil
	}
	if targetWidth > 1920 {
		return 1920, nil
	}
	return targetWidth, nil
}

func resolveWordCloudAutoOptions(img image.Image, opts wordCloudOptions) wordCloudOptions {
	if strings.TrimSpace(opts.packingProfile) != "" {
		return opts
	}

	if looksLikeHighContrastPoster(img) {
		opts.packingProfile = wordcloud.PackingProfileBinarySilhouette
		if opts.maskType == "" {
			opts.maskType = wordcloud.MaskTypeContrast
		}
		if opts.maskThreshold < 0 {
			opts.maskThreshold = 0.16
		}
		if opts.qualityPreset == "" {
			opts.qualityPreset = wordcloud.QualityPresetPoster
		}
		if opts.minFontSize == 0 {
			opts.minFontSize = 3
		}
		if opts.maxFontSize == 0 {
			opts.maxFontSize = 72
		}
		if opts.fillerWordCount == 0 && opts.densityAlias == 0 {
			opts.fillerWordCount = 4200
		}
		if opts.finalFillPasses < 0 {
			opts.finalFillPasses = 6
		}
		if opts.finalFillFontSize == 0 {
			opts.finalFillFontSize = 3
		}
		if opts.fillerMaxScale == 0 {
			opts.fillerMaxScale = 0.18
		}
		if opts.wordPadding < 0 {
			opts.wordPadding = 0
		}
		if opts.colorMode == "" {
			opts.colorMode = wordcloud.ColorModeRandomPalette
		}
		if strings.TrimSpace(opts.palette) == "" {
			opts.derivePaletteFromSource = true
		}
		if strings.TrimSpace(opts.background) == "" || opts.background == "#ffffff" {
			opts.background = ""
		}
		return opts
	}

	if looksLikeForegroundOnLightBackground(img) {
		opts.packingProfile = wordcloud.PackingProfileTonalDetail
		if opts.maskType == "" {
			opts.maskType = wordcloud.MaskTypeAll
		}
		if opts.qualityPreset == "" {
			opts.qualityPreset = wordcloud.QualityPresetBalanced
		}
		if opts.minFontSize == 0 {
			opts.minFontSize = 5
		}
		if opts.maxFontSize == 0 {
			opts.maxFontSize = 36
		}
		if opts.fillerWordCount == 0 && opts.densityAlias == 0 {
			opts.fillerWordCount = 1200
		}
		if opts.finalFillPasses < 0 {
			opts.finalFillPasses = 1
		}
		if opts.finalFillFontSize == 0 {
			opts.finalFillFontSize = 5
		}
		if opts.fillerMaxScale == 0 {
			opts.fillerMaxScale = 0.20
		}
		if opts.wordPadding < 0 {
			opts.wordPadding = 1
		}
		if opts.colorMode == "" {
			opts.colorMode = wordcloud.ColorModeSource
		}
		return opts
	}

	opts.packingProfile = wordcloud.PackingProfileForegroundBackground
	if opts.qualityPreset == "" {
		opts.qualityPreset = wordcloud.QualityPresetBalanced
	}
	if opts.minFontSize == 0 {
		opts.minFontSize = 5
	}
	if opts.maxFontSize == 0 {
		opts.maxFontSize = 34
	}
	if opts.fillerWordCount == 0 && opts.densityAlias == 0 {
		opts.fillerWordCount = 1300
	}
	if opts.finalFillPasses < 0 {
		opts.finalFillPasses = 2
	}
	if opts.finalFillFontSize == 0 {
		opts.finalFillFontSize = 5
	}
	if opts.fillerMaxScale == 0 {
		opts.fillerMaxScale = 0.18
	}
	if opts.wordPadding < 0 {
		opts.wordPadding = 1
	}
	if opts.colorMode == "" {
		opts.colorMode = wordcloud.ColorModeSource
	}
	return opts
}

func looksLikeHighContrastPoster(img image.Image) bool {
	bounds := img.Bounds()
	w, h := bounds.Dx(), bounds.Dy()
	if w == 0 || h == 0 {
		return false
	}

	step := max(1, min(w, h)/96)
	patchW := max(step, w/10)
	patchH := max(step, h/10)
	corners := []image.Rectangle{
		image.Rect(bounds.Min.X, bounds.Min.Y, bounds.Min.X+patchW, bounds.Min.Y+patchH),
		image.Rect(bounds.Max.X-patchW, bounds.Min.Y, bounds.Max.X, bounds.Min.Y+patchH),
		image.Rect(bounds.Min.X, bounds.Max.Y-patchH, bounds.Min.X+patchW, bounds.Max.Y),
		image.Rect(bounds.Max.X-patchW, bounds.Max.Y-patchH, bounds.Max.X, bounds.Max.Y),
	}

	darkCorners := 0
	for _, corner := range corners {
		if averagePatchLuminance(img, corner, step) < 0.08 {
			darkCorners++
		}
	}
	if darkCorners < 3 {
		return false
	}

	center := image.Rect(
		bounds.Min.X+w/5,
		bounds.Min.Y+h/8,
		bounds.Max.X-w/5,
		bounds.Max.Y-h/8,
	)
	return averagePatchSaturation(img, center, step) > 0.32 && averagePatchLuminance(img, center, step) > 0.08
}

func looksLikeForegroundOnLightBackground(img image.Image) bool {
	bounds := img.Bounds()
	w, h := bounds.Dx(), bounds.Dy()
	if w == 0 || h == 0 {
		return false
	}

	step := max(1, min(w, h)/96)
	patchW := max(step, w/10)
	patchH := max(step, h/10)
	corners := []image.Rectangle{
		image.Rect(bounds.Min.X, bounds.Min.Y, bounds.Min.X+patchW, bounds.Min.Y+patchH),
		image.Rect(bounds.Max.X-patchW, bounds.Min.Y, bounds.Max.X, bounds.Min.Y+patchH),
		image.Rect(bounds.Min.X, bounds.Max.Y-patchH, bounds.Min.X+patchW, bounds.Max.Y),
		image.Rect(bounds.Max.X-patchW, bounds.Max.Y-patchH, bounds.Max.X, bounds.Max.Y),
	}
	brightCorners := 0
	for _, corner := range corners {
		if averagePatchLuminance(img, corner, step) > 0.86 {
			brightCorners++
		}
	}
	return brightCorners >= 3
}

func averagePatchLuminance(img image.Image, rect image.Rectangle, step int) float64 {
	bounds := img.Bounds()
	rect = rect.Intersect(bounds)
	var sum float64
	var samples int
	for y := rect.Min.Y; y < rect.Max.Y; y += step {
		for x := rect.Min.X; x < rect.Max.X; x += step {
			sum += imageLuminance(img.At(x, y))
			samples++
		}
	}
	if samples == 0 {
		return 0
	}
	return sum / float64(samples)
}

func imageLuminance(c color.Color) float64 {
	r, g, b, _ := c.RGBA()
	return (0.299*float64(r) + 0.587*float64(g) + 0.114*float64(b)) / 65535.0
}

func averagePatchSaturation(img image.Image, rect image.Rectangle, step int) float64 {
	bounds := img.Bounds()
	rect = rect.Intersect(bounds)
	var sum float64
	var samples int
	for y := rect.Min.Y; y < rect.Max.Y; y += step {
		for x := rect.Min.X; x < rect.Max.X; x += step {
			sum += imageSaturation(img.At(x, y))
			samples++
		}
	}
	if samples == 0 {
		return 0
	}
	return sum / float64(samples)
}

func imageSaturation(c color.Color) float64 {
	r16, g16, b16, _ := c.RGBA()
	r := float64(r16) / 65535.0
	g := float64(g16) / 65535.0
	b := float64(b16) / 65535.0
	maxV := math.Max(r, math.Max(g, b))
	if maxV == 0 {
		return 0
	}
	minV := math.Min(r, math.Min(g, b))
	return (maxV - minV) / maxV
}

func deriveSourcePalette(img image.Image, maxColors int) []color.Color {
	if maxColors <= 0 {
		return nil
	}
	bounds := img.Bounds()
	step := max(1, min(bounds.Dx(), bounds.Dy())/96)
	bgR, bgG, bgB := averageImageBorderRGB(img, step)

	type sample struct {
		color color.NRGBA
		lum   float64
	}
	samples := make([]sample, 0)
	for y := bounds.Min.Y; y < bounds.Max.Y; y += step {
		for x := bounds.Min.X; x < bounds.Max.X; x += step {
			c := img.At(x, y)
			r16, g16, b16, a16 := c.RGBA()
			r := float64(r16) / 65535.0
			g := float64(g16) / 65535.0
			b := float64(b16) / 65535.0
			contrast := math.Abs(imageLuminance(c) - imageLuminance(color.NRGBA{
				R: uint8(bgR * 255),
				G: uint8(bgG * 255),
				B: uint8(bgB * 255),
				A: 255,
			}))
			distance := math.Sqrt((r-bgR)*(r-bgR)+(g-bgG)*(g-bgG)+(b-bgB)*(b-bgB)) / math.Sqrt(3)
			if math.Max(contrast, distance) < 0.10 {
				continue
			}
			samples = append(samples, sample{
				color: color.NRGBA{
					R: uint8(r * 255),
					G: uint8(g * 255),
					B: uint8(b * 255),
					A: uint8(float64(a16) / 65535.0 * 255),
				},
				lum: imageLuminance(c),
			})
		}
	}
	if len(samples) == 0 {
		return nil
	}

	sort.Slice(samples, func(i, j int) bool {
		return samples[i].lum < samples[j].lum
	})
	palette := make([]color.Color, 0, maxColors)
	for i := 0; i < maxColors; i++ {
		index := int(math.Round(float64(i) * float64(len(samples)-1) / float64(max(1, maxColors-1))))
		palette = append(palette, boostPaletteColorFromBackground(samples[index].color, bgR, bgG, bgB, 0.34))
	}
	return palette
}

func boostPaletteColorFromBackground(c color.NRGBA, bgR, bgG, bgB float64, minDiff float64) color.NRGBA {
	lum := imageLuminance(c)
	bgLum := imageLuminance(color.NRGBA{
		R: uint8(bgR * 255),
		G: uint8(bgG * 255),
		B: uint8(bgB * 255),
		A: 255,
	})
	if math.Abs(lum-bgLum) >= minDiff {
		return c
	}
	if bgLum >= 0.5 {
		target := math.Max(0, bgLum-minDiff)
		scale := target / math.Max(lum, 0.001)
		return color.NRGBA{
			R: uint8(math.Min(255, float64(c.R)*scale)),
			G: uint8(math.Min(255, float64(c.G)*scale)),
			B: uint8(math.Min(255, float64(c.B)*scale)),
			A: c.A,
		}
	}
	target := math.Min(1, bgLum+minDiff)
	scale := (target - lum) / math.Max(1-lum, 0.001)
	return color.NRGBA{
		R: uint8(math.Min(255, float64(c.R)+(255-float64(c.R))*scale)),
		G: uint8(math.Min(255, float64(c.G)+(255-float64(c.G))*scale)),
		B: uint8(math.Min(255, float64(c.B)+(255-float64(c.B))*scale)),
		A: c.A,
	}
}

func averageImageBorderRGB(img image.Image, step int) (float64, float64, float64) {
	bounds := img.Bounds()
	var rSum, gSum, bSum, samples float64
	add := func(x, y int) {
		r, g, b, _ := img.At(x, y).RGBA()
		rSum += float64(r) / 65535.0
		gSum += float64(g) / 65535.0
		bSum += float64(b) / 65535.0
		samples++
	}
	for x := bounds.Min.X; x < bounds.Max.X; x += step {
		add(x, bounds.Min.Y)
		add(x, bounds.Max.Y-1)
	}
	for y := bounds.Min.Y; y < bounds.Max.Y; y += step {
		add(bounds.Min.X, y)
		add(bounds.Max.X-1, y)
	}
	if samples == 0 {
		return 1, 1, 1
	}
	return rSum / samples, gSum / samples, bSum / samples
}

func resolveForegroundMask(img image.Image, opts wordCloudOptions) (image.Image, error) {
	if strings.TrimSpace(opts.foregroundMaskPath) != "" {
		mask, err := imaging.Open(opts.foregroundMaskPath)
		if err != nil {
			return nil, fmt.Errorf("open foreground mask %q: %w", opts.foregroundMaskPath, err)
		}
		return mask, nil
	}

	if opts.packingProfile != wordcloud.PackingProfileForegroundBackground {
		if opts.packingProfile == wordcloud.PackingProfileTonalDetail && opts.maskType == wordcloud.MaskTypeAll {
			return imageutil.LightBackgroundForegroundMask(img, imageutil.LightBackgroundMaskConfig{}), nil
		}
		return nil, nil
	}

	rect, err := parseOptionalRect(opts.foregroundRect)
	if err != nil {
		return nil, err
	}
	mask, err := imageutil.GrabCutForegroundMask(img, imageutil.GrabCutConfig{
		Rect:                      rect,
		IterCount:                 opts.grabCutIterations,
		IncludeProbableForeground: opts.includeProbableForeground,
	})
	if err != nil {
		return nil, err
	}

	if strings.TrimSpace(opts.saveForegroundMaskPath) != "" {
		if err := savePNG(mask, opts.saveForegroundMaskPath); err != nil {
			return nil, err
		}
	}

	return mask, nil
}

func parseOptionalRect(value string) (image.Rectangle, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return image.Rectangle{}, nil
	}
	parts := strings.Split(value, ",")
	if len(parts) != 4 {
		return image.Rectangle{}, errors.New("foreground rect must be x,y,width,height")
	}
	nums := [4]int{}
	for i, part := range parts {
		n, err := strconv.Atoi(strings.TrimSpace(part))
		if err != nil {
			return image.Rectangle{}, fmt.Errorf("parse foreground rect %q: %w", value, err)
		}
		nums[i] = n
	}
	if nums[2] <= 0 || nums[3] <= 0 {
		return image.Rectangle{}, errors.New("foreground rect width and height must be positive")
	}
	return image.Rect(nums[0], nums[1], nums[0]+nums[2], nums[1]+nums[3]), nil
}

func savePNG(img image.Image, path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return fmt.Errorf("create foreground mask directory: %w", err)
	}
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("create foreground mask %q: %w", path, err)
	}
	defer f.Close()
	if err := png.Encode(f, img); err != nil {
		return fmt.Errorf("write foreground mask %q: %w", path, err)
	}
	return nil
}

func optionalUnitFloat(value float64, name string) (*float64, error) {
	if value < 0 {
		return nil, nil
	}
	if value > 1 {
		return nil, fmt.Errorf("%s must be between 0 and 1", name)
	}
	return &value, nil
}

func resolveFillerWordCount(fillerWordCount, densityAlias int) (int, error) {
	if fillerWordCount > 0 && densityAlias > 0 && fillerWordCount != densityAlias {
		return 0, errors.New("use either -filler-word-count or -density, not both with different values")
	}
	if fillerWordCount > 0 {
		return fillerWordCount, nil
	}
	if densityAlias > 0 {
		return densityAlias, nil
	}
	return 0, nil
}

func parsePalette(value string) ([]color.Color, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil, nil
	}

	parts := strings.Split(value, ",")
	palette := make([]color.Color, 0, len(parts))
	for _, part := range parts {
		c, err := parseHexColor(part)
		if err != nil {
			return nil, fmt.Errorf("parse palette color %q: %w", part, err)
		}
		palette = append(palette, c)
	}
	return palette, nil
}

func optionalColor(value string) (color.Color, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil, nil
	}
	return parseHexColor(value)
}

func parseHexColor(value string) (color.Color, error) {
	value = strings.TrimSpace(strings.TrimPrefix(value, "#"))
	if len(value) != 6 && len(value) != 8 {
		return nil, errors.New("expected #RRGGBB or #RRGGBBAA")
	}

	r, err := parseHexByte(value[0:2])
	if err != nil {
		return nil, err
	}
	g, err := parseHexByte(value[2:4])
	if err != nil {
		return nil, err
	}
	b, err := parseHexByte(value[4:6])
	if err != nil {
		return nil, err
	}

	a := uint8(255)
	if len(value) == 8 {
		a, err = parseHexByte(value[6:8])
		if err != nil {
			return nil, err
		}
	}

	return color.NRGBA{R: r, G: g, B: b, A: a}, nil
}

func parseHexByte(value string) (uint8, error) {
	parsed, err := strconv.ParseUint(value, 16, 8)
	if err != nil {
		return 0, err
	}
	return uint8(parsed), nil
}

func parseStopWords(value string) map[string]struct{} {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}

	words := make(map[string]struct{})
	for _, part := range strings.Split(value, ",") {
		word := strings.TrimSpace(part)
		if word != "" {
			words[word] = struct{}{}
		}
	}
	return words
}
