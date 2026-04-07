package handler

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"

	"github.com/zaffka/jigsaw/internal/crypto"
	"github.com/zaffka/jigsaw/internal/middleware"
	"github.com/zaffka/jigsaw/internal/store"
	"github.com/zaffka/jigsaw/pkg/s3"
	"go.uber.org/zap"
)

const maxAudioSize = 30 << 20  // 30 MB
const maxVideoSize = 100 << 20 // 100 MB

type parentPuzzleResponse struct {
	ID         string         `json:"id"`
	Title      string         `json:"title"`
	Locale     string         `json:"locale"`
	ImageKey   string         `json:"image_key"`
	Status     string         `json:"status"`
	Config     map[string]any `json:"config"`
	Category   *string        `json:"category"`
	Difficulty string         `json:"difficulty"`
	CreatedAt  string         `json:"created_at"`
}

func parentPuzzleToResponse(p *store.ParentPuzzle) parentPuzzleResponse {
	return parentPuzzleResponse{
		ID:         p.ID,
		Title:      p.Title,
		Locale:     p.Locale,
		ImageKey:   p.ImageKey,
		Status:     p.Status,
		Config:     p.Config,
		Category:   p.CategorySlug,
		Difficulty: p.Difficulty,
		CreatedAt:  p.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
}

type layerResponse struct {
	ID        string  `json:"id"`
	PuzzleID  string  `json:"puzzle_id"`
	SortOrder int     `json:"sort_order"`
	Type      string  `json:"type"`
	Text      *string `json:"text"`
	AudioKey  *string `json:"audio_key"`
	TTSKey    *string `json:"tts_key"`
	VideoKey  *string `json:"video_key"`
	CreatedAt string  `json:"created_at"`
}

func layerToResponse(l *store.PuzzleLayer) layerResponse {
	return layerResponse{
		ID:        l.ID,
		PuzzleID:  l.PuzzleID,
		SortOrder: l.SortOrder,
		Type:      l.Type,
		Text:      l.Text,
		AudioKey:  l.AudioKey,
		TTSKey:    l.TTSKey,
		VideoKey:  l.VideoKey,
		CreatedAt: l.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
}

// encryptIfNeeded returns the data encrypted with h.MediaEncryptionKey if set,
// otherwise returns data unchanged.
func (h *Handler) encryptIfNeeded(data []byte) ([]byte, error) {
	if len(h.MediaEncryptionKey) == 0 {
		return data, nil
	}
	enc, err := crypto.Encrypt(h.MediaEncryptionKey, data)
	if err != nil {
		return nil, fmt.Errorf("encrypt: %w", err)
	}
	return enc, nil
}

// HandleParentListPuzzles GET /api/parent/puzzles
func (h *Handler) HandleParentListPuzzles(w http.ResponseWriter, r *http.Request) {
	user := middleware.UserFromContext(r.Context())

	list, err := h.Store.ListParentPuzzles(r.Context(), user.ID)
	if err != nil {
		h.Log.Error("list parent puzzles", zap.Error(err))
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	resp := make([]parentPuzzleResponse, 0, len(list))
	for _, p := range list {
		resp = append(resp, parentPuzzleToResponse(p))
	}
	writeJSON(w, http.StatusOK, resp)
}

// HandleParentCreatePuzzle POST /api/parent/puzzles
func (h *Handler) HandleParentCreatePuzzle(w http.ResponseWriter, r *http.Request) {
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

	var categoryID *string
	if v := r.FormValue("category_id"); v != "" {
		categoryID = &v
	}

	ext := ".jpg"
	if ct == "image/png" {
		ext = ".png"
	}
	imageKey := "originals/" + newObjectKey() + ext

	data, err := io.ReadAll(io.LimitReader(file, maxImageSize))
	if err != nil {
		h.Log.Error("create parent puzzle: read image", zap.Error(err))
		writeError(w, http.StatusInternalServerError, "failed to read image")
		return
	}

	if _, err := h.S3.PutObject(r.Context(), imageKey, bytes.NewReader(data), int64(len(data)), s3.PutObjectOptions{ContentType: ct}); err != nil {
		h.Log.Error("create parent puzzle: upload to s3", zap.Error(err))
		writeError(w, http.StatusInternalServerError, "failed to upload image")
		return
	}

	user := middleware.UserFromContext(r.Context())

	p, err := h.Store.CreateParentPuzzle(r.Context(), user.ID, title, locale, imageKey, config, categoryID)
	if err != nil {
		h.Log.Error("create parent puzzle: db", zap.Error(err))
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	writeJSON(w, http.StatusCreated, parentPuzzleToResponse(p))
}

// HandleParentGetPuzzle GET /api/parent/puzzles/{id}
func (h *Handler) HandleParentGetPuzzle(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	user := middleware.UserFromContext(r.Context())

	p, err := h.Store.GetParentPuzzle(r.Context(), id, user.ID)
	if err != nil {
		if err == store.ErrNotFound {
			writeError(w, http.StatusNotFound, "not found")
			return
		}
		h.Log.Error("get parent puzzle", zap.Error(err))
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	writeJSON(w, http.StatusOK, parentPuzzleToResponse(p))
}

// HandleParentUpdatePuzzle PUT /api/parent/puzzles/{id}
func (h *Handler) HandleParentUpdatePuzzle(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	user := middleware.UserFromContext(r.Context())

	var req struct {
		Title      string  `json:"title"`
		CategoryID *string `json:"category_id"`
	}
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := h.Store.UpdateParentPuzzle(r.Context(), id, user.ID, req.Title, req.CategoryID); err != nil {
		if err == store.ErrNotFound {
			writeError(w, http.StatusNotFound, "not found")
			return
		}
		h.Log.Error("update parent puzzle", zap.Error(err))
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	p, err := h.Store.GetParentPuzzle(r.Context(), id, user.ID)
	if err != nil {
		h.Log.Error("update parent puzzle: get after update", zap.Error(err))
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	writeJSON(w, http.StatusOK, parentPuzzleToResponse(p))
}

// HandleParentDeletePuzzle DELETE /api/parent/puzzles/{id}
func (h *Handler) HandleParentDeletePuzzle(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	user := middleware.UserFromContext(r.Context())

	if err := h.Store.DeleteParentPuzzle(r.Context(), id, user.ID); err != nil {
		if err == store.ErrNotFound {
			writeError(w, http.StatusNotFound, "not found")
			return
		}
		h.Log.Error("delete parent puzzle", zap.Error(err))
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// HandleParentListLayers GET /api/parent/puzzles/{id}/layers
func (h *Handler) HandleParentListLayers(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	user := middleware.UserFromContext(r.Context())

	puzzle, err := h.Store.GetParentPuzzle(r.Context(), id, user.ID)
	if err != nil {
		if err == store.ErrNotFound {
			writeError(w, http.StatusNotFound, "not found")
			return
		}
		h.Log.Error("list layers: get puzzle", zap.Error(err))
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	list, err := h.Store.ListPuzzleLayers(r.Context(), puzzle.ID)
	if err != nil {
		h.Log.Error("list layers", zap.Error(err))
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	resp := make([]layerResponse, 0, len(list))
	for _, l := range list {
		resp = append(resp, layerToResponse(l))
	}
	writeJSON(w, http.StatusOK, resp)
}

// HandleParentCreateLayer POST /api/parent/puzzles/{id}/layers
func (h *Handler) HandleParentCreateLayer(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	user := middleware.UserFromContext(r.Context())

	puzzle, err := h.Store.GetParentPuzzle(r.Context(), id, user.ID)
	if err != nil {
		if err == store.ErrNotFound {
			writeError(w, http.StatusNotFound, "not found")
			return
		}
		h.Log.Error("create layer: get puzzle", zap.Error(err))
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	maxFormSize := int64(maxVideoSize)
	if err := r.ParseMultipartForm(maxFormSize); err != nil {
		writeError(w, http.StatusBadRequest, "failed to parse form")
		return
	}

	layerType := r.FormValue("type")
	sortOrder, _ := strconv.Atoi(r.FormValue("sort_order"))

	var text *string
	if v := r.FormValue("text"); v != "" {
		text = &v
	}

	var audioKey, videoKey *string

	switch layerType {
	case "audio":
		file, header, err := r.FormFile("audio")
		if err == nil {
			defer file.Close()
			ct := header.Header.Get("Content-Type")
			if ct != "audio/mpeg" && ct != "audio/wav" && ct != "audio/x-wav" && ct != "audio/mp3" {
				writeError(w, http.StatusBadRequest, "only MP3 and WAV audio is supported")
				return
			}
			ext := ".mp3"
			if ct == "audio/wav" || ct == "audio/x-wav" {
				ext = ".wav"
			}
			data, err := io.ReadAll(io.LimitReader(file, maxAudioSize))
			if err != nil {
				h.Log.Error("create layer: read audio", zap.Error(err))
				writeError(w, http.StatusInternalServerError, "failed to read audio")
				return
			}
			data, err = h.encryptIfNeeded(data)
			if err != nil {
				h.Log.Error("create layer: encrypt audio", zap.Error(err))
				writeError(w, http.StatusInternalServerError, "failed to encrypt audio")
				return
			}
			key := "audio/" + newObjectKey() + ext
			if _, err := h.S3.PutObject(r.Context(), key, bytes.NewReader(data), int64(len(data)), s3.PutObjectOptions{ContentType: ct}); err != nil {
				h.Log.Error("create layer: upload audio", zap.Error(err))
				writeError(w, http.StatusInternalServerError, "failed to upload audio")
				return
			}
			audioKey = &key
		}
	case "video":
		file, header, err := r.FormFile("video")
		if err == nil {
			defer file.Close()
			ct := header.Header.Get("Content-Type")
			if ct != "video/mp4" {
				writeError(w, http.StatusBadRequest, "only MP4 video is supported")
				return
			}
			data, err := io.ReadAll(io.LimitReader(file, maxVideoSize))
			if err != nil {
				h.Log.Error("create layer: read video", zap.Error(err))
				writeError(w, http.StatusInternalServerError, "failed to read video")
				return
			}
			data, err = h.encryptIfNeeded(data)
			if err != nil {
				h.Log.Error("create layer: encrypt video", zap.Error(err))
				writeError(w, http.StatusInternalServerError, "failed to encrypt video")
				return
			}
			key := "video/" + newObjectKey() + ".mp4"
			if _, err := h.S3.PutObject(r.Context(), key, bytes.NewReader(data), int64(len(data)), s3.PutObjectOptions{ContentType: ct}); err != nil {
				h.Log.Error("create layer: upload video", zap.Error(err))
				writeError(w, http.StatusInternalServerError, "failed to upload video")
				return
			}
			videoKey = &key
		}
	}

	layer, err := h.Store.CreatePuzzleLayer(r.Context(), puzzle.ID, layerType, text, audioKey, videoKey, sortOrder)
	if err != nil {
		h.Log.Error("create layer: db", zap.Error(err))
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	writeJSON(w, http.StatusCreated, layerToResponse(layer))
}

// HandleParentUpdateLayer PUT /api/parent/puzzles/{id}/layers/{lid}
func (h *Handler) HandleParentUpdateLayer(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	lid := r.PathValue("lid")
	user := middleware.UserFromContext(r.Context())

	// Verify puzzle ownership
	_, err := h.Store.GetParentPuzzle(r.Context(), id, user.ID)
	if err != nil {
		if err == store.ErrNotFound {
			writeError(w, http.StatusNotFound, "not found")
			return
		}
		h.Log.Error("update layer: get puzzle", zap.Error(err))
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	maxFormSize := int64(maxVideoSize)
	if err := r.ParseMultipartForm(maxFormSize); err != nil {
		writeError(w, http.StatusBadRequest, "failed to parse form")
		return
	}

	layerType := r.FormValue("type")
	sortOrder, _ := strconv.Atoi(r.FormValue("sort_order"))

	var text *string
	if v := r.FormValue("text"); v != "" {
		text = &v
	}

	var audioKey, videoKey *string

	switch layerType {
	case "audio":
		file, header, err := r.FormFile("audio")
		if err == nil {
			defer file.Close()
			ct := header.Header.Get("Content-Type")
			if ct != "audio/mpeg" && ct != "audio/wav" && ct != "audio/x-wav" && ct != "audio/mp3" {
				writeError(w, http.StatusBadRequest, "only MP3 and WAV audio is supported")
				return
			}
			ext := ".mp3"
			if ct == "audio/wav" || ct == "audio/x-wav" {
				ext = ".wav"
			}
			data, err := io.ReadAll(io.LimitReader(file, maxAudioSize))
			if err != nil {
				h.Log.Error("update layer: read audio", zap.Error(err))
				writeError(w, http.StatusInternalServerError, "failed to read audio")
				return
			}
			data, err = h.encryptIfNeeded(data)
			if err != nil {
				h.Log.Error("update layer: encrypt audio", zap.Error(err))
				writeError(w, http.StatusInternalServerError, "failed to encrypt audio")
				return
			}
			key := "audio/" + newObjectKey() + ext
			if _, err := h.S3.PutObject(r.Context(), key, bytes.NewReader(data), int64(len(data)), s3.PutObjectOptions{ContentType: ct}); err != nil {
				h.Log.Error("update layer: upload audio", zap.Error(err))
				writeError(w, http.StatusInternalServerError, "failed to upload audio")
				return
			}
			audioKey = &key
		}
	case "video":
		file, header, err := r.FormFile("video")
		if err == nil {
			defer file.Close()
			ct := header.Header.Get("Content-Type")
			if ct != "video/mp4" {
				writeError(w, http.StatusBadRequest, "only MP4 video is supported")
				return
			}
			data, err := io.ReadAll(io.LimitReader(file, maxVideoSize))
			if err != nil {
				h.Log.Error("update layer: read video", zap.Error(err))
				writeError(w, http.StatusInternalServerError, "failed to read video")
				return
			}
			data, err = h.encryptIfNeeded(data)
			if err != nil {
				h.Log.Error("update layer: encrypt video", zap.Error(err))
				writeError(w, http.StatusInternalServerError, "failed to encrypt video")
				return
			}
			key := "video/" + newObjectKey() + ".mp4"
			if _, err := h.S3.PutObject(r.Context(), key, bytes.NewReader(data), int64(len(data)), s3.PutObjectOptions{ContentType: ct}); err != nil {
				h.Log.Error("update layer: upload video", zap.Error(err))
				writeError(w, http.StatusInternalServerError, "failed to upload video")
				return
			}
			videoKey = &key
		}
	}

	layer, err := h.Store.UpdatePuzzleLayer(r.Context(), lid, layerType, text, audioKey, videoKey, sortOrder)
	if err != nil {
		h.Log.Error("update layer: db", zap.Error(err))
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	writeJSON(w, http.StatusOK, layerToResponse(layer))
}

// HandleParentDeleteLayer DELETE /api/parent/puzzles/{id}/layers/{lid}
func (h *Handler) HandleParentDeleteLayer(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	lid := r.PathValue("lid")
	user := middleware.UserFromContext(r.Context())

	// Verify puzzle ownership
	_, err := h.Store.GetParentPuzzle(r.Context(), id, user.ID)
	if err != nil {
		if err == store.ErrNotFound {
			writeError(w, http.StatusNotFound, "not found")
			return
		}
		h.Log.Error("delete layer: get puzzle", zap.Error(err))
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	if err := h.Store.DeletePuzzleLayer(r.Context(), lid); err != nil {
		h.Log.Error("delete layer: db", zap.Error(err))
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// HandleParentReorderLayers POST /api/parent/puzzles/{id}/layers/reorder
func (h *Handler) HandleParentReorderLayers(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	user := middleware.UserFromContext(r.Context())

	// Verify puzzle ownership
	_, err := h.Store.GetParentPuzzle(r.Context(), id, user.ID)
	if err != nil {
		if err == store.ErrNotFound {
			writeError(w, http.StatusNotFound, "not found")
			return
		}
		h.Log.Error("reorder layers: get puzzle", zap.Error(err))
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	var items []store.LayerOrderItem
	if err := decodeJSON(r, &items); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := h.Store.ReorderPuzzleLayers(r.Context(), items); err != nil {
		h.Log.Error("reorder layers: db", zap.Error(err))
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}
