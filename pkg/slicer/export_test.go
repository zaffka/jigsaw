package slicer

import (
	"encoding/json"
	"os"
	"strings"
	"testing"
)

func TestExportMeta(t *testing.T) {
	src := testImage(300, 300)
	pieces, err := Puzzle(src, PuzzleOpts{Cols: 3, Rows: 3, Seed: 1})
	if err != nil {
		t.Fatal(err)
	}

	meta := ExportMeta(pieces)
	if len(meta) != 9 {
		t.Fatalf("got %d meta entries, want 9", len(meta))
	}

	// Verify fields are populated.
	for _, m := range meta {
		if m.SVGPath == "" {
			t.Errorf("piece %d: empty SVG path", m.ID)
		}
		if m.Bounds.W <= 0 || m.Bounds.H <= 0 {
			t.Errorf("piece %d: invalid bounds %+v", m.ID, m.Bounds)
		}
	}
}

func TestExportMetaJSON(t *testing.T) {
	src := testImage(200, 200)
	pieces, err := Grid(src, GridOpts{Cols: 2, Rows: 2})
	if err != nil {
		t.Fatal(err)
	}

	data, err := ExportMetaJSON(pieces)
	if err != nil {
		t.Fatal(err)
	}

	// Must be valid JSON.
	var parsed []PieceMeta
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if len(parsed) != 4 {
		t.Fatalf("got %d entries, want 4", len(parsed))
	}
	// Verify grid positions.
	if parsed[0].GridPos != (CellMeta{Col: 0, Row: 0}) {
		t.Errorf("piece 0 grid_pos = %+v", parsed[0].GridPos)
	}
	if parsed[3].GridPos != (CellMeta{Col: 1, Row: 1}) {
		t.Errorf("piece 3 grid_pos = %+v", parsed[3].GridPos)
	}
}

func TestSilhouette(t *testing.T) {
	src := testImage(200, 200)
	pieces, err := Grid(src, GridOpts{Cols: 2, Rows: 2})
	if err != nil {
		t.Fatal(err)
	}

	svg := Silhouette(pieces, 200, 200, nil)

	if !strings.Contains(svg, `<svg xmlns=`) {
		t.Error("missing svg element")
	}
	if !strings.Contains(svg, `viewBox="0 0 200 200"`) {
		t.Error("wrong viewBox")
	}
	if strings.Count(svg, "<path") != 4 {
		t.Errorf("expected 4 path elements, got %d", strings.Count(svg, "<path"))
	}
	if !strings.Contains(svg, `data-id="0"`) {
		t.Error("missing data-id attribute")
	}
	if !strings.Contains(svg, `stroke="#cccccc"`) {
		t.Error("missing default stroke")
	}
}

func TestSilhouetteCustomOpts(t *testing.T) {
	src := testImage(200, 200)
	pieces, err := Grid(src, GridOpts{Cols: 2, Rows: 2})
	if err != nil {
		t.Fatal(err)
	}

	svg := Silhouette(pieces, 200, 200, &SilhouetteOpts{
		Stroke:      "#ff0000",
		StrokeWidth: 3,
		Fill:        "rgba(0,0,0,0.05)",
		Class:       "piece-outline",
	})

	if !strings.Contains(svg, `stroke="#ff0000"`) {
		t.Error("custom stroke not applied")
	}
	if !strings.Contains(svg, `stroke-width="3.0"`) {
		t.Error("custom stroke-width not applied")
	}
	if !strings.Contains(svg, `fill="rgba(0,0,0,0.05)"`) {
		t.Error("custom fill not applied")
	}
	if !strings.Contains(svg, `class="piece-outline"`) {
		t.Error("custom class not applied")
	}
}

func TestSilhouetteDucksFile(t *testing.T) {
	src := loadJPEG(t, "ducks.jpg")
	pieces, err := Puzzle(src, PuzzleOpts{Cols: 3, Rows: 3, Seed: 99, TabSize: 0.22})
	if err != nil {
		t.Fatal(err)
	}

	svg := Silhouette(pieces, 1024, 1024, &SilhouetteOpts{
		Stroke:      "#666666",
		StrokeWidth: 2,
		Fill:        "rgba(200,200,200,0.1)",
		Class:       "target",
	})

	if err := os.MkdirAll("testdata", 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile("testdata/ducks_puzzle_silhouette.svg", []byte(svg), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Logf("saved testdata/ducks_puzzle_silhouette.svg (%d bytes)", len(svg))

	// Verify SVG contains cubic bezier curves.
	if !strings.Contains(svg, " C ") {
		t.Error("puzzle silhouette should contain bezier curves")
	}
}

func TestExportMetaDucksJSON(t *testing.T) {
	src := loadJPEG(t, "ducks.jpg")
	pieces, err := Puzzle(src, PuzzleOpts{Cols: 3, Rows: 3, Seed: 99})
	if err != nil {
		t.Fatal(err)
	}

	data, err := ExportMetaJSON(pieces)
	if err != nil {
		t.Fatal(err)
	}

	if err := os.MkdirAll("testdata", 0o755); err != nil {
		t.Fatal(err)
	}

	// Pretty-print for inspection.
	var raw any
	json.Unmarshal(data, &raw)
	pretty, _ := json.MarshalIndent(raw, "", "  ")
	if err := os.WriteFile("testdata/ducks_puzzle_meta.json", pretty, 0o644); err != nil {
		t.Fatal(err)
	}
	t.Logf("saved testdata/ducks_puzzle_meta.json (%d bytes)", len(pretty))
}
