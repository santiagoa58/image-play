package wordcloud

import (
	"errors"
	"fmt"
	"image"
	"image/color"
	"log/slog"
	"math"
	"math/rand"
	"sort"
	"strings"
	"unicode"

	"github.com/disintegration/imaging"
	"github.com/fogleman/gg"
)

const (
	PackingProfileBinarySilhouette     = "binary-silhouette"
	PackingProfileTonalDetail          = "tonal-detail"
	PackingProfileForegroundBackground = "foreground-background"

	QualityPresetFast     = "fast"
	QualityPresetBalanced = "balanced"
	QualityPresetDense    = "dense"
	QualityPresetPoster   = "poster"

	ColorModeSource            = "source"
	ColorModePalette           = "palette"
	ColorModeLuminancePalette  = "luminance-palette"
	ColorModeRandomPalette     = "random-palette"
	ColorModeSequentialPalette = "sequential-palette"

	MaskTypeLight = "light"
	MaskTypeDark  = "dark"
	MaskTypeAll   = "all"
	// MaskTypeContrast fills pixels that visually separate from the inferred or
	// explicit background by luminance contrast or color distance.
	MaskTypeContrast = "contrast"

	defaultMaxFontSize            = 96
	defaultMinFontSize            = 8
	defaultMaskThreshold          = 0.50
	defaultAlphaThreshold         = 0.10
	defaultGlyphAlphaThreshold    = 0.05
	defaultFillerWordCount        = 1200
	defaultMaxPlacementAttempts   = 3000
	defaultFillerMaxScale         = 0.25
	defaultFinalFillAnchorSamples = 32
	defaultMaxPixels              = 32_000_000
	defaultSizeExponent           = 0.50
	defaultHeroWordCount          = 6
	defaultDetailPlacementBias    = 0.0
	defaultForegroundThreshold    = 0.50

	maxColorChannelValue = 65535.0
)

var defaultPalette = []color.Color{
	color.RGBA{30, 30, 30, 255},
	color.RGBA{82, 82, 82, 255},
	color.RGBA{132, 132, 132, 255},
	color.RGBA{185, 185, 185, 255},
}

// Config defines parameters needed to generate a dense shape word cloud.
type Config struct {
	Logger     *slog.Logger
	Text       string
	InputImage image.Image
	// ForegroundMask is an optional semantic/depth/focus mask. White/opaque pixels
	// are foreground; black/transparent pixels are background. When omitted, the
	// foreground-background profile derives a deterministic focus/position mask.
	ForegroundMask image.Image
	TargetWidth    int
	FontPath       string
	// PackingProfile selects a technical preset over the same glyph-packing engine.
	// Supported values are "binary-silhouette", "tonal-detail", and
	// "foreground-background".
	PackingProfile string
	// QualityPreset selects a performance/density preset. Supported values are
	// "fast", "balanced", "dense", and "poster".
	QualityPreset string

	MinFontSize int
	MaxFontSize int
	// SizeExponent controls the frequency-to-font-size curve. 0.5 produces a balanced
	// hierarchy; larger values emphasize dominant words more strongly.
	SizeExponent float64

	// MaskThreshold is the luminance threshold used to extract the shape. A nil value
	// uses the default; a pointer allows zero to be used intentionally.
	MaskThreshold *float64
	// AlphaThreshold is the source alpha cutoff used for mask extraction. A nil value
	// uses the default.
	AlphaThreshold *float64
	// GlyphAlphaThreshold is the rendered glyph alpha cutoff for collision/stamping.
	GlyphAlphaThreshold *float64
	MaskType            string // "light", "dark", "all", or "contrast"
	// ForegroundThreshold is the luminance threshold for ForegroundMask. Nil uses
	// the default; a pointer allows zero to be used intentionally.
	ForegroundThreshold *float64
	// ForegroundBackground enables two-layer packing: a sparse/larger background
	// layer and a denser foreground layer. The foreground layer requires
	// ForegroundMask.
	ForegroundBackground bool

	// FillerWordCount is the number of progressively smaller filler candidates.
	FillerWordCount int
	// DensityLevel is kept as a backwards-compatible alias for FillerWordCount.
	DensityLevel int

	MaxPlacementAttempts int
	// MaxHeroPlacementAttempts and MaxFillerPlacementAttempts override MaxPlacementAttempts
	// for primary hero words and filler/final-fill words.
	MaxHeroPlacementAttempts   int
	MaxFillerPlacementAttempts int
	WordPadding                int
	WordPaddingSet             bool
	MaxPixels                  int

	ColorMode string
	Palette   []color.Color
	Seed      int64
	// SourceColorMinContrast pushes source-sampled colors away from the background.
	// It is useful for full-frame photo word clouds where pale or dark source regions
	// would otherwise disappear into the canvas.
	SourceColorMinContrast float64
	// InvertLuminancePalette maps bright source regions to earlier palette entries.
	InvertLuminancePalette bool

	Background      color.Color
	InferBackground bool

	HeroWordCount          int
	FinalFillPasses        int
	FinalFillPassesSet     bool
	FinalFillFontSize      int
	FinalFillAnchorSamples int
	FillerMaxScale         float64
	// DetailPlacementBias increases final-fill and tonal-profile placement preference
	// for local luminance detail. 0 disables detail-biased scoring.
	DetailPlacementBias float64

	MinWordLength int
	MaxWordLength int
	StopWords     map[string]struct{}
}

type Result struct {
	Image image.Image
	Stats Stats
}

type Stats struct {
	AttemptedWords             int
	PlacedWords                int
	RejectedWords              int
	PlacementChecks            int
	MaskCheckFailures          int
	CollisionCheckFailures     int
	WordsRejectedAfterAttempts int
	WordsRejectedTooLarge      int
	WordsRejectedRenderFailure int
	MaskCoverage               float64
	OccupiedCoverage           float64
	ForegroundMaskCoverage     float64
	ForegroundOccupiedCoverage float64
	BackgroundOccupiedCoverage float64
}

type settings struct {
	logger                 *slog.Logger
	minFontSize            int
	maxFontSize            int
	sizeExponent           float64
	maskThreshold          float64
	foregroundThreshold    float64
	alphaThreshold         float64
	glyphAlphaThreshold    float64
	maskType               string
	fillerWordCount        int
	maxPlacementAttempts   int
	maxHeroAttempts        int
	maxFillerAttempts      int
	wordPadding            int
	maxPixels              int
	colorMode              string
	palette                []color.Color
	seed                   int64
	sourceColorMinContrast float64
	invertLuminancePalette bool
	inferBackground        bool
	heroWordCount          int
	finalFillPasses        int
	finalFillFontSize      int
	finalFillAnchorSamples int
	fillerMaxScale         float64
	detailPlacementBias    float64
	foregroundBackground   bool
	minWordLength          int
	maxWordLength          int
	stopWords              map[string]struct{}
}

type wordStat struct {
	Word  string
	Count int
}

type wordCandidate struct {
	Word string
	Size float64
}

type glyphPixel struct {
	x     int
	y     int
	alpha uint8
}

type glyphSpan struct {
	y  int
	x1 int
	x2 int
}

type glyphBitmap struct {
	word    string
	size    float64
	width   int
	height  int
	pixels  []glyphPixel
	spans   []glyphSpan
	rotated bool
}

type glyphKey struct {
	word           string
	size           int
	rotated        bool
	alphaThreshold int
}

type placedWord struct {
	glyph *glyphBitmap
	x     int
	y     int
	box   rect
	color color.Color
}

type shapeMask struct {
	width           int
	height          int
	bits            []bool
	luminance       []float64
	detail          []float64
	playableIndexes []int
	playablePx      int
	minX            int
	minY            int
	maxX            int
	maxY            int
	centerX         float64
	centerY         float64
	bgColor         color.Color
}

type pointF struct {
	x float64
	y float64
}

type rect struct {
	left   int
	top    int
	right  int
	bottom int
}

type spatialHash struct {
	cellSize int
	cells    map[cell][]int
	boxes    []rect
}

type cell struct {
	x int
	y int
}

type packer struct {
	mask                   *shapeMask
	source                 image.Image
	fontPath               string
	minFontSize            int
	wordPadding            int
	glyphAlphaThreshold    float64
	maxAttempts            int
	maxHeroAttempts        int
	maxFillerAttempts      int
	heroWordCount          int
	finalFillPasses        int
	finalFillFontSize      int
	finalFillAnchorSamples int
	colorMode              string
	palette                []color.Color
	sourceColorMinContrast float64
	invertLuminancePalette bool
	layoutRNG              *rand.Rand
	colorRNG               *rand.Rand
	detailPlacementBias    float64
	occupancy              []bool
	hash                   *spatialHash
	glyphs                 map[glyphKey]*glyphBitmap
	anchors                []pointF
	stats                  Stats
}

// Generate creates a shape-constrained word cloud and returns only the image.
func Generate(conf Config) (image.Image, error) {
	result, err := GenerateResult(conf)
	if err != nil {
		return nil, err
	}
	return result.Image, nil
}

// GenerateResult creates a shape-constrained word cloud and returns quality/debug stats.
func GenerateResult(conf Config) (Result, error) {
	cfg, err := normalizeConfig(conf)
	if err != nil {
		return Result{}, err
	}

	img := conf.InputImage
	if conf.TargetWidth > 0 {
		img = imaging.Resize(img, conf.TargetWidth, 0, imaging.Lanczos)
	}
	bounds := img.Bounds()
	if bounds.Dx()*bounds.Dy() > cfg.maxPixels {
		return Result{}, fmt.Errorf("input image has %d pixels after resize, max is %d; lower the render width or raise the max pixel limit for direct large renders", bounds.Dx()*bounds.Dy(), cfg.maxPixels)
	}

	mask, err := buildMask(img, cfg.maskType, cfg.maskThreshold, cfg.alphaThreshold, cfg.inferBackground)
	if err != nil {
		return Result{}, err
	}
	if conf.Background != nil {
		mask.bgColor = conf.Background
	}
	if conf.ForegroundMask != nil && !cfg.foregroundBackground {
		foregroundBits, err := buildForegroundBits(conf.ForegroundMask, mask, cfg.foregroundThreshold, cfg.alphaThreshold)
		if err != nil {
			return Result{}, err
		}
		mask = deriveMask(mask, func(index int) bool {
			return mask.bits[index] && foregroundBits[index]
		})
		if mask.playablePx == 0 {
			return Result{}, errors.New("foreground clip mask has no playable pixels")
		}
	}

	stats, err := parseWordStats(conf.Text, cfg)
	if err != nil {
		return Result{}, err
	}

	if cfg.foregroundBackground {
		if conf.ForegroundMask == nil {
			return Result{}, errors.New("foreground-background packing requires a foreground mask")
		}
		foregroundBits, err := buildForegroundBits(conf.ForegroundMask, mask, cfg.foregroundThreshold, cfg.alphaThreshold)
		if err != nil {
			return Result{}, err
		}
		return generateForegroundBackgroundResult(mask, img, conf.FontPath, stats, cfg, foregroundBits)
	}

	cfg.logger.Debug(
		"running dense glyph word cloud packer",
		"words", len(stats),
		"filler_words", cfg.fillerWordCount,
		"size", fmt.Sprintf("%dx%d", mask.width, mask.height),
	)

	candidates := buildHierarchy(stats, cfg.minFontSize, cfg.maxFontSize, cfg.fillerWordCount, cfg.sizeExponent, cfg.fillerMaxScale, rand.New(rand.NewSource(cfg.seed)))
	p := newPacker(mask, img, conf.FontPath, cfg)
	placed := p.pack(candidates)
	if cfg.finalFillPasses > 0 {
		placed = append(placed, p.finalFill(stats)...)
	}
	if len(placed) == 0 {
		return Result{Stats: p.finalStats()}, errors.New("could not place any words inside the image mask")
	}

	finalStats := p.finalStats()
	return Result{
		Image: drawWords(mask.width, mask.height, mask.bgColor, placed),
		Stats: finalStats,
	}, nil
}

func generateForegroundBackgroundResult(baseMask *shapeMask, img image.Image, fontPath string, stats []wordStat, cfg settings, foregroundBits []bool) (Result, error) {
	foregroundMask := deriveMask(baseMask, func(index int) bool {
		return baseMask.bits[index] && foregroundBits[index]
	})
	if foregroundMask.playablePx == 0 {
		return Result{}, errors.New("foreground mask has no playable pixels")
	}

	backgroundMask := deriveMask(baseMask, func(index int) bool {
		return baseMask.bits[index] && !foregroundBits[index]
	})

	placed := make([]placedWord, 0)
	combined := Stats{}

	if backgroundMask.playablePx > 0 {
		backgroundCfg := backgroundLayerSettings(cfg, len(stats))
		backgroundCandidates := buildHierarchy(stats, backgroundCfg.minFontSize, backgroundCfg.maxFontSize, backgroundCfg.fillerWordCount, backgroundCfg.sizeExponent, backgroundCfg.fillerMaxScale, rand.New(rand.NewSource(seedOffset(backgroundCfg.seed, 3))))
		backgroundPacker := newPacker(backgroundMask, img, fontPath, backgroundCfg)
		backgroundPlaced := backgroundPacker.pack(backgroundCandidates)
		placed = append(placed, backgroundPlaced...)
		backgroundStats := backgroundPacker.finalStats()
		combined = addStats(combined, backgroundStats)
		combined.BackgroundOccupiedCoverage = backgroundStats.OccupiedCoverage
	}

	foregroundCandidates := buildHierarchy(stats, cfg.minFontSize, cfg.maxFontSize, cfg.fillerWordCount, cfg.sizeExponent, cfg.fillerMaxScale, rand.New(rand.NewSource(seedOffset(cfg.seed, 4))))
	foregroundPacker := newPacker(foregroundMask, img, fontPath, cfg)
	foregroundPlaced := foregroundPacker.pack(foregroundCandidates)
	if cfg.finalFillPasses > 0 {
		foregroundPlaced = append(foregroundPlaced, foregroundPacker.finalFill(stats)...)
	}
	placed = append(placed, foregroundPlaced...)
	foregroundStats := foregroundPacker.finalStats()
	combined = addStats(combined, foregroundStats)
	combined.ForegroundOccupiedCoverage = foregroundStats.OccupiedCoverage

	if len(placed) == 0 {
		return Result{Stats: combined}, errors.New("could not place any words in foreground-background layers")
	}

	totalPlayable := baseMask.playablePx
	foregroundPlayable := foregroundMask.playablePx
	backgroundPlayable := backgroundMask.playablePx
	foregroundOccupied := int(math.Round(foregroundStats.OccupiedCoverage * float64(foregroundPlayable)))
	backgroundOccupied := 0
	if backgroundPlayable > 0 {
		backgroundOccupied = int(math.Round(combined.BackgroundOccupiedCoverage * float64(backgroundPlayable)))
	}
	combined.MaskCoverage = float64(totalPlayable) / float64(baseMask.width*baseMask.height)
	combined.ForegroundMaskCoverage = float64(foregroundPlayable) / float64(baseMask.width*baseMask.height)
	if totalPlayable > 0 {
		combined.OccupiedCoverage = float64(foregroundOccupied+backgroundOccupied) / float64(totalPlayable)
	}

	return Result{
		Image: drawWords(baseMask.width, baseMask.height, baseMask.bgColor, placed),
		Stats: combined,
	}, nil
}

func backgroundLayerSettings(cfg settings, uniqueWords int) settings {
	bg := cfg
	bg.minFontSize = max(cfg.minFontSize+2, int(math.Round(float64(cfg.maxFontSize)*0.45)))
	bg.maxFontSize = cfg.maxFontSize
	if bg.minFontSize > bg.maxFontSize {
		bg.minFontSize = max(1, bg.maxFontSize/2)
	}
	bg.fillerWordCount = max(uniqueWords*3, cfg.fillerWordCount/8)
	bg.finalFillPasses = 0
	bg.finalFillFontSize = bg.minFontSize
	bg.fillerMaxScale = max(cfg.fillerMaxScale, 0.45)
	bg.maxFillerAttempts = max(150, cfg.maxFillerAttempts/2)
	bg.wordPadding = max(1, cfg.wordPadding)
	bg.detailPlacementBias = cfg.detailPlacementBias * 0.25
	return bg
}

func addStats(a, b Stats) Stats {
	a.AttemptedWords += b.AttemptedWords
	a.PlacedWords += b.PlacedWords
	a.RejectedWords += b.RejectedWords
	a.PlacementChecks += b.PlacementChecks
	a.MaskCheckFailures += b.MaskCheckFailures
	a.CollisionCheckFailures += b.CollisionCheckFailures
	a.WordsRejectedAfterAttempts += b.WordsRejectedAfterAttempts
	a.WordsRejectedTooLarge += b.WordsRejectedTooLarge
	a.WordsRejectedRenderFailure += b.WordsRejectedRenderFailure
	return a
}

func normalizeConfig(conf Config) (settings, error) {
	var err error
	conf, err = applyQualityPresetDefaults(conf)
	if err != nil {
		return settings{}, err
	}
	conf, err = applyPackingProfileDefaults(conf)
	if err != nil {
		return settings{}, err
	}

	cfg := settings{
		logger:                 conf.Logger,
		minFontSize:            conf.MinFontSize,
		maxFontSize:            conf.MaxFontSize,
		sizeExponent:           conf.SizeExponent,
		maskThreshold:          defaultMaskThreshold,
		foregroundThreshold:    defaultForegroundThreshold,
		alphaThreshold:         defaultAlphaThreshold,
		glyphAlphaThreshold:    defaultGlyphAlphaThreshold,
		maskType:               conf.MaskType,
		fillerWordCount:        conf.FillerWordCount,
		maxPlacementAttempts:   conf.MaxPlacementAttempts,
		maxHeroAttempts:        conf.MaxHeroPlacementAttempts,
		maxFillerAttempts:      conf.MaxFillerPlacementAttempts,
		wordPadding:            conf.WordPadding,
		maxPixels:              conf.MaxPixels,
		colorMode:              conf.ColorMode,
		palette:                append([]color.Color(nil), conf.Palette...),
		seed:                   conf.Seed,
		sourceColorMinContrast: conf.SourceColorMinContrast,
		invertLuminancePalette: conf.InvertLuminancePalette,
		inferBackground:        conf.InferBackground,
		heroWordCount:          conf.HeroWordCount,
		finalFillPasses:        conf.FinalFillPasses,
		finalFillFontSize:      conf.FinalFillFontSize,
		finalFillAnchorSamples: conf.FinalFillAnchorSamples,
		fillerMaxScale:         conf.FillerMaxScale,
		detailPlacementBias:    conf.DetailPlacementBias,
		foregroundBackground:   conf.ForegroundBackground,
		minWordLength:          conf.MinWordLength,
		maxWordLength:          conf.MaxWordLength,
		stopWords:              conf.StopWords,
	}
	if cfg.logger == nil {
		cfg.logger = slog.Default()
	}
	if conf.InputImage == nil {
		return cfg, errors.New("input image is required")
	}
	bounds := conf.InputImage.Bounds()
	if bounds.Dx() <= 0 || bounds.Dy() <= 0 {
		return cfg, errors.New("input image must have positive dimensions")
	}
	if conf.FontPath == "" {
		return cfg, errors.New("font path is required")
	}
	if conf.TargetWidth < 0 {
		return cfg, errors.New("target width cannot be negative")
	}
	if cfg.maxFontSize <= 0 {
		cfg.maxFontSize = defaultMaxFontSize
	}
	if cfg.minFontSize <= 0 {
		cfg.minFontSize = defaultMinFontSize
	}
	if cfg.minFontSize > cfg.maxFontSize {
		return cfg, errors.New("min font size cannot be greater than max font size")
	}
	if cfg.sizeExponent == 0 {
		cfg.sizeExponent = defaultSizeExponent
	}
	if cfg.sizeExponent <= 0 {
		return cfg, errors.New("size exponent must be greater than zero")
	}
	if conf.MaskThreshold != nil {
		cfg.maskThreshold = *conf.MaskThreshold
	}
	if cfg.maskThreshold < 0 || cfg.maskThreshold > 1 {
		return cfg, errors.New("mask threshold must be between 0 and 1")
	}
	if conf.ForegroundThreshold != nil {
		cfg.foregroundThreshold = *conf.ForegroundThreshold
	}
	if cfg.foregroundThreshold < 0 || cfg.foregroundThreshold > 1 {
		return cfg, errors.New("foreground threshold must be between 0 and 1")
	}
	if conf.AlphaThreshold != nil {
		cfg.alphaThreshold = *conf.AlphaThreshold
	}
	if cfg.alphaThreshold < 0 || cfg.alphaThreshold > 1 {
		return cfg, errors.New("alpha threshold must be between 0 and 1")
	}
	if conf.GlyphAlphaThreshold != nil {
		cfg.glyphAlphaThreshold = *conf.GlyphAlphaThreshold
	}
	if cfg.glyphAlphaThreshold <= 0 || cfg.glyphAlphaThreshold > 1 {
		return cfg, errors.New("glyph alpha threshold must be greater than 0 and less than or equal to 1")
	}
	if cfg.maskType == "" {
		cfg.maskType = MaskTypeLight
	}
	if cfg.maskType != MaskTypeLight && cfg.maskType != MaskTypeDark && cfg.maskType != MaskTypeAll && cfg.maskType != MaskTypeContrast {
		return cfg, errors.New(`mask type must be "light", "dark", "all", or "contrast"`)
	}
	if cfg.fillerWordCount <= 0 {
		cfg.fillerWordCount = conf.DensityLevel
	}
	if cfg.fillerWordCount <= 0 {
		cfg.fillerWordCount = defaultFillerWordCount
	}
	if cfg.maxPlacementAttempts <= 0 {
		cfg.maxPlacementAttempts = defaultMaxPlacementAttempts
	}
	if cfg.maxHeroAttempts <= 0 {
		cfg.maxHeroAttempts = cfg.maxPlacementAttempts
	}
	if cfg.maxFillerAttempts <= 0 {
		cfg.maxFillerAttempts = max(250, cfg.maxPlacementAttempts/5)
	}
	if cfg.wordPadding < 0 {
		return cfg, errors.New("word padding cannot be negative")
	}
	if cfg.maxPixels <= 0 {
		cfg.maxPixels = defaultMaxPixels
	}
	if cfg.colorMode == "" {
		cfg.colorMode = ColorModePalette
	}
	switch cfg.colorMode {
	case ColorModeSource, ColorModePalette, ColorModeLuminancePalette, ColorModeRandomPalette, ColorModeSequentialPalette:
	default:
		return cfg, fmt.Errorf("unknown color mode %q", cfg.colorMode)
	}
	if len(cfg.palette) == 0 {
		cfg.palette = append([]color.Color(nil), defaultPalette...)
	}
	if cfg.sourceColorMinContrast < 0 || cfg.sourceColorMinContrast > 1 {
		return cfg, errors.New("source color min contrast must be between 0 and 1")
	}
	if cfg.heroWordCount < 0 {
		return cfg, errors.New("hero word count cannot be negative")
	}
	if cfg.heroWordCount == 0 {
		cfg.heroWordCount = defaultHeroWordCount
	}
	if cfg.finalFillPasses < 0 {
		return cfg, errors.New("final fill passes cannot be negative")
	}
	if cfg.finalFillFontSize < 0 {
		return cfg, errors.New("final fill font size cannot be negative")
	}
	if cfg.finalFillFontSize == 0 {
		cfg.finalFillFontSize = cfg.minFontSize
	}
	if cfg.finalFillAnchorSamples <= 0 {
		cfg.finalFillAnchorSamples = defaultFinalFillAnchorSamples
	}
	if cfg.fillerMaxScale == 0 {
		cfg.fillerMaxScale = defaultFillerMaxScale
	}
	if cfg.fillerMaxScale < 0 || cfg.fillerMaxScale > 1 {
		return cfg, errors.New("filler max scale must be between 0 and 1")
	}
	if cfg.detailPlacementBias == 0 {
		cfg.detailPlacementBias = defaultDetailPlacementBias
	}
	if cfg.detailPlacementBias < 0 {
		return cfg, errors.New("detail placement bias cannot be negative")
	}
	if cfg.minWordLength <= 0 {
		cfg.minWordLength = 2
	}
	if cfg.maxWordLength < 0 {
		return cfg, errors.New("max word length cannot be negative")
	}
	cfg.stopWords = normalizeStopWords(cfg.stopWords)
	return cfg, nil
}

func applyQualityPresetDefaults(conf Config) (Config, error) {
	switch conf.QualityPreset {
	case "":
		return conf, nil
	case QualityPresetFast:
		if conf.FillerWordCount == 0 && conf.DensityLevel == 0 {
			conf.FillerWordCount = 300
		}
		if conf.MaxPlacementAttempts == 0 {
			conf.MaxPlacementAttempts = 1200
		}
		if conf.MaxFillerPlacementAttempts == 0 {
			conf.MaxFillerPlacementAttempts = 180
		}
		if !conf.FinalFillPassesSet && conf.FinalFillPasses == 0 {
			conf.FinalFillPasses = 0
		}
		return conf, nil
	case QualityPresetBalanced:
		if conf.FillerWordCount == 0 && conf.DensityLevel == 0 {
			conf.FillerWordCount = 900
		}
		if conf.MaxPlacementAttempts == 0 {
			conf.MaxPlacementAttempts = defaultMaxPlacementAttempts
		}
		if conf.MaxFillerPlacementAttempts == 0 {
			conf.MaxFillerPlacementAttempts = 450
		}
		if !conf.FinalFillPassesSet && conf.FinalFillPasses == 0 {
			conf.FinalFillPasses = 1
		}
		return conf, nil
	case QualityPresetDense:
		if conf.FillerWordCount == 0 && conf.DensityLevel == 0 {
			conf.FillerWordCount = 1800
		}
		if conf.MaxPlacementAttempts == 0 {
			conf.MaxPlacementAttempts = 4500
		}
		if conf.MaxFillerPlacementAttempts == 0 {
			conf.MaxFillerPlacementAttempts = 700
		}
		if !conf.FinalFillPassesSet && conf.FinalFillPasses == 0 {
			conf.FinalFillPasses = 3
		}
		return conf, nil
	case QualityPresetPoster:
		if conf.FillerWordCount == 0 && conf.DensityLevel == 0 {
			conf.FillerWordCount = 2600
		}
		if conf.MaxPlacementAttempts == 0 {
			conf.MaxPlacementAttempts = 6500
		}
		if conf.MaxFillerPlacementAttempts == 0 {
			conf.MaxFillerPlacementAttempts = 900
		}
		if !conf.FinalFillPassesSet && conf.FinalFillPasses == 0 {
			conf.FinalFillPasses = 5
		}
		return conf, nil
	default:
		return conf, fmt.Errorf("unknown quality preset %q", conf.QualityPreset)
	}
}

func applyPackingProfileDefaults(conf Config) (Config, error) {
	switch conf.PackingProfile {
	case "":
		return conf, nil
	case PackingProfileBinarySilhouette:
		if conf.ColorMode == "" {
			conf.ColorMode = ColorModePalette
		}
		if conf.FillerMaxScale == 0 {
			conf.FillerMaxScale = 0.22
		}
		if !conf.FinalFillPassesSet && conf.FinalFillPasses == 0 {
			conf.FinalFillPasses = 1
		}
		if conf.HeroWordCount == 0 {
			conf.HeroWordCount = 5
		}
		return conf, nil
	case PackingProfileTonalDetail:
		if conf.ColorMode == "" {
			conf.ColorMode = ColorModeLuminancePalette
		}
		if conf.FillerMaxScale == 0 {
			conf.FillerMaxScale = 0.16
		}
		if !conf.FinalFillPassesSet && conf.FinalFillPasses == 0 {
			conf.FinalFillPasses = 3
		}
		if !conf.WordPaddingSet && conf.WordPadding == 0 {
			conf.WordPadding = 1
		}
		if conf.HeroWordCount == 0 {
			conf.HeroWordCount = 8
		}
		if conf.MaxFillerPlacementAttempts == 0 && conf.MaxPlacementAttempts > 0 {
			conf.MaxFillerPlacementAttempts = max(600, conf.MaxPlacementAttempts/2)
		}
		if conf.DetailPlacementBias == 0 {
			conf.DetailPlacementBias = 1.0
		}
		return conf, nil
	case PackingProfileForegroundBackground:
		if conf.MaskType == "" {
			conf.MaskType = MaskTypeAll
		}
		if conf.ColorMode == "" {
			conf.ColorMode = ColorModeSource
		}
		if conf.FillerMaxScale == 0 {
			conf.FillerMaxScale = 0.12
		}
		if !conf.FinalFillPassesSet && conf.FinalFillPasses == 0 {
			conf.FinalFillPasses = 4
		}
		if !conf.WordPaddingSet && conf.WordPadding == 0 {
			conf.WordPadding = 0
		}
		if conf.HeroWordCount == 0 {
			conf.HeroWordCount = 4
		}
		if conf.DetailPlacementBias == 0 {
			conf.DetailPlacementBias = 1.5
		}
		if conf.SourceColorMinContrast == 0 {
			conf.SourceColorMinContrast = 0.22
		}
		conf.ForegroundBackground = true
		return conf, nil
	default:
		return conf, fmt.Errorf("unknown packing profile %q", conf.PackingProfile)
	}
}

func buildMask(img image.Image, maskType string, threshold float64, alphaThreshold float64, inferBackground bool) (*shapeMask, error) {
	bounds := img.Bounds()
	w, h := bounds.Dx(), bounds.Dy()
	bgR, bgG, bgB, bgA := estimatedBackgroundRGBA(img)
	mask := &shapeMask{
		width:     w,
		height:    h,
		bits:      make([]bool, w*h),
		luminance: make([]float64, w*h),
		detail:    make([]float64, w*h),
		minX:      w,
		minY:      h,
		maxX:      -1,
		maxY:      -1,
		bgColor: color.RGBA{
			R: uint8(math.Round(bgR * 255)),
			G: uint8(math.Round(bgG * 255)),
			B: uint8(math.Round(bgB * 255)),
			A: uint8(math.Round(bgA * 255)),
		},
	}

	var sumX, sumY float64
	var inferredBgR, inferredBgG, inferredBgB, inferredBgA, bgCount float64
	if threshold == defaultMaskThreshold && maskType == MaskTypeContrast {
		threshold = 0.18
	}

	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			index := y*w + x
			c := img.At(bounds.Min.X+x, bounds.Min.Y+y)
			r, g, b, a := c.RGBA()
			lum := (0.299*float64(r) + 0.587*float64(g) + 0.114*float64(b)) / maxColorChannelValue
			alpha := float64(a) / maxColorChannelValue
			mask.luminance[index] = lum

			playable := alpha >= alphaThreshold
			switch maskType {
			case MaskTypeAll:
			case MaskTypeLight:
				playable = playable && lum >= threshold
			case MaskTypeDark:
				playable = playable && lum <= threshold
			case MaskTypeContrast:
				playable = playable && backgroundContrastScore(float64(r)/maxColorChannelValue, float64(g)/maxColorChannelValue, float64(b)/maxColorChannelValue, bgR, bgG, bgB) >= threshold
			}

			if playable {
				mask.bits[index] = true
				mask.playableIndexes = append(mask.playableIndexes, index)
				mask.minX = min(mask.minX, x)
				mask.minY = min(mask.minY, y)
				mask.maxX = max(mask.maxX, x)
				mask.maxY = max(mask.maxY, y)
				sumX += float64(x)
				sumY += float64(y)
				mask.playablePx++
				continue
			}

			if inferBackground {
				inferredBgR += float64(r) / 257.0
				inferredBgG += float64(g) / 257.0
				inferredBgB += float64(b) / 257.0
				inferredBgA += float64(a) / 257.0
				bgCount++
			}
		}
	}

	if mask.playablePx == 0 {
		return nil, errors.New("mask has no playable pixels; adjust mask type or threshold")
	}

	mask.computeDetail()
	mask.centerX = sumX / float64(mask.playablePx)
	mask.centerY = sumY / float64(mask.playablePx)
	if inferBackground && bgCount > 0 {
		mask.bgColor = color.RGBA{
			R: uint8(math.Round(inferredBgR / bgCount)),
			G: uint8(math.Round(inferredBgG / bgCount)),
			B: uint8(math.Round(inferredBgB / bgCount)),
			A: uint8(math.Round(inferredBgA / bgCount)),
		}
	}

	return mask, nil
}

func estimatedBackgroundRGBA(img image.Image) (float64, float64, float64, float64) {
	bounds := img.Bounds()
	step := max(1, min(bounds.Dx(), bounds.Dy())/128)
	var rSum, gSum, bSum, aSum float64
	var samples float64
	add := func(x, y int) {
		r, g, b, a := img.At(x, y).RGBA()
		rSum += float64(r) / maxColorChannelValue
		gSum += float64(g) / maxColorChannelValue
		bSum += float64(b) / maxColorChannelValue
		aSum += float64(a) / maxColorChannelValue
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
		return 1, 1, 1, 1
	}
	return rSum / samples, gSum / samples, bSum / samples, aSum / samples
}

func backgroundContrastScore(r, g, b, bgR, bgG, bgB float64) float64 {
	lum := relativeLuminance(r, g, b)
	bgLum := relativeLuminance(bgR, bgG, bgB)
	contrastRatio := (math.Max(lum, bgLum) + 0.05) / (math.Min(lum, bgLum) + 0.05)
	luminanceScore := math.Min(1, (contrastRatio-1)/6)
	colorDistance := math.Sqrt((r-bgR)*(r-bgR)+(g-bgG)*(g-bgG)+(b-bgB)*(b-bgB)) / math.Sqrt(3)
	return math.Max(luminanceScore, colorDistance)
}

func relativeLuminance(r, g, b float64) float64 {
	linear := func(v float64) float64 {
		if v <= 0.03928 {
			return v / 12.92
		}
		return math.Pow((v+0.055)/1.055, 2.4)
	}
	return 0.2126*linear(r) + 0.7152*linear(g) + 0.0722*linear(b)
}

func buildForegroundBits(maskImg image.Image, baseMask *shapeMask, threshold float64, alphaThreshold float64) ([]bool, error) {
	bounds := maskImg.Bounds()
	if bounds.Dx() != baseMask.width || bounds.Dy() != baseMask.height {
		maskImg = imaging.Resize(maskImg, baseMask.width, baseMask.height, imaging.NearestNeighbor)
		bounds = maskImg.Bounds()
	}

	bits := make([]bool, baseMask.width*baseMask.height)
	var count int
	for y := 0; y < baseMask.height; y++ {
		for x := 0; x < baseMask.width; x++ {
			index := y*baseMask.width + x
			if !baseMask.bits[index] {
				continue
			}
			r, g, b, a := maskImg.At(bounds.Min.X+x, bounds.Min.Y+y).RGBA()
			lum := (0.299*float64(r) + 0.587*float64(g) + 0.114*float64(b)) / maxColorChannelValue
			alpha := float64(a) / maxColorChannelValue
			if alpha >= alphaThreshold && lum >= threshold {
				bits[index] = true
				count++
			}
		}
	}
	if count == 0 {
		return nil, errors.New("foreground mask did not contain any foreground pixels")
	}
	return bits, nil
}

func deriveMask(base *shapeMask, include func(index int) bool) *shapeMask {
	mask := &shapeMask{
		width:     base.width,
		height:    base.height,
		bits:      make([]bool, len(base.bits)),
		luminance: base.luminance,
		detail:    base.detail,
		minX:      base.width,
		minY:      base.height,
		maxX:      -1,
		maxY:      -1,
		bgColor:   base.bgColor,
	}

	var sumX, sumY float64
	for index := range base.bits {
		if !include(index) {
			continue
		}
		x := index % base.width
		y := index / base.width
		mask.bits[index] = true
		mask.playableIndexes = append(mask.playableIndexes, index)
		mask.minX = min(mask.minX, x)
		mask.minY = min(mask.minY, y)
		mask.maxX = max(mask.maxX, x)
		mask.maxY = max(mask.maxY, y)
		sumX += float64(x)
		sumY += float64(y)
		mask.playablePx++
	}

	if mask.playablePx > 0 {
		mask.centerX = sumX / float64(mask.playablePx)
		mask.centerY = sumY / float64(mask.playablePx)
	}
	return mask
}

func (m *shapeMask) computeDetail() {
	if len(m.detail) == 0 || len(m.luminance) == 0 {
		return
	}
	for y := 1; y < m.height-1; y++ {
		for x := 1; x < m.width-1; x++ {
			index := y*m.width + x
			gx := math.Abs(m.luminance[index-1] - m.luminance[index+1])
			gy := math.Abs(m.luminance[index-m.width] - m.luminance[index+m.width])
			m.detail[index] = math.Min(1, gx+gy)
		}
	}
}

func parseWordStats(text string, cfg settings) ([]wordStat, error) {
	fields := strings.FieldsFunc(text, func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsNumber(r) && r != '\'' && r != '-'
	})

	counts := make(map[string]int)
	for _, field := range fields {
		word := normalizeWord(field)
		length := len([]rune(word))
		if length < cfg.minWordLength {
			continue
		}
		if cfg.maxWordLength > 0 && length > cfg.maxWordLength {
			continue
		}
		if _, blocked := cfg.stopWords[word]; blocked {
			continue
		}
		counts[word]++
	}
	if len(counts) == 0 {
		return nil, errors.New("text must contain at least one usable word")
	}

	stats := make([]wordStat, 0, len(counts))
	for word, count := range counts {
		stats = append(stats, wordStat{Word: word, Count: count})
	}
	sort.Slice(stats, func(i, j int) bool {
		if stats[i].Count == stats[j].Count {
			return stats[i].Word < stats[j].Word
		}
		return stats[i].Count > stats[j].Count
	})

	return stats, nil
}

func normalizeWord(field string) string {
	word := strings.ToUpper(strings.Trim(field, "'- "))
	word = strings.ReplaceAll(word, "’", "'")
	return word
}

func normalizeStopWords(stopWords map[string]struct{}) map[string]struct{} {
	if len(stopWords) == 0 {
		return nil
	}
	normalized := make(map[string]struct{}, len(stopWords))
	for word := range stopWords {
		word = normalizeWord(word)
		if word != "" {
			normalized[word] = struct{}{}
		}
	}
	return normalized
}

func buildHierarchy(stats []wordStat, minFontSize, maxFontSize, fillerWordCount int, exponent float64, fillerMaxScale float64, rng *rand.Rand) []wordCandidate {
	maxCount := stats[0].Count
	candidates := make([]wordCandidate, 0, len(stats)+fillerWordCount)

	for _, stat := range stats {
		weight := math.Pow(float64(stat.Count)/float64(maxCount), exponent)
		size := float64(minFontSize) + weight*float64(maxFontSize-minFontSize)
		candidates = append(candidates, wordCandidate{Word: stat.Word, Size: size})
	}

	if fillerWordCount <= 0 {
		return candidates
	}

	totalWeight := 0
	for _, stat := range stats {
		totalWeight += stat.Count
	}
	sizeRange := float64(maxFontSize - minFontSize)
	for i := 0; i < fillerWordCount; i++ {
		level := float64(i) / float64(max(1, fillerWordCount-1))
		size := float64(minFontSize) + sizeRange*fillerMaxScale*math.Pow(1-level, 2)
		candidates = append(candidates, wordCandidate{
			Word: weightedWord(stats, totalWeight, rng),
			Size: size,
		})
	}

	return candidates
}

func weightedWord(stats []wordStat, totalWeight int, rng *rand.Rand) string {
	if totalWeight <= 0 {
		return stats[0].Word
	}
	n := rng.Intn(totalWeight)
	for _, stat := range stats {
		if n < stat.Count {
			return stat.Word
		}
		n -= stat.Count
	}
	return stats[len(stats)-1].Word
}

func newPacker(mask *shapeMask, source image.Image, fontPath string, cfg settings) *packer {
	cellSize := max(cfg.minFontSize*3, 12)
	anchors := mask.anchors()
	if cfg.detailPlacementBias > 0 {
		anchors = appendUniqueAnchors(anchors, mask.detailPeakAnchors(24))
	}
	return &packer{
		mask:                   mask,
		source:                 source,
		fontPath:               fontPath,
		minFontSize:            cfg.minFontSize,
		wordPadding:            cfg.wordPadding,
		glyphAlphaThreshold:    cfg.glyphAlphaThreshold,
		maxAttempts:            cfg.maxPlacementAttempts,
		maxHeroAttempts:        cfg.maxHeroAttempts,
		maxFillerAttempts:      cfg.maxFillerAttempts,
		heroWordCount:          cfg.heroWordCount,
		finalFillPasses:        cfg.finalFillPasses,
		finalFillFontSize:      cfg.finalFillFontSize,
		finalFillAnchorSamples: cfg.finalFillAnchorSamples,
		colorMode:              cfg.colorMode,
		palette:                cfg.palette,
		sourceColorMinContrast: cfg.sourceColorMinContrast,
		invertLuminancePalette: cfg.invertLuminancePalette,
		layoutRNG:              rand.New(rand.NewSource(seedOffset(cfg.seed, 1))),
		colorRNG:               rand.New(rand.NewSource(seedOffset(cfg.seed, 2))),
		detailPlacementBias:    cfg.detailPlacementBias,
		occupancy:              make([]bool, mask.width*mask.height),
		hash:                   newSpatialHash(cellSize),
		glyphs:                 make(map[glyphKey]*glyphBitmap),
		anchors:                anchors,
	}
}

func seedOffset(seed int64, offset int64) int64 {
	return seed + offset*1_000_003
}

func appendUniqueAnchors(base []pointF, extra []pointF) []pointF {
	for _, point := range extra {
		keep := true
		for _, existing := range base {
			if math.Hypot(existing.x-point.x, existing.y-point.y) < 3 {
				keep = false
				break
			}
		}
		if keep {
			base = append(base, point)
		}
	}
	return base
}

func (p *packer) pack(candidates []wordCandidate) []placedWord {
	placed := make([]placedWord, 0, len(candidates))
	for i, candidate := range candidates {
		p.stats.AttemptedWords++
		word, ok := p.place(candidate, i)
		if !ok {
			p.stats.RejectedWords++
			continue
		}
		p.stamp(word)
		p.hash.add(word.box.expand(p.wordPadding))
		p.stats.PlacedWords++
		placed = append(placed, word)
	}
	return placed
}

func (p *packer) finalFill(stats []wordStat) []placedWord {
	var placed []placedWord
	if len(stats) == 0 || p.finalFillPasses <= 0 {
		return placed
	}

	total := p.finalFillPasses * len(stats)
	for pass := 0; pass < total; pass++ {
		anchor, ok := p.findFinalFillAnchor()
		if !ok {
			return placed
		}
		stat := stats[pass%len(stats)]
		candidate := wordCandidate{Word: stat.Word, Size: float64(p.finalFillFontSize)}
		p.stats.AttemptedWords++
		word, ok := p.placeFromAnchor(candidate, pass+p.stats.AttemptedWords, anchor)
		if !ok {
			p.stats.RejectedWords++
			continue
		}
		p.stamp(word)
		p.hash.add(word.box.expand(p.wordPadding))
		p.stats.PlacedWords++
		placed = append(placed, word)
	}
	return placed
}

func (p *packer) findFinalFillAnchor() (pointF, bool) {
	if len(p.mask.playableIndexes) == 0 {
		return pointF{}, false
	}

	bestIndex := -1
	bestScore := -1
	for i := 0; i < p.finalFillAnchorSamples; i++ {
		index := p.mask.playableIndexes[p.layoutRNG.Intn(len(p.mask.playableIndexes))]
		if p.occupancy[index] {
			continue
		}
		score := p.freeNeighborhoodScore(index%p.mask.width, index/p.mask.width, max(2, p.finalFillFontSize))
		if score > bestScore {
			bestScore = score
			bestIndex = index
		}
	}
	if bestIndex < 0 {
		return pointF{}, false
	}
	return pointF{
		x: float64(bestIndex % p.mask.width),
		y: float64(bestIndex / p.mask.width),
	}, true
}

func (p *packer) freeNeighborhoodScore(x, y int, radius int) int {
	score := 0.0
	for yy := y - radius; yy <= y+radius; yy++ {
		if yy < 0 || yy >= p.mask.height {
			continue
		}
		for xx := x - radius; xx <= x+radius; xx++ {
			if xx < 0 || xx >= p.mask.width {
				continue
			}
			if (xx-x)*(xx-x)+(yy-y)*(yy-y) > radius*radius {
				continue
			}
			index := yy*p.mask.width + xx
			if p.mask.bits[index] && !p.occupancy[index] {
				score += 1 + p.detailPlacementBias*p.mask.detail[index]*8
			}
		}
	}
	return int(math.Round(score))
}

func (p *packer) place(candidate wordCandidate, index int) (placedWord, bool) {
	return p.placeWithAnchor(candidate, index, nil)
}

func (p *packer) placeFromAnchor(candidate wordCandidate, index int, anchor pointF) (placedWord, bool) {
	return p.placeWithAnchor(candidate, index, &anchor)
}

func (p *packer) placeWithAnchor(candidate wordCandidate, index int, preferredAnchor *pointF) (placedWord, bool) {
	renderFailed := false
	tooLarge := false
	attemptFailed := false
	attemptLimit := p.maxFillerAttempts
	if index < p.heroWordCount {
		attemptLimit = p.maxHeroAttempts
	}
	for _, size := range retrySizes(candidate.Size, float64(p.minFontSize)) {
		rotations := []bool{false, true}
		if p.layoutRNG.Intn(2) == 1 {
			rotations[0], rotations[1] = rotations[1], rotations[0]
		}

		for _, rotated := range rotations {
			glyph, err := p.renderGlyph(candidate.Word, size, rotated)
			if err != nil {
				renderFailed = true
				continue
			}
			if glyph.width >= p.mask.width || glyph.height >= p.mask.height {
				tooLarge = true
				continue
			}

			x, y, ok := p.search(glyph, index, attemptLimit, preferredAnchor)
			if !ok {
				attemptFailed = true
				continue
			}

			word := placedWord{
				glyph: glyph,
				x:     x,
				y:     y,
				box: rect{
					left:   x,
					top:    y,
					right:  x + glyph.width,
					bottom: y + glyph.height,
				},
				color: p.chooseColor(glyph, x, y, index, size),
			}
			return word, true
		}
	}

	switch {
	case renderFailed:
		p.stats.WordsRejectedRenderFailure++
	case tooLarge:
		p.stats.WordsRejectedTooLarge++
	case attemptFailed:
		p.stats.WordsRejectedAfterAttempts++
	}
	return placedWord{}, false
}

func retrySizes(size, minSize float64) []float64 {
	sizes := []float64{size}
	for _, factor := range []float64{0.82, 0.66, 0.50} {
		next := math.Max(minSize, math.Round(size*factor))
		if next < sizes[len(sizes)-1] {
			sizes = append(sizes, next)
		}
	}
	if sizes[len(sizes)-1] != minSize {
		sizes = append(sizes, minSize)
	}
	return sizes
}

func (p *packer) renderGlyph(word string, size float64, rotated bool) (*glyphBitmap, error) {
	key := glyphKey{
		word:           word,
		size:           int(math.Round(size * 10)),
		rotated:        rotated,
		alphaThreshold: int(math.Round(p.glyphAlphaThreshold * 1000)),
	}
	if glyph, ok := p.glyphs[key]; ok {
		return glyph, nil
	}

	glyph, err := renderGlyphBitmap(p.fontPath, word, size, p.glyphAlphaThreshold)
	if err != nil {
		return nil, err
	}
	if rotated {
		glyph = rotateGlyph(glyph)
	}
	glyph.rotated = rotated
	p.glyphs[key] = glyph
	return glyph, nil
}

func renderGlyphBitmap(fontPath string, word string, size float64, alphaThreshold float64) (*glyphBitmap, error) {
	measureCtx := gg.NewContext(1, 1)
	if err := measureCtx.LoadFontFace(fontPath, size); err != nil {
		return nil, fmt.Errorf("load font face %q: %w", fontPath, err)
	}
	textW, textH := measureCtx.MeasureString(word)
	if textW <= 0 || textH <= 0 {
		return nil, errors.New("word rendered with empty bounds")
	}

	padding := int(math.Ceil(size * 1.5))
	canvasW := max(1, int(math.Ceil(textW))+2*padding)
	canvasH := max(1, int(math.Ceil(size*3.0))+2*padding)
	img := image.NewRGBA(image.Rect(0, 0, canvasW, canvasH))
	ctx := gg.NewContextForRGBA(img)
	if err := ctx.LoadFontFace(fontPath, size); err != nil {
		return nil, fmt.Errorf("load font face %q: %w", fontPath, err)
	}
	ctx.SetRGBA(0, 0, 0, 1)
	ctx.DrawStringAnchored(word, float64(canvasW)/2, float64(canvasH)/2, 0.5, 0.5)

	minAlpha := uint8(math.Round(alphaThreshold * 255))
	minX, minY := canvasW, canvasH
	maxX, maxY := -1, -1
	for y := 0; y < canvasH; y++ {
		for x := 0; x < canvasW; x++ {
			if alphaAt(img, x, y) <= minAlpha {
				continue
			}
			minX = min(minX, x)
			minY = min(minY, y)
			maxX = max(maxX, x)
			maxY = max(maxY, y)
		}
	}
	if maxX < minX || maxY < minY {
		return nil, errors.New("word rendered no visible glyph pixels")
	}

	width := maxX - minX + 1
	height := maxY - minY + 1
	pixels := make([]glyphPixel, 0, width*height/2)
	for y := minY; y <= maxY; y++ {
		for x := minX; x <= maxX; x++ {
			alpha := alphaAt(img, x, y)
			if alpha <= minAlpha {
				continue
			}
			pixels = append(pixels, glyphPixel{
				x:     x - minX,
				y:     y - minY,
				alpha: alpha,
			})
		}
	}

	return &glyphBitmap{
		word:   word,
		size:   size,
		width:  width,
		height: height,
		pixels: pixels,
		spans:  pixelsToSpans(pixels),
	}, nil
}

func rotateGlyph(src *glyphBitmap) *glyphBitmap {
	pixels := make([]glyphPixel, len(src.pixels))
	for i, pixel := range src.pixels {
		pixels[i] = glyphPixel{
			x:     src.height - 1 - pixel.y,
			y:     pixel.x,
			alpha: pixel.alpha,
		}
	}
	sort.Slice(pixels, func(i, j int) bool {
		if pixels[i].y == pixels[j].y {
			return pixels[i].x < pixels[j].x
		}
		return pixels[i].y < pixels[j].y
	})
	return &glyphBitmap{
		word:    src.word,
		size:    src.size,
		width:   src.height,
		height:  src.width,
		pixels:  pixels,
		spans:   pixelsToSpans(pixels),
		rotated: true,
	}
}

func pixelsToSpans(pixels []glyphPixel) []glyphSpan {
	if len(pixels) == 0 {
		return nil
	}
	spans := make([]glyphSpan, 0)
	current := glyphSpan{y: pixels[0].y, x1: pixels[0].x, x2: pixels[0].x}
	for _, pixel := range pixels[1:] {
		if pixel.y == current.y && pixel.x == current.x2+1 {
			current.x2 = pixel.x
			continue
		}
		spans = append(spans, current)
		current = glyphSpan{y: pixel.y, x1: pixel.x, x2: pixel.x}
	}
	spans = append(spans, current)
	return spans
}

func alphaAt(img *image.RGBA, x int, y int) uint8 {
	offset := img.PixOffset(x, y)
	return img.Pix[offset+3]
}

func (p *packer) search(glyph *glyphBitmap, index int, attemptLimit int, preferredAnchor *pointF) (int, int, bool) {
	anchors := p.anchors
	if preferredAnchor != nil {
		anchors = append([]pointF{*preferredAnchor}, p.anchors...)
	}
	if len(anchors) == 0 {
		return 0, 0, false
	}

	anchorOffset := index % len(anchors)
	if preferredAnchor != nil {
		anchorOffset = 0
	} else if index >= p.heroWordCount {
		anchorOffset = p.layoutRNG.Intn(len(anchors))
	}

	maxRadius := math.Hypot(float64(p.mask.width), float64(p.mask.height))
	spacing := math.Max(1.0, math.Min(float64(glyph.width), float64(glyph.height))*0.20)
	thetaLimit := maxRadius/spacing + 2*math.Pi
	maxDim := math.Max(float64(glyph.width), float64(glyph.height))
	thetaStep := math.Max(0.045, math.Min(0.22, 1.5/math.Max(8, maxDim)))
	phase := float64(index%19)*0.37 + p.layoutRNG.Float64()*0.20
	if preferredAnchor != nil {
		phase = 0
	}
	attempts := 0

	for pass := 0; pass < len(anchors); pass++ {
		anchor := anchors[(anchorOffset+pass)%len(anchors)]
		for theta := phase; theta <= thetaLimit+phase; theta += thetaStep {
			if attempts >= attemptLimit {
				return 0, 0, false
			}
			attempts++
			p.stats.PlacementChecks++
			radius := spacing * theta
			centerX := anchor.x + radius*math.Cos(theta)
			centerY := anchor.y + radius*math.Sin(theta)
			x := int(math.Round(centerX - float64(glyph.width)/2))
			y := int(math.Round(centerY - float64(glyph.height)/2))

			if p.canPlace(glyph, x, y) {
				return x, y, true
			}
		}
	}

	return 0, 0, false
}

func (m *shapeMask) anchors() []pointF {
	points := []pointF{{x: m.centerX, y: m.centerY}}
	addPoint := func(x, y int) {
		if !m.playable(x, y) {
			return
		}
		for _, existing := range points {
			if math.Abs(existing.x-float64(x)) < 2 && math.Abs(existing.y-float64(y)) < 2 {
				return
			}
		}
		points = append(points, pointF{x: float64(x), y: float64(y)})
	}

	addPoint((m.minX+m.maxX)/2, (m.minY+m.maxY)/2)
	for gy := 1; gy <= 9; gy++ {
		for gx := 1; gx <= 9; gx++ {
			x := m.minX + gx*(m.maxX-m.minX)/10
			y := m.minY + gy*(m.maxY-m.minY)/10
			addPoint(x, y)
		}
	}
	for _, point := range m.distancePeakAnchors(24) {
		addPoint(int(math.Round(point.x)), int(math.Round(point.y)))
	}
	return points
}

func (m *shapeMask) distancePeakAnchors(limit int) []pointF {
	type candidate struct {
		x     int
		y     int
		score int
	}
	step := max(4, min(m.width, m.height)/48)
	candidates := make([]candidate, 0)
	for y := m.minY; y <= m.maxY; y += step {
		for x := m.minX; x <= m.maxX; x += step {
			if !m.playable(x, y) {
				continue
			}
			score := m.approxDistanceToBoundary(x, y, step)
			candidates = append(candidates, candidate{x: x, y: y, score: score})
		}
	}
	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].score > candidates[j].score
	})

	anchors := make([]pointF, 0, min(limit, len(candidates)))
	minSeparation := float64(step * 3)
	for _, candidate := range candidates {
		if len(anchors) >= limit {
			break
		}
		keep := true
		for _, existing := range anchors {
			if math.Hypot(existing.x-float64(candidate.x), existing.y-float64(candidate.y)) < minSeparation {
				keep = false
				break
			}
		}
		if keep {
			anchors = append(anchors, pointF{x: float64(candidate.x), y: float64(candidate.y)})
		}
	}
	return anchors
}

func (m *shapeMask) detailPeakAnchors(limit int) []pointF {
	type candidate struct {
		x     int
		y     int
		score float64
	}
	if len(m.detail) == 0 {
		return nil
	}
	step := max(4, min(m.width, m.height)/56)
	candidates := make([]candidate, 0)
	for y := m.minY; y <= m.maxY; y += step {
		for x := m.minX; x <= m.maxX; x += step {
			if !m.playable(x, y) {
				continue
			}
			score := m.localDetailScore(x, y, step)
			if score > 0 {
				candidates = append(candidates, candidate{x: x, y: y, score: score})
			}
		}
	}
	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].score > candidates[j].score
	})

	anchors := make([]pointF, 0, min(limit, len(candidates)))
	minSeparation := float64(step * 3)
	for _, candidate := range candidates {
		if len(anchors) >= limit {
			break
		}
		keep := true
		for _, existing := range anchors {
			if math.Hypot(existing.x-float64(candidate.x), existing.y-float64(candidate.y)) < minSeparation {
				keep = false
				break
			}
		}
		if keep {
			anchors = append(anchors, pointF{x: float64(candidate.x), y: float64(candidate.y)})
		}
	}
	return anchors
}

func (m *shapeMask) localDetailScore(x, y int, radius int) float64 {
	var score float64
	for yy := y - radius; yy <= y+radius; yy++ {
		if yy < 0 || yy >= m.height {
			continue
		}
		for xx := x - radius; xx <= x+radius; xx++ {
			if xx < 0 || xx >= m.width {
				continue
			}
			index := yy*m.width + xx
			if m.bits[index] {
				score += m.detail[index]
			}
		}
	}
	return score
}

func (m *shapeMask) approxDistanceToBoundary(x, y int, step int) int {
	for radius := step; radius < max(m.width, m.height); radius += step {
		if !m.playable(x-radius, y) || !m.playable(x+radius, y) ||
			!m.playable(x, y-radius) || !m.playable(x, y+radius) ||
			!m.playable(x-radius, y-radius) || !m.playable(x+radius, y-radius) ||
			!m.playable(x-radius, y+radius) || !m.playable(x+radius, y+radius) {
			return radius
		}
	}
	return max(m.width, m.height)
}

func (p *packer) canPlace(glyph *glyphBitmap, x int, y int) bool {
	box := rect{left: x, top: y, right: x + glyph.width, bottom: y + glyph.height}
	if box.left < 0 || box.top < 0 || box.right > p.mask.width || box.bottom > p.mask.height {
		p.stats.MaskCheckFailures++
		return false
	}
	if p.hash.hasOverlap(box.expand(p.wordPadding)) {
		if p.glyphOverlapsOccupied(glyph, x, y) {
			p.stats.CollisionCheckFailures++
			return false
		}
	}
	if !p.glyphInsideMask(glyph, x, y) {
		p.stats.MaskCheckFailures++
		return false
	}
	return true
}

func (p *packer) glyphInsideMask(glyph *glyphBitmap, x int, y int) bool {
	for _, pixel := range glyph.pixels {
		if !p.mask.playable(x+pixel.x, y+pixel.y) {
			return false
		}
	}
	return true
}

func (p *packer) glyphOverlapsOccupied(glyph *glyphBitmap, x int, y int) bool {
	padding := p.wordPadding
	for _, pixel := range glyph.pixels {
		for dy := -padding; dy <= padding; dy++ {
			targetY := y + pixel.y + dy
			if targetY < 0 || targetY >= p.mask.height {
				continue
			}
			for dx := -padding; dx <= padding; dx++ {
				if padding > 0 && dx*dx+dy*dy > padding*padding {
					continue
				}
				targetX := x + pixel.x + dx
				if targetX < 0 || targetX >= p.mask.width {
					continue
				}
				if p.occupancy[targetY*p.mask.width+targetX] {
					return true
				}
			}
		}
	}
	return false
}

func (p *packer) stamp(word placedWord) {
	padding := p.wordPadding
	for _, pixel := range word.glyph.pixels {
		for dy := -padding; dy <= padding; dy++ {
			targetY := word.y + pixel.y + dy
			if targetY < 0 || targetY >= p.mask.height {
				continue
			}
			for dx := -padding; dx <= padding; dx++ {
				if padding > 0 && dx*dx+dy*dy > padding*padding {
					continue
				}
				targetX := word.x + pixel.x + dx
				if targetX < 0 || targetX >= p.mask.width {
					continue
				}
				index := targetY*p.mask.width + targetX
				if p.mask.bits[index] {
					p.occupancy[index] = true
				}
			}
		}
	}
}

func (m *shapeMask) playable(x, y int) bool {
	if x < 0 || y < 0 || x >= m.width || y >= m.height {
		return false
	}
	return m.bits[y*m.width+x]
}

func (m *shapeMask) lum(x, y int) float64 {
	if x < 0 || y < 0 || x >= m.width || y >= m.height {
		return 0
	}
	return m.luminance[y*m.width+x]
}

func (p *packer) chooseColor(glyph *glyphBitmap, x int, y int, index int, size float64) color.Color {
	switch p.colorMode {
	case ColorModeSource:
		return pushColorFromBackground(averageSourceColor(p.source, glyph, x, y), p.mask.bgColor, p.sourceColorMinContrast)
	case ColorModeLuminancePalette:
		return p.paletteByLuminance(glyph, x, y)
	case ColorModePalette:
		paletteIndex := stablePaletteIndex(glyph.word, index, len(p.palette))
		return p.palette[paletteIndex]
	case ColorModeRandomPalette:
		return p.palette[p.colorRNG.Intn(len(p.palette))]
	case ColorModeSequentialPalette:
		return p.palette[index%len(p.palette)]
	default:
		paletteIndex := (index*31 + int(math.Round(size))*17) % len(p.palette)
		return p.palette[paletteIndex]
	}
}

func pushColorFromBackground(c color.Color, bg color.Color, minDiff float64) color.Color {
	if minDiff <= 0 {
		return c
	}
	r, g, b, a := rgba8(c)
	bgR, bgG, bgB, _ := rgba8(bg)
	lum := luminance8(r, g, b)
	bgLum := luminance8(bgR, bgG, bgB)
	if math.Abs(lum-bgLum) >= minDiff {
		return c
	}

	if bgLum >= 0.5 {
		target := math.Max(0, bgLum-minDiff)
		scale := target / math.Max(lum, 0.001)
		return color.NRGBA{R: uint8(clampFloat(float64(r)*scale, 0, 255)), G: uint8(clampFloat(float64(g)*scale, 0, 255)), B: uint8(clampFloat(float64(b)*scale, 0, 255)), A: a}
	}

	target := math.Min(1, bgLum+minDiff)
	scale := (target - lum) / math.Max(1-lum, 0.001)
	return color.NRGBA{
		R: uint8(clampFloat(float64(r)+(255-float64(r))*scale, 0, 255)),
		G: uint8(clampFloat(float64(g)+(255-float64(g))*scale, 0, 255)),
		B: uint8(clampFloat(float64(b)+(255-float64(b))*scale, 0, 255)),
		A: a,
	}
}

func luminance8(r, g, b uint8) float64 {
	return (0.299*float64(r) + 0.587*float64(g) + 0.114*float64(b)) / 255
}

func clampFloat(v, minV, maxV float64) float64 {
	if v < minV {
		return minV
	}
	if v > maxV {
		return maxV
	}
	return v
}

func stablePaletteIndex(word string, index int, length int) int {
	if length <= 1 {
		return 0
	}
	hash := uint32(2166136261)
	for _, r := range word {
		hash ^= uint32(r)
		hash *= 16777619
	}
	hash ^= uint32(index * 374761393)
	return int(hash % uint32(length))
}

func (p *packer) paletteByLuminance(glyph *glyphBitmap, x int, y int) color.Color {
	if len(p.palette) == 1 {
		return p.palette[0]
	}
	lum := p.averageLuminance(glyph, x, y)
	if p.invertLuminancePalette {
		lum = 1 - lum
	}
	index := int(math.Round(lum * float64(len(p.palette)-1)))
	return p.palette[min(max(index, 0), len(p.palette)-1)]
}

func (p *packer) averageLuminance(glyph *glyphBitmap, x int, y int) float64 {
	if len(glyph.pixels) == 0 {
		return p.mask.lum(x, y)
	}
	var weightedLum float64
	var weight float64
	for _, pixel := range glyph.pixels {
		alpha := float64(pixel.alpha) / 255
		weightedLum += p.mask.lum(x+pixel.x, y+pixel.y) * alpha
		weight += alpha
	}
	if weight == 0 {
		return p.mask.lum(x, y)
	}
	return weightedLum / weight
}

func (p *packer) finalStats() Stats {
	stats := p.stats
	stats.MaskCoverage = float64(p.mask.playablePx) / float64(p.mask.width*p.mask.height)
	var occupied int
	for i, ok := range p.occupancy {
		if ok && p.mask.bits[i] {
			occupied++
		}
	}
	if p.mask.playablePx > 0 {
		stats.OccupiedCoverage = float64(occupied) / float64(p.mask.playablePx)
	}
	return stats
}

func drawWords(width int, height int, bg color.Color, words []placedWord) image.Image {
	out := image.NewRGBA(image.Rect(0, 0, width, height))
	fill(out, bg)
	for _, word := range words {
		stampGlyph(out, word)
	}
	return out
}

func fill(img *image.RGBA, c color.Color) {
	r, g, b, a := rgba8(c)
	for y := img.Bounds().Min.Y; y < img.Bounds().Max.Y; y++ {
		for x := img.Bounds().Min.X; x < img.Bounds().Max.X; x++ {
			img.SetRGBA(x, y, color.RGBA{R: r, G: g, B: b, A: a})
		}
	}
}

func stampGlyph(out *image.RGBA, word placedWord) {
	sr, sg, sb, sa := rgba8(word.color)
	for _, pixel := range word.glyph.pixels {
		x := word.x + pixel.x
		y := word.y + pixel.y
		effectiveAlpha := uint8((uint16(sa) * uint16(pixel.alpha)) / 255)
		blendPixel(out, x, y, sr, sg, sb, effectiveAlpha)
	}
}

func blendPixel(img *image.RGBA, x int, y int, sr uint8, sg uint8, sb uint8, sa uint8) {
	offset := img.PixOffset(x, y)
	dr := img.Pix[offset]
	dg := img.Pix[offset+1]
	db := img.Pix[offset+2]
	da := img.Pix[offset+3]

	invA := 255 - int(sa)
	outA := int(sa) + int(da)*invA/255
	if outA == 0 {
		img.Pix[offset] = 0
		img.Pix[offset+1] = 0
		img.Pix[offset+2] = 0
		img.Pix[offset+3] = 0
		return
	}

	dstPremulR := int(dr) * int(da) / 255
	dstPremulG := int(dg) * int(da) / 255
	dstPremulB := int(db) * int(da) / 255
	outPremulR := int(sr)*int(sa)/255 + dstPremulR*invA/255
	outPremulG := int(sg)*int(sa)/255 + dstPremulG*invA/255
	outPremulB := int(sb)*int(sa)/255 + dstPremulB*invA/255

	img.Pix[offset] = uint8(outPremulR * 255 / outA)
	img.Pix[offset+1] = uint8(outPremulG * 255 / outA)
	img.Pix[offset+2] = uint8(outPremulB * 255 / outA)
	img.Pix[offset+3] = uint8(outA)
}

func rgba8(c color.Color) (uint8, uint8, uint8, uint8) {
	if c == nil {
		return 255, 255, 255, 255
	}
	r, g, b, a := c.RGBA()
	return uint8(r / 257), uint8(g / 257), uint8(b / 257), uint8(a / 257)
}

func sampleColor(img image.Image, x, y int) color.Color {
	bounds := img.Bounds()
	x = min(max(x, 0), bounds.Dx()-1)
	y = min(max(y, 0), bounds.Dy()-1)
	return img.At(bounds.Min.X+x, bounds.Min.Y+y)
}

func averageSourceColor(img image.Image, glyph *glyphBitmap, x, y int) color.Color {
	if len(glyph.pixels) == 0 {
		return sampleColor(img, x, y)
	}
	var rSum, gSum, bSum, aSum, weightSum float64
	bounds := img.Bounds()
	for _, pixel := range glyph.pixels {
		targetX := min(max(x+pixel.x, 0), bounds.Dx()-1)
		targetY := min(max(y+pixel.y, 0), bounds.Dy()-1)
		r, g, b, a := img.At(bounds.Min.X+targetX, bounds.Min.Y+targetY).RGBA()
		weight := float64(pixel.alpha) / 255
		rSum += (float64(r) / 257.0) * weight
		gSum += (float64(g) / 257.0) * weight
		bSum += (float64(b) / 257.0) * weight
		aSum += (float64(a) / 257.0) * weight
		weightSum += weight
	}
	if weightSum == 0 {
		return sampleColor(img, x, y)
	}
	return color.RGBA{
		R: uint8(math.Round(rSum / weightSum)),
		G: uint8(math.Round(gSum / weightSum)),
		B: uint8(math.Round(bSum / weightSum)),
		A: uint8(math.Round(aSum / weightSum)),
	}
}

func newSpatialHash(cellSize int) *spatialHash {
	return &spatialHash{
		cellSize: max(1, cellSize),
		cells:    make(map[cell][]int),
	}
}

func (s *spatialHash) add(box rect) {
	index := len(s.boxes)
	s.boxes = append(s.boxes, box)
	minCellX, minCellY, maxCellX, maxCellY := s.cellRange(box)
	for y := minCellY; y <= maxCellY; y++ {
		for x := minCellX; x <= maxCellX; x++ {
			c := cell{x: x, y: y}
			s.cells[c] = append(s.cells[c], index)
		}
	}
}

func (s *spatialHash) hasOverlap(box rect) bool {
	minCellX, minCellY, maxCellX, maxCellY := s.cellRange(box)
	for y := minCellY; y <= maxCellY; y++ {
		for x := minCellX; x <= maxCellX; x++ {
			for _, index := range s.cells[cell{x: x, y: y}] {
				if box.overlaps(s.boxes[index]) {
					return true
				}
			}
		}
	}
	return false
}

func (s *spatialHash) cellRange(box rect) (int, int, int, int) {
	return int(math.Floor(float64(box.left) / float64(s.cellSize))),
		int(math.Floor(float64(box.top) / float64(s.cellSize))),
		int(math.Floor(float64(box.right) / float64(s.cellSize))),
		int(math.Floor(float64(box.bottom) / float64(s.cellSize)))
}

func (a rect) overlaps(b rect) bool {
	return a.left < b.right && a.right > b.left && a.top < b.bottom && a.bottom > b.top
}

func (a rect) expand(padding int) rect {
	if padding <= 0 {
		return a
	}
	return rect{
		left:   a.left - padding,
		top:    a.top - padding,
		right:  a.right + padding,
		bottom: a.bottom + padding,
	}
}
