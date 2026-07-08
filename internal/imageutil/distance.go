package imageutil

import (
	"image"
	"image/color"

	"gocv.io/x/gocv"
)

func FindCenters(m Mask) []image.Point {
	centers := []image.Point{}
	distCopy := m.distMat.Clone()
	defer distCopy.Close()

	_, globalMax, _, _ := gocv.MinMaxLoc(*m.distMat)
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
	h, w := m.distMat.Rows(), m.distMat.Cols()
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

// computeDistanceTransform runs the Euclidean distance transform.
func ComputeDistanceTransform(th gocv.Mat) (*gocv.Mat, error) {
	dist := gocv.NewMat()
	labels := gocv.NewMat()
	defer labels.Close()
	if err := gocv.DistanceTransform(th, &dist, &labels, gocv.DistL2, gocv.DistanceMaskPrecise, gocv.DistanceLabelPixel); err != nil {
		dist.Close()
		return nil, err
	}
	return &dist, nil
}
