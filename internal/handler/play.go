package handler

import (
	"net/http"
)

// HandlePlayComplete POST /api/play/:id/complete
// Best-effort stub: always returns 200, no auth required.
// In a later phase this will record play_results.
func (h *Handler) HandlePlayComplete(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}
