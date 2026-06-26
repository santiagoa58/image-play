package imageutil

import (
	"fmt"
	"image"
	"image/color"

	"gocv.io/x/gocv"
)

const (
	grabCutBackground         = uint8(0)
	grabCutForeground         = uint8(1)
	grabCutProbableBackground = uint8(2)
	grabCutProbableForeground = uint8(3)
)

type GrabCutConfig struct {
	Rect                      image.Rectangle
	IterCount                 int
	IncludeProbableForeground bool
	// MinComponentAreaRatio removes detached foreground components smaller than
	// this fraction of the largest component. Negative disables cleanup.
	MinComponentAreaRatio float64
}

func GrabCutForegroundMask(img image.Image, conf GrabCutConfig) (image.Image, error) {
	mat, err := gocv.ImageToMatRGB(img)
	if err != nil {
		return nil, fmt.Errorf("convert image to OpenCV mat: %w", err)
	}
	defer mat.Close()
	if mat.Empty() {
		return nil, fmt.Errorf("input image is empty")
	}

	rect := conf.Rect
	if rect.Empty() {
		rect = defaultGrabCutRect(mat.Cols(), mat.Rows())
	}
	rect = rect.Intersect(image.Rect(0, 0, mat.Cols(), mat.Rows()))
	if rect.Empty() {
		return nil, fmt.Errorf("foreground rectangle is outside image bounds")
	}

	iterCount := conf.IterCount
	if iterCount <= 0 {
		iterCount = 5
	}

	mask := gocv.NewMatWithSize(mat.Rows(), mat.Cols(), gocv.MatTypeCV8U)
	defer mask.Close()
	bgdModel := gocv.NewMat()
	defer bgdModel.Close()
	fgdModel := gocv.NewMat()
	defer fgdModel.Close()

	if err := gocv.GrabCut(mat, &mask, rect, &bgdModel, &fgdModel, iterCount, gocv.GCInitWithRect); err != nil {
		return nil, fmt.Errorf("run OpenCV GrabCut: %w", err)
	}

	out := image.NewGray(image.Rect(0, 0, mat.Cols(), mat.Rows()))
	includeProbable := conf.IncludeProbableForeground
	for y := 0; y < mat.Rows(); y++ {
		for x := 0; x < mat.Cols(); x++ {
			class := mask.GetUCharAt(y, x)
			foreground := class == grabCutForeground || (includeProbable && class == grabCutProbableForeground)
			if foreground {
				out.SetGray(x, y, color.Gray{Y: 255})
				continue
			}
			if class != grabCutBackground && class != grabCutProbableBackground && class != grabCutProbableForeground {
				out.SetGray(x, y, color.Gray{Y: 0})
			}
		}
	}

	return filterForegroundComponents(out, conf.MinComponentAreaRatio), nil
}

func defaultGrabCutRect(width, height int) image.Rectangle {
	marginX := max(1, width/16)
	top := max(1, height*28/100)
	bottomMargin := max(1, height/40)
	return image.Rect(marginX, top, width-marginX, height-bottomMargin)
}

func filterForegroundComponents(mask *image.Gray, minAreaRatio float64) *image.Gray {
	if minAreaRatio < 0 {
		return mask
	}
	if minAreaRatio == 0 {
		minAreaRatio = 0.12
	}

	bounds := mask.Bounds()
	width, height := bounds.Dx(), bounds.Dy()
	total := width * height
	if total == 0 {
		return mask
	}

	labels := make([]int, total)
	areas := []int{0}
	queue := make([]int, 0, total/8)
	label := 0
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			index := y*width + x
			if labels[index] != 0 || mask.GrayAt(bounds.Min.X+x, bounds.Min.Y+y).Y == 0 {
				continue
			}

			label++
			areas = append(areas, 0)
			labels[index] = label
			queue = append(queue[:0], index)
			for head := 0; head < len(queue); head++ {
				current := queue[head]
				areas[label]++
				cx := current % width
				cy := current / width
				neighbors := [4][2]int{
					{cx - 1, cy},
					{cx + 1, cy},
					{cx, cy - 1},
					{cx, cy + 1},
				}
				for _, neighbor := range neighbors {
					nx, ny := neighbor[0], neighbor[1]
					if nx < 0 || nx >= width || ny < 0 || ny >= height {
						continue
					}
					next := ny*width + nx
					if labels[next] != 0 || mask.GrayAt(bounds.Min.X+nx, bounds.Min.Y+ny).Y == 0 {
						continue
					}
					labels[next] = label
					queue = append(queue, next)
				}
			}
		}
	}

	if label <= 1 {
		return mask
	}

	largest := 0
	for _, area := range areas[1:] {
		if area > largest {
			largest = area
		}
	}
	if largest == 0 {
		return mask
	}

	minArea := int(float64(largest) * minAreaRatio)
	if minArea < 1 {
		minArea = 1
	}

	filtered := image.NewGray(bounds)
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			index := y*width + x
			component := labels[index]
			if component > 0 && areas[component] >= minArea {
				filtered.SetGray(bounds.Min.X+x, bounds.Min.Y+y, color.Gray{Y: 255})
			}
		}
	}
	return filtered
}
