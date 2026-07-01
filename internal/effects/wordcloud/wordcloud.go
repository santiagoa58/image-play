package wordcloud

import (
	"fmt"

	"github.com/santiagoa58/image-play/internal/imageutil"
	"github.com/santiagoa58/image-play/internal/textutils"
)

func GenWordCloud(inpath, outpath, textpath string) error {
	outputPath, err := textutils.ResolveOutputPath(inpath, outpath, "wordcloud")
	if err != nil {
		return fmt.Errorf("resolve output path: %w", err)
	}

	mask, err := imageutil.PrepareMask(inpath)
	if err != nil {
		return fmt.Errorf("prepare mask: %w", err)
	}
	defer mask.Close()

	countHeap, err := textutils.CountWords(textpath)
	if err != nil {
		return fmt.Errorf("count words: %w", err)
	}

	if err := mask.IMWrite(outputPath); err != nil {
		return fmt.Errorf("write thresholded image to %q: %w", outputPath, err)
	}

	fmt.Printf("Word count heap:\n%s\n", countHeap.RankedString(10))
	fmt.Printf("✅ Done! Saved to %s\n", outputPath)

	return nil
}
