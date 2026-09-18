package imageutil

import (
	"image/color"

	"gocv.io/x/gocv"
)

type ImageFields struct {
	Color     *gocv.Mat // original BGR/BGRA pixels
	Luminance *gocv.Mat // optional tonal/detail field
	Subject   *gocv.Mat // binary foreground from segmentation/alpha
	Placement *gocv.Mat // binary area currently allowed for words
	Distance  *gocv.Mat // distance inside Placement or Subject
}

func SampleColor(img gocv.Mat, x, y int) color.RGBA {
	p := img.GetVecbAt(x, y)
	return color.RGBA{
		R: p[2],
		G: p[1],
		B: p[0],
		A: 255,
	}
}
