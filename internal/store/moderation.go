package store

import (
	"context"
	"fmt"
	"time"
)

// Submission represents a catalog_submissions row, optionally joined with puzzles.
type Submission struct {
	ID           string
	PuzzleID     string
	PuzzleTitle  string     // joined from puzzles (ListModerationQueue, ListParentNotifications)
	ImageKey     string     // joined from puzzles
	OwnerID      string     // puzzles.user_id
	Status       string
	AdminComment *string
	NotifiedAt   *time.Time
	CreatedAt    time.Time
	ReviewedAt   *time.Time
}

// scanSubmission scans 7 fields: id, puzzle_id, status, admin_comment, notified_at, created_at, reviewed_at.
func scanSubmission(row interface {
	Scan(...any) error
}, sub *Submission) error {
	return row.Scan(
		&sub.ID, &sub.PuzzleID, &sub.Status,
		&sub.AdminComment, &sub.NotifiedAt,
		&sub.CreatedAt, &sub.ReviewedAt,
	)
}

// scanSubmissionJoined scans 10 fields (joined with puzzles):
// id, puzzle_id, title, image_key, user_id, status, admin_comment, notified_at, created_at, reviewed_at.
func scanSubmissionJoined(row interface {
	Scan(...any) error
}) (*Submission, error) {
	var sub Submission
	err := row.Scan(
		&sub.ID, &sub.PuzzleID, &sub.PuzzleTitle, &sub.ImageKey, &sub.OwnerID,
		&sub.Status, &sub.AdminComment, &sub.NotifiedAt,
		&sub.CreatedAt, &sub.ReviewedAt,
	)
	if err != nil {
		return nil, err
	}
	return &sub, nil
}

// SubmitPuzzle creates or re-submits a catalog_submission for the given puzzle.
// Ownership check: puzzle must be owned by userID.
// If already pending or approved → ErrConflict.
// If previously rejected → resets to pending.
func (s *Store) SubmitPuzzle(ctx context.Context, puzzleID, userID string) (*Submission, error) {
	// 1. Verify ownership.
	var exists bool
	err := s.db.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM puzzles WHERE id = $1 AND user_id = $2)`,
		puzzleID, userID,
	).Scan(&exists)
	if err != nil {
		return nil, fmt.Errorf("submit puzzle: check ownership: %w", err)
	}
	if !exists {
		return nil, ErrNotFound
	}

	// 2. Try fresh insert.
	var sub Submission
	err = scanSubmission(s.db.QueryRow(ctx, `
		INSERT INTO catalog_submissions (puzzle_id) VALUES ($1)
		RETURNING id, puzzle_id, status, admin_comment, notified_at, created_at, reviewed_at
	`, puzzleID), &sub)
	if err == nil {
		return &sub, nil
	}
	if !isUniqueViolation(err) {
		return nil, fmt.Errorf("submit puzzle: %w", err)
	}

	// 3. Conflict — read existing status.
	err = scanSubmission(s.db.QueryRow(ctx, `
		SELECT id, puzzle_id, status, admin_comment, notified_at, created_at, reviewed_at
		FROM catalog_submissions WHERE puzzle_id = $1
	`, puzzleID), &sub)
	if err != nil {
		return nil, fmt.Errorf("submit puzzle: read existing: %w", err)
	}

	if sub.Status == "pending" || sub.Status == "approved" {
		return nil, ErrConflict
	}

	// 4. Was rejected — reset to pending.
	err = scanSubmission(s.db.QueryRow(ctx, `
		UPDATE catalog_submissions
		SET status = 'pending', reviewed_at = NULL, admin_comment = NULL, notified_at = NULL
		WHERE id = $1
		RETURNING id, puzzle_id, status, admin_comment, notified_at, created_at, reviewed_at
	`, sub.ID), &sub)
	if err != nil {
		return nil, fmt.Errorf("submit puzzle: reset: %w", err)
	}
	return &sub, nil
}

// ListModerationQueue returns pending submissions ordered by created_at ASC.
func (s *Store) ListModerationQueue(ctx context.Context) ([]*Submission, error) {
	rows, err := s.db.Query(ctx, `
		SELECT cs.id, cs.puzzle_id, p.title, p.image_key, p.user_id,
		       cs.status, cs.admin_comment, cs.notified_at, cs.created_at, cs.reviewed_at
		FROM catalog_submissions cs
		JOIN puzzles p ON p.id = cs.puzzle_id
		WHERE cs.status = 'pending'
		ORDER BY cs.created_at ASC
	`)
	if err != nil {
		return nil, fmt.Errorf("list moderation queue: %w", err)
	}
	defer rows.Close()
	var list []*Submission
	for rows.Next() {
		sub, err := scanSubmissionJoined(rows)
		if err != nil {
			return nil, err
		}
		list = append(list, sub)
	}
	return list, rows.Err()
}

// GetSubmission returns a submission by its ID.
func (s *Store) GetSubmission(ctx context.Context, id string) (*Submission, error) {
	var sub Submission
	err := scanSubmission(s.db.QueryRow(ctx, `
		SELECT id, puzzle_id, status, admin_comment, notified_at, created_at, reviewed_at
		FROM catalog_submissions WHERE id = $1
	`, id), &sub)
	if err != nil {
		return nil, err
	}
	return &sub, nil
}

// ApprovePuzzle marks a submission as approved, sets puzzle visibility to 'public',
// and creates a catalog_puzzles entry. All in one transaction.
func (s *Store) ApprovePuzzle(ctx context.Context, submissionID, reviewerID string) error {
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("approve puzzle: begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	// 1. Mark as approved (only if currently pending).
	tag, err := tx.Exec(ctx, `
		UPDATE catalog_submissions
		SET status = 'approved', reviewer_id = $2, reviewed_at = now()
		WHERE id = $1 AND status = 'pending'
	`, submissionID, reviewerID)
	if err != nil {
		return fmt.Errorf("approve puzzle: update submission: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}

	// 2. Get puzzle_id.
	var puzzleID string
	err = tx.QueryRow(ctx,
		`SELECT puzzle_id FROM catalog_submissions WHERE id = $1`,
		submissionID,
	).Scan(&puzzleID)
	if err != nil {
		return fmt.Errorf("approve puzzle: get puzzle_id: %w", err)
	}

	// 3. Set puzzle visibility to public.
	_, err = tx.Exec(ctx,
		`UPDATE puzzles SET visibility = 'public' WHERE id = $1`,
		puzzleID,
	)
	if err != nil {
		return fmt.Errorf("approve puzzle: set visibility: %w", err)
	}

	// 4. Create catalog_puzzles entry (idempotent).
	_, err = tx.Exec(ctx,
		`INSERT INTO catalog_puzzles (puzzle_id) VALUES ($1) ON CONFLICT DO NOTHING`,
		puzzleID,
	)
	if err != nil {
		return fmt.Errorf("approve puzzle: insert catalog_puzzles: %w", err)
	}

	return tx.Commit(ctx)
}

// RejectPuzzle marks a pending submission as rejected with an admin comment.
// Returns ErrNotFound if the submission does not exist or is not pending.
func (s *Store) RejectPuzzle(ctx context.Context, submissionID, reviewerID, comment string) error {
	tag, err := s.db.Exec(ctx, `
		UPDATE catalog_submissions
		SET status = 'rejected', reviewer_id = $2, admin_comment = $3, reviewed_at = now()
		WHERE id = $1 AND status = 'pending'
	`, submissionID, reviewerID, comment)
	if err != nil {
		return fmt.Errorf("reject puzzle: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// ListParentNotifications returns reviewed (approved/rejected) submissions for puzzles
// owned by userID that have not yet been marked as notified.
func (s *Store) ListParentNotifications(ctx context.Context, userID string) ([]*Submission, error) {
	rows, err := s.db.Query(ctx, `
		SELECT cs.id, cs.puzzle_id, p.title, p.image_key, p.user_id,
		       cs.status, cs.admin_comment, cs.notified_at, cs.created_at, cs.reviewed_at
		FROM catalog_submissions cs
		JOIN puzzles p ON p.id = cs.puzzle_id
		WHERE p.user_id = $1
		  AND cs.status IN ('approved', 'rejected')
		  AND cs.notified_at IS NULL
		ORDER BY cs.reviewed_at DESC
	`, userID)
	if err != nil {
		return nil, fmt.Errorf("list parent notifications: %w", err)
	}
	defer rows.Close()
	var list []*Submission
	for rows.Next() {
		sub, err := scanSubmissionJoined(rows)
		if err != nil {
			return nil, err
		}
		list = append(list, sub)
	}
	return list, rows.Err()
}

// MarkNotified sets notified_at = now() for the given submission.
func (s *Store) MarkNotified(ctx context.Context, submissionID string) error {
	_, err := s.db.Exec(ctx,
		`UPDATE catalog_submissions SET notified_at = now() WHERE id = $1`,
		submissionID,
	)
	return err
}
