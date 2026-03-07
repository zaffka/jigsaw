package slicer

import (
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"os"
	"path/filepath"
	"testing"
)

func loadJPEG(t *testing.T, path string) image.Image {
	t.Helper()
	f, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	img, err := jpeg.Decode(f)
	if err != nil {
		t.Fatal(err)
	}
	return img
}

func savePieces(t *testing.T, dir string, pieces []Piece) {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	for _, p := range pieces {
		name := filepath.Join(dir, fmt.Sprintf("piece_%03d.png", p.ID))
		f, err := os.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		if err := png.Encode(f, p.Image); err != nil {
			f.Close()
			t.Fatal(err)
		}
		f.Close()
	}
	t.Logf("%s: saved %d pieces", dir, len(pieces))
}

func TestCosmicGrid(t *testing.T) {
	src := loadJPEG(t, "cosmic.jpg")
	pieces, err := Grid(src, GridOpts{Cols: 4, Rows: 4})
	if err != nil {
		t.Fatal(err)
	}
	if len(pieces) != 16 {
		t.Fatalf("got %d pieces, want 16", len(pieces))
	}
	savePieces(t, "testdata/cosmic_grid", pieces)
}

func TestCosmicMerge(t *testing.T) {
	src := loadJPEG(t, "cosmic.jpg")
	pieces, err := Merge(src, MergeOpts{Cols: 4, Rows: 4, Seed: 7, MergeRatio: 0.6})
	if err != nil {
		t.Fatal(err)
	}
	if len(pieces) == 0 {
		t.Fatal("no pieces")
	}
	savePieces(t, "testdata/cosmic_merge", pieces)
}

func TestCosmicTriangles(t *testing.T) {
	src := loadJPEG(t, "cosmic.jpg")
	pieces, err := Geometry(src, GeometryOpts{Cols: 3, Rows: 3, Shape: Triangles})
	if err != nil {
		t.Fatal(err)
	}
	if len(pieces) != 18 {
		t.Fatalf("got %d pieces, want 18", len(pieces))
	}
	savePieces(t, "testdata/cosmic_triangles", pieces)
}

func TestCosmicDiamonds(t *testing.T) {
	src := loadJPEG(t, "cosmic.jpg")
	pieces, err := Geometry(src, GeometryOpts{Cols: 3, Rows: 3, Shape: Diamonds})
	if err != nil {
		t.Fatal(err)
	}
	if len(pieces) != 36 {
		t.Fatalf("got %d pieces, want 36", len(pieces))
	}
	savePieces(t, "testdata/cosmic_diamonds", pieces)
}

func TestCosmicTrapezoids(t *testing.T) {
	src := loadJPEG(t, "cosmic.jpg")
	pieces, err := Geometry(src, GeometryOpts{Cols: 3, Rows: 3, Shape: Trapezoids})
	if err != nil {
		t.Fatal(err)
	}
	if len(pieces) != 18 {
		t.Fatalf("got %d pieces, want 18", len(pieces))
	}
	savePieces(t, "testdata/cosmic_trapezoids", pieces)
}

func TestCosmicParallelograms(t *testing.T) {
	src := loadJPEG(t, "cosmic.jpg")
	pieces, err := Geometry(src, GeometryOpts{Cols: 4, Rows: 3, Shape: Parallelograms})
	if err != nil {
		t.Fatal(err)
	}
	// 4 parallelograms + 1 left triangle per row, 3 rows = 15
	if len(pieces) != 15 {
		t.Fatalf("got %d pieces, want 15", len(pieces))
	}
	savePieces(t, "testdata/cosmic_parallelograms", pieces)
}

func TestCosmicMixed(t *testing.T) {
	src := loadJPEG(t, "cosmic.jpg")
	pieces, err := Geometry(src, GeometryOpts{Cols: 4, Rows: 4, Shape: Mixed, Seed: 55})
	if err != nil {
		t.Fatal(err)
	}
	if len(pieces) == 0 {
		t.Fatal("no pieces")
	}
	savePieces(t, "testdata/cosmic_mixed", pieces)
}

func TestCosmicPuzzle(t *testing.T) {
	src := loadJPEG(t, "cosmic.jpg")
	pieces, err := Puzzle(src, PuzzleOpts{Cols: 4, Rows: 4, Seed: 42, TabSize: 0.2})
	if err != nil {
		t.Fatal(err)
	}
	if len(pieces) != 16 {
		t.Fatalf("got %d pieces, want 16", len(pieces))
	}
	savePieces(t, "testdata/cosmic_puzzle", pieces)
}
