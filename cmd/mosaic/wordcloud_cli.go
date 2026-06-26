package main

import (
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"log/slog"
	"os"
	"path/filepath"
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
	colorMode                  string
	palette                    string
	background                 string
	inferBackground            bool
	seed                       int64
	minWordLength              int
	maxWordLength              int
	stopWords                  string
}

func generateWordCloud(logger *slog.Logger, img image.Image, text string, fontPath string, targetWidth int, opts wordCloudOptions) (wordcloud.Result, error) {
	opts = resolveWordCloudAutoOptions(img, opts)
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
		TargetWidth:                targetWidth,
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
		ColorMode:                  opts.colorMode,
		Palette:                    palette,
		Background:                 background,
		InferBackground:            opts.inferBackground,
		Seed:                       opts.seed,
		MinWordLength:              opts.minWordLength,
		MaxWordLength:              opts.maxWordLength,
		StopWords:                  stopWords,
	}
	if opts.wordPadding >= 0 {
		conf.WordPadding = opts.wordPadding
		conf.WordPaddingSet = true
	}
	if opts.finalFillPasses >= 0 {
		conf.FinalFillPasses = opts.finalFillPasses
		conf.FinalFillPassesSet = true
	}

	return wordcloud.GenerateResult(conf)
}

func resolveWordCloudAutoOptions(img image.Image, opts wordCloudOptions) wordCloudOptions {
	if strings.TrimSpace(opts.packingProfile) != "" {
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
