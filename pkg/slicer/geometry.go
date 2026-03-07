package slicer

import (
	"fmt"
	"image"
	"math/rand/v2"
)

// ShapeType selects the tessellation pattern for geometric cutting.
type ShapeType int

const (
	// Triangles splits each grid cell diagonally into 2 right triangles.
	Triangles ShapeType = iota
	// Diamonds splits each grid cell by both diagonals into 4 right triangles.
	Diamonds
	// Trapezoids splits each grid cell with a sloped horizontal cut.
	Trapezoids
	// Parallelograms creates a sheared grid of parallelogram pieces.
	Parallelograms
	// Mixed randomly assigns a shape to each grid cell from all available types.
	Mixed
)

// GeometryOpts configures geometric shape cutting.
type GeometryOpts struct {
	Cols, Rows int
	Shape      ShapeType
	Seed       uint64 // random seed (used by Mixed mode)
}

// Geometry cuts the source image into geometric shapes.
func Geometry(src image.Image, opts GeometryOpts) ([]Piece, error) {
	if opts.Cols <= 0 || opts.Rows <= 0 {
		return nil, fmt.Errorf("slicer: cols and rows must be positive, got %dx%d", opts.Cols, opts.Rows)
	}

	if opts.Shape == Parallelograms {
		return parallelograms(src, opts)
	}

	b := src.Bounds()
	cellW := float64(b.Dx()) / float64(opts.Cols)
	cellH := float64(b.Dy()) / float64(opts.Rows)
	ox, oy := float64(b.Min.X), float64(b.Min.Y)

	var rng *rand.Rand
	if opts.Shape == Mixed {
		rng = rand.New(rand.NewPCG(opts.Seed, opts.Seed^0xfacade))
	}

	var pieces []Piece
	id := 0

	for r := 0; r < opts.Rows; r++ {
		for c := 0; c < opts.Cols; c++ {
			x0 := ox + float64(c)*cellW
			y0 := oy + float64(r)*cellH
			x1 := x0 + cellW
			y1 := y0 + cellH

			shape := opts.Shape
			if shape == Mixed {
				shape = ShapeType(rng.IntN(4)) // Triangles..Parallelograms
			}

			polys := cellPolygons(shape, c, r, x0, y0, x1, y1, cellW, cellH)

			for _, poly := range polys {
				outline := pathFromPolygon(poly)
				pieces = append(pieces, clipPiece(src, id, outline, image.Pt(c, r)))
				id++
			}
		}
	}
	return pieces, nil
}

// cellPolygons returns the polygon set for a single grid cell with the given shape.
func cellPolygons(shape ShapeType, c, r int, x0, y0, x1, y1, cellW, cellH float64) [][]Point {
	mx := (x0 + x1) / 2
	my := (y0 + y1) / 2

	switch shape {
	case Triangles:
		if (c+r)%2 == 0 {
			return [][]Point{
				{{x0, y0}, {x1, y0}, {x0, y1}},
				{{x1, y0}, {x1, y1}, {x0, y1}},
			}
		}
		return [][]Point{
			{{x0, y0}, {x1, y0}, {x1, y1}},
			{{x0, y0}, {x1, y1}, {x0, y1}},
		}

	case Diamonds:
		return [][]Point{
			{{x0, y0}, {x1, y0}, {mx, my}},
			{{x1, y0}, {x1, y1}, {mx, my}},
			{{x1, y1}, {x0, y1}, {mx, my}},
			{{x0, y1}, {x0, y0}, {mx, my}},
		}

	case Trapezoids:
		if r%2 == 0 {
			cutY0 := my - cellH*0.15
			cutY1 := my + cellH*0.15
			return [][]Point{
				{{x0, y0}, {x1, y0}, {x1, cutY1}, {x0, cutY0}},
				{{x0, cutY0}, {x1, cutY1}, {x1, y1}, {x0, y1}},
			}
		}
		cutY0 := my + cellH*0.15
		cutY1 := my - cellH*0.15
		return [][]Point{
			{{x0, y0}, {x1, y0}, {x1, cutY1}, {x0, cutY0}},
			{{x0, cutY0}, {x1, cutY1}, {x1, y1}, {x0, y1}},
		}

	case Parallelograms:
		// Within a single cell: parallelogram + 2 corner triangles.
		s := cellW * 0.2
		return [][]Point{
			// left corner triangle
			{{x0, y0}, {x0 + s, y0}, {x0, y1}},
			// central parallelogram
			{{x0 + s, y0}, {x1, y0}, {x1 - s, y1}, {x0, y1}},
			// right corner triangle
			{{x1, y0}, {x1, y1}, {x1 - s, y1}},
		}
	}
	return nil
}

// parallelograms generates a sheared grid of parallelogram pieces with
// triangular filler pieces at the left and right edges.
func parallelograms(src image.Image, opts GeometryOpts) ([]Piece, error) {
	b := src.Bounds()
	cellW := float64(b.Dx()) / float64(opts.Cols)
	cellH := float64(b.Dy()) / float64(opts.Rows)
	ox, oy := float64(b.Min.X), float64(b.Min.Y)
	shear := cellW * 0.25

	var pieces []Piece
	id := 0

	for r := 0; r < opts.Rows; r++ {
		y0 := oy + float64(r)*cellH
		y1 := y0 + cellH

		// Even rows: top edge shifted right. Odd rows: bottom edge shifted right.
		var topOff, botOff float64
		if r%2 == 0 {
			topOff, botOff = shear, 0
		} else {
			topOff, botOff = 0, shear
		}

		// Left edge filler triangle.
		if topOff > 0 {
			// Gap at top-left: (ox, y0), (ox+shear, y0), (ox, y1)
			tri := []Point{{ox, y0}, {ox + shear, y0}, {ox, y1}}
			pieces = append(pieces, clipPiece(src, id, pathFromPolygon(tri), image.Pt(0, r)))
			id++
		} else {
			// Gap at bottom-left: (ox, y0), (ox+shear, y1), (ox, y1)
			tri := []Point{{ox, y0}, {ox + shear, y1}, {ox, y1}}
			pieces = append(pieces, clipPiece(src, id, pathFromPolygon(tri), image.Pt(0, r)))
			id++
		}

		// Main parallelograms.
		for c := 0; c < opts.Cols; c++ {
			x0 := ox + float64(c)*cellW
			x1 := x0 + cellW
			poly := []Point{
				{x0 + topOff, y0},
				{x1 + topOff, y0},
				{x1 + botOff, y1},
				{x0 + botOff, y1},
			}
			pieces = append(pieces, clipPiece(src, id, pathFromPolygon(poly), image.Pt(c, r)))
			id++
		}

		// The rightmost parallelogram extends past the image edge by shear;
		// clipPiece handles that naturally (out-of-bounds pixels stay transparent).
	}
	return pieces, nil
}
