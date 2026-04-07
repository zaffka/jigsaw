package store

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"
)

type Store struct {
	db *pgxpool.Pool
}

func New(db *pgxpool.Pool) *Store {
	return &Store{db: db}
}

// --- User ---

type User struct {
	ID           string
	Email        string
	PasswordHash string
	Role         string
	Locale       string
	CreatedAt    time.Time
	Blocked      bool
}

var ErrNotFound = pgx.ErrNoRows
var ErrEmailTaken = fmt.Errorf("email already taken")

func (s *Store) CreateUser(ctx context.Context, email, password, locale string) (*User, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("hash password: %w", err)
	}
	var u User
	err = s.db.QueryRow(ctx, `
		INSERT INTO users (email, password_hash, locale)
		VALUES ($1, $2, $3)
		RETURNING id, email, password_hash, role, locale, created_at
	`, email, string(hash), locale).Scan(&u.ID, &u.Email, &u.PasswordHash, &u.Role, &u.Locale, &u.CreatedAt)
	if err != nil {
		if isDuplicateEmail(err) {
			return nil, ErrEmailTaken
		}
		return nil, fmt.Errorf("create user: %w", err)
	}
	return &u, nil
}

func (s *Store) GetUserByEmail(ctx context.Context, email string) (*User, error) {
	var u User
	err := s.db.QueryRow(ctx, `
		SELECT id, email, password_hash, role, locale, created_at, blocked
		FROM users WHERE email = $1
	`, email).Scan(&u.ID, &u.Email, &u.PasswordHash, &u.Role, &u.Locale, &u.CreatedAt, &u.Blocked)
	return &u, err
}

func (s *Store) GetUserByID(ctx context.Context, id string) (*User, error) {
	var u User
	err := s.db.QueryRow(ctx, `
		SELECT id, email, password_hash, role, locale, created_at, blocked
		FROM users WHERE id = $1
	`, id).Scan(&u.ID, &u.Email, &u.PasswordHash, &u.Role, &u.Locale, &u.CreatedAt, &u.Blocked)
	return &u, err
}

func (s *Store) ListUsers(ctx context.Context) ([]*User, error) {
	rows, err := s.db.Query(ctx, `
		SELECT id, email, password_hash, role, locale, created_at
		FROM users ORDER BY created_at DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var users []*User
	for rows.Next() {
		var u User
		if err := rows.Scan(&u.ID, &u.Email, &u.PasswordHash, &u.Role, &u.Locale, &u.CreatedAt); err != nil {
			return nil, err
		}
		users = append(users, &u)
	}
	return users, rows.Err()
}

func (s *Store) CheckPassword(hash, password string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) == nil
}

// --- Session ---

type Session struct {
	ID        string
	UserID    string
	Token     string
	ExpiresAt time.Time
}

func (s *Store) CreateSession(ctx context.Context, userID string) (*Session, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return nil, fmt.Errorf("generate token: %w", err)
	}
	token := hex.EncodeToString(b)
	expiresAt := time.Now().Add(30 * 24 * time.Hour)
	var sess Session
	err := s.db.QueryRow(ctx, `
		INSERT INTO sessions (user_id, token, expires_at)
		VALUES ($1, $2, $3)
		RETURNING id, user_id, token, expires_at
	`, userID, token, expiresAt).Scan(&sess.ID, &sess.UserID, &sess.Token, &sess.ExpiresAt)
	if err != nil {
		return nil, fmt.Errorf("create session: %w", err)
	}
	return &sess, nil
}

func (s *Store) GetSessionByToken(ctx context.Context, token string) (*Session, error) {
	var sess Session
	err := s.db.QueryRow(ctx, `
		SELECT id, user_id, token, expires_at
		FROM sessions
		WHERE token = $1 AND expires_at > now()
	`, token).Scan(&sess.ID, &sess.UserID, &sess.Token, &sess.ExpiresAt)
	return &sess, err
}

func (s *Store) DeleteSession(ctx context.Context, token string) error {
	_, err := s.db.Exec(ctx, `DELETE FROM sessions WHERE token = $1`, token)
	return err
}

// --- Catalog ---

type CatalogPuzzle struct {
	ID           string
	PuzzleID     string
	Title        string
	Locale       string
	ImageKey     string
	Status       string
	Config       map[string]any
	Featured     bool
	SortOrder    int
	CreatedAt    time.Time
	CategorySlug *string
	Difficulty   string
}

func (s *Store) CreateCatalogPuzzle(ctx context.Context, adminUserID, title, locale, imageKey string, config map[string]any) (*CatalogPuzzle, error) {
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
		INSERT INTO puzzles (user_id, title, locale, image_key, status, config)
		VALUES ($1, $2, $3, $4, 'processing', $5)
		RETURNING id
	`, adminUserID, title, locale, imageKey, configJSON).Scan(&puzzleID)
	if err != nil {
		return nil, fmt.Errorf("create puzzle: %w", err)
	}

	var cp CatalogPuzzle
	err = tx.QueryRow(ctx, `
		INSERT INTO catalog_puzzles (puzzle_id) VALUES ($1)
		RETURNING id, puzzle_id, featured, sort_order, created_at
	`, puzzleID).Scan(&cp.ID, &cp.PuzzleID, &cp.Featured, &cp.SortOrder, &cp.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("create catalog entry: %w", err)
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

	cp.Title = title
	cp.Locale = locale
	cp.ImageKey = imageKey
	cp.Status = "processing"
	cp.Config = config
	return &cp, nil
}

func scanCatalogPuzzle(row interface {
	Scan(...any) error
}) (*CatalogPuzzle, error) {
	var cp CatalogPuzzle
	var configRaw []byte
	err := row.Scan(&cp.ID, &cp.PuzzleID, &cp.Title, &cp.Locale, &cp.ImageKey, &cp.Status, &configRaw, &cp.Featured, &cp.SortOrder, &cp.CreatedAt, &cp.CategorySlug, &cp.Difficulty)
	if err != nil {
		return nil, err
	}
	json.Unmarshal(configRaw, &cp.Config)
	return &cp, nil
}

const catalogPuzzleSelect = `
	SELECT cp.id, cp.puzzle_id, p.title, p.locale, p.image_key, p.status, p.config, cp.featured, cp.sort_order, cp.created_at, cat.slug, p.difficulty
	FROM catalog_puzzles cp
	JOIN puzzles p ON p.id = cp.puzzle_id
	LEFT JOIN categories cat ON cat.id = p.category_id`

func (s *Store) ListCatalogPuzzles(ctx context.Context) ([]*CatalogPuzzle, error) {
	rows, err := s.db.Query(ctx, catalogPuzzleSelect+`
		ORDER BY cp.sort_order ASC, cp.created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []*CatalogPuzzle
	for rows.Next() {
		cp, err := scanCatalogPuzzle(rows)
		if err != nil {
			return nil, err
		}
		list = append(list, cp)
	}
	return list, rows.Err()
}

func (s *Store) GetCatalogPuzzle(ctx context.Context, id string) (*CatalogPuzzle, error) {
	return scanCatalogPuzzle(s.db.QueryRow(ctx,
		catalogPuzzleSelect+` WHERE cp.id = $1`, id))
}

func (s *Store) UpdateCatalogPuzzle(ctx context.Context, id, title string, featured bool, sortOrder int) error {
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	_, err = tx.Exec(ctx, `
		UPDATE catalog_puzzles SET featured = $2, sort_order = $3 WHERE id = $1
	`, id, featured, sortOrder)
	if err != nil {
		return fmt.Errorf("update catalog_puzzles: %w", err)
	}

	_, err = tx.Exec(ctx, `
		UPDATE puzzles SET title = $2 WHERE id = (SELECT puzzle_id FROM catalog_puzzles WHERE id = $1)
	`, id, title)
	if err != nil {
		return fmt.Errorf("update puzzle title: %w", err)
	}

	return tx.Commit(ctx)
}

func (s *Store) DeleteCatalogPuzzle(ctx context.Context, id string) error {
	_, err := s.db.Exec(ctx, `
		DELETE FROM puzzles WHERE id = (SELECT puzzle_id FROM catalog_puzzles WHERE id = $1)
	`, id)
	return err
}

// CatalogFilters holds optional filter parameters for ListPublicCatalog.
type CatalogFilters struct {
	CategorySlug string // empty = no filter
	Difficulty   string // empty = no filter
}

func (s *Store) ListPublicCatalog(ctx context.Context, locale string, filters CatalogFilters) ([]*CatalogPuzzle, error) {
	query := catalogPuzzleSelect + `
		WHERE p.status = 'ready' AND p.visibility = 'public' AND p.locale = $1`
	args := []any{locale}

	if filters.CategorySlug != "" {
		args = append(args, filters.CategorySlug)
		query += fmt.Sprintf(` AND cat.slug = $%d`, len(args))
	}
	if filters.Difficulty != "" {
		args = append(args, filters.Difficulty)
		query += fmt.Sprintf(` AND p.difficulty = $%d`, len(args))
	}
	query += `
		ORDER BY cp.featured DESC, cp.sort_order ASC, cp.created_at DESC`

	rows, err := s.db.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []*CatalogPuzzle
	for rows.Next() {
		cp, err := scanCatalogPuzzle(rows)
		if err != nil {
			return nil, err
		}
		list = append(list, cp)
	}
	return list, rows.Err()
}

// --- Reward ---

type Reward struct {
	ID        string
	PuzzleID  string
	VideoKey  *string
	Word      *string
	TTSKey    *string
	Animation string
	CreatedAt time.Time
}

func (s *Store) UpsertReward(ctx context.Context, puzzleID string, videoKey, word *string, animation string) (*Reward, error) {
	if animation == "" {
		animation = "confetti"
	}
	var r Reward
	err := s.db.QueryRow(ctx, `
		INSERT INTO rewards (puzzle_id, video_key, word, animation)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (puzzle_id) DO UPDATE
			SET video_key = EXCLUDED.video_key,
			    word      = EXCLUDED.word,
			    animation = EXCLUDED.animation
		RETURNING id, puzzle_id, video_key, word, tts_key, animation, created_at
	`, puzzleID, videoKey, word, animation).Scan(
		&r.ID, &r.PuzzleID, &r.VideoKey, &r.Word, &r.TTSKey, &r.Animation, &r.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("upsert reward: %w", err)
	}
	return &r, nil
}

func (s *Store) GetRewardByPuzzleID(ctx context.Context, puzzleID string) (*Reward, error) {
	var r Reward
	err := s.db.QueryRow(ctx, `
		SELECT id, puzzle_id, video_key, word, tts_key, animation, created_at
		FROM rewards WHERE puzzle_id = $1
	`, puzzleID).Scan(&r.ID, &r.PuzzleID, &r.VideoKey, &r.Word, &r.TTSKey, &r.Animation, &r.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &r, nil
}

// --- Puzzle pieces ---

type PuzzlePiece struct {
	ID       string
	PuzzleID string
	ImageKey string
	PathSVG  string
	GridX    int
	GridY    int
	Bounds   map[string]any
}

func (s *Store) GetPuzzlePieces(ctx context.Context, puzzleID string) ([]*PuzzlePiece, error) {
	rows, err := s.db.Query(ctx, `
		SELECT id, puzzle_id, image_key, path_svg, grid_x, grid_y, bounds
		FROM puzzle_pieces WHERE puzzle_id = $1 ORDER BY grid_y, grid_x
	`, puzzleID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []*PuzzlePiece
	for rows.Next() {
		var p PuzzlePiece
		var boundsRaw []byte
		if err := rows.Scan(&p.ID, &p.PuzzleID, &p.ImageKey, &p.PathSVG, &p.GridX, &p.GridY, &boundsRaw); err != nil {
			return nil, err
		}
		json.Unmarshal(boundsRaw, &p.Bounds)
		list = append(list, &p)
	}
	return list, rows.Err()
}

func isDuplicateEmail(err error) bool {
	return isUniqueViolation(err)
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}

// --- Category ---

type Category struct {
	ID        string
	Slug      string
	Name      map[string]string
	Icon      string
	SortOrder int
	CreatedAt time.Time
}

var ErrConflict = fmt.Errorf("conflict")

func scanCategory(row interface {
	Scan(...any) error
}) (*Category, error) {
	var c Category
	var nameRaw []byte
	if err := row.Scan(&c.ID, &c.Slug, &nameRaw, &c.Icon, &c.SortOrder, &c.CreatedAt); err != nil {
		return nil, err
	}
	if err := json.Unmarshal(nameRaw, &c.Name); err != nil {
		return nil, fmt.Errorf("unmarshal category name: %w", err)
	}
	return &c, nil
}

func (s *Store) ListCategories(ctx context.Context) ([]*Category, error) {
	rows, err := s.db.Query(ctx, `
		SELECT id, slug, name, icon, sort_order, created_at
		FROM categories ORDER BY sort_order ASC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []*Category
	for rows.Next() {
		c, err := scanCategory(rows)
		if err != nil {
			return nil, err
		}
		list = append(list, c)
	}
	return list, rows.Err()
}

func (s *Store) CreateCategory(ctx context.Context, slug string, name map[string]string, icon string, sortOrder int) (*Category, error) {
	nameJSON, err := json.Marshal(name)
	if err != nil {
		return nil, fmt.Errorf("marshal name: %w", err)
	}
	c, err := scanCategory(s.db.QueryRow(ctx, `
		INSERT INTO categories (slug, name, icon, sort_order)
		VALUES ($1, $2, $3, $4)
		RETURNING id, slug, name, icon, sort_order, created_at
	`, slug, nameJSON, icon, sortOrder))
	if isUniqueViolation(err) {
		return nil, ErrConflict
	}
	return c, err
}

func (s *Store) UpdateCategory(ctx context.Context, id string, slug string, name map[string]string, icon string, sortOrder int) (*Category, error) {
	nameJSON, err := json.Marshal(name)
	if err != nil {
		return nil, fmt.Errorf("marshal name: %w", err)
	}
	c, err := scanCategory(s.db.QueryRow(ctx, `
		UPDATE categories SET slug = $2, name = $3, icon = $4, sort_order = $5
		WHERE id = $1
		RETURNING id, slug, name, icon, sort_order, created_at
	`, id, slug, nameJSON, icon, sortOrder))
	if isUniqueViolation(err) {
		return nil, ErrConflict
	}
	return c, err
}

func (s *Store) DeleteCategory(ctx context.Context, id string) error {
	var pgErr *pgconn.PgError
	_, err := s.db.Exec(ctx, `DELETE FROM categories WHERE id = $1`, id)
	if errors.As(err, &pgErr) && pgErr.Code == "23503" {
		return ErrConflict
	}
	return err
}
