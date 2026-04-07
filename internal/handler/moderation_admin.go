package handler

import (
	"net/http"

	"github.com/zaffka/jigsaw/internal/middleware"
	"github.com/zaffka/jigsaw/internal/store"
	"go.uber.org/zap"
)

type moderationItemResponse struct {
	ID           string             `json:"id"`
	PuzzleID     string             `json:"puzzle_id"`
	PuzzleTitle  string             `json:"puzzle_title"`
	ImageKey     string             `json:"image_key"`
	Status       string             `json:"status"`
	AdminComment *string            `json:"admin_comment"`
	CreatedAt    string             `json:"created_at"`
	Layers       []modLayerResponse `json:"layers"`
}

type modLayerResponse struct {
	ID        string  `json:"id"`
	Type      string  `json:"type"`
	SortOrder int     `json:"sort_order"`
	Text      *string `json:"text"`
	AudioKey  *string `json:"audio_key"`
	VideoKey  *string `json:"video_key"`
}

// HandleAdminListModeration GET /api/admin/moderation
func (h *Handler) HandleAdminListModeration(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	subs, err := h.Store.ListModerationQueue(ctx)
	if err != nil {
		h.Log.Error("list moderation queue", zap.Error(err))
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	resp := make([]moderationItemResponse, 0, len(subs))
	for _, sub := range subs {
		layers, err := h.Store.ListPuzzleLayers(ctx, sub.PuzzleID)
		if err != nil {
			h.Log.Error("list puzzle layers for moderation", zap.String("puzzle_id", sub.PuzzleID), zap.Error(err))
			writeError(w, http.StatusInternalServerError, "internal error")
			return
		}

		modLayers := make([]modLayerResponse, 0, len(layers))
		for _, l := range layers {
			modLayers = append(modLayers, modLayerResponse{
				ID:        l.ID,
				Type:      l.Type,
				SortOrder: l.SortOrder,
				Text:      l.Text,
				AudioKey:  l.AudioKey,
				VideoKey:  l.VideoKey,
			})
		}

		resp = append(resp, moderationItemResponse{
			ID:           sub.ID,
			PuzzleID:     sub.PuzzleID,
			PuzzleTitle:  sub.PuzzleTitle,
			ImageKey:     sub.ImageKey,
			Status:       sub.Status,
			AdminComment: sub.AdminComment,
			CreatedAt:    sub.CreatedAt.UTC().Format("2006-01-02T15:04:05Z"),
			Layers:       modLayers,
		})
	}

	writeJSON(w, http.StatusOK, resp)
}

// HandleAdminApprove POST /api/admin/moderation/{id}/approve
func (h *Handler) HandleAdminApprove(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	user := middleware.UserFromContext(r.Context())

	if err := h.Store.ApprovePuzzle(r.Context(), id, user.ID); err != nil {
		if err == store.ErrNotFound {
			writeError(w, http.StatusNotFound, "submission not found or not pending")
			return
		}
		h.Log.Error("approve puzzle", zap.String("submission_id", id), zap.Error(err))
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

// HandleAdminReject POST /api/admin/moderation/{id}/reject
func (h *Handler) HandleAdminReject(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	user := middleware.UserFromContext(r.Context())

	var req struct {
		Comment string `json:"comment"`
	}
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := h.Store.RejectPuzzle(r.Context(), id, user.ID, req.Comment); err != nil {
		if err == store.ErrNotFound {
			writeError(w, http.StatusNotFound, "submission not found or not pending")
			return
		}
		h.Log.Error("reject puzzle", zap.String("submission_id", id), zap.Error(err))
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}
