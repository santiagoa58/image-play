package wordcloud

import (
	"fmt"
	"log/slog"

	"github.com/fogleman/gg"
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
	mask, err := imageutil.PrepareMask(cfg.InputPath, cfg.MaskSource, cfg.AlphaThreshold)
	if err != nil {
		return fmt.Errorf("prepare mask: %w", err)
	}
	defer mask.Close()
	maskOut, err := textutil.ResolveOutputPath(cfg.InputPath, cfg.OutputPath, "mask")
	if err != nil {
		return fmt.Errorf("resolve output path: %w", err)
	}
	// write out the mask
	imageutil.IMWrite(maskOut, mask.BinaryMat)
	// 3. Count words from text
	logger.Info("counting words", "path", cfg.TextPath)
	wordHeap, err := textutil.CountWords(cfg.TextPath)
	if err != nil {
		return fmt.Errorf("count words: %w", err)
	}

	// 4. Convert counts to sized words (sorted largest → smallest)
	words, err := textutil.WordsWithMeasurements(wordHeap, cfg.MinFontSize, cfg.MaxFontSize, cfg.FontPath, cfg.WordLimit)
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

	// 6. Place words
	logger.Info("placing words")
	var placed []PlacedWord
	for _, w := range words {
		p, ok := placeCtx.TryPlace(w, cfg.MaxAttemptsPerCenter)
		if ok {
			placed = append(placed, p)
		}
	}
	// 7. Render final image
	logger.Info("rendering word cloud", "output", outputPath)
	dc := gg.NewContext(mask.Width, mask.Height)
	dc.SetRGB(1, 1, 1) // white background
	dc.Clear()

	for _, p := range placed {
		if err := dc.LoadFontFace(cfg.FontPath, p.Word.FontSize); err != nil {
			logger.Warn("failed to load font", "word", p.Word, "error", err)
			continue
		}
		dc.SetRGB(0, 0, 0) // black text
		dc.DrawStringAnchored(p.Word.Text, p.X, p.Y, 0.5, 0.5)
	}

	if err := dc.SavePNG(outputPath); err != nil {
		return fmt.Errorf("save word cloud: %w", err)
	}

	fmt.Printf("✅ Word cloud saved to %s\n", outputPath)
	return nil
}
