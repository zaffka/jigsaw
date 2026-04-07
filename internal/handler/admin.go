package handler

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"path"

	"github.com/zaffka/jigsaw/internal/middleware"
	"github.com/zaffka/jigsaw/internal/store"
	"github.com/zaffka/jigsaw/pkg/s3"
	"go.uber.org/zap"
)

const maxImageSize = 10 << 20 // 10 MB

type catalogPuzzleResponse struct {
	ID           string         `json:"id"`
	PuzzleID     string         `json:"puzzle_id"`
	Title        string         `json:"title"`
	Locale       string         `json:"locale"`
	ImageKey     string         `json:"image_key"`
	Status       string         `json:"status"`
	Config       map[string]any `json:"config"`
	Featured     bool           `json:"featured"`
	SortOrder    int            `json:"sort_order"`
	CreatedAt    string         `json:"created_at"`
	Category     *string        `json:"category"`
	Difficulty   string         `json:"difficulty"`
}

func catalogPuzzleToResponse(cp *store.CatalogPuzzle) catalogPuzzleResponse {
	return catalogPuzzleResponse{
		ID:        cp.ID,
		PuzzleID:  cp.PuzzleID,
		Title:     cp.Title,
		Locale:    cp.Locale,
		ImageKey:  cp.ImageKey,
		Status:    cp.Status,
		Config:    cp.Config,
		Featured:  cp.Featured,
		SortOrder: cp.SortOrder,
		CreatedAt: cp.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		Category:  cp.CategorySlug,
		Difficulty: cp.Difficulty,
	}
}

// HandleAdminListCatalog GET /api/admin/catalog/puzzles
func (h *Handler) HandleAdminListCatalog(w http.ResponseWriter, r *http.Request) {
	list, err := h.Store.ListCatalogPuzzles(r.Context())
	if err != nil {
		h.Log.Error("admin list catalog", zap.Error(err))
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	resp := make([]catalogPuzzleResponse, 0, len(list))
	for _, cp := range list {
		resp = append(resp, catalogPuzzleToResponse(cp))
	}
	writeJSON(w, http.StatusOK, resp)
}

// HandleAdminCreateCatalogPuzzle POST /api/admin/catalog/puzzles
func (h *Handler) HandleAdminCreateCatalogPuzzle(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseMultipartForm(maxImageSize); err != nil {
		writeError(w, http.StatusBadRequest, "failed to parse form")
		return
	}

	file, header, err := r.FormFile("image")
	if err != nil {
		writeError(w, http.StatusBadRequest, "image file is required")
		return
	}
	defer file.Close()

	ct := header.Header.Get("Content-Type")
	if ct != "image/jpeg" && ct != "image/png" {
		writeError(w, http.StatusBadRequest, "only JPEG and PNG images are supported")
		return
	}

	title := r.FormValue("title")
	locale := r.FormValue("locale")
	if locale == "" {
		locale = "ru"
	}

	var config map[string]any
	if v := r.FormValue("config"); v != "" {
		if err := json.Unmarshal([]byte(v), &config); err != nil {
			writeError(w, http.StatusBadRequest, "invalid config JSON")
			return
		}
	}
	if config == nil {
		config = map[string]any{}
	}

	ext := ".jpg"
	if ct == "image/png" {
		ext = ".png"
	}
	imageKey := "originals/" + newObjectKey() + ext

	data, err := io.ReadAll(io.LimitReader(file, maxImageSize))
	if err != nil {
		h.Log.Error("admin create puzzle: read image", zap.Error(err))
		writeError(w, http.StatusInternalServerError, "failed to read image")
		return
	}

	if _, err := h.S3.PutObject(r.Context(), imageKey, bytes.NewReader(data), int64(len(data)), s3.PutObjectOptions{ContentType: ct}); err != nil {
		h.Log.Error("admin create puzzle: upload to s3", zap.Error(err))
		writeError(w, http.StatusInternalServerError, "failed to upload image")
		return
	}

	user := middleware.UserFromContext(r.Context())

	cp, err := h.Store.CreateCatalogPuzzle(r.Context(), user.ID, title, locale, imageKey, config)
	if err != nil {
		h.Log.Error("admin create puzzle: db", zap.Error(err))
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	writeJSON(w, http.StatusCreated, catalogPuzzleToResponse(cp))
}

// HandleAdminGetCatalogPuzzle GET /api/admin/catalog/puzzles/{id}
func (h *Handler) HandleAdminGetCatalogPuzzle(w http.ResponseWriter, r *http.Request) {
	id := path.Base(r.URL.Path)
	cp, err := h.Store.GetCatalogPuzzle(r.Context(), id)
	if err != nil {
		if err == store.ErrNotFound {
			writeError(w, http.StatusNotFound, "not found")
			return
		}
		h.Log.Error("admin get catalog puzzle", zap.Error(err))
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	writeJSON(w, http.StatusOK, catalogPuzzleToResponse(cp))
}

// HandleAdminUpdateCatalogPuzzle PUT /api/admin/catalog/puzzles/{id}
func (h *Handler) HandleAdminUpdateCatalogPuzzle(w http.ResponseWriter, r *http.Request) {
	id := path.Base(r.URL.Path)

	var req struct {
		Title     string `json:"title"`
		Featured  bool   `json:"featured"`
		SortOrder int    `json:"sort_order"`
	}
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := h.Store.UpdateCatalogPuzzle(r.Context(), id, req.Title, req.Featured, req.SortOrder); err != nil {
		h.Log.Error("admin update catalog puzzle", zap.Error(err))
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	cp, err := h.Store.GetCatalogPuzzle(r.Context(), id)
	if err != nil {
		h.Log.Error("admin update: get after update", zap.Error(err))
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	writeJSON(w, http.StatusOK, catalogPuzzleToResponse(cp))
}

// HandleAdminDeleteCatalogPuzzle DELETE /api/admin/catalog/puzzles/{id}
func (h *Handler) HandleAdminDeleteCatalogPuzzle(w http.ResponseWriter, r *http.Request) {
	id := path.Base(r.URL.Path)
	if err := h.Store.DeleteCatalogPuzzle(r.Context(), id); err != nil {
		h.Log.Error("admin delete catalog puzzle", zap.Error(err))
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// --- Users ---

type userAdminResponse struct {
	ID        string `json:"id"`
	Email     string `json:"email"`
	Role      string `json:"role"`
	Locale    string `json:"locale"`
	CreatedAt string `json:"created_at"`
}

// HandleAdminListUsers GET /api/admin/users
func (h *Handler) HandleAdminListUsers(w http.ResponseWriter, r *http.Request) {
	users, err := h.Store.ListUsers(r.Context())
	if err != nil {
		h.Log.Error("admin list users", zap.Error(err))
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	resp := make([]userAdminResponse, 0, len(users))
	for _, u := range users {
		resp = append(resp, userAdminResponse{
			ID:        u.ID,
			Email:     u.Email,
			Role:      u.Role,
			Locale:    u.Locale,
			CreatedAt: u.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		})
	}
	writeJSON(w, http.StatusOK, resp)
}

func newObjectKey() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		panic(err)
	}
	return hex.EncodeToString(b)
}
