package wordcloud

import (
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/santiagoa58/image-play/internal/imageutil"
	"github.com/santiagoa58/image-play/internal/output"
)

// Generate creates a word cloud from cfg and writes the final PNG.
//
// The pipeline is intentionally split into policy and mechanics:
//   - imageutil derives the silhouette and distance transform;
//   - textutil counts, sizes, and measures candidate words;
//   - this package chooses sizes, shape regions, orientations, and positions;
//   - layout.FreeSpace supplies exact rectangular fit and reserves either
//     rectangular or rendered-glyph footprints;
//   - the renderer draws the accepted layout.
//
// A candidate that cannot fit at the minimum size is skipped; WordLimit is a
// source pool, not a required placement count.
func Generate(cfg Config) error {
	if err := cfg.Validate(); err != nil {
		return fmt.Errorf("validate config: %w", err)
	}

	started := time.Now()
	logger := slog.Default()

	outputPath, err := output.PreparePNG(cfg.InputPath, cfg.OutputPath, "wordcloud")
	if err != nil {
		return err
	}

	logger.Info("preparing mask", "path", cfg.InputPath)
	mask, err := imageutil.PrepareMask(cfg.InputPath, cfg.AlphaThreshold)
	if err != nil {
		return fmt.Errorf("prepare mask: %w", err)
	}
	defer mask.Close()

	if err := writeMaskDebug(cfg.Debug, outputPath, mask); err != nil {
		return fmt.Errorf("write mask diagnostics: %w", err)
	}
	if err := writeDistanceDebug(cfg.Debug, outputPath, mask); err != nil {
		return fmt.Errorf("write distance diagnostics: %w", err)
	}

	words, minFontSize, err := prepareWords(mask, cfg, logger)
	if err != nil {
		return err
	}

	logger.Info("creating placement context")
	placeCtx, err := NewPlacementContext(mask, cfg)
	if err != nil {
		return fmt.Errorf("create placement context: %w", err)
	}
	defer placeCtx.Close()
	if err := writeRegionsDebug(cfg.Debug, outputPath, placeCtx.regions); err != nil {
		return fmt.Errorf("write region diagnostics: %w", err)
	}

	placed, err := placeWords(placeCtx, words, minFontSize, cfg.Debug, logger)
	if err != nil {
		return err
	}
	if err := writePlacementDebug(
		cfg.Debug,
		outputPath,
		*placeCtx.safeZone,
		*placeCtx.occupancy,
	); err != nil {
		return fmt.Errorf("write placement diagnostics: %w", err)
	}

	logger.Info("rendering word cloud", "output", outputPath)
	if len(placed) == 0 {
		return errors.New("no words could be placed inside the image shape")
	}
	if err := Render(mask.Source, mask.DarkBackground, cfg.FontPath, placed, outputPath, cfg.Debug); err != nil {
		return err
	}

	logger.Info(
		"word cloud generated",
		"output", outputPath,
		"total_duration", time.Since(started),
	)
	return nil
}
