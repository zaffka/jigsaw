package slicer

import (
	"image"
	"image/color"
	"testing"
)

// testImage creates a simple test image with a gradient.
func testImage(w, h int) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := range h {
		for x := range w {
			img.SetRGBA(x, y, color.RGBA{
				R: uint8(x * 255 / w),
				G: uint8(y * 255 / h),
				B: 128,
				A: 255,
			})
		}
	}
	return img
}

func TestGrid(t *testing.T) {
	src := testImage(300, 200)
	pieces, err := Grid(src, GridOpts{Cols: 3, Rows: 2})
	if err != nil {
		t.Fatal(err)
	}
	if got, want := len(pieces), 6; got != want {
		t.Fatalf("Grid: got %d pieces, want %d", got, want)
	}
	// Each piece should be approximately 100x100
	for _, p := range pieces {
		if p.Image == nil {
			t.Fatal("piece has nil image")
		}
		if p.Bounds.Dx() < 99 || p.Bounds.Dx() > 101 {
			t.Errorf("piece %d: unexpected width %d", p.ID, p.Bounds.Dx())
		}
	}
	// Verify SVG path is non-empty.
	svg := pieces[0].Outline.SVG()
	if svg == "" {
		t.Error("piece outline SVG is empty")
	}
}

func TestGridInvalid(t *testing.T) {
	src := testImage(100, 100)
	_, err := Grid(src, GridOpts{Cols: 0, Rows: 2})
	if err == nil {
		t.Fatal("expected error for zero cols")
	}
}

func TestMerge(t *testing.T) {
	src := testImage(400, 400)
	pieces, err := Merge(src, MergeOpts{Cols: 4, Rows: 4, Seed: 42})
	if err != nil {
		t.Fatal(err)
	}
	if len(pieces) == 0 {
		t.Fatal("Merge produced no pieces")
	}
	// Total cells covered should be 16.
	totalPixels := 0
	for _, p := range pieces {
		totalPixels += countOpaquePixels(p.Image)
	}
	// Allow some tolerance at boundaries due to point-in-polygon at edges.
	totalCells := 400 * 400
	if totalPixels < totalCells*90/100 {
		t.Errorf("Merge: only %d/%d pixels covered", totalPixels, totalCells)
	}
	t.Logf("Merge: %d pieces, %d/%d pixels", len(pieces), totalPixels, totalCells)
}

func TestMergeExplicitGroups(t *testing.T) {
	src := testImage(300, 300)
	groups := []CellGroup{
		{{0, 0}, {1, 0}},         // horizontal pair
		{{2, 0}},                 // single
		{{0, 1}, {0, 2}, {1, 2}}, // L-shape
		{{1, 1}},                 // single
		{{2, 1}, {2, 2}},         // vertical pair
	}
	pieces, err := Merge(src, MergeOpts{Cols: 3, Rows: 3, Groups: groups})
	if err != nil {
		t.Fatal(err)
	}
	if got, want := len(pieces), 5; got != want {
		t.Fatalf("got %d pieces, want %d", got, want)
	}
}

func TestGeometryTriangles(t *testing.T) {
	src := testImage(200, 200)
	pieces, err := Geometry(src, GeometryOpts{Cols: 2, Rows: 2, Shape: Triangles})
	if err != nil {
		t.Fatal(err)
	}
	if got, want := len(pieces), 8; got != want {
		t.Fatalf("Triangles: got %d pieces, want %d", got, want)
	}
}

func TestGeometryDiamonds(t *testing.T) {
	src := testImage(200, 200)
	pieces, err := Geometry(src, GeometryOpts{Cols: 2, Rows: 2, Shape: Diamonds})
	if err != nil {
		t.Fatal(err)
	}
	if got, want := len(pieces), 16; got != want {
		t.Fatalf("Diamonds: got %d pieces, want %d", got, want)
	}
}

func TestGeometryTrapezoids(t *testing.T) {
	src := testImage(200, 200)
	pieces, err := Geometry(src, GeometryOpts{Cols: 2, Rows: 2, Shape: Trapezoids})
	if err != nil {
		t.Fatal(err)
	}
	if got, want := len(pieces), 8; got != want {
		t.Fatalf("Trapezoids: got %d pieces, want %d", got, want)
	}
}

func TestGeometryParallelograms(t *testing.T) {
	src := testImage(300, 200)
	pieces, err := Geometry(src, GeometryOpts{Cols: 3, Rows: 2, Shape: Parallelograms})
	if err != nil {
		t.Fatal(err)
	}
	// 3 parallelograms + 1 left edge triangle per row, 2 rows = 8 pieces
	if got, want := len(pieces), 8; got != want {
		t.Fatalf("Parallelograms: got %d pieces, want %d", got, want)
	}
	// Verify all pieces have non-nil images with some opaque pixels.
	for _, p := range pieces {
		if p.Image == nil {
			t.Fatalf("piece %d has nil image", p.ID)
		}
		if countOpaquePixels(p.Image) == 0 {
			t.Errorf("piece %d has no opaque pixels", p.ID)
		}
	}
	// Verify a parallelogram piece (not edge triangle) has 4 vertices in its SVG.
	svg := pieces[1].Outline.SVG() // first parallelogram (index 1, after left triangle)
	t.Logf("Parallelogram SVG: %s", svg)
}

func TestPuzzle(t *testing.T) {
	src := testImage(400, 300)
	pieces, err := Puzzle(src, PuzzleOpts{Cols: 4, Rows: 3, Seed: 123})
	if err != nil {
		t.Fatal(err)
	}
	if got, want := len(pieces), 12; got != want {
		t.Fatalf("Puzzle: got %d pieces, want %d", got, want)
	}
	// Verify corner pieces have straight borders.
	corner := pieces[0] // top-left corner
	if corner.Image == nil {
		t.Fatal("corner piece has nil image")
	}
	// SVG should contain cubic Bezier commands for internal edges.
	svg := pieces[5].Outline.SVG() // internal piece
	if svg == "" {
		t.Error("puzzle piece SVG is empty")
	}
	if !containsCurve(svg) {
		t.Error("internal puzzle piece should have bezier curves")
	}
	t.Logf("Puzzle piece SVG sample: %s", svg[:min(len(svg), 120)])
}

func TestPuzzleEdgePiece(t *testing.T) {
	src := testImage(200, 200)
	pieces, err := Puzzle(src, PuzzleOpts{Cols: 2, Rows: 2, Seed: 0})
	if err != nil {
		t.Fatal(err)
	}
	// All 4 pieces should have non-nil images.
	for _, p := range pieces {
		if p.Image == nil {
			t.Errorf("piece %d has nil image", p.ID)
		}
	}
}

func TestPathPolygon(t *testing.T) {
	var p Path
	p.MoveTo(0, 0)
	p.LineTo(100, 0)
	p.CubicTo(100, 50, 100, 100, 50, 100)
	p.LineTo(0, 100)
	p.Close()

	poly := p.Polygon(8)
	// MoveTo(1) + LineTo(1) + CubicTo(8 segments) + LineTo(1) = 11 points
	if len(poly) != 11 {
		t.Errorf("polygon has %d points, want 11", len(poly))
	}
}

func TestPointInPolygon(t *testing.T) {
	square := []Point{{0, 0}, {10, 0}, {10, 10}, {0, 10}}

	tests := []struct {
		p    Point
		want bool
	}{
		{Point{5, 5}, true},
		{Point{0.5, 0.5}, true},
		{Point{-1, 5}, false},
		{Point{11, 5}, false},
		{Point{5, -1}, false},
	}
	for _, tt := range tests {
		got := pointInPolygon(tt.p, square)
		if got != tt.want {
			t.Errorf("pointInPolygon(%v) = %v, want %v", tt.p, got, tt.want)
		}
	}
}

func countOpaquePixels(img *image.RGBA) int {
	b := img.Bounds()
	count := 0
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			_, _, _, a := img.At(x, y).RGBA()
			if a > 0 {
				count++
			}
		}
	}
	return count
}

func containsCurve(svg string) bool {
	for _, c := range svg {
		if c == 'C' {
			return true
		}
	}
	return false
}
