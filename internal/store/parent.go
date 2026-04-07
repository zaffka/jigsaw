package store

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	"golang.org/x/crypto/bcrypt"
)

// --- Child ---

type Child struct {
	ID          string
	UserID      string
	Name        string
	PinHash     string
	AvatarEmoji string
	CreatedAt   time.Time
}

// ListChildren returns children belonging to userID, ordered by created_at ASC.
func (s *Store) ListChildren(ctx context.Context, userID string) ([]*Child, error) {
	rows, err := s.db.Query(ctx, `
		SELECT id, user_id, name, COALESCE(pin_hash, ''), avatar_emoji, created_at
		FROM children
		WHERE user_id = $1
		ORDER BY created_at ASC
	`, userID)
	if err != nil {
		return nil, fmt.Errorf("list children: %w", err)
	}
	defer rows.Close()
	var list []*Child
	for rows.Next() {
		var c Child
		if err := rows.Scan(&c.ID, &c.UserID, &c.Name, &c.PinHash, &c.AvatarEmoji, &c.CreatedAt); err != nil {
			return nil, err
		}
		list = append(list, &c)
	}
	return list, rows.Err()
}

// CreateChild creates a child profile. pin is bcrypt-hashed before storage.
// If pin == "", PinHash is stored as "".
func (s *Store) CreateChild(ctx context.Context, userID, name, pin, avatarEmoji string) (*Child, error) {
	pinHash := ""
	if pin != "" {
		h, err := bcrypt.GenerateFromPassword([]byte(pin), bcrypt.DefaultCost)
		if err != nil {
			return nil, fmt.Errorf("hash pin: %w", err)
		}
		pinHash = string(h)
	}
	var c Child
	err := s.db.QueryRow(ctx, `
		INSERT INTO children (user_id, name, pin_hash, avatar_emoji)
		VALUES ($1, $2, $3, $4)
		RETURNING id, user_id, name, COALESCE(pin_hash, ''), avatar_emoji, created_at
	`, userID, name, pinHash, avatarEmoji).Scan(
		&c.ID, &c.UserID, &c.Name, &c.PinHash, &c.AvatarEmoji, &c.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("create child: %w", err)
	}
	return &c, nil
}

// GetChild returns a child by ID (any owner).
func (s *Store) GetChild(ctx context.Context, id string) (*Child, error) {
	var c Child
	err := s.db.QueryRow(ctx, `
		SELECT id, user_id, name, COALESCE(pin_hash, ''), avatar_emoji, created_at
		FROM children WHERE id = $1
	`, id).Scan(&c.ID, &c.UserID, &c.Name, &c.PinHash, &c.AvatarEmoji, &c.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &c, nil
}

// UpdateChild updates name and avatarEmoji.
// If pin != "", re-hashes and updates pin_hash too.
// If pin == "", leaves pin_hash unchanged.
func (s *Store) UpdateChild(ctx context.Context, id, name, pin, avatarEmoji string) (*Child, error) {
	var c Child
	var scanErr error
	if pin != "" {
		h, err := bcrypt.GenerateFromPassword([]byte(pin), bcrypt.DefaultCost)
		if err != nil {
			return nil, fmt.Errorf("hash pin: %w", err)
		}
		scanErr = s.db.QueryRow(ctx, `
			UPDATE children
			SET name = $2, avatar_emoji = $3, pin_hash = $4
			WHERE id = $1
			RETURNING id, user_id, name, COALESCE(pin_hash, ''), avatar_emoji, created_at
		`, id, name, avatarEmoji, string(h)).Scan(
			&c.ID, &c.UserID, &c.Name, &c.PinHash, &c.AvatarEmoji, &c.CreatedAt,
		)
	} else {
		scanErr = s.db.QueryRow(ctx, `
			UPDATE children
			SET name = $2, avatar_emoji = $3
			WHERE id = $1
			RETURNING id, user_id, name, COALESCE(pin_hash, ''), avatar_emoji, created_at
		`, id, name, avatarEmoji).Scan(
			&c.ID, &c.UserID, &c.Name, &c.PinHash, &c.AvatarEmoji, &c.CreatedAt,
		)
	}
	if scanErr != nil {
		return nil, scanErr
	}
	return &c, nil
}

// DeleteChild deletes a child by ID.
func (s *Store) DeleteChild(ctx context.Context, id string) error {
	_, err := s.db.Exec(ctx, `DELETE FROM children WHERE id = $1`, id)
	return err
}

// --- Child sessions ---

// CreateChildSession generates a random token, stores it, returns the token.
func (s *Store) CreateChildSession(ctx context.Context, childID string) (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("generate token: %w", err)
	}
	token := hex.EncodeToString(b)
	_, err := s.db.Exec(ctx, `
		INSERT INTO child_sessions (child_id, token) VALUES ($1, $2)
	`, childID, token)
	if err != nil {
		return "", fmt.Errorf("create child session: %w", err)
	}
	return token, nil
}

// GetChildByToken returns the child associated with the session token.
// Returns ErrNotFound if token doesn't exist.
func (s *Store) GetChildByToken(ctx context.Context, token string) (*Child, error) {
	var c Child
	err := s.db.QueryRow(ctx, `
		SELECT ch.id, ch.user_id, ch.name, COALESCE(ch.pin_hash, ''), ch.avatar_emoji, ch.created_at
		FROM child_sessions cs
		JOIN children ch ON ch.id = cs.child_id
		WHERE cs.token = $1
	`, token).Scan(&c.ID, &c.UserID, &c.Name, &c.PinHash, &c.AvatarEmoji, &c.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &c, nil
}

// DeleteChildSession removes a session by token.
func (s *Store) DeleteChildSession(ctx context.Context, token string) error {
	_, err := s.db.Exec(ctx, `DELETE FROM child_sessions WHERE token = $1`, token)
	return err
}

// VerifyChildPIN checks a plaintext PIN against the stored hash.
// Returns false (no error) if hash is empty or PIN doesn't match.
func (s *Store) VerifyChildPIN(ctx context.Context, childID, pin string) (bool, error) {
	var pinHash string
	err := s.db.QueryRow(ctx, `
		SELECT COALESCE(pin_hash, '') FROM children WHERE id = $1
	`, childID).Scan(&pinHash)
	if err != nil {
		return false, err
	}
	if pinHash == "" {
		return false, nil
	}
	err = bcrypt.CompareHashAndPassword([]byte(pinHash), []byte(pin))
	if err != nil {
		return false, nil
	}
	return true, nil
}

// --- Parent puzzles ---

type ParentPuzzle struct {
	ID           string
	UserID       string
	Title        string
	Locale       string
	ImageKey     string
	Status       string
	Config       map[string]any
	CategoryID   *string
	CategorySlug *string
	Difficulty   string
	CreatedAt    time.Time
}

func scanParentPuzzle(row interface {
	Scan(...any) error
}) (*ParentPuzzle, error) {
	var p ParentPuzzle
	var configRaw []byte
	err := row.Scan(
		&p.ID, &p.UserID, &p.Title, &p.Locale, &p.ImageKey, &p.Status,
		&configRaw, &p.CategoryID, &p.CategorySlug, &p.Difficulty, &p.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(configRaw, &p.Config); err != nil {
		return nil, fmt.Errorf("unmarshal config: %w", err)
	}
	return &p, nil
}

const parentPuzzleSelect = `
	SELECT p.id, p.user_id, p.title, p.locale, p.image_key, p.status, p.config,
	       p.category_id, cat.slug, p.difficulty, p.created_at
	FROM puzzles p
	LEFT JOIN categories cat ON cat.id = p.category_id`

// CreateParentPuzzle inserts into puzzles (owner_type='parent', visibility='private')
// and immediately enqueues a 'process_image' task. Returns the new ParentPuzzle.
func (s *Store) CreateParentPuzzle(ctx context.Context, userID, title, locale, imageKey string, config map[string]any, categoryID *string) (*ParentPuzzle, error) {
	configJSON, err := json.Marshal(config)
	if err != nil {
		return nil, fmt.Errorf("marshal config: %w", err)
	}

	tx, err := s.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	var puzzleID string
	err = tx.QueryRow(ctx, `
		INSERT INTO puzzles (user_id, title, locale, image_key, status, config, category_id, owner_type, visibility)
		VALUES ($1, $2, $3, $4, 'processing', $5, $6, 'parent', 'private')
		RETURNING id
	`, userID, title, locale, imageKey, configJSON, categoryID).Scan(&puzzleID)
	if err != nil {
		return nil, fmt.Errorf("create puzzle: %w", err)
	}

	taskPayload, _ := json.Marshal(map[string]string{"puzzle_id": puzzleID})
	_, err = tx.Exec(ctx, `
		INSERT INTO tasks (type, payload) VALUES ('process_image', $1)
	`, taskPayload)
	if err != nil {
		return nil, fmt.Errorf("create task: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit: %w", err)
	}

	p := &ParentPuzzle{
		ID:         puzzleID,
		UserID:     userID,
		Title:      title,
		Locale:     locale,
		ImageKey:   imageKey,
		Status:     "processing",
		Config:     config,
		CategoryID: categoryID,
		Difficulty: "",
		CreatedAt:  time.Now(),
	}
	return p, nil
}

// ListParentPuzzles returns all parent puzzles owned by userID, ordered by created_at DESC.
func (s *Store) ListParentPuzzles(ctx context.Context, userID string) ([]*ParentPuzzle, error) {
	rows, err := s.db.Query(ctx, parentPuzzleSelect+`
		WHERE p.owner_type = 'parent' AND p.user_id = $1
		ORDER BY p.created_at DESC
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []*ParentPuzzle
	for rows.Next() {
		p, err := scanParentPuzzle(rows)
		if err != nil {
			return nil, err
		}
		list = append(list, p)
	}
	return list, rows.Err()
}

// GetParentPuzzle returns a parent puzzle by ID, only if owned by userID.
// Returns ErrNotFound otherwise.
func (s *Store) GetParentPuzzle(ctx context.Context, id, userID string) (*ParentPuzzle, error) {
	return scanParentPuzzle(s.db.QueryRow(ctx,
		parentPuzzleSelect+` WHERE p.id = $1 AND p.user_id = $2 AND p.owner_type = 'parent'`,
		id, userID,
	))
}

// UpdateParentPuzzle updates title and categoryID (only if owned by userID).
// Returns ErrNotFound if not found/not owner.
func (s *Store) UpdateParentPuzzle(ctx context.Context, id, userID, title string, categoryID *string) error {
	tag, err := s.db.Exec(ctx, `
		UPDATE puzzles
		SET title = $3, category_id = $4
		WHERE id = $1 AND user_id = $2 AND owner_type = 'parent'
	`, id, userID, title, categoryID)
	if err != nil {
		return fmt.Errorf("update parent puzzle: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// DeleteParentPuzzle deletes puzzle (and cascades) only if owned by userID.
// Returns ErrNotFound if not found/not owner.
func (s *Store) DeleteParentPuzzle(ctx context.Context, id, userID string) error {
	tag, err := s.db.Exec(ctx, `
		DELETE FROM puzzles
		WHERE id = $1 AND user_id = $2 AND owner_type = 'parent'
	`, id, userID)
	if err != nil {
		return fmt.Errorf("delete parent puzzle: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// --- Puzzle layers ---

type PuzzleLayer struct {
	ID        string
	PuzzleID  string
	SortOrder int
	Type      string
	Text      *string
	AudioKey  *string
	TTSKey    *string
	VideoKey  *string
	CreatedAt time.Time
}

type LayerOrderItem struct {
	ID        string
	SortOrder int
}

func scanPuzzleLayer(row interface {
	Scan(...any) error
}) (*PuzzleLayer, error) {
	var l PuzzleLayer
	err := row.Scan(
		&l.ID, &l.PuzzleID, &l.SortOrder, &l.Type,
		&l.Text, &l.AudioKey, &l.TTSKey, &l.VideoKey, &l.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &l, nil
}

// ListPuzzleLayers returns layers for a puzzle, ordered by sort_order ASC.
func (s *Store) ListPuzzleLayers(ctx context.Context, puzzleID string) ([]*PuzzleLayer, error) {
	rows, err := s.db.Query(ctx, `
		SELECT id, puzzle_id, sort_order, type, text, audio_key, tts_key, video_key, created_at
		FROM puzzle_layers
		WHERE puzzle_id = $1
		ORDER BY sort_order ASC
	`, puzzleID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []*PuzzleLayer
	for rows.Next() {
		l, err := scanPuzzleLayer(rows)
		if err != nil {
			return nil, err
		}
		list = append(list, l)
	}
	return list, rows.Err()
}

// CreatePuzzleLayer inserts a new layer.
func (s *Store) CreatePuzzleLayer(ctx context.Context, puzzleID, layerType string, text, audioKey, videoKey *string, sortOrder int) (*PuzzleLayer, error) {
	return scanPuzzleLayer(s.db.QueryRow(ctx, `
		INSERT INTO puzzle_layers (puzzle_id, type, text, audio_key, video_key, sort_order)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, puzzle_id, sort_order, type, text, audio_key, tts_key, video_key, created_at
	`, puzzleID, layerType, text, audioKey, videoKey, sortOrder))
}

// UpdatePuzzleLayer updates a layer by ID.
func (s *Store) UpdatePuzzleLayer(ctx context.Context, id, layerType string, text, audioKey, videoKey *string, sortOrder int) (*PuzzleLayer, error) {
	return scanPuzzleLayer(s.db.QueryRow(ctx, `
		UPDATE puzzle_layers
		SET type = $2, text = $3, audio_key = $4, video_key = $5, sort_order = $6
		WHERE id = $1
		RETURNING id, puzzle_id, sort_order, type, text, audio_key, tts_key, video_key, created_at
	`, id, layerType, text, audioKey, videoKey, sortOrder))
}

// DeletePuzzleLayer deletes a layer by ID.
func (s *Store) DeletePuzzleLayer(ctx context.Context, id string) error {
	_, err := s.db.Exec(ctx, `DELETE FROM puzzle_layers WHERE id = $1`, id)
	return err
}

// ReorderPuzzleLayers batch-updates sort_order for multiple layers in a transaction.
func (s *Store) ReorderPuzzleLayers(ctx context.Context, items []LayerOrderItem) error {
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	for _, item := range items {
		_, err := tx.Exec(ctx, `
			UPDATE puzzle_layers SET sort_order = $2 WHERE id = $1
		`, item.ID, item.SortOrder)
		if err != nil {
			return fmt.Errorf("update layer order: %w", err)
		}
	}

	return tx.Commit(ctx)
}

