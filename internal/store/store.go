package store

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
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
		SELECT id, email, password_hash, role, locale, created_at
		FROM users WHERE email = $1
	`, email).Scan(&u.ID, &u.Email, &u.PasswordHash, &u.Role, &u.Locale, &u.CreatedAt)
	return &u, err
}

func (s *Store) GetUserByID(ctx context.Context, id string) (*User, error) {
	var u User
	err := s.db.QueryRow(ctx, `
		SELECT id, email, password_hash, role, locale, created_at
		FROM users WHERE id = $1
	`, id).Scan(&u.ID, &u.Email, &u.PasswordHash, &u.Role, &u.Locale, &u.CreatedAt)
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
	ID        string
	PuzzleID  string
	Titles    map[string]string
	ImageKey  string
	Status    string
	Config    map[string]any
	Featured  bool
	SortOrder int
	CreatedAt time.Time
}

func (s *Store) CreateCatalogPuzzle(ctx context.Context, adminUserID string, titles map[string]string, imageKey string, config map[string]any) (*CatalogPuzzle, error) {
	titlesJSON, err := json.Marshal(titles)
	if err != nil {
		return nil, fmt.Errorf("marshal titles: %w", err)
	}
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
		INSERT INTO puzzles (user_id, titles, image_key, status, config)
		VALUES ($1, $2, $3, 'processing', $4)
		RETURNING id
	`, adminUserID, titlesJSON, imageKey, configJSON).Scan(&puzzleID)
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

	cp.Titles = titles
	cp.ImageKey = imageKey
	cp.Status = "processing"
	cp.Config = config
	return &cp, nil
}

func (s *Store) ListCatalogPuzzles(ctx context.Context) ([]*CatalogPuzzle, error) {
	rows, err := s.db.Query(ctx, `
		SELECT cp.id, cp.puzzle_id, p.titles, p.image_key, p.status, p.config, cp.featured, cp.sort_order, cp.created_at
		FROM catalog_puzzles cp
		JOIN puzzles p ON p.id = cp.puzzle_id
		ORDER BY cp.sort_order ASC, cp.created_at DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []*CatalogPuzzle
	for rows.Next() {
		var cp CatalogPuzzle
		var titlesRaw, configRaw []byte
		if err := rows.Scan(&cp.ID, &cp.PuzzleID, &titlesRaw, &cp.ImageKey, &cp.Status, &configRaw, &cp.Featured, &cp.SortOrder, &cp.CreatedAt); err != nil {
			return nil, err
		}
		json.Unmarshal(titlesRaw, &cp.Titles)
		json.Unmarshal(configRaw, &cp.Config)
		list = append(list, &cp)
	}
	return list, rows.Err()
}

func (s *Store) GetCatalogPuzzle(ctx context.Context, id string) (*CatalogPuzzle, error) {
	var cp CatalogPuzzle
	var titlesRaw, configRaw []byte
	err := s.db.QueryRow(ctx, `
		SELECT cp.id, cp.puzzle_id, p.titles, p.image_key, p.status, p.config, cp.featured, cp.sort_order, cp.created_at
		FROM catalog_puzzles cp
		JOIN puzzles p ON p.id = cp.puzzle_id
		WHERE cp.id = $1
	`, id).Scan(&cp.ID, &cp.PuzzleID, &titlesRaw, &cp.ImageKey, &cp.Status, &configRaw, &cp.Featured, &cp.SortOrder, &cp.CreatedAt)
	if err != nil {
		return nil, err
	}
	json.Unmarshal(titlesRaw, &cp.Titles)
	json.Unmarshal(configRaw, &cp.Config)
	return &cp, nil
}

func (s *Store) UpdateCatalogPuzzle(ctx context.Context, id string, titles map[string]string, featured bool, sortOrder int) error {
	titlesJSON, err := json.Marshal(titles)
	if err != nil {
		return fmt.Errorf("marshal titles: %w", err)
	}

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
		UPDATE puzzles SET titles = $2 WHERE id = (SELECT puzzle_id FROM catalog_puzzles WHERE id = $1)
	`, id, titlesJSON)
	if err != nil {
		return fmt.Errorf("update puzzles titles: %w", err)
	}

	return tx.Commit(ctx)
}

func (s *Store) DeleteCatalogPuzzle(ctx context.Context, id string) error {
	_, err := s.db.Exec(ctx, `
		DELETE FROM puzzles WHERE id = (SELECT puzzle_id FROM catalog_puzzles WHERE id = $1)
	`, id)
	return err
}

func (s *Store) ListPublicCatalog(ctx context.Context) ([]*CatalogPuzzle, error) {
	rows, err := s.db.Query(ctx, `
		SELECT cp.id, cp.puzzle_id, p.titles, p.image_key, p.status, p.config, cp.featured, cp.sort_order, cp.created_at
		FROM catalog_puzzles cp
		JOIN puzzles p ON p.id = cp.puzzle_id
		WHERE p.status = 'ready'
		ORDER BY cp.featured DESC, cp.sort_order ASC, cp.created_at DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []*CatalogPuzzle
	for rows.Next() {
		var cp CatalogPuzzle
		var titlesRaw, configRaw []byte
		if err := rows.Scan(&cp.ID, &cp.PuzzleID, &titlesRaw, &cp.ImageKey, &cp.Status, &configRaw, &cp.Featured, &cp.SortOrder, &cp.CreatedAt); err != nil {
			return nil, err
		}
		json.Unmarshal(titlesRaw, &cp.Titles)
		json.Unmarshal(configRaw, &cp.Config)
		list = append(list, &cp)
	}
	return list, rows.Err()
}

func isDuplicateEmail(err error) bool {
	return err != nil && strings.Contains(err.Error(), "unique") && strings.Contains(err.Error(), "email")
}
