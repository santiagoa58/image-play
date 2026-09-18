package wordcloud

import (
	"image"

	"github.com/santiagoa58/image-play/internal/textutil"
)

type WordFootprint struct {
	Word   textutil.Word
	Angle  int
	Image  *image.NRGBA
	Mask   *image.Alpha
	Width  int
	Height int
}

/*

For each word:
Render it onto a transparent temporary canvas.
Read its alpha pixels.
Crop empty outer rows and columns.
Dilate the alpha mask by WordPadding.
Cache the footprint.

```text
footprint pixel + forbidden safe-zone pixel → reject
footprint pixel + occupied pixel            → reject
otherwise                                   → commit footprint pixels
```

Do not stretch the word height. Tighter packing comes from no longer reserving transparent rectangle corners.
Change PlacedWord to store the exact accepted footprint and top-left position. Final rendering must composite that stored image instead of measuring and drawing the word again.
*/
