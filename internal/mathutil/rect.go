package mathutil

import (
	"image"
	"math"
)

func CenteredRect(cx, cy, w, h float64) image.Rectangle {
	left := int(math.Round(cx - w/2))
	top := int(math.Round(cy - h/2))
	return image.Rect(left, top, left+int(math.Ceil(w)), top+int(math.Ceil(h)))
}
