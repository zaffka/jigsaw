package handler

import (
	"net/http"
	"strings"

	"go.uber.org/zap"
)

// HandlePlayComplete POST /api/play/{id}/complete
// id is the catalog_puzzles.id.
// Reads child token from X-Child-Token header, Authorization header, or child_session cookie.
// Records play result if child token is valid.
func (h *Handler) HandlePlayComplete(w http.ResponseWriter, r *http.Request) {
	catalogID := r.PathValue("id")

	token := childTokenFromRequest(r)
	if token != "" {
		child, err := h.Store.GetChildByToken(r.Context(), token)
		if err == nil {
			cp, err := h.Store.GetCatalogPuzzle(r.Context(), catalogID)
			if err == nil {
				if err := h.Store.RecordPlayResult(r.Context(), child.ID, cp.PuzzleID); err != nil {
					h.Log.Warn("record play result", zap.Error(err))
				}
			}
		}
	}

	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

// HandlePlayCompleted GET /api/play/completed
// Returns list of catalog puzzle IDs completed by the current child.
// Requires X-Child-Token header, Authorization header, or child_session cookie.
func (h *Handler) HandlePlayCompleted(w http.ResponseWriter, r *http.Request) {
	token := childTokenFromRequest(r)
	if token == "" {
		writeJSON(w, http.StatusOK, []string{})
		return
	}
	child, err := h.Store.GetChildByToken(r.Context(), token)
	if err != nil {
		writeJSON(w, http.StatusOK, []string{})
		return
	}
	ids, err := h.Store.GetCompletedPuzzleIDs(r.Context(), child.ID)
	if err != nil {
		h.Log.Error("get completed puzzle ids", zap.Error(err))
		writeJSON(w, http.StatusOK, []string{})
		return
	}
	if ids == nil {
		ids = []string{}
	}
	writeJSON(w, http.StatusOK, ids)
}

func childTokenFromRequest(r *http.Request) string {
	// Check X-Child-Token header first
	if t := r.Header.Get("X-Child-Token"); t != "" {
		return t
	}
	// Check Authorization header for "Child <token>"
	if auth := r.Header.Get("Authorization"); strings.HasPrefix(auth, "Child ") {
		return strings.TrimPrefix(auth, "Child ")
	}
	// Check cookie
	if c, err := r.Cookie("child_session"); err == nil {
		return c.Value
	}
	return ""
}
