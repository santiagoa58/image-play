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

	// MinFontSize overrides the 6px minimum when positive. Zero uses 6px.
	MinFontSize float64
	// MaxFontSize overrides automatic maximum-size calibration when positive.
	// Zero lets the layout determine a sensible maximum from the image shape.
	MaxFontSize float64
	// WordLimit is the maximum number of candidate words considered. It is not
	// a promise that every candidate will be placed.
	WordLimit int

	// SafeZoneErodeSize shrinks the binary mask before placement so rendered
	// words have a small safety margin from the silhouette edge. It must be odd.
	SafeZoneErodeSize int

	// AlphaThreshold treats source pixels at or below this alpha as invisible.
	AlphaThreshold uint8
	// WordPadding expands each rectangular footprint before collision checks.
	WordPadding int
	// Angles lists allowed clockwise rotations in preference order within the
	// selected shape region. Placement supports 0 and 90 degrees.
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
		MinFontSize:       0,
		WordLimit:         500,
		SafeZoneErodeSize: 3,
		WordPadding:       1,
		Angles:            []int{0, 90},
		AlphaThreshold:    8,
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
	case cfg.MinFontSize < 0:
		return errors.New("minimum font size must be zero (automatic) or positive")
	case cfg.MaxFontSize < 0:
		return errors.New("maximum font size cannot be negative")
	case cfg.MaxFontSize > 0 && cfg.MaxFontSize < cfg.MinFontSize:
		return errors.New("maximum font size must be zero (automatic) or >= minimum")
	case cfg.WordLimit <= 0:
		return errors.New("word limit must be positive")
	case cfg.SafeZoneErodeSize <= 0 || cfg.SafeZoneErodeSize%2 == 0:
		return errors.New("safe-zone erosion size must be positive and odd")
	case cfg.WordPadding < 0:
		return errors.New("word padding cannot be negative")
	}
	return nil
}

// WithInputPath sets the source image path.
func WithInputPath(path string) Option { return func(cfg *Config) { cfg.InputPath = path } }

// WithOutputPath sets the output image path.
func WithOutputPath(path string) Option { return func(cfg *Config) { cfg.OutputPath = path } }

// WithTextPath sets the text source path.
func WithTextPath(path string) Option { return func(cfg *Config) { cfg.TextPath = path } }

// WithFontPath sets the TTF/OTF font path.
func WithFontPath(path string) Option { return func(cfg *Config) { cfg.FontPath = path } }

// WithMinFontSize sets the minimum placement font size.
func WithMinFontSize(size float64) Option {
	return func(cfg *Config) { cfg.MinFontSize = size }
}

// WithMaxFontSize overrides automatic maximum-size calibration.
// Pass zero to use automatic sizing.
func WithMaxFontSize(size float64) Option {
	return func(cfg *Config) { cfg.MaxFontSize = size }
}

// WithWordLimit sets the maximum number of candidate words.
func WithWordLimit(limit int) Option {
	return func(cfg *Config) { cfg.WordLimit = limit }
}

// WithSafeZoneErodeSize sets the odd-kernel erosion size for the placement mask.
func WithSafeZoneErodeSize(size int) Option {
	return func(cfg *Config) { cfg.SafeZoneErodeSize = size }
}

// WithDebug enables or disables intermediate diagnostic images.
func WithDebug(debug bool) Option {
	return func(cfg *Config) { cfg.Debug = debug }
}
