package slicer

import (
	"fmt"
	"image"
	"math/rand/v2"
)

// CellGroup is a set of grid cells that form a single merged piece.
type CellGroup []image.Point

// MergeOpts configures merged-cell cutting.
type MergeOpts struct {
	Cols, Rows int
	Groups     []CellGroup // explicit merge groups; nil = auto-generate
	Seed       uint64      // random seed for auto-generation
	MergeRatio float64     // 0.0-1.0, probability of attempting a merge (default 0.5)
}

// Merge cuts the source image into pieces where some grid cells are merged
// into rectangles and L-shapes.
func Merge(src image.Image, opts MergeOpts) ([]Piece, error) {
	if opts.Cols <= 0 || opts.Rows <= 0 {
		return nil, fmt.Errorf("slicer: cols and rows must be positive, got %dx%d", opts.Cols, opts.Rows)
	}

	groups := opts.Groups
	if groups == nil {
		groups = autoMergeGroups(opts.Cols, opts.Rows, opts.Seed, opts.MergeRatio)
	}

	b := src.Bounds()
	cellW := b.Dx() / opts.Cols
	cellH := b.Dy() / opts.Rows

	pieces := make([]Piece, 0, len(groups))
	for id, group := range groups {
		outline := groupOutline(group, cellW, cellH, b.Min)
		path := pathFromPolygon(outline)
		// Use the top-left cell as the grid position.
		gridPos := group[0]
		for _, c := range group[1:] {
			if c.Y < gridPos.Y || (c.Y == gridPos.Y && c.X < gridPos.X) {
				gridPos = c
			}
		}
		pieces = append(pieces, clipPiece(src, id, path, gridPos))
	}
	return pieces, nil
}

// merge templates: relative cell offsets.
var mergeTemplates = [][]image.Point{
	// 2-cell pairs
	{{0, 0}, {1, 0}}, // horizontal
	{{0, 0}, {0, 1}}, // vertical
	// 3-cell L-shapes (4 rotations)
	{{0, 0}, {1, 0}, {0, 1}},
	{{0, 0}, {1, 0}, {1, 1}},
	{{0, 0}, {0, 1}, {1, 1}},
	{{1, 0}, {0, 1}, {1, 1}},
}

func autoMergeGroups(cols, rows int, seed uint64, mergeRatio float64) []CellGroup {
	if mergeRatio <= 0 {
		mergeRatio = 0.5
	}
	if mergeRatio > 1 {
		mergeRatio = 1
	}

	rng := rand.New(rand.NewPCG(seed, seed^0xdeadbeef))

	used := make([][]bool, cols)
	for i := range used {
		used[i] = make([]bool, rows)
	}

	// Randomized cell order
	cells := make([]image.Point, 0, cols*rows)
	for r := 0; r < rows; r++ {
		for c := 0; c < cols; c++ {
			cells = append(cells, image.Pt(c, r))
		}
	}
	rng.Shuffle(len(cells), func(i, j int) { cells[i], cells[j] = cells[j], cells[i] })

	var groups []CellGroup

	for _, cell := range cells {
		if used[cell.X][cell.Y] {
			continue
		}
		if rng.Float64() >= mergeRatio {
			// Single cell
			used[cell.X][cell.Y] = true
			groups = append(groups, CellGroup{cell})
			continue
		}

		// Try random templates
		placed := false
		order := rng.Perm(len(mergeTemplates))
		for _, ti := range order {
			tmpl := mergeTemplates[ti]
			ok := true
			for _, off := range tmpl {
				cx, cy := cell.X+off.X, cell.Y+off.Y
				if cx < 0 || cx >= cols || cy < 0 || cy >= rows || used[cx][cy] {
					ok = false
					break
				}
			}
			if ok {
				group := make(CellGroup, len(tmpl))
				for i, off := range tmpl {
					cx, cy := cell.X+off.X, cell.Y+off.Y
					used[cx][cy] = true
					group[i] = image.Pt(cx, cy)
				}
				groups = append(groups, group)
				placed = true
				break
			}
		}
		if !placed {
			used[cell.X][cell.Y] = true
			groups = append(groups, CellGroup{cell})
		}
	}
	return groups
}

// groupOutline computes the outer boundary polygon of a set of grid cells.
func groupOutline(cells []image.Point, cellW, cellH int, origin image.Point) []Point {
	cellSet := make(map[image.Point]bool, len(cells))
	for _, c := range cells {
		cellSet[c] = true
	}

	type ipt struct{ x, y int }
	type seg struct{ from, to ipt }

	var segments []seg
	for _, c := range cells {
		x := origin.X + c.X*cellW
		y := origin.Y + c.Y*cellH

		// Top edge (if no neighbor above)
		if !cellSet[image.Pt(c.X, c.Y-1)] {
			segments = append(segments, seg{ipt{x, y}, ipt{x + cellW, y}})
		}
		// Right edge
		if !cellSet[image.Pt(c.X+1, c.Y)] {
			segments = append(segments, seg{ipt{x + cellW, y}, ipt{x + cellW, y + cellH}})
		}
		// Bottom edge
		if !cellSet[image.Pt(c.X, c.Y+1)] {
			segments = append(segments, seg{ipt{x + cellW, y + cellH}, ipt{x, y + cellH}})
		}
		// Left edge
		if !cellSet[image.Pt(c.X-1, c.Y)] {
			segments = append(segments, seg{ipt{x, y + cellH}, ipt{x, y}})
		}
	}

	if len(segments) == 0 {
		return nil
	}

	// Build adjacency map and trace contour
	adj := make(map[ipt]ipt, len(segments))
	for _, s := range segments {
		adj[s.from] = s.to
	}

	start := segments[0].from
	var result []Point
	cur := start
	for {
		result = append(result, Point{float64(cur.x), float64(cur.y)})
		next := adj[cur]
		if next == start {
			break
		}
		cur = next
	}
	return result
}
