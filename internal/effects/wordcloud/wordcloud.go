package wordcloud

import (
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/santiagoa58/image-play/internal/imageutil"
	"github.com/santiagoa58/image-play/internal/textutil"
)

// Generate creates a word cloud from cfg and writes the final PNG.
//
// The pipeline is intentionally split into policy and mechanics:
//   - imageutil derives the silhouette and distance transform;
//   - textutil counts, sizes, and measures candidate words;
//   - this package chooses centers, sizes, orientations, and search order;
//   - layout.Space performs fast mask-containment and collision checks;
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

	// 4. Resolve an image-appropriate maximum size, then measure candidates.
	maxFontSize, err := resolveMaxFontSize(mask, wordHeap, cfg)
	if err != nil {
		return fmt.Errorf("resolve maximum font size: %w", err)
	}
	logger.Info(
		"resolved font range",
		"min", cfg.MinFontSize,
		"max", maxFontSize,
		"automatic_max", cfg.MaxFontSize == 0,
	)

	words, err := textutil.MeasureWords(
		wordHeap,
		textutil.WordMeasurementConfig{
			FontPath:    cfg.FontPath,
			MinFontSize: cfg.MinFontSize,
			MaxFontSize: maxFontSize,
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
	placementStarted := time.Now()
	var placed []PlacedWord
	skipped := 0
	horizontal := 0
	vertical := 0

	for i, w := range words {
		percent := 100 * (i + 1) / len(words)
		fmt.Printf("\rProgress: [%3d%%] %d/%d", percent, i+1, len(words))
		prevFontSize := maxFontSize
		if len(placed) > 0 {
			last := placed[len(placed)-1]
			prevFontSize = last.Word.FontSize
		}

		p, err := placeCtx.Place(w, prevFontSize, cfg.MinFontSize, placementStepRatio)
		if err != nil {
			if errors.Is(err, errNoPlacement) {
				skipped++
				if cfg.Debug {
					logger.Info("word skipped", "word", w.Text, "reason", err)
				}
				continue
			}
			return fmt.Errorf("place word %q: %w", w.Text, err)
		}

		placed = append(placed, p)
		if p.Angle == 90 {
			vertical++
		} else {
			horizontal++
		}
	}
	placementDuration := time.Since(placementStarted)
	fmt.Println()

	logger.Info(
		"placement complete",
		"prepared", len(words),
		"placed", len(placed),
		"skipped", skipped,
		"horizontal", horizontal,
		"vertical", vertical,
		"duration", placementDuration,
	)

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

	logger.Info(
		"word cloud generated",
		"output", outputPath,
		"total_duration", time.Since(started),
	)
	return nil
}
