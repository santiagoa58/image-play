package wordcloud

import (
	"errors"
	"image"
	"image/color"

	"github.com/santiagoa58/image-play/internal/imageutil"
	"gocv.io/x/gocv"
)

type Center struct {
	Point image.Point
	Depth float32
}

func FindCenters(m *imageutil.Mask, cfg Config) ([]Center, error) {
	if err := validate(m, cfg); err != nil {
		return nil, err
	}
	centers := []Center{}
	distCopy := m.DistMat.Clone()
	defer distCopy.Close()
	radius := cfg.CenterSuppressionRadius
	if radius == 0 {
		// Automatically determine how far apart centers should be.
		radius = max(8, min(m.DistMat.Cols(), m.DistMat.Rows())/40)
	}

	_, globalMax, _, _ := gocv.MinMaxLoc(*m.DistMat)
	if globalMax <= 0 {
		return centers, nil
	}

	minDepth := globalMax * float32(cfg.MinCenterDepthRatio)

	for {
		// Find the deepest remaining pixel.
		_, maxDepth, _, point := gocv.MinMaxLoc(distCopy)
		if maxDepth <= 0 || maxDepth < minDepth {
			// Stop when only shallow or zero-valued pixels remain
			break
		}
		centers = append(centers, Center{
			Point: point,
			Depth: maxDepth,
		})

		// Suppress this region so we find centers in other parts of the shape
		gocv.Circle(&distCopy, point, radius, color.RGBA{}, -1)
	}

	return centers, nil
}

func validate(m *imageutil.Mask, c Config) error {
	if m == nil {
		return errors.New("mask cannot be nil")
	}
	if m.DistMat == nil {
		return errors.New("mask distance matrix cannot be nil")
	}
	if m.DistMat.Empty() {
		return errors.New("mask distance matrix cannot be empty")
	}
	if c.MinCenterDepthRatio <= 0 || c.MinCenterDepthRatio > 1 {
		return errors.New("center depth ratio must be in (0, 1]")
	}
	if c.CenterSuppressionRadius < 0 {
		return errors.New("center suppression radius cannot be negative")
	}
	return nil
}
