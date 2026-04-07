package handler

import (
	"net/http"

	"github.com/zaffka/jigsaw/internal/middleware"
	"github.com/zaffka/jigsaw/internal/store"
	"go.uber.org/zap"
)

type childResponse struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	AvatarEmoji string `json:"avatar_emoji"`
	CreatedAt   string `json:"created_at"`
}

func childToResponse(c *store.Child) childResponse {
	return childResponse{
		ID:          c.ID,
		Name:        c.Name,
		AvatarEmoji: c.AvatarEmoji,
		CreatedAt:   c.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
}

// HandleParentListChildren GET /api/parent/children
func (h *Handler) HandleParentListChildren(w http.ResponseWriter, r *http.Request) {
	user := middleware.UserFromContext(r.Context())

	list, err := h.Store.ListChildren(r.Context(), user.ID)
	if err != nil {
		h.Log.Error("list children", zap.Error(err))
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	resp := make([]childResponse, 0, len(list))
	for _, c := range list {
		resp = append(resp, childToResponse(c))
	}
	writeJSON(w, http.StatusOK, resp)
}

// HandleParentCreateChild POST /api/parent/children
func (h *Handler) HandleParentCreateChild(w http.ResponseWriter, r *http.Request) {
	user := middleware.UserFromContext(r.Context())

	var req struct {
		Name        string `json:"name"`
		Pin         string `json:"pin"`
		AvatarEmoji string `json:"avatar_emoji"`
	}
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Name == "" {
		writeError(w, http.StatusBadRequest, "name is required")
		return
	}
	if req.AvatarEmoji == "" {
		req.AvatarEmoji = "🧒"
	}

	c, err := h.Store.CreateChild(r.Context(), user.ID, req.Name, req.Pin, req.AvatarEmoji)
	if err != nil {
		h.Log.Error("create child", zap.Error(err))
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	writeJSON(w, http.StatusCreated, childToResponse(c))
}

// HandleParentGetChild GET /api/parent/children/{id}
func (h *Handler) HandleParentGetChild(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	child, err := h.Store.GetChild(r.Context(), id)
	if err != nil {
		if err == store.ErrNotFound {
			writeError(w, http.StatusNotFound, "not found")
			return
		}
		h.Log.Error("get child", zap.Error(err))
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	user := middleware.UserFromContext(r.Context())
	if child.UserID != user.ID {
		writeError(w, http.StatusNotFound, "not found")
		return
	}

	writeJSON(w, http.StatusOK, childToResponse(child))
}

// HandleParentUpdateChild PUT /api/parent/children/{id}
func (h *Handler) HandleParentUpdateChild(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	child, err := h.Store.GetChild(r.Context(), id)
	if err != nil {
		if err == store.ErrNotFound {
			writeError(w, http.StatusNotFound, "not found")
			return
		}
		h.Log.Error("get child for update", zap.Error(err))
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	user := middleware.UserFromContext(r.Context())
	if child.UserID != user.ID {
		writeError(w, http.StatusNotFound, "not found")
		return
	}

	var req struct {
		Name        string `json:"name"`
		Pin         string `json:"pin"`
		AvatarEmoji string `json:"avatar_emoji"`
	}
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	updated, err := h.Store.UpdateChild(r.Context(), id, req.Name, req.Pin, req.AvatarEmoji)
	if err != nil {
		if err == store.ErrNotFound {
			writeError(w, http.StatusNotFound, "not found")
			return
		}
		h.Log.Error("update child", zap.Error(err))
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	writeJSON(w, http.StatusOK, childToResponse(updated))
}

// HandleParentDeleteChild DELETE /api/parent/children/{id}
func (h *Handler) HandleParentDeleteChild(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	child, err := h.Store.GetChild(r.Context(), id)
	if err != nil {
		if err == store.ErrNotFound {
			writeError(w, http.StatusNotFound, "not found")
			return
		}
		h.Log.Error("get child for delete", zap.Error(err))
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	user := middleware.UserFromContext(r.Context())
	if child.UserID != user.ID {
		writeError(w, http.StatusNotFound, "not found")
		return
	}

	if err := h.Store.DeleteChild(r.Context(), id); err != nil {
		h.Log.Error("delete child", zap.Error(err))
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
