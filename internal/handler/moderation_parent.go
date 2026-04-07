package handler

import (
	"net/http"

	"github.com/zaffka/jigsaw/internal/middleware"
	"github.com/zaffka/jigsaw/internal/store"
	"go.uber.org/zap"
)

type submissionResponse struct {
	ID           string  `json:"id"`
	PuzzleID     string  `json:"puzzle_id"`
	PuzzleTitle  string  `json:"puzzle_title"`
	ImageKey     string  `json:"image_key"`
	Status       string  `json:"status"`
	AdminComment *string `json:"admin_comment"`
	CreatedAt    string  `json:"created_at"`
	ReviewedAt   *string `json:"reviewed_at"`
}

func submissionToResponse(s *store.Submission) submissionResponse {
	r := submissionResponse{
		ID:           s.ID,
		PuzzleID:     s.PuzzleID,
		PuzzleTitle:  s.PuzzleTitle,
		ImageKey:     s.ImageKey,
		Status:       s.Status,
		AdminComment: s.AdminComment,
		CreatedAt:    s.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
	if s.ReviewedAt != nil {
		t := s.ReviewedAt.Format("2006-01-02T15:04:05Z07:00")
		r.ReviewedAt = &t
	}
	return r
}

// HandleParentSubmitPuzzle POST /api/parent/puzzles/{id}/submit
func (h *Handler) HandleParentSubmitPuzzle(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	user := middleware.UserFromContext(r.Context())

	sub, err := h.Store.SubmitPuzzle(r.Context(), id, user.ID)
	if err != nil {
		if err == store.ErrNotFound {
			writeError(w, http.StatusNotFound, "puzzle not found")
			return
		}
		if err == store.ErrConflict {
			writeError(w, http.StatusConflict, "already submitted")
			return
		}
		h.Log.Error("submit puzzle", zap.Error(err))
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	writeJSON(w, http.StatusOK, submissionToResponse(sub))
}

// HandleParentListNotifications GET /api/parent/notifications
func (h *Handler) HandleParentListNotifications(w http.ResponseWriter, r *http.Request) {
	user := middleware.UserFromContext(r.Context())

	list, err := h.Store.ListParentNotifications(r.Context(), user.ID)
	if err != nil {
		h.Log.Error("list parent notifications", zap.Error(err))
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	resp := make([]submissionResponse, 0, len(list))
	for _, sub := range list {
		resp = append(resp, submissionToResponse(sub))
	}

	for _, sub := range list {
		if err := h.Store.MarkNotified(r.Context(), sub.ID); err != nil {
			h.Log.Warn("mark notified", zap.String("id", sub.ID), zap.Error(err))
		}
	}

	writeJSON(w, http.StatusOK, resp)
}
