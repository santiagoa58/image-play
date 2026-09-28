package wordcloud

import (
	"errors"
	"image"
	"image/color"

	"gocv.io/x/gocv"
)

const maxShapeRegions = 24

// regionPolicy partitions the silhouette around deep, separated points. A
// multi-source flood fills through the silhouette, so regions cannot jump
// across empty pixels. Regions rank legal placements; they never limit fit.
type regionPolicy struct {
	width, height int
	labels        []int32
	area          []int
	occupied      []int
	depth         []float32
}

type orientedCenters struct {
	angle   int
	centers gocv.Mat
}

type regionCandidate struct {
	point image.Point
	depth float32
	valid bool
}

func newRegionPolicy(safe, distance gocv.Mat) (*regionPolicy, error) {
	if safe.Empty() || distance.Empty() || safe.Type() != gocv.MatTypeCV8UC1 ||
		distance.Type() != gocv.MatTypeCV32FC1 || safe.Rows() != distance.Rows() ||
		safe.Cols() != distance.Cols() {
		return nil, errors.New("regions require matching binary and float distance maps")
	}
	width, height := safe.Cols(), safe.Rows()
	pixels, err := safe.DataPtrUint8()
	if err != nil {
		return nil, err
	}
	depth, err := distance.DataPtrFloat32()
	if err != nil {
		return nil, err
	}
	policy := &regionPolicy{
		width: width, height: height,
		labels: make([]int32, width*height), depth: depth,
	}

	peaks := distance.Clone()
	defer peaks.Close()
	peakPixels, err := peaks.DataPtrFloat32()
	if err != nil {
		return nil, err
	}
	for i, pixel := range pixels {
		policy.labels[i] = -1
		if pixel == 0 {
			peakPixels[i] = 0
		} else {
			policy.labels[i] = -2 // free, awaiting a region
		}
	}

	queue := make([]int32, 0, len(pixels)/2)
	radius := max(4, min(width, height)/6)
	for len(policy.area) < maxShapeRegions {
		_, value, _, point := gocv.MinMaxLoc(peaks)
		if value <= 0 {
			break
		}
		id := int32(len(policy.area))
		index := point.Y*width + point.X
		policy.labels[index] = id
		policy.area = append(policy.area, 0)
		policy.occupied = append(policy.occupied, 0)
		queue = append(queue, int32(index))
		if err := gocv.Circle(&peaks, point, radius, color.RGBA{}, -1); err != nil {
			return nil, err
		}
	}
	if len(queue) == 0 {
		return nil, errors.New("placement silhouette has no free pixels")
	}

	// Any component without a peak gets a seed too, after the first flood.
	policy.flood(queue)
	for index, label := range policy.labels {
		if label != -2 {
			continue
		}
		id := int32(len(policy.area))
		policy.labels[index] = id
		policy.area = append(policy.area, 0)
		policy.occupied = append(policy.occupied, 0)
		queue = append(queue[:0], int32(index))
		policy.flood(queue)
	}
	return policy, nil
}

func (p *regionPolicy) flood(queue []int32) {
	for head := 0; head < len(queue); head++ {
		index := int(queue[head])
		id := p.labels[index]
		p.area[id]++
		x, y := index%p.width, index/p.width
		for dy := -1; dy <= 1; dy++ {
			for dx := -1; dx <= 1; dx++ {
				nx, ny := x+dx, y+dy
				if (dx == 0 && dy == 0) || nx < 0 || ny < 0 || nx >= p.width || ny >= p.height {
					continue
				}
				next := ny*p.width + nx
				if p.labels[next] == -2 {
					p.labels[next] = id
					queue = append(queue, int32(next))
				}
			}
		}
	}
}

// choose scans each legal-center bitmap once. It first favors the region with
// the lowest occupied fraction, then configured orientation, then shape depth.
func (p *regionPolicy) choose(options []orientedCenters) (image.Point, int, bool, error) {
	best := make([][]regionCandidate, len(p.area))
	for id := range best {
		best[id] = make([]regionCandidate, len(options))
	}
	for angleIndex, option := range options {
		pixels, err := option.centers.DataPtrUint8()
		if err != nil {
			return image.Point{}, 0, false, err
		}
		for index, pixel := range pixels {
			if pixel == 0 || p.labels[index] < 0 {
				continue
			}
			candidate := &best[p.labels[index]][angleIndex]
			if !candidate.valid || p.depth[index] > candidate.depth {
				*candidate = regionCandidate{
					point: image.Pt(index%p.width, index/p.width),
					depth: p.depth[index], valid: true,
				}
			}
		}
	}

	chosen := -1
	for id, candidates := range best {
		hasPosition := false
		for _, candidate := range candidates {
			hasPosition = hasPosition || candidate.valid
		}
		if !hasPosition {
			continue
		}
		if chosen < 0 ||
			p.occupied[id]*p.area[chosen] < p.occupied[chosen]*p.area[id] ||
			(p.occupied[id]*p.area[chosen] == p.occupied[chosen]*p.area[id] && p.area[id] > p.area[chosen]) {
			chosen = id
		}
	}
	if chosen < 0 {
		return image.Point{}, 0, false, nil
	}
	for i, candidate := range best[chosen] {
		if candidate.valid {
			return candidate.point, options[i].angle, true, nil
		}
	}
	return image.Point{}, 0, false, errors.New("selected region has no legal center")
}

// reserve credits every region intersected by the actual footprint.
func (p *regionPolicy) reserve(rect image.Rectangle) {
	for y := rect.Min.Y; y < rect.Max.Y; y++ {
		for x := rect.Min.X; x < rect.Max.X; x++ {
			if id := p.labels[y*p.width+x]; id >= 0 {
				p.occupied[id]++
			}
		}
	}
}
