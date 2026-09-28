package layout

import (
	"image"
	"testing"

	"gocv.io/x/gocv"
)

func TestValidCentersMatchesExhaustiveRectangleCheck(t *testing.T) {
	shapes := map[string][]string{
		"full":         {"########", "########", "########", "########", "########"},
		"concave":      {"########", "##....##", "##....##", "########", "########"},
		"disconnected": {"###..###", "###..###", "........", "###..###", "###..###"},
		"narrow":       {"..##....", "..##....", "..######", "..##....", "..##...."},
	}
	sizes := []image.Point{
		image.Pt(1, 1), image.Pt(2, 2), image.Pt(3, 2),
		image.Pt(2, 3), image.Pt(7, 1), image.Pt(1, 7), image.Pt(9, 2),
	}
	for name, rows := range shapes {
		t.Run(name, func(t *testing.T) {
			allowed := maskFromRows(rows)
			defer allowed.Close()
			space, err := NewFreeSpace(allowed)
			if err != nil {
				t.Fatal(err)
			}
			defer space.Close()

			for _, size := range sizes {
				got, err := space.ValidCenters(size)
				if err != nil {
					t.Fatal(err)
				}
				for y := range rows {
					for x := range rows[y] {
						want := bruteForceFits(rows, size, image.Pt(x, y))
						valid := got.GetUCharAt(y, x) != 0
						if valid != want {
							t.Errorf("size %v center (%d,%d): valid=%v, want %v", size, x, y, valid, want)
						}
					}
				}
				got.Close()
			}
		})
	}
}

func TestReserveChangesOnlyOwnedSpace(t *testing.T) {
	allowed := maskFromRows([]string{"######", "######", "######", "######"})
	defer allowed.Close()
	space, err := NewFreeSpace(allowed)
	if err != nil {
		t.Fatal(err)
	}
	defer space.Close()

	if !space.Reserve(image.Pt(2, 2), image.Pt(1, 1)) {
		t.Fatal("first rectangle should fit")
	}
	if space.Reserve(image.Pt(2, 2), image.Pt(2, 1)) {
		t.Fatal("overlapping rectangle should not fit")
	}
	if space.Reserve(image.Pt(2, 2), image.Pt(0, 0)) {
		t.Fatal("out-of-bounds rectangle should not fit")
	}
	if !space.Reserve(image.Pt(2, 2), image.Pt(3, 1)) {
		t.Fatal("edge-touching rectangle should fit")
	}
	if allowed.GetUCharAt(0, 0) == 0 {
		t.Fatal("the input mask was mutated")
	}

	got, err := space.ValidCenters(image.Pt(2, 2))
	if err != nil {
		t.Fatal(err)
	}
	defer got.Close()
	remaining := []string{"....##", "....##", "######", "######"}
	for y := 0; y < 4; y++ {
		for x := 0; x < 6; x++ {
			want := bruteForceFits(remaining, image.Pt(2, 2), image.Pt(x, y))
			if (got.GetUCharAt(y, x) != 0) != want {
				t.Errorf("center (%d,%d) validity differs after reservation", x, y)
			}
		}
	}
}

func TestReserveMaskKeepsGapsAvailableToRectangles(t *testing.T) {
	allowed := maskFromRows([]string{
		"########", "########", "########", "########", "########", "########",
	})
	defer allowed.Close()
	space, err := NewFreeSpace(allowed)
	if err != nil {
		t.Fatal(err)
	}
	defer space.Close()
	glyph := maskFromRows([]string{"#...#", "#...#", "#####", "#...#", "#...#"})
	defer glyph.Close()
	if err := space.ReserveMask(glyph, image.Pt(1, 0)); err != nil {
		t.Fatal(err)
	}
	if !space.Reserve(image.Pt(3, 2), image.Pt(3, 1)) {
		t.Fatal("small rectangle should fit between glyph strokes")
	}
	if space.Reserve(image.Pt(2, 2), image.Pt(1, 1)) {
		t.Fatal("rectangle should not overlap glyph strokes")
	}
	if err := space.ReserveMask(glyph, image.Pt(5, 2)); err != nil {
		t.Fatal(err)
	}
	if space.free.GetUCharAt(2, 5) != 0 {
		t.Fatal("visible portion of an overhanging glyph should be reserved")
	}
}

func maskFromRows(rows []string) gocv.Mat {
	mask := gocv.NewMatWithSize(len(rows), len(rows[0]), gocv.MatTypeCV8UC1)
	for y, row := range rows {
		for x := range row {
			if row[x] == '#' {
				mask.SetUCharAt(y, x, 255)
			}
		}
	}
	return mask
}

func bruteForceFits(rows []string, size, center image.Point) bool {
	rect := RectAt(center, size)
	if !rect.In(image.Rect(0, 0, len(rows[0]), len(rows))) {
		return false
	}
	for y := rect.Min.Y; y < rect.Max.Y; y++ {
		for x := rect.Min.X; x < rect.Max.X; x++ {
			if rows[y][x] != '#' {
				return false
			}
		}
	}
	return true
}
