// internal/wordcloud/placement.go
package wordcloud

import (
	"fmt"
	"image"
	"image/color"

	"gocv.io/x/gocv"
)

type PlacedWord struct {
	Word string
	X, Y float64 // center position
	Size float64 // font size in points
}

func FindCenters(dist *gocv.Mat) []image.Point {
	centers := []image.Point{}
	distCopy := dist.Clone()
	defer distCopy.Close()

	_, globalMax, _, _ := gocv.MinMaxLoc(*dist)
	if globalMax <= 0 {
		return centers
	}

	// Define a threshold to filter out insignificant local maxima (~15% of the global maximum).
	minUsefulDepth := globalMax * 0.15

	// Automatically determine how far apart centers should be.
	// We calculate the radius as ~5% of the smaller image dimension so that
	// the separation scales reasonably with image resolution.
	// We also enforce a minimum of 25 pixels to prevent centers from being
	// placed too close together on smaller images.
	h, w := dist.Rows(), dist.Cols()
	suppressionRadius := max(min(h, w)/20, 25)

	for {
		_, maxVal, _, maxLoc := gocv.MinMaxLoc(distCopy)
		if maxVal < minUsefulDepth {
			break
		}

		centers = append(centers, maxLoc)

		// Suppress this region so we find centers in other parts of the shape
		gocv.Circle(&distCopy, maxLoc, suppressionRadius, color.RGBA{0, 0, 0, 0}, -1)
	}

	return centers
}

// PrepareMasks creates two masks used for fast validation during placement.
//
// safeZone: A shrunk version of the binary mask. Any rectangle that fits
//
//	completely inside this mask is guaranteed to have enough
//	breathing room from the actual shape boundary.
//
// occupancy: Starts empty. We mark areas as occupied when we place words.
//
//	This lets us quickly check if a candidate position overlaps
//	any already placed word.
func PrepareMasks(binaryMask gocv.Mat, padding int) (safeZone, occupancy gocv.Mat, err error) {
	safeZone = gocv.NewMat()

	// Create a kernel size based on padding.
	// A kernel of size (padding*2 + 1) roughly erodes by `padding` pixels.
	kSize := max(padding*2+1, 3)
	kernel := gocv.GetStructuringElement(gocv.MorphRect, image.Point{kSize, kSize})
	defer kernel.Close()

	if err := gocv.Erode(binaryMask, &safeZone, kernel); err != nil {
		safeZone.Close()
		return gocv.Mat{}, gocv.Mat{}, fmt.Errorf("failed to create safe zone: %w", err)
	}

	occupancy = gocv.NewMatWithSize(
		binaryMask.Rows(),
		binaryMask.Cols(),
		gocv.MatTypeCV8UC1,
	)

	return safeZone, occupancy, nil
}
