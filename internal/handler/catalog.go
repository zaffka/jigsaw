package handler

import (
	"net/http"
	"path"

	"github.com/zaffka/jigsaw/internal/store"
	"go.uber.org/zap"
)

// HandleListCatalog GET /api/catalog
func (h *Handler) HandleListCatalog(w http.ResponseWriter, r *http.Request) {
	list, err := h.Store.ListPublicCatalog(r.Context())
	if err != nil {
		h.Log.Error("list catalog", zap.Error(err))
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	resp := make([]catalogPuzzleResponse, 0, len(list))
	for _, cp := range list {
		resp = append(resp, catalogPuzzleToResponse(cp))
	}
	writeJSON(w, http.StatusOK, resp)
}

// HandleGetCatalogPuzzle GET /api/catalog/{id}
func (h *Handler) HandleGetCatalogPuzzle(w http.ResponseWriter, r *http.Request) {
	id := path.Base(r.URL.Path)
	cp, err := h.Store.GetCatalogPuzzle(r.Context(), id)
	if err != nil {
		if err == store.ErrNotFound {
			writeError(w, http.StatusNotFound, "not found")
			return
		}
		h.Log.Error("get catalog puzzle", zap.Error(err))
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	writeJSON(w, http.StatusOK, catalogPuzzleToResponse(cp))
}
