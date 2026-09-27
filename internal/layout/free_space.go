package layout

import (
	"errors"
	"image"

	"gocv.io/x/gocv"
)

// FreeSpace owns the remaining pixels available for rectangular placements.
// Nonzero pixels in the initial mask are free. The caller must Close it.
type FreeSpace struct {
	free gocv.Mat
}

func NewFreeSpace(allowed gocv.Mat) (*FreeSpace, error) {
	if allowed.Empty() || allowed.Type() != gocv.MatTypeCV8UC1 {
		return nil, errors.New("free space requires a nonempty 8-bit single-channel mask")
	}
	return &FreeSpace{free: allowed.Clone()}, nil
}

func (s *FreeSpace) Close() error { return s.free.Close() }

// RectAt is the exact rectangle represented by a center and footprint. The
// same convention is used as the explicit erosion anchor, including for even
// widths and heights.
func RectAt(center, size image.Point) image.Rectangle {
	left := center.X - size.X/2
	top := center.Y - size.Y/2
	return image.Rect(left, top, left+size.X, top+size.Y)
}

// ValidCenters returns an 8-bit mask of all integer centers at which size fits
// wholly within the remaining free space. The caller must Close the result.
func (s *FreeSpace) ValidCenters(size image.Point) (gocv.Mat, error) {
	if size.X <= 0 || size.Y <= 0 {
		return gocv.NewMat(), errors.New("footprint dimensions must be positive")
	}
	if size.X > s.free.Cols() || size.Y > s.free.Rows() {
		return gocv.NewMatWithSize(s.free.Rows(), s.free.Cols(), gocv.MatTypeCV8UC1), nil
	}

	kernel := gocv.GetStructuringElement(gocv.MorphRect, size)
	defer kernel.Close()
	result := gocv.NewMat()
	err := gocv.ErodeWithParamsAndBorderValue(
		s.free, &result, kernel, image.Pt(size.X/2, size.Y/2),
		1, gocv.BorderConstant, gocv.NewScalar(0, 0, 0, 0),
	)
	if err != nil {
		result.Close()
		return gocv.NewMat(), err
	}
	return result, nil
}

// Reserve removes a footprint from free space. It returns false without
// changing anything when the proposed rectangle is outside or occupied.
func (s *FreeSpace) Reserve(size, center image.Point) bool {
	if size.X <= 0 || size.Y <= 0 {
		return false
	}
	rect := RectAt(center, size)
	if !rect.In(image.Rect(0, 0, s.free.Cols(), s.free.Rows())) {
		return false
	}
	area := s.free.Region(rect)
	defer area.Close()
	if gocv.CountNonZero(area) != size.X*size.Y {
		return false
	}
	area.SetTo(gocv.NewScalar(0, 0, 0, 0))
	return true
}

// Snapshot returns a copy of the remaining free pixels for diagnostics.
// The caller must Close it.
func (s *FreeSpace) Snapshot() gocv.Mat { return s.free.Clone() }
