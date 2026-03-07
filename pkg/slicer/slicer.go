// Package slicer cuts images into pieces of various shapes for puzzle games.
//
// Four difficulty levels are supported:
//   - Grid: simple square grid
//   - Merge: combined squares forming rectangles and L-shapes
//   - Geometry: triangles, diamonds, trapezoids
//   - Puzzle: jigsaw pieces with tabs and blanks
package slicer

import "image"

// Point is a 2D point with float64 coordinates.
type Point struct {
	X, Y float64
}

// Piece represents a single cut piece of the source image.
type Piece struct {
	ID      int
	Image   *image.RGBA     // piece image with transparency outside the outline
	Outline Path            // outline path (for SVG export and rendering)
	Bounds  image.Rectangle // bounding box in source image coordinates
	GridPos image.Point     // grid position (meaningful for grid-based modes)
}
