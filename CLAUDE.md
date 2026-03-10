# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Jigsaw is a Go library (`pkg/slicer`) that cuts images into puzzle pieces for a children's educational web game. The backend computes all geometry server-side; a future frontend client receives piece metadata and PNG images for rendering.

## Commands

```bash
# Run all tests
go test ./...

# Run a single test
go test ./pkg/slicer -run TestDucksPuzzle

# Run tests with verbose output (useful for seeing saved testdata paths)
go test ./pkg/slicer -v
```

No build step — this is a library package with no `main`.

## Architecture

All code lives in a single package: `pkg/slicer`.

**Four cutting modes**, each with its own opts struct and entry function:
- `Grid(src, GridOpts)` — rectangular grid
- `Merge(src, MergeOpts)` — merged cells forming rectangles and L-shapes
- `Geometry(src, GeometryOpts)` — triangles, diamonds, trapezoids, parallelograms, mixed
- `Puzzle(src, PuzzleOpts)` — jigsaw pieces with cubic Bezier tabs/blanks

**Core types:**
- `Piece` — output of every cutting mode: contains `*image.RGBA`, `Path` (outline), `Bounds`, `GridPos`
- `Path` / `PathCmd` — vector path with MoveTo/LineTo/CubicTo/Close commands; can export to SVG string or flatten to polygon
- `Point` — float64 2D point used throughout geometry calculations

**Key pipeline:** each mode builds a `Path` outline per piece, then calls `clipPiece()` which flattens the path to a polygon, computes bounds, and rasterizes pixels inside the outline via ray-casting (`pointInPolygon`).

**Export layer** (`export.go`): `ExportMeta`/`ExportMetaJSON` convert pieces to client-ready JSON; `Silhouette` generates an SVG document with all piece outlines.

## Tests

Tests use JPEG fixtures (`ducks.jpg`, `cosmic.jpg`) in `pkg/slicer/`. Image-based tests save PNG output to `pkg/slicer/testdata/` for visual inspection. Test helpers `loadJPEG` and `savePieces` are in `cosmic_test.go`.

## Языки

- **Общение с пользователем**: русский
- **Документация** (CLAUDE.md, docs/, коммиты, PR): русский
- **Комментарии в коде, имена переменных, API**: английский
