package layout

import (
	"errors"
	"image"
)

const (
	// minimumSpatialCellSize avoids creating extremely fine grids for tiny
	// images, where the map overhead would outweigh the collision savings.
	minimumSpatialCellSize = 8

	// targetCellsPerAxis keeps spatial-hash cells roughly proportional to the
	// canvas size instead of relying on one magic pixel size for every image.
	targetCellsPerAxis = 40
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

// integralMask is a summed-area table over the static allowed-pixel mask.
type integralMask struct {
	// stride is width+1 because the table includes an empty top row and left
	// column. That padding keeps rectangle-sum queries branch-free.
	stride int
	sums   []int
}

// newIntegralMask preprocesses the mask once so Contains can answer a
// rectangle-containment query in constant time.
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

// Contains reports whether every pixel in rect is allowed.
//
// The four table lookups are the standard summed-area-table inclusion/
// exclusion formula. Comparing the sum with rect's area works because each
// allowed pixel contributes exactly one to the table.
func (m *integralMask) Contains(rect image.Rectangle) bool {
	x0, y0 := rect.Min.X, rect.Min.Y
	x1, y1 := rect.Max.X, rect.Max.Y

	sum := m.sums[y1*m.stride+x1] -
		m.sums[y0*m.stride+x1] -
		m.sums[y1*m.stride+x0] +
		m.sums[y0*m.stride+x0]

	return sum == rect.Dx()*rect.Dy()
}

// gridCell identifies one bucket in the spatial hash.
type gridCell struct {
	x int
	y int
}

// spatialIndex partitions the canvas into coarse cells and indexes each
// occupied rectangle into every cell it touches. Candidate rectangles only
// need to compare themselves with rectangles from their nearby cells.
type spatialIndex struct {
	cellSize int
	cells    map[gridCell][]int
	rects    []image.Rectangle
}

// newSpatialIndex creates an empty collision index using square cells.
func newSpatialIndex(cellSize int) *spatialIndex {
	return &spatialIndex{
		cellSize: cellSize,
		cells:    make(map[gridCell][]int),
	}
}

// Add records rect in the index after a successful placement.
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

// Overlaps reports whether rect intersects an already occupied rectangle.
//
// A rectangle can appear in more than one bucket. Duplicate checks are harmless
// because this method returns on the first actual intersection.
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

// cellRange maps a pixel rectangle to the inclusive range of spatial-hash
// cells it touches. Max is exclusive for image.Rectangle, hence the -1.
func (s *spatialIndex) cellRange(rect image.Rectangle) (minX, maxX, minY, maxY int) {
	minX = rect.Min.X / s.cellSize
	maxX = (rect.Max.X - 1) / s.cellSize
	minY = rect.Min.Y / s.cellSize
	maxY = (rect.Max.Y - 1) / s.cellSize
	return
}
