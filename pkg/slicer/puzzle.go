package slicer

import (
	"fmt"
	"image"
	"math/rand/v2"
)

// PuzzleOpts configures jigsaw puzzle cutting.
type PuzzleOpts struct {
	Cols, Rows int
	TabSize    float64 // tab size relative to edge length (0.15-0.30, default 0.20)
	Seed       uint64  // random seed for tab direction
}

// Puzzle cuts the source image into jigsaw puzzle pieces with tabs and blanks.
func Puzzle(src image.Image, opts PuzzleOpts) ([]Piece, error) {
	if opts.Cols <= 0 || opts.Rows <= 0 {
		return nil, fmt.Errorf("slicer: cols and rows must be positive, got %dx%d", opts.Cols, opts.Rows)
	}
	if opts.TabSize <= 0 {
		opts.TabSize = 0.20
	}

	rng := rand.New(rand.NewPCG(opts.Seed, opts.Seed^0xcafebabe))

	b := src.Bounds()
	cellW := float64(b.Dx()) / float64(opts.Cols)
	cellH := float64(b.Dy()) / float64(opts.Rows)
	ox, oy := float64(b.Min.X), float64(b.Min.Y)

	// Generate tab directions for each internal edge.
	// +1 means tab protrudes in the positive direction (right/down).
	// -1 means tab protrudes in the negative direction (left/up).

	// Horizontal edges: between rows r and r+1, for each column c.
	// hEdges[r][c]: r in [0, rows-1), c in [0, cols)
	hEdges := make([][]int, opts.Rows-1)
	for r := range hEdges {
		hEdges[r] = make([]int, opts.Cols)
		for c := range hEdges[r] {
			if rng.IntN(2) == 0 {
				hEdges[r][c] = 1
			} else {
				hEdges[r][c] = -1
			}
		}
	}

	// Vertical edges: between columns c and c+1, for each row r.
	// vEdges[r][c]: r in [0, rows), c in [0, cols-1)
	vEdges := make([][]int, opts.Rows)
	for r := range vEdges {
		vEdges[r] = make([]int, opts.Cols-1)
		for c := range vEdges[r] {
			if rng.IntN(2) == 0 {
				vEdges[r][c] = 1
			} else {
				vEdges[r][c] = -1
			}
		}
	}

	pieces := make([]Piece, 0, opts.Cols*opts.Rows)
	id := 0

	for r := 0; r < opts.Rows; r++ {
		for c := 0; c < opts.Cols; c++ {
			x0 := ox + float64(c)*cellW
			y0 := oy + float64(r)*cellH
			x1 := x0 + cellW
			y1 := y0 + cellH

			var path Path
			path.MoveTo(x0, y0)

			// Top edge (left to right)
			if r == 0 {
				path.LineTo(x1, y0)
			} else {
				// Tab direction: hEdges[r-1][c], from piece above.
				// For the top edge of this piece, the tab goes upward if dir=1,
				// downward if dir=-1.
				puzzleEdge(&path, x0, y0, x1, y0, -hEdges[r-1][c], opts.TabSize)
			}

			// Right edge (top to bottom)
			if c == opts.Cols-1 {
				path.LineTo(x1, y1)
			} else {
				puzzleEdge(&path, x1, y0, x1, y1, vEdges[r][c], opts.TabSize)
			}

			// Bottom edge (right to left)
			if r == opts.Rows-1 {
				path.LineTo(x0, y1)
			} else {
				puzzleEdge(&path, x1, y1, x0, y1, hEdges[r][c], opts.TabSize)
			}

			// Left edge (bottom to top)
			if c == 0 {
				path.LineTo(x0, y0)
			} else {
				puzzleEdge(&path, x0, y1, x0, y0, -vEdges[r][c-1], opts.TabSize)
			}

			path.Close()
			pieces = append(pieces, clipPiece(src, id, path, image.Pt(c, r)))
			id++
		}
	}

	return pieces, nil
}

// puzzleEdge adds path commands for a puzzle edge from (x1,y1) to (x2,y2).
// tabDir: +1 tab protrudes to the left of travel direction, -1 to the right, 0 straight.
func puzzleEdge(p *Path, x1, y1, x2, y2 float64, tabDir int, tabSize float64) {
	if tabDir == 0 {
		p.LineTo(x2, y2)
		return
	}

	// Edge vector
	dx, dy := x2-x1, y2-y1
	// Perpendicular (90 degrees CCW): (-dy, dx)
	perpX, perpY := -dy, dx

	d := float64(tabDir) * tabSize

	// Transform helper: t = fraction along edge, n = fraction perpendicular.
	pt := func(t, n float64) (float64, float64) {
		return x1 + dx*t + perpX*n*d, y1 + dy*t + perpY*n*d
	}

	// Neck start (35% along the edge)
	nx1, ny1 := pt(0.35, 0)
	p.LineTo(nx1, ny1)

	// First bezier: from neck to tab peak
	cx1, cy1 := pt(0.35, 0.5)
	cx2, cy2 := pt(0.35, 1.0)
	tx, ty := pt(0.50, 1.0)
	p.CubicTo(cx1, cy1, cx2, cy2, tx, ty)

	// Second bezier: from tab peak to neck end
	cx3, cy3 := pt(0.65, 1.0)
	cx4, cy4 := pt(0.65, 0.5)
	nx2, ny2 := pt(0.65, 0)
	p.CubicTo(cx3, cy3, cx4, cy4, nx2, ny2)

	// Line to edge end
	p.LineTo(x2, y2)
}
