package imageutil

import (
	"gocv.io/x/gocv"
)

// ComputeDistanceTransform returns the Euclidean distance from every non-zero
// mask pixel to the nearest zero pixel.
//
// Word-cloud placement uses larger values as a proxy for "roomier" interior
// locations. The caller owns the returned matrix.
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
