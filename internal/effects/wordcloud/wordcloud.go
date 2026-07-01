package wordcloud

import (
	"fmt"

	"github.com/santiagoa58/image-play/internal/imageutil"
	"github.com/santiagoa58/image-play/internal/util"
)

func GenWordCloud(inpath, outpath, textpath string) error {
	// temp test code to show the mask image
	mask, err := imageutil.PrepareMask(inpath)
	if err != nil {
		return err
	}
	defer mask.Close()
	countHeap, err := util.CountWords(textpath)
	if err != nil {
		return err
	}
	fmt.Printf("Word count heap: %v\n", countHeap)
	mask.IMWrite(outpath)
	return nil
}
