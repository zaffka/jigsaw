package slicer

import (
	"encoding/json"
	"fmt"
	"strings"
)

// PieceMeta contains piece metadata for the game client.
type PieceMeta struct {
	ID      int       `json:"id"`
	SVGPath string    `json:"svg_path"`
	Bounds  RectMeta  `json:"bounds"`
	Target  PointMeta `json:"target"`
	GridPos CellMeta  `json:"grid_pos"`
}

// RectMeta is a JSON-friendly rectangle.
type RectMeta struct {
	X int `json:"x"`
	Y int `json:"y"`
	W int `json:"w"`
	H int `json:"h"`
}

// PointMeta is a JSON-friendly point.
type PointMeta struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
}

// CellMeta is a JSON-friendly grid cell position.
type CellMeta struct {
	Col int `json:"col"`
	Row int `json:"row"`
}

// ExportMeta converts a slice of pieces into a slice of client-ready metadata.
func ExportMeta(pieces []Piece) []PieceMeta {
	out := make([]PieceMeta, len(pieces))
	for i, p := range pieces {
		out[i] = PieceMeta{
			ID:      p.ID,
			SVGPath: p.Outline.SVG(),
			Bounds: RectMeta{
				X: p.Bounds.Min.X,
				Y: p.Bounds.Min.Y,
				W: p.Bounds.Dx(),
				H: p.Bounds.Dy(),
			},
			Target: PointMeta{
				X: float64(p.Bounds.Min.X),
				Y: float64(p.Bounds.Min.Y),
			},
			GridPos: CellMeta{
				Col: p.GridPos.X,
				Row: p.GridPos.Y,
			},
		}
	}
	return out
}

// ExportMetaJSON returns the piece metadata as a JSON byte slice.
func ExportMetaJSON(pieces []Piece) ([]byte, error) {
	return json.Marshal(ExportMeta(pieces))
}

// Silhouette generates a complete SVG document with piece outlines.
// The SVG uses the source image dimensions as the viewBox.
// Options control the visual style of the outlines.
func Silhouette(pieces []Piece, width, height int, opts *SilhouetteOpts) string {
	o := defaultSilhouetteOpts
	if opts != nil {
		if opts.Stroke != "" {
			o.Stroke = opts.Stroke
		}
		if opts.StrokeWidth > 0 {
			o.StrokeWidth = opts.StrokeWidth
		}
		if opts.Fill != "" {
			o.Fill = opts.Fill
		}
		if opts.Class != "" {
			o.Class = opts.Class
		}
	}

	var b strings.Builder
	fmt.Fprintf(&b, `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 %d %d">`+"\n", width, height)
	for _, p := range pieces {
		b.WriteString("  <path")
		if o.Class != "" {
			fmt.Fprintf(&b, ` class="%s"`, o.Class)
		}
		fmt.Fprintf(&b, ` data-id="%d"`, p.ID)
		fmt.Fprintf(&b, ` data-col="%d" data-row="%d"`, p.GridPos.X, p.GridPos.Y)
		fmt.Fprintf(&b, ` d="%s"`, p.Outline.SVG())
		fmt.Fprintf(&b, ` fill="%s" stroke="%s" stroke-width="%.1f"`, o.Fill, o.Stroke, o.StrokeWidth)
		b.WriteString("/>\n")
	}
	b.WriteString("</svg>\n")
	return b.String()
}

// SilhouetteOpts controls the visual style of the silhouette SVG.
type SilhouetteOpts struct {
	Stroke      string  // stroke color (default "#cccccc")
	StrokeWidth float64 // stroke width (default 2)
	Fill        string  // fill color (default "none")
	Class       string  // CSS class added to each <path> element
}

var defaultSilhouetteOpts = SilhouetteOpts{
	Stroke:      "#cccccc",
	StrokeWidth: 2,
	Fill:        "none",
}
