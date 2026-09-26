package mathutil

import (
	"image"
	"math"
)

// CenteredRect converts floating-point center coordinates and dimensions into
// an axis-aligned integer rectangle suitable for mask and collision checks.
//
// Width and height are rounded outward so placement never reserves less space
// than the measured footprint.
func CenteredRect(cx, cy, w, h float64) image.Rectangle {
	left := int(math.Round(cx - w/2))
	top := int(math.Round(cy - h/2))
	return image.Rect(left, top, left+int(math.Ceil(w)), top+int(math.Ceil(h)))
}
