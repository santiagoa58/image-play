package wordcloud

import (
	"errors"
	"strings"
)

type MaskSource int

const (
	MaskSourceLuminance MaskSource = iota
	MaskSourceAlpha
)

type Config struct {
	InputPath  string
	OutputPath string
	TextPath   string
	FontPath   string

	MinFontSize float64
	MaxFontSize float64
	WordLimit   int

	MaxAttemptsPerCenter    int
	CenterDepthRatio        float64
	CenterSuppressionRadius int
	SafeZoneErodeSize       int

	MaskSource     MaskSource
	AlphaThreshold uint8
	WordPadding    int
	Angles         []int // degrees; initially 0 and 90

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
		WordLimit:               1000,
		MaxAttemptsPerCenter:    500,
		CenterDepthRatio:        0.01,
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
	case cfg.MaxAttemptsPerCenter <= 0:
		return errors.New("attempt count must be positive")
	case cfg.CenterDepthRatio <= 0 || cfg.CenterDepthRatio > 1:
		return errors.New("center depth ratio must be in (0, 1]")
	case cfg.CenterSuppressionRadius < 0:
		return errors.New("suppression radius cannot be negative")
	case cfg.SafeZoneErodeSize <= 0 || cfg.SafeZoneErodeSize%2 == 0:
		return errors.New("safe-zone erosion size must be positive and odd")
	}
	return nil
}

func WithInputPath(path string) Option  { return func(cfg *Config) { cfg.InputPath = path } }
func WithOutputPath(path string) Option { return func(cfg *Config) { cfg.OutputPath = path } }
func WithTextPath(path string) Option   { return func(cfg *Config) { cfg.TextPath = path } }
func WithFontPath(path string) Option   { return func(cfg *Config) { cfg.FontPath = path } }

func WithMinFontSize(size float64) Option {
	return func(cfg *Config) { cfg.MinFontSize = size }
}

func WithMaxFontSize(size float64) Option {
	return func(cfg *Config) { cfg.MaxFontSize = size }
}

func WithWordLimit(limit int) Option {
	return func(cfg *Config) { cfg.WordLimit = limit }
}

func WithMaxAttemptsPerCenter(attempts int) Option {
	return func(cfg *Config) { cfg.MaxAttemptsPerCenter = attempts }
}

func WithCenterDepthRatio(ratio float64) Option {
	return func(cfg *Config) { cfg.CenterDepthRatio = ratio }
}

func WithCenterSuppressionRadius(radius int) Option {
	return func(cfg *Config) { cfg.CenterSuppressionRadius = radius }
}

func WithSafeZoneErodeSize(size int) Option {
	return func(cfg *Config) { cfg.SafeZoneErodeSize = size }
}

func WithDebug(debug bool) Option {
	return func(cfg *Config) { cfg.Debug = debug }
}
