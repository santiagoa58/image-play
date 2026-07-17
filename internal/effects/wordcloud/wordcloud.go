package wordcloud

import (
	"fmt"
	"log/slog"

	"github.com/santiagoa58/image-play/internal/imageutil"
	"github.com/santiagoa58/image-play/internal/textutil"
)

func Generate(cfg Config) error {
	if err := cfg.Validate(); err != nil {
		return fmt.Errorf("validate config: %w", err)
	}

	logger := slog.Default()

	// 1. Resolve output path
	outputPath, err := textutil.ResolveOutputPath(cfg.InputPath, cfg.OutputPath, "wordcloud")
	if err != nil {
		return fmt.Errorf("resolve output path: %w", err)
	}

	// 2. Prepare mask (binary + distance)
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

	// 3. Count words from text
	logger.Info("counting words", "path", cfg.TextPath)
	wordHeap, err := textutil.CountWords(cfg.TextPath)
	if err != nil {
		return fmt.Errorf("count words: %w", err)
	}

	// 4. Convert counts to sized words (sorted largest → smallest)
	words, err := textutil.MeasureWords(
		wordHeap,
		textutil.WordMeasurementConfig{
			FontPath:    cfg.FontPath,
			MinFontSize: cfg.MinFontSize,
			MaxFontSize: cfg.MaxFontSize,
			Limit:       cfg.WordLimit,
		},
	)
	if err != nil {
		return err
	}
	logger.Info("prepared words", "count", len(words))

	// 5. Create placement context (owns safeZone + occupancy)
	logger.Info("creating placement context")
	placeCtx, err := NewPlacementContext(mask, cfg)
	if err != nil {
		return fmt.Errorf("create placement context: %w", err)
	}
	defer placeCtx.Close()

	if err := writeCentersDebug(
		cfg.Debug,
		outputPath,
		mask,
		placeCtx.centers,
	); err != nil {
		return fmt.Errorf("write center diagnostics: %w", err)
	}

	// 6. Place words
	logger.Info("placing words")
	var placed []PlacedWord
	for _, w := range words {
		p, ok := placeCtx.TryPlace(w)
		if ok {
			placed = append(placed, p)
		}
	}

	if err := writePlacementDebug(
		cfg.Debug,
		outputPath,
		*placeCtx.safeZone,
		*placeCtx.occupancy,
	); err != nil {
		return fmt.Errorf("write placement diagnostics: %w", err)
	}

	// 7. Render final image
	logger.Info("rendering word cloud", "output", outputPath)
	if err := RenderRectangles(mask.Width, mask.Height, cfg.FontPath, placed, outputPath, cfg.Debug); err != nil {
		return err
	}

	fmt.Printf("✅ Word cloud saved to %s\n", outputPath)
	return nil
}
