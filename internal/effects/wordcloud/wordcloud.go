package wordcloud

import (
	"fmt"
	"log/slog"

	"github.com/fogleman/gg"
	"github.com/santiagoa58/image-play/internal/imageutil"
	"github.com/santiagoa58/image-play/internal/textutil"
)

const (
	minFontSize = 8
	maxFontSize = 72
)

func GenWordCloud(inpath, outpath, textpath, fontpath string) error {
	logger := slog.Default()

	// 1. Resolve output path
	outputPath, err := textutil.ResolveOutputPath(inpath, outpath, "wordcloud")
	if err != nil {
		return fmt.Errorf("resolve output path: %w", err)
	}

	// 2. Prepare mask (binary + distance)
	logger.Info("preparing mask", "path", inpath)
	mask, err := imageutil.PrepareMask(inpath)
	if err != nil {
		return fmt.Errorf("prepare mask: %w", err)
	}
	defer mask.Close()

	// 3. Count words from text
	logger.Info("counting words", "path", textpath)
	wordHeap, err := textutil.CountWords(textpath)
	if err != nil {
		return fmt.Errorf("count words: %w", err)
	}

	// 4. Convert counts to sized words (sorted largest → smallest)
	words := textutil.WordsFromCounts(wordHeap, minFontSize, maxFontSize)
	logger.Info("prepared words", "count", len(words))

	// 5. Create placement context (owns safeZone + occupancy)
	logger.Info("creating placement context")
	placeCtx, err := NewPlacementContext(mask, fontpath)
	if err != nil {
		return fmt.Errorf("create placement context: %w", err)
	}
	defer placeCtx.Close()

	// 6. Place words
	logger.Info("placing words")
	var placed []PlacedWord
	for _, w := range words {
		if p, ok := placeCtx.TryPlace(w); ok {
			placed = append(placed, p)
			logger.Debug("placed word", "word", w.Text, "size", p.Size)
		}
	}
	logger.Info("finished placement", "placed", len(placed), "skipped", len(words)-len(placed))

	// 7. Render final image
	logger.Info("rendering word cloud", "output", outputPath)
	dc := gg.NewContext(mask.Width, mask.Height)
	dc.SetRGB(1, 1, 1) // white background
	dc.Clear()

	for _, p := range placed {
		if err := dc.LoadFontFace(fontpath, p.Size); err != nil {
			logger.Warn("failed to load font", "word", p.Word, "error", err)
			continue
		}
		dc.SetRGB(0, 0, 0) // black text
		dc.DrawStringAnchored(p.Word, p.X, p.Y, 0.5, 0.5)
	}

	if err := dc.SavePNG(outputPath); err != nil {
		return fmt.Errorf("save word cloud: %w", err)
	}

	fmt.Printf("✅ Word cloud saved to %s\n", outputPath)
	return nil
}
