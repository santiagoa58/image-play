package layout

import (
	"errors"
	"image"

	"gocv.io/x/gocv"
)

// FreeSpace owns the remaining pixels available for word placements.
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

// ValidCentersMask returns centers where every nonzero mask pixel fits in free
// space. Offset locates the mask's top-left corner relative to the word center.
func (s *FreeSpace) ValidCentersMask(mask gocv.Mat, offset image.Point) (gocv.Mat, error) {
	if mask.Empty() || mask.Type() != gocv.MatTypeCV8UC1 {
		return gocv.NewMat(), errors.New("glyph footprint requires a nonempty binary mask")
	}
	minX, minY := min(0, offset.X), min(0, offset.Y)
	maxX, maxY := max(1, offset.X+mask.Cols()), max(1, offset.Y+mask.Rows())
	width, height := maxX-minX, maxY-minY
	if width > s.free.Cols() || height > s.free.Rows() {
		return gocv.NewMatWithSize(s.free.Rows(), s.free.Cols(), gocv.MatTypeCV8UC1), nil
	}
	kernel := gocv.NewMatWithSize(height, width, gocv.MatTypeCV8UC1)
	defer kernel.Close()
	region := kernel.Region(image.Rect(offset.X-minX, offset.Y-minY, offset.X-minX+mask.Cols(), offset.Y-minY+mask.Rows()))
	if err := mask.CopyTo(&region); err != nil {
		region.Close()
		return gocv.NewMat(), err
	}
	region.Close()
	result := gocv.NewMat()
	err := gocv.ErodeWithParamsAndBorderValue(
		s.free, &result, kernel, image.Pt(-minX, -minY),
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

// ReserveMask removes only the nonzero pixels of a glyph footprint. Origin is
// the top-left position of the mask in the free-space coordinate system.
func (s *FreeSpace) ReserveMask(mask gocv.Mat, origin image.Point) (bool, error) {
	if mask.Empty() || mask.Type() != gocv.MatTypeCV8UC1 {
		return false, errors.New("glyph footprint requires a nonempty binary mask")
	}
	rect := image.Rectangle{Min: origin, Max: origin.Add(image.Pt(mask.Cols(), mask.Rows()))}
	if !rect.In(image.Rect(0, 0, s.free.Cols(), s.free.Rows())) {
		return false, nil
	}
	area := s.free.Region(rect)
	defer area.Close()
	intersection := gocv.NewMat()
	defer intersection.Close()
	if err := gocv.BitwiseAnd(area, mask, &intersection); err != nil {
		return false, err
	}
	if gocv.CountNonZero(intersection) != gocv.CountNonZero(mask) {
		return false, nil
	}
	inverse := gocv.NewMat()
	defer inverse.Close()
	if err := gocv.BitwiseNot(mask, &inverse); err != nil {
		return false, err
	}
	remaining := gocv.NewMat()
	defer remaining.Close()
	if err := gocv.BitwiseAnd(area, inverse, &remaining); err != nil {
		return false, err
	}
	return true, remaining.CopyTo(&area)
}

// Snapshot returns a copy of the remaining free pixels for diagnostics.
// The caller must Close it.
func (s *FreeSpace) Snapshot() gocv.Mat { return s.free.Clone() }
