package wordcloud

import (
	"errors"
	"strings"
)

// Config controls word-cloud generation.
//
// Defaults are provided by NewConfig. Most callers only need to supply the
// input image, text file, font, and optional output path.
type Config struct {
	// InputPath is the source image used to derive the placement silhouette.
	InputPath string
	// OutputPath is the target PNG. When empty, Generate derives one from InputPath.
	OutputPath string
	// TextPath is the UTF-8 text source whose word frequencies drive the cloud.
	TextPath string
	// FontPath points to the TTF/OTF font used for measurement and rendering.
	FontPath string

	// MinFontSize is the smallest size placement is allowed to try.
	MinFontSize float64
	// MaxFontSize is the initial upper bound for the most important words.
	MaxFontSize float64
	// WordLimit is the maximum number of candidate words considered. It is not
	// a promise that every candidate will be placed.
	WordLimit int

	// SpiralStepsPerCenter limits how many spiral positions are sampled around
	// each placement center for one orientation and font size.
	SpiralStepsPerCenter int
	// MinCenterDepthRatio discards candidate centers shallower than this
	// fraction of the maximum distance-transform depth.
	MinCenterDepthRatio float64
	// CenterSuppressionRadius controls spacing between candidate centers. Zero
	// selects an image-size-dependent default.
	CenterSuppressionRadius int
	// SafeZoneErodeSize shrinks the binary mask before placement so rendered
	// words have a small safety margin from the silhouette edge. It must be odd.
	SafeZoneErodeSize int

	// AlphaThreshold treats source pixels at or below this alpha as invisible.
	AlphaThreshold uint8
	// WordPadding expands each rectangular footprint before collision checks.
	WordPadding int
	// Angles lists allowed clockwise rotations in preference order. Placement
	// currently supports 0 and 90 degrees.
	Angles []int

	// Debug writes intermediate mask, distance, center, occupancy, and output images.
	Debug bool
}

// Option changes a Config created by NewConfig.
//
// Options allow callers to override only the fields they care about, without
// treating a zero value as "not set".
type Option func(*Config)

// NewConfig returns the default configuration with options applied.
//
// For example:
//
//	cfg := wordcloud.NewConfig(
//		wordcloud.WithInputPath("portrait.png"),
//		wordcloud.WithWordLimit(500),
//	)
func NewConfig(options ...Option) Config {
	cfg := Config{
		MinFontSize:             6,
		MaxFontSize:             48,
		WordLimit:               500,
		SpiralStepsPerCenter:    500,
		MinCenterDepthRatio:     0.01,
		CenterSuppressionRadius: 0, // automatic
		SafeZoneErodeSize:       3,
		WordPadding:             1,
		Angles:                  []int{0, 90},
		AlphaThreshold:          8,
	}
	for _, option := range options {
		if option != nil {
			option(&cfg)
		}
	}
	return cfg
}

// Validate reports whether cfg has the external inputs required to generate a
// word cloud. OutputPath is optional: an output path is derived from InputPath
// when it is empty.
func (cfg Config) Validate() error {
	switch {
	case strings.TrimSpace(cfg.InputPath) == "":
		return errors.New("input path is required")
	case strings.TrimSpace(cfg.TextPath) == "":
		return errors.New("text path is required")
	case strings.TrimSpace(cfg.FontPath) == "":
		return errors.New("font path is required")
	case cfg.MinFontSize <= 0:
		return errors.New("minimum font size must be positive")
	case cfg.MaxFontSize < cfg.MinFontSize:
		return errors.New("maximum font size must be >= minimum")
	case cfg.WordLimit <= 0:
		return errors.New("word limit must be positive")
	case cfg.SpiralStepsPerCenter <= 0:
		return errors.New("attempt count must be positive")
	case cfg.MinCenterDepthRatio <= 0 || cfg.MinCenterDepthRatio > 1:
		return errors.New("center depth ratio must be in (0, 1]")
	case cfg.CenterSuppressionRadius < 0:
		return errors.New("suppression radius cannot be negative")
	case cfg.SafeZoneErodeSize <= 0 || cfg.SafeZoneErodeSize%2 == 0:
		return errors.New("safe-zone erosion size must be positive and odd")
	}
	return nil
}

// WithInputPath sets the source image path.
func WithInputPath(path string) Option  { return func(cfg *Config) { cfg.InputPath = path } }
// WithOutputPath sets the output image path.
func WithOutputPath(path string) Option { return func(cfg *Config) { cfg.OutputPath = path } }
// WithTextPath sets the text source path.
func WithTextPath(path string) Option   { return func(cfg *Config) { cfg.TextPath = path } }
// WithFontPath sets the TTF/OTF font path.
func WithFontPath(path string) Option   { return func(cfg *Config) { cfg.FontPath = path } }

// WithMinFontSize sets the minimum placement font size.
func WithMinFontSize(size float64) Option {
	return func(cfg *Config) { cfg.MinFontSize = size }
}

// WithMaxFontSize sets the maximum initial font size.
func WithMaxFontSize(size float64) Option {
	return func(cfg *Config) { cfg.MaxFontSize = size }
}

// WithWordLimit sets the maximum number of candidate words.
func WithWordLimit(limit int) Option {
	return func(cfg *Config) { cfg.WordLimit = limit }
}

// WithMaxAttemptsPerCenter sets the number of spiral samples per center.
func WithMaxAttemptsPerCenter(attempts int) Option {
	return func(cfg *Config) { cfg.SpiralStepsPerCenter = attempts }
}

// WithMinCenterDepthRatio sets the minimum relative depth for search centers.
func WithMinCenterDepthRatio(ratio float64) Option {
	return func(cfg *Config) { cfg.MinCenterDepthRatio = ratio }
}

// WithCenterSuppressionRadius sets center spacing; zero keeps automatic spacing.
func WithCenterSuppressionRadius(radius int) Option {
	return func(cfg *Config) { cfg.CenterSuppressionRadius = radius }
}

// WithSafeZoneErodeSize sets the odd-kernel erosion size for the placement mask.
func WithSafeZoneErodeSize(size int) Option {
	return func(cfg *Config) { cfg.SafeZoneErodeSize = size }
}

// WithDebug enables or disables intermediate diagnostic images.
func WithDebug(debug bool) Option {
	return func(cfg *Config) { cfg.Debug = debug }
}
