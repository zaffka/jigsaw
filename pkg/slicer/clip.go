package slicer

import (
	"image"
	"image/color"
)

// clipPiece extracts a piece from the source image using a path outline.
func clipPiece(src image.Image, id int, outline Path, gridPos image.Point) Piece {
	polygon := outline.Polygon(16)
	bounds := polygonBounds(polygon)
	srcBounds := src.Bounds()

	// Clamp bounds to source image
	if bounds.Min.X < srcBounds.Min.X {
		bounds.Min.X = srcBounds.Min.X
	}
	if bounds.Min.Y < srcBounds.Min.Y {
		bounds.Min.Y = srcBounds.Min.Y
	}
	if bounds.Max.X > srcBounds.Max.X {
		bounds.Max.X = srcBounds.Max.X
	}
	if bounds.Max.Y > srcBounds.Max.Y {
		bounds.Max.Y = srcBounds.Max.Y
	}

	dst := image.NewRGBA(image.Rect(0, 0, bounds.Dx(), bounds.Dy()))
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			// Test pixel center
			if pointInPolygon(Point{float64(x) + 0.5, float64(y) + 0.5}, polygon) {
				r, g, b, a := src.At(x, y).RGBA()
				dst.SetRGBA(x-bounds.Min.X, y-bounds.Min.Y, color.RGBA{
					R: uint8(r >> 8),
					G: uint8(g >> 8),
					B: uint8(b >> 8),
					A: uint8(a >> 8),
				})
			}
		}
	}

	return Piece{
		ID:      id,
		Image:   dst,
		Outline: outline,
		Bounds:  bounds,
		GridPos: gridPos,
	}
}

// pointInPolygon tests if a point is inside a polygon using ray casting.
func pointInPolygon(p Point, polygon []Point) bool {
	n := len(polygon)
	if n < 3 {
		return false
	}
	inside := false
	j := n - 1
	for i := 0; i < n; i++ {
		yi, yj := polygon[i].Y, polygon[j].Y
		if (yi > p.Y) != (yj > p.Y) {
			xi, xj := polygon[i].X, polygon[j].X
			if p.X < (xj-xi)*(p.Y-yi)/(yj-yi)+xi {
				inside = !inside
			}
		}
		j = i
	}
	return inside
}
