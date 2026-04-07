package handler

import (
	"net/http"

	"github.com/zaffka/jigsaw/internal/store"
	"go.uber.org/zap"
)

type categoryResponse struct {
	ID        string            `json:"id"`
	Slug      string            `json:"slug"`
	Name      map[string]string `json:"name"`
	Icon      string            `json:"icon"`
	SortOrder int               `json:"sort_order"`
}

func categoryToResponse(c *store.Category) categoryResponse {
	return categoryResponse{
		ID:        c.ID,
		Slug:      c.Slug,
		Name:      c.Name,
		Icon:      c.Icon,
		SortOrder: c.SortOrder,
	}
}

// HandleListCategories GET /api/categories
func (h *Handler) HandleListCategories(w http.ResponseWriter, r *http.Request) {
	list, err := h.Store.ListCategories(r.Context())
	if err != nil {
		h.Log.Error("list categories", zap.Error(err))
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	resp := make([]categoryResponse, 0, len(list))
	for _, c := range list {
		resp = append(resp, categoryToResponse(c))
	}
	writeJSON(w, http.StatusOK, resp)
}

// HandleAdminListCategories GET /api/admin/categories
func (h *Handler) HandleAdminListCategories(w http.ResponseWriter, r *http.Request) {
	h.HandleListCategories(w, r)
}

// HandleAdminCreateCategory POST /api/admin/categories
func (h *Handler) HandleAdminCreateCategory(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Slug      string            `json:"slug"`
		Name      map[string]string `json:"name"`
		Icon      string            `json:"icon"`
		SortOrder int               `json:"sort_order"`
	}
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	c, err := h.Store.CreateCategory(r.Context(), req.Slug, req.Name, req.Icon, req.SortOrder)
	if err != nil {
		h.Log.Error("admin create category", zap.Error(err))
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	writeJSON(w, http.StatusCreated, categoryToResponse(c))
}

// HandleAdminUpdateCategory PUT /api/admin/categories/{id}
func (h *Handler) HandleAdminUpdateCategory(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	var req struct {
		Slug      string            `json:"slug"`
		Name      map[string]string `json:"name"`
		Icon      string            `json:"icon"`
		SortOrder int               `json:"sort_order"`
	}
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	c, err := h.Store.UpdateCategory(r.Context(), id, req.Slug, req.Name, req.Icon, req.SortOrder)
	if err != nil {
		if err == store.ErrNotFound {
			writeError(w, http.StatusNotFound, "not found")
			return
		}
		h.Log.Error("admin update category", zap.Error(err))
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	writeJSON(w, http.StatusOK, categoryToResponse(c))
}

// HandleAdminDeleteCategory DELETE /api/admin/categories/{id}
func (h *Handler) HandleAdminDeleteCategory(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	if err := h.Store.DeleteCategory(r.Context(), id); err != nil {
		if err == store.ErrConflict {
			writeError(w, http.StatusConflict, "category is referenced by puzzles")
			return
		}
		h.Log.Error("admin delete category", zap.Error(err))
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
