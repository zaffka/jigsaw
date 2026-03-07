package slicer

import (
	"fmt"
	"image"
	"math"
	"strings"
)

// PathCmdType enumerates path command types.
type PathCmdType int

const (
	MoveToCmd  PathCmdType = iota
	LineToCmd
	CubicToCmd
	CloseCmd
)

// PathCmd is a single path command.
type PathCmd struct {
	Type   PathCmdType
	Points []Point // 0 for Close, 1 for MoveTo/LineTo, 3 for CubicTo
}

// Path is a sequence of path commands defining a shape outline.
type Path struct {
	Cmds []PathCmd
}

// MoveTo starts a new subpath at the given point.
func (p *Path) MoveTo(x, y float64) {
	p.Cmds = append(p.Cmds, PathCmd{Type: MoveToCmd, Points: []Point{{x, y}}})
}

// LineTo adds a straight line to the given point.
func (p *Path) LineTo(x, y float64) {
	p.Cmds = append(p.Cmds, PathCmd{Type: LineToCmd, Points: []Point{{x, y}}})
}

// CubicTo adds a cubic Bezier curve with two control points and an endpoint.
func (p *Path) CubicTo(cx1, cy1, cx2, cy2, x, y float64) {
	p.Cmds = append(p.Cmds, PathCmd{
		Type:   CubicToCmd,
		Points: []Point{{cx1, cy1}, {cx2, cy2}, {x, y}},
	})
}

// Close closes the current subpath.
func (p *Path) Close() {
	p.Cmds = append(p.Cmds, PathCmd{Type: CloseCmd})
}

// Polygon flattens the path to a polygon by sampling cubic Bezier curves.
// curveSegments controls how many line segments approximate each curve (default 16).
func (p Path) Polygon(curveSegments int) []Point {
	if curveSegments <= 0 {
		curveSegments = 16
	}
	var pts []Point
	var cur Point
	for _, cmd := range p.Cmds {
		switch cmd.Type {
		case MoveToCmd:
			cur = cmd.Points[0]
			pts = append(pts, cur)
		case LineToCmd:
			cur = cmd.Points[0]
			pts = append(pts, cur)
		case CubicToCmd:
			cp1, cp2, end := cmd.Points[0], cmd.Points[1], cmd.Points[2]
			for i := 1; i <= curveSegments; i++ {
				t := float64(i) / float64(curveSegments)
				pts = append(pts, cubicBezier(t, cur, cp1, cp2, end))
			}
			cur = end
		case CloseCmd:
			// polygon is implicitly closed
		}
	}
	return pts
}

// SVG returns the SVG path data string for this path.
func (p Path) SVG() string {
	var b strings.Builder
	for _, cmd := range p.Cmds {
		switch cmd.Type {
		case MoveToCmd:
			fmt.Fprintf(&b, "M %.2f %.2f ", cmd.Points[0].X, cmd.Points[0].Y)
		case LineToCmd:
			fmt.Fprintf(&b, "L %.2f %.2f ", cmd.Points[0].X, cmd.Points[0].Y)
		case CubicToCmd:
			fmt.Fprintf(&b, "C %.2f %.2f %.2f %.2f %.2f %.2f ",
				cmd.Points[0].X, cmd.Points[0].Y,
				cmd.Points[1].X, cmd.Points[1].Y,
				cmd.Points[2].X, cmd.Points[2].Y)
		case CloseCmd:
			b.WriteString("Z ")
		}
	}
	return strings.TrimSpace(b.String())
}

// cubicBezier evaluates a cubic Bezier curve at parameter t.
func cubicBezier(t float64, p0, p1, p2, p3 Point) Point {
	u := 1 - t
	return Point{
		X: u*u*u*p0.X + 3*u*u*t*p1.X + 3*u*t*t*p2.X + t*t*t*p3.X,
		Y: u*u*u*p0.Y + 3*u*u*t*p1.Y + 3*u*t*t*p2.Y + t*t*t*p3.Y,
	}
}

// polygonBounds returns the bounding rectangle of a polygon.
func polygonBounds(pts []Point) image.Rectangle {
	if len(pts) == 0 {
		return image.Rectangle{}
	}
	minX, minY := pts[0].X, pts[0].Y
	maxX, maxY := pts[0].X, pts[0].Y
	for _, p := range pts[1:] {
		minX = math.Min(minX, p.X)
		minY = math.Min(minY, p.Y)
		maxX = math.Max(maxX, p.X)
		maxY = math.Max(maxY, p.Y)
	}
	return image.Rect(
		int(math.Floor(minX)),
		int(math.Floor(minY)),
		int(math.Ceil(maxX)),
		int(math.Ceil(maxY)),
	)
}

// pathFromPolygon creates a Path from a polygon (slice of points).
func pathFromPolygon(pts []Point) Path {
	var p Path
	if len(pts) == 0 {
		return p
	}
	p.MoveTo(pts[0].X, pts[0].Y)
	for _, pt := range pts[1:] {
		p.LineTo(pt.X, pt.Y)
	}
	p.Close()
	return p
}
