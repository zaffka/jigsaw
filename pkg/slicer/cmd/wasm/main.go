//go:build js && wasm

package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"syscall/js"

	"github.com/zaffka/jigsaw/pkg/slicer"
)

func main() {
	js.Global().Set("slicer", js.ValueOf(map[string]any{
		"grid":       js.FuncOf(grid),
		"merge":      js.FuncOf(merge),
		"geometry":   js.FuncOf(geometry),
		"puzzle":     js.FuncOf(puzzle),
		"silhouette": js.FuncOf(silhouette),
	}))

	// Keep the Go runtime alive.
	select {}
}

// decodeImage decodes a JS Uint8Array into an image.Image.
func decodeImage(jsData js.Value) (image.Image, error) {
	length := jsData.Get("length").Int()
	buf := make([]byte, length)
	js.CopyBytesToGo(buf, jsData)

	img, _, err := image.Decode(bytes.NewReader(buf))
	if err != nil {
		return nil, fmt.Errorf("decode image: %w", err)
	}
	return img, nil
}

// pieceResult converts a slice of pieces into a JS-friendly result.
func pieceResult(pieces []slicer.Piece) (any, error) {
	type jsRect struct {
		X int `json:"x"`
		Y int `json:"y"`
		W int `json:"w"`
		H int `json:"h"`
	}
	type jsPiece struct {
		ID       int    `json:"id"`
		SVGPath  string `json:"svg_path"`
		Bounds   jsRect `json:"bounds"`
		TargetX  int    `json:"target_x"`
		TargetY  int    `json:"target_y"`
		Col      int    `json:"col"`
		Row      int    `json:"row"`
		ImageB64 string `json:"image_b64"`
	}

	result := make([]jsPiece, len(pieces))
	for i, p := range pieces {
		var buf bytes.Buffer
		if err := png.Encode(&buf, p.Image); err != nil {
			return nil, fmt.Errorf("encode piece %d: %w", p.ID, err)
		}

		result[i] = jsPiece{
			ID:      p.ID,
			SVGPath: p.Outline.SVG(),
			Bounds: jsRect{
				X: p.Bounds.Min.X,
				Y: p.Bounds.Min.Y,
				W: p.Bounds.Dx(),
				H: p.Bounds.Dy(),
			},
			TargetX:  p.Bounds.Min.X,
			TargetY:  p.Bounds.Min.Y,
			Col:      p.GridPos.X,
			Row:      p.GridPos.Y,
			ImageB64: base64.StdEncoding.EncodeToString(buf.Bytes()),
		}
	}

	data, err := json.Marshal(result)
	if err != nil {
		return nil, err
	}
	return string(data), nil
}

func grid(_ js.Value, args []js.Value) any {
	if len(args) < 3 {
		return map[string]any{"error": "usage: grid(imageData, cols, rows)"}
	}
	img, err := decodeImage(args[0])
	if err != nil {
		return map[string]any{"error": err.Error()}
	}
	pieces, err := slicer.Grid(img, slicer.GridOpts{
		Cols: args[1].Int(),
		Rows: args[2].Int(),
	})
	if err != nil {
		return map[string]any{"error": err.Error()}
	}
	result, err := pieceResult(pieces)
	if err != nil {
		return map[string]any{"error": err.Error()}
	}
	return map[string]any{"data": result}
}

func merge(_ js.Value, args []js.Value) any {
	if len(args) < 3 {
		return map[string]any{"error": "usage: merge(imageData, cols, rows)"}
	}
	img, err := decodeImage(args[0])
	if err != nil {
		return map[string]any{"error": err.Error()}
	}
	pieces, err := slicer.Merge(img, slicer.MergeOpts{
		Cols: args[1].Int(),
		Rows: args[2].Int(),
		Seed: uint64(args[3].Int()),
	})
	if err != nil {
		return map[string]any{"error": err.Error()}
	}
	result, err := pieceResult(pieces)
	if err != nil {
		return map[string]any{"error": err.Error()}
	}
	return map[string]any{"data": result}
}

func geometry(_ js.Value, args []js.Value) any {
	if len(args) < 4 {
		return map[string]any{"error": "usage: geometry(imageData, cols, rows, shape)"}
	}
	img, err := decodeImage(args[0])
	if err != nil {
		return map[string]any{"error": err.Error()}
	}

	shapeNames := map[string]slicer.ShapeType{
		"triangles":      slicer.Triangles,
		"diamonds":       slicer.Diamonds,
		"trapezoids":     slicer.Trapezoids,
		"parallelograms": slicer.Parallelograms,
		"mixed":          slicer.Mixed,
	}
	shape, ok := shapeNames[args[3].String()]
	if !ok {
		return map[string]any{"error": "unknown shape: " + args[3].String()}
	}

	pieces, err := slicer.Geometry(img, slicer.GeometryOpts{
		Cols:  args[1].Int(),
		Rows:  args[2].Int(),
		Shape: shape,
	})
	if err != nil {
		return map[string]any{"error": err.Error()}
	}
	result, err := pieceResult(pieces)
	if err != nil {
		return map[string]any{"error": err.Error()}
	}
	return map[string]any{"data": result}
}

func puzzle(_ js.Value, args []js.Value) any {
	if len(args) < 3 {
		return map[string]any{"error": "usage: puzzle(imageData, cols, rows)"}
	}
	img, err := decodeImage(args[0])
	if err != nil {
		return map[string]any{"error": err.Error()}
	}
	pieces, err := slicer.Puzzle(img, slicer.PuzzleOpts{
		Cols: args[1].Int(),
		Rows: args[2].Int(),
	})
	if err != nil {
		return map[string]any{"error": err.Error()}
	}
	result, err := pieceResult(pieces)
	if err != nil {
		return map[string]any{"error": err.Error()}
	}
	return map[string]any{"data": result}
}

func silhouette(_ js.Value, args []js.Value) any {
	if len(args) < 3 {
		return map[string]any{"error": "usage: silhouette(imageData, cols, rows)"}
	}
	img, err := decodeImage(args[0])
	if err != nil {
		return map[string]any{"error": err.Error()}
	}
	pieces, err := slicer.Puzzle(img, slicer.PuzzleOpts{
		Cols: args[1].Int(),
		Rows: args[2].Int(),
	})
	if err != nil {
		return map[string]any{"error": err.Error()}
	}
	b := img.Bounds()
	svg := slicer.Silhouette(pieces, b.Dx(), b.Dy(), nil)
	return map[string]any{"data": svg}
}

// Register JPEG decoder.
func init() {
	_ = jpeg.Decode
}
