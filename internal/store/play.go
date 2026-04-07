package store

import (
	"context"
	"fmt"
)

// RecordPlayResult records a puzzle completion for a child.
func (s *Store) RecordPlayResult(ctx context.Context, childID, puzzleID string) error {
	_, err := s.db.Exec(ctx, `
		INSERT INTO play_results (child_id, puzzle_id, completed)
		VALUES ($1, $2, true)
	`, childID, puzzleID)
	if err != nil {
		return fmt.Errorf("record play result: %w", err)
	}
	return nil
}

// GetCompletedPuzzleIDs returns all catalog puzzle IDs this child has completed.
// Returns catalog_puzzles.id values (the public catalog id), not internal puzzle IDs.
func (s *Store) GetCompletedPuzzleIDs(ctx context.Context, childID string) ([]string, error) {
	rows, err := s.db.Query(ctx, `
		SELECT DISTINCT COALESCE(cp.id, pr.puzzle_id::text)
		FROM play_results pr
		LEFT JOIN catalog_puzzles cp ON cp.puzzle_id = pr.puzzle_id
		WHERE pr.child_id = $1 AND pr.completed = true
	`, childID)
	if err != nil {
		return nil, fmt.Errorf("get completed puzzles: %w", err)
	}
	defer rows.Close()
	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}
