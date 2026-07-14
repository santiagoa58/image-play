package imageutil

import (
	"gocv.io/x/gocv"
)

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
