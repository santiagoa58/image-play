package textmosaic

import (
	"errors"
	"math"
	"strings"
)

const (
	defaultBaseFontSize  = 14.0
	defaultLetterSpacing = -0.08
	defaultWordSpacing   = 0.10
	defaultOutputScale   = 2.0
)

// Config controls text-mosaic generation.
//
// Defaults are provided by NewConfig. Most callers only need to supply the
// input image, text file, font, and optional output path.
type Config struct {
	// InputPath is the source image sampled by the effect.
	InputPath string
	// OutputPath is the target PNG. When empty, Generate derives one from InputPath.
	OutputPath string
	// TextPath is the UTF-8 text source repeated across the mosaic.
	TextPath string
	// FontPath points to the TTF/OTF font used for rendering.
	FontPath string

	// TargetWidth resizes the source before the final output scale is applied.
	// Zero keeps the original source width.
	TargetWidth int
	// OutputScale enlarges the prepared image and text for a sharper PNG export.
	// Upscaling stops when the longest output side reaches 4096 pixels.
	OutputScale float64
	// BaseFontSize is the base font size before resolution-aware scaling.
	BaseFontSize float64
	// ContrastPercent adjusts source contrast before clipping through text.
	ContrastPercent float64
	// Uppercase converts the text to uppercase before repeating it.
	Uppercase bool
	// LetterSpacing changes the advance after every character in font-size units (em).
	LetterSpacing float64
	// WordSpacing changes the advance after spaces, in addition to LetterSpacing.
	WordSpacing float64
}

// Option changes a Config created by NewConfig.
type Option func(*Config)

// NewConfig returns the default configuration with options applied.
func NewConfig(options ...Option) Config {
	cfg := Config{
		BaseFontSize:  defaultBaseFontSize,
		LetterSpacing: defaultLetterSpacing,
		WordSpacing:   defaultWordSpacing,
		OutputScale:   defaultOutputScale,
	}

	for _, option := range options {
		if option != nil {
			option(&cfg)
		}
	}

	return cfg
}

// Validate reports whether cfg has the required external inputs and valid
// rendering parameters.
func (cfg Config) Validate() error {
	switch {
	case strings.TrimSpace(cfg.InputPath) == "":
		return errors.New("input path is required")
	case strings.TrimSpace(cfg.TextPath) == "":
		return errors.New("text path is required")
	case strings.TrimSpace(cfg.FontPath) == "":
		return errors.New("font path is required")
	case cfg.TargetWidth < 0:
		return errors.New("target width cannot be negative")
	case math.IsNaN(cfg.OutputScale) || math.IsInf(cfg.OutputScale, 0) || cfg.OutputScale < 1:
		return errors.New("output scale must be finite and at least 1")
	case cfg.BaseFontSize <= 0:
		return errors.New("base font size must be positive")
	case cfg.ContrastPercent < -100 || cfg.ContrastPercent > 100:
		return errors.New("contrast percent must be between -100 and 100")
	case math.IsNaN(cfg.LetterSpacing) || math.IsInf(cfg.LetterSpacing, 0):
		return errors.New("letter spacing must be finite")
	case math.IsNaN(cfg.WordSpacing) || math.IsInf(cfg.WordSpacing, 0):
		return errors.New("word spacing must be finite")
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

// WithTargetWidth sets the working image width while preserving aspect ratio.
// OutputScale is applied afterward. Zero keeps the source image width.
func WithTargetWidth(width int) Option {
	return func(cfg *Config) { cfg.TargetWidth = width }
}

// WithOutputScale scales both the prepared image and text for a larger PNG.
// The effective scale is capped to keep the longest output side at most 4096
// pixels unless the input is already larger.
func WithOutputScale(scale float64) Option {
	return func(cfg *Config) { cfg.OutputScale = scale }
}

// WithBaseFontSize sets the base size used before resolution-aware scaling.
func WithBaseFontSize(size float64) Option {
	return func(cfg *Config) { cfg.BaseFontSize = size }
}

// WithContrastPercent sets the source contrast adjustment.
func WithContrastPercent(percent float64) Option {
	return func(cfg *Config) { cfg.ContrastPercent = percent }
}

// WithUppercase renders the source text in uppercase.
func WithUppercase(uppercase bool) Option {
	return func(cfg *Config) { cfg.Uppercase = uppercase }
}

// WithLetterSpacing sets the extra character advance in font-size units (em).
func WithLetterSpacing(spacing float64) Option {
	return func(cfg *Config) { cfg.LetterSpacing = spacing }
}

// WithWordSpacing sets the extra advance after spaces in font-size units (em).
func WithWordSpacing(spacing float64) Option {
	return func(cfg *Config) { cfg.WordSpacing = spacing }
}
