package textmosaic

import (
	"errors"
	"strings"
)

const defaultBaseFontSize = 14.0

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
	// FontPath points to the monospace TTF/OTF font used for rendering.
	FontPath string

	// TargetWidth resizes the source before rendering. Zero keeps the original width.
	TargetWidth int
	// BaseFontSize is the base font size before resolution-aware scaling.
	BaseFontSize float64
	// ContrastPercent adjusts source contrast before grayscale conversion.
	ContrastPercent float64
}

// Option changes a Config created by NewConfig.
type Option func(*Config)

// NewConfig returns the default configuration with options applied.
func NewConfig(options ...Option) Config {
	cfg := Config{
		BaseFontSize: defaultBaseFontSize,
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
	case cfg.BaseFontSize <= 0:
		return errors.New("base font size must be positive")
	case cfg.ContrastPercent < -100 || cfg.ContrastPercent > 100:
		return errors.New("contrast percent must be between -100 and 100")
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

// WithTargetWidth sets the output width while preserving aspect ratio.
// Zero keeps the source image width.
func WithTargetWidth(width int) Option {
	return func(cfg *Config) { cfg.TargetWidth = width }
}

// WithBaseFontSize sets the base size used before resolution-aware scaling.
func WithBaseFontSize(size float64) Option {
	return func(cfg *Config) { cfg.BaseFontSize = size }
}

// WithContrastPercent sets the source contrast adjustment.
func WithContrastPercent(percent float64) Option {
	return func(cfg *Config) { cfg.ContrastPercent = percent }
}
