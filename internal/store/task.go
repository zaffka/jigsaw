package store

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
)

// Task represents a background task record.
type Task struct {
	ID       string
	Type     string
	Status   string
	Payload  json.RawMessage
	Error    string
	Attempts int
	UpdatedAt time.Time
}

// ClaimTask atomically claims one pending task using SKIP LOCKED.
// Returns ErrNotFound if no pending tasks.
func (s *Store) ClaimTask(ctx context.Context) (*Task, error) {
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	var t Task
	err = tx.QueryRow(ctx, `
		SELECT id, type, payload, attempts
		FROM tasks
		WHERE status = 'pending'
		ORDER BY created_at ASC
		LIMIT 1
		FOR UPDATE SKIP LOCKED
	`).Scan(&t.ID, &t.Type, &t.Payload, &t.Attempts)
	if err != nil {
		return nil, err // ErrNotFound if no rows
	}

	_, err = tx.Exec(ctx, `
		UPDATE tasks
		SET status = 'processing', attempts = attempts + 1, updated_at = now()
		WHERE id = $1
	`, t.ID)
	if err != nil {
		return nil, fmt.Errorf("claim task: %w", err)
	}

	t.Attempts++
	return &t, tx.Commit(ctx)
}

// CompleteTask marks a task as completed.
func (s *Store) CompleteTask(ctx context.Context, id string) error {
	_, err := s.db.Exec(ctx, `
		UPDATE tasks SET status = 'completed', error = NULL, updated_at = now()
		WHERE id = $1
	`, id)
	return err
}

// RetryOrFailTask puts a task back to pending (if attempts < max) or marks it failed.
func (s *Store) RetryOrFailTask(ctx context.Context, id string, errMsg string, maxAttempts int) error {
	_, err := s.db.Exec(ctx, `
		UPDATE tasks
		SET status     = CASE WHEN attempts >= $3 THEN 'failed' ELSE 'pending' END,
		    error      = $2,
		    updated_at = now()
		WHERE id = $1
	`, id, errMsg, maxAttempts)
	return err
}

// --- Puzzle queries used by worker ---

// Puzzle holds data needed for image processing.
type Puzzle struct {
	ID       string
	ImageKey string
	Status   string
	Config   map[string]any
}

// GetPuzzleByID returns a puzzle by its UUID.
func (s *Store) GetPuzzleByID(ctx context.Context, id string) (*Puzzle, error) {
	var p Puzzle
	var configRaw []byte
	err := s.db.QueryRow(ctx, `
		SELECT id, image_key, status, config FROM puzzles WHERE id = $1
	`, id).Scan(&p.ID, &p.ImageKey, &p.Status, &configRaw)
	if err != nil {
		return nil, err
	}
	json.Unmarshal(configRaw, &p.Config)
	return &p, nil
}

// SetPuzzleStatus updates puzzle status.
func (s *Store) SetPuzzleStatus(ctx context.Context, id, status string) error {
	_, err := s.db.Exec(ctx, `UPDATE puzzles SET status = $2 WHERE id = $1`, id, status)
	return err
}

// SetPuzzleReady marks a puzzle as ready and sets its difficulty.
func (s *Store) SetPuzzleReady(ctx context.Context, id, difficulty string) error {
	_, err := s.db.Exec(ctx, `
		UPDATE puzzles SET status = 'ready', difficulty = $2 WHERE id = $1
	`, id, difficulty)
	return err
}

// PuzzlePieceRecord holds data for one piece to be inserted.
type PuzzlePieceRecord struct {
	PuzzleID string
	ImageKey string
	PathSVG  string
	GridX    int
	GridY    int
	Bounds   map[string]int
}

// CreatePuzzlePieces batch-inserts puzzle pieces.
func (s *Store) CreatePuzzlePieces(ctx context.Context, pieces []PuzzlePieceRecord) error {
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	for _, p := range pieces {
		boundsJSON, _ := json.Marshal(p.Bounds)
		_, err := tx.Exec(ctx, `
			INSERT INTO puzzle_pieces (puzzle_id, image_key, path_svg, grid_x, grid_y, bounds)
			VALUES ($1, $2, $3, $4, $5, $6)
		`, p.PuzzleID, p.ImageKey, p.PathSVG, p.GridX, p.GridY, boundsJSON)
		if err != nil {
			return fmt.Errorf("insert piece: %w", err)
		}
	}

	return tx.Commit(ctx)
}
