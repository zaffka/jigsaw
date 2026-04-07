package worker

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"io"

	"golang.org/x/image/draw"

	"github.com/zaffka/jigsaw/internal/store"
	"github.com/zaffka/jigsaw/pkg/s3"
	"github.com/zaffka/jigsaw/pkg/slicer"
	"go.uber.org/zap"
)

const maxImageDimension = 2048

type processImagePayload struct {
	PuzzleID string `json:"puzzle_id"`
}

func (w *Worker) processImage(ctx context.Context, task *store.Task) (retErr error) {
	var payload processImagePayload
	if err := json.Unmarshal(task.Payload, &payload); err != nil {
		return fmt.Errorf("parse payload: %w", err)
	}

	puzzle, err := w.store.GetPuzzleByID(ctx, payload.PuzzleID)
	if err != nil {
		return fmt.Errorf("get puzzle %s: %w", payload.PuzzleID, err)
	}

	// On any error, mark the puzzle as failed.
	defer func() {
		if retErr != nil {
			if failErr := w.store.SetPuzzleStatus(ctx, puzzle.ID, "failed"); failErr != nil {
				w.log.Error("set puzzle failed", zap.String("puzzle_id", puzzle.ID), zap.Error(failErr))
			}
		}
	}()

	// 1. Download original image from S3.
	obj, err := w.s3.GetObject(ctx, puzzle.ImageKey)
	if err != nil {
		return fmt.Errorf("download image %s: %w", puzzle.ImageKey, err)
	}
	defer obj.Body.Close()

	data, err := io.ReadAll(obj.Body)
	if err != nil {
		return fmt.Errorf("read image: %w", err)
	}

	// 2. Decode.
	src, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("decode image: %w", err)
	}

	// 3. Resize if too large.
	src = resizeIfNeeded(src, maxImageDimension)

	// 4. Slice.
	pieces, err := sliceImage(src, puzzle.Config)
	if err != nil {
		return fmt.Errorf("slice image: %w", err)
	}

	w.log.Info("image sliced", zap.String("puzzle_id", puzzle.ID), zap.Int("pieces", len(pieces)))

	// 5. Upload pieces and collect records.
	records := make([]store.PuzzlePieceRecord, 0, len(pieces))
	for _, p := range pieces {
		key, err := w.uploadPiece(ctx, puzzle.ID, p)
		if err != nil {
			return fmt.Errorf("upload piece %d: %w", p.ID, err)
		}
		records = append(records, store.PuzzlePieceRecord{
			PuzzleID: puzzle.ID,
			ImageKey: key,
			PathSVG:  p.Outline.SVG(),
			GridX:    p.GridPos.X,
			GridY:    p.GridPos.Y,
			Bounds: map[string]int{
				"x": p.Bounds.Min.X,
				"y": p.Bounds.Min.Y,
				"w": p.Bounds.Dx(),
				"h": p.Bounds.Dy(),
			},
		})
	}

	// 6. Save piece records.
	if err := w.store.CreatePuzzlePieces(ctx, records); err != nil {
		return fmt.Errorf("save pieces: %w", err)
	}

	// 7. Compute difficulty from config.
	cols := configInt(puzzle.Config, "cols", 4)
	rows := configInt(puzzle.Config, "rows", 3)
	difficulty := computeDifficulty(cols, rows)

	// 8. Mark puzzle ready with difficulty.
	if err := w.store.SetPuzzleReady(ctx, puzzle.ID, difficulty); err != nil {
		return fmt.Errorf("set puzzle ready: %w", err)
	}

	w.log.Info("puzzle ready",
		zap.String("puzzle_id", puzzle.ID),
		zap.String("difficulty", difficulty),
	)
	return nil
}

// computeDifficulty returns difficulty level based on total piece count.
// ≤6 → "easy", ≤16 → "medium", >16 → "hard".
func computeDifficulty(cols, rows int) string {
	total := cols * rows
	switch {
	case total <= 6:
		return "easy"
	case total <= 16:
		return "medium"
	default:
		return "hard"
	}
}

func (w *Worker) uploadPiece(ctx context.Context, puzzleID string, p slicer.Piece) (string, error) {
	var buf bytes.Buffer
	if err := png.Encode(&buf, p.Image); err != nil {
		return "", fmt.Errorf("encode piece: %w", err)
	}

	key := fmt.Sprintf("puzzle-pieces/%s/%d.png", puzzleID, p.ID)
	data := buf.Bytes()
	if _, err := w.s3.PutObject(ctx, key, bytes.NewReader(data), int64(len(data)),
		s3.PutObjectOptions{ContentType: "image/png"}); err != nil {
		return "", err
	}

	return key, nil
}

func sliceImage(src image.Image, config map[string]any) ([]slicer.Piece, error) {
	mode, _ := config["mode"].(string)
	cols := configInt(config, "cols", 4)
	rows := configInt(config, "rows", 3)
	seed := configUint64(config, "seed", 0)

	switch mode {
	case "merge":
		return slicer.Merge(src, slicer.MergeOpts{
			Cols:       cols,
			Rows:       rows,
			Seed:       seed,
			MergeRatio: configFloat(config, "merge_ratio", 0.5),
		})
	case "geometry":
		return slicer.Geometry(src, slicer.GeometryOpts{
			Cols:  cols,
			Rows:  rows,
			Seed:  seed,
			Shape: slicer.ShapeType(configInt(config, "shape", 0)),
		})
	case "puzzle":
		return slicer.Puzzle(src, slicer.PuzzleOpts{
			Cols:    cols,
			Rows:    rows,
			Seed:    seed,
			TabSize: configFloat(config, "tab_size", 0.20),
		})
	default: // "grid" or empty
		return slicer.Grid(src, slicer.GridOpts{Cols: cols, Rows: rows})
	}
}

// resizeIfNeeded scales down the image so neither dimension exceeds maxDim.
// Returns the original if already within bounds.
func resizeIfNeeded(src image.Image, maxDim int) image.Image {
	b := src.Bounds()
	w, h := b.Dx(), b.Dy()
	if w <= maxDim && h <= maxDim {
		return src
	}

	scale := float64(maxDim) / float64(max(w, h)) //nolint:predeclared
	newW := int(float64(w) * scale)
	newH := int(float64(h) * scale)

	dst := image.NewRGBA(image.Rect(0, 0, newW, newH))
	draw.BiLinear.Scale(dst, dst.Bounds(), src, src.Bounds(), draw.Over, nil)
	return dst
}

func configInt(cfg map[string]any, key string, def int) int {
	if v, ok := cfg[key]; ok {
		switch n := v.(type) {
		case float64:
			return int(n)
		case int:
			return n
		}
	}
	return def
}

func configUint64(cfg map[string]any, key string, def uint64) uint64 {
	if v, ok := cfg[key]; ok {
		if n, ok := v.(float64); ok {
			return uint64(n)
		}
	}
	return def
}

func configFloat(cfg map[string]any, key string, def float64) float64 {
	if v, ok := cfg[key]; ok {
		if n, ok := v.(float64); ok {
			return n
		}
	}
	return def
}

// generateTTS is a stub — real implementation in phase 5.
func (w *Worker) generateTTS(_ context.Context, task *store.Task) error {
	w.log.Info("generate_tts: stub, skipping", zap.String("task_id", task.ID))
	return nil
}

// Ensure image formats are registered.
var _ = jpeg.Decode
