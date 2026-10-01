package textmosaic

import (
	"fmt"
	"log/slog"
	"time"

	"github.com/disintegration/imaging"
	"github.com/santiagoa58/image-play/internal/output"
)

// Generate creates a text mosaic from cfg and writes the final PNG.
//
// File I/O and effect-specific processing live here so callers do not need to
// assemble the pipeline themselves.
func Generate(cfg Config) error {
	if err := cfg.Validate(); err != nil {
		return fmt.Errorf("validate config: %w", err)
	}

	started := time.Now()
	logger := slog.Default()

	outputPath, err := output.PreparePNG(cfg.InputPath, cfg.OutputPath, "textmosaic")
	if err != nil {
		return err
	}

	text, err := loadText(cfg.TextPath)
	if err != nil {
		return err
	}

	logger.Info("loading source image", "path", cfg.InputPath)
	source, err := imaging.Open(cfg.InputPath)
	if err != nil {
		return fmt.Errorf("open input image %q: %w", cfg.InputPath, err)
	}

	result, err := generateImage(source, text, cfg)
	if err != nil {
		return fmt.Errorf("generate text mosaic: %w", err)
	}

	if err := imaging.Save(result, outputPath); err != nil {
		return fmt.Errorf("save text mosaic %q: %w", outputPath, err)
	}

	logger.Info(
		"text mosaic generated",
		"output", outputPath,
		"width", result.Bounds().Dx(),
		"height", result.Bounds().Dy(),
		"duration", time.Since(started),
	)

	return nil
}
