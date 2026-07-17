package wordcloud

import (
	"fmt"
	"log/slog"

	"github.com/fogleman/gg"
)

func RenderRectangles(
	width, height int,
	fontPath string,
	placed []PlacedWord,
	outputPath string,
	debug bool,
) error {
	logger := slog.Default()
	dc := gg.NewContext(width, height)
	dc.SetRGB(1, 1, 1) // white background
	dc.Clear()

	for _, p := range placed {
		if err := dc.LoadFontFace(fontPath, p.Word.FontSize); err != nil {
			logger.Warn("failed to load font", "word", p.Word, "error", err)
			continue
		}
		dc.SetRGB(0, 0, 0) // black text
		dc.DrawStringAnchored(p.Word.Text, p.X, p.Y, 0.5, 0.5)
	}

	if err := dc.SavePNG(outputPath); err != nil {
		return fmt.Errorf("save word cloud: %w", err)
	}
	if err := writeWordcloudDebug(
		debug,
		outputPath,
		dc.Image(),
	); err != nil {
		return fmt.Errorf("write word-cloud diagnostics: %w", err)
	}
	return nil
}
