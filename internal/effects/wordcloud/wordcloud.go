package wordcloud

import (
	"fmt"

	"github.com/santiagoa58/image-play/internal/imageutil"
	"github.com/santiagoa58/image-play/internal/textutils"
)

func GenWordCloud(inpath, outpath, textpath string) error {
	// temp test code to show the mask image
	mask, err := imageutil.PrepareMask(inpath)
	if err != nil {
		return err
	}
	defer mask.Close()
	countHeap, err := textutils.CountWords(textpath)
	if err != nil {
		return err
	}
	if err := mask.IMWrite(outpath); err != nil {
		return err
	}
	fmt.Printf("Word count heap:\n%v\n", countHeap.RankedString(10))

	return nil
}
