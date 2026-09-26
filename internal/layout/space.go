// Package layout contains the low-level geometry used by word placement.
//
// It deliberately keeps artistic policy out. The summed-area mask follows the
// integral-image approach used by amueller/word_cloud, while dynamic collision
// lookup follows the spatial-hash approach used by psykhi/wordclouds.
package layout

import (
	"errors"
	"image"
)

const (
	minimumSpatialCellSize = 8
	targetCellsPerAxis      = 40
)

// Space tracks where rectangular items may be placed.
//
// The fixed allowed-area mask uses a summed-area table for constant-time
// rectangle checks. Dynamic collisions use a spatial hash so each candidate
// only checks nearby occupied rectangles.
type Space struct {
	bounds   image.Rectangle
	allowed  *integralMask
	occupied *spatialIndex
}

// NewSpace builds a placement space from a row-major binary mask.
// Non-zero pixels are allowed; zero pixels are excluded.
func NewSpace(width, height int, pixels []uint8) (*Space, error) {
	if width <= 0 || height <= 0 {
		return nil, errors.New("layout dimensions must be positive")
	}
	if len(pixels) < width*height {
		return nil, errors.New("layout mask is smaller than its dimensions")
	}

	cellSize := min(width, height) / targetCellsPerAxis
	cellSize = max(cellSize, minimumSpatialCellSize)

	return &Space{
		bounds:   image.Rect(0, 0, width, height),
		allowed:  newIntegralMask(width, height, pixels),
		occupied: newSpatialIndex(cellSize),
	}, nil
}

// TryPlace reserves rect when it is fully inside the allowed mask and does not
// overlap any previously placed rectangle.
func (s *Space) TryPlace(rect image.Rectangle) bool {
	if rect.Empty() || !rect.In(s.bounds) {
		return false
	}
	if !s.allowed.Contains(rect) || s.occupied.Overlaps(rect) {
		return false
	}

	s.occupied.Add(rect)
	return true
}

type integralMask struct {
	stride int
	sums   []int
}

func newIntegralMask(width, height int, pixels []uint8) *integralMask {
	stride := width + 1
	sums := make([]int, (height+1)*stride)

	for y := 1; y <= height; y++ {
		rowSum := 0
		for x := 1; x <= width; x++ {
			if pixels[(y-1)*width+x-1] != 0 {
				rowSum++
			}
			sums[y*stride+x] = sums[(y-1)*stride+x] + rowSum
		}
	}

	return &integralMask{
		stride: stride,
		sums:   sums,
	}
}

func (m *integralMask) Contains(rect image.Rectangle) bool {
	x0, y0 := rect.Min.X, rect.Min.Y
	x1, y1 := rect.Max.X, rect.Max.Y

	sum := m.sums[y1*m.stride+x1] -
		m.sums[y0*m.stride+x1] -
		m.sums[y1*m.stride+x0] +
		m.sums[y0*m.stride+x0]

	return sum == rect.Dx()*rect.Dy()
}

type gridCell struct {
	x int
	y int
}

type spatialIndex struct {
	cellSize int
	cells    map[gridCell][]int
	rects    []image.Rectangle
}

func newSpatialIndex(cellSize int) *spatialIndex {
	return &spatialIndex{
		cellSize: cellSize,
		cells:    make(map[gridCell][]int),
	}
}

func (s *spatialIndex) Add(rect image.Rectangle) {
	id := len(s.rects)
	s.rects = append(s.rects, rect)

	minX, maxX, minY, maxY := s.cellRange(rect)
	for y := minY; y <= maxY; y++ {
		for x := minX; x <= maxX; x++ {
			cell := gridCell{x: x, y: y}
			s.cells[cell] = append(s.cells[cell], id)
		}
	}
}

func (s *spatialIndex) Overlaps(rect image.Rectangle) bool {
	minX, maxX, minY, maxY := s.cellRange(rect)
	for y := minY; y <= maxY; y++ {
		for x := minX; x <= maxX; x++ {
			for _, id := range s.cells[gridCell{x: x, y: y}] {
				if s.rects[id].Overlaps(rect) {
					return true
				}
			}
		}
	}

	return false
}

func (s *spatialIndex) cellRange(rect image.Rectangle) (minX, maxX, minY, maxY int) {
	minX = rect.Min.X / s.cellSize
	maxX = (rect.Max.X - 1) / s.cellSize
	minY = rect.Min.Y / s.cellSize
	maxY = (rect.Max.Y - 1) / s.cellSize
	return
}
