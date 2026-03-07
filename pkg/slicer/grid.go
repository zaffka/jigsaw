package slicer

import (
	"fmt"
	"image"
)

// GridOpts configures square grid cutting.
type GridOpts struct {
	Cols, Rows int
}

// Grid cuts the source image into a grid of rectangular pieces.
func Grid(src image.Image, opts GridOpts) ([]Piece, error) {
	if opts.Cols <= 0 || opts.Rows <= 0 {
		return nil, fmt.Errorf("slicer: cols and rows must be positive, got %dx%d", opts.Cols, opts.Rows)
	}

	b := src.Bounds()
	cellW := float64(b.Dx()) / float64(opts.Cols)
	cellH := float64(b.Dy()) / float64(opts.Rows)

	pieces := make([]Piece, 0, opts.Cols*opts.Rows)
	id := 0
	for r := 0; r < opts.Rows; r++ {
		for c := 0; c < opts.Cols; c++ {
			x0 := float64(b.Min.X) + float64(c)*cellW
			y0 := float64(b.Min.Y) + float64(r)*cellH
			x1 := x0 + cellW
			y1 := y0 + cellH

			outline := pathFromPolygon([]Point{
				{x0, y0}, {x1, y0}, {x1, y1}, {x0, y1},
			})

			pieces = append(pieces, clipPiece(src, id, outline, image.Pt(c, r)))
			id++
		}
	}
	return pieces, nil
}
