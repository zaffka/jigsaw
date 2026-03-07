package slicer

import "testing"

func TestDucksGrid(t *testing.T) {
	src := loadJPEG(t, "ducks.jpg")
	pieces, err := Grid(src, GridOpts{Cols: 3, Rows: 3})
	if err != nil {
		t.Fatal(err)
	}
	if len(pieces) != 9 {
		t.Fatalf("got %d pieces, want 9", len(pieces))
	}
	savePieces(t, "testdata/ducks_grid", pieces)
}

func TestDucksMerge(t *testing.T) {
	src := loadJPEG(t, "ducks.jpg")
	pieces, err := Merge(src, MergeOpts{Cols: 4, Rows: 4, Seed: 13, MergeRatio: 0.7})
	if err != nil {
		t.Fatal(err)
	}
	if len(pieces) == 0 {
		t.Fatal("no pieces")
	}
	savePieces(t, "testdata/ducks_merge", pieces)
}

func TestDucksTriangles(t *testing.T) {
	src := loadJPEG(t, "ducks.jpg")
	pieces, err := Geometry(src, GeometryOpts{Cols: 3, Rows: 3, Shape: Triangles})
	if err != nil {
		t.Fatal(err)
	}
	if len(pieces) != 18 {
		t.Fatalf("got %d pieces, want 18", len(pieces))
	}
	savePieces(t, "testdata/ducks_triangles", pieces)
}

func TestDucksDiamonds(t *testing.T) {
	src := loadJPEG(t, "ducks.jpg")
	pieces, err := Geometry(src, GeometryOpts{Cols: 3, Rows: 3, Shape: Diamonds})
	if err != nil {
		t.Fatal(err)
	}
	if len(pieces) != 36 {
		t.Fatalf("got %d pieces, want 36", len(pieces))
	}
	savePieces(t, "testdata/ducks_diamonds", pieces)
}

func TestDucksTrapezoids(t *testing.T) {
	src := loadJPEG(t, "ducks.jpg")
	pieces, err := Geometry(src, GeometryOpts{Cols: 3, Rows: 3, Shape: Trapezoids})
	if err != nil {
		t.Fatal(err)
	}
	if len(pieces) != 18 {
		t.Fatalf("got %d pieces, want 18", len(pieces))
	}
	savePieces(t, "testdata/ducks_trapezoids", pieces)
}

func TestDucksParallelograms(t *testing.T) {
	src := loadJPEG(t, "ducks.jpg")
	pieces, err := Geometry(src, GeometryOpts{Cols: 4, Rows: 3, Shape: Parallelograms})
	if err != nil {
		t.Fatal(err)
	}
	if len(pieces) != 15 {
		t.Fatalf("got %d pieces, want 15", len(pieces))
	}
	savePieces(t, "testdata/ducks_parallelograms", pieces)
}

func TestDucksMixed(t *testing.T) {
	src := loadJPEG(t, "ducks.jpg")
	pieces, err := Geometry(src, GeometryOpts{Cols: 4, Rows: 4, Shape: Mixed, Seed: 77})
	if err != nil {
		t.Fatal(err)
	}
	if len(pieces) == 0 {
		t.Fatal("no pieces")
	}
	savePieces(t, "testdata/ducks_mixed", pieces)
}

func TestDucksPuzzle(t *testing.T) {
	src := loadJPEG(t, "ducks.jpg")
	pieces, err := Puzzle(src, PuzzleOpts{Cols: 3, Rows: 3, Seed: 99, TabSize: 0.22})
	if err != nil {
		t.Fatal(err)
	}
	if len(pieces) != 9 {
		t.Fatalf("got %d pieces, want 9", len(pieces))
	}
	savePieces(t, "testdata/ducks_puzzle", pieces)
}
