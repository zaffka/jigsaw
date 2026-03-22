package handler

import (
	"io"
	"net/http"
	"strings"
)

// HandleMedia streams an S3 object to the client.
// Route: GET /api/media/{path...}
// The path is the S3 object key (e.g. "originals/abc.jpg", "pieces/uuid/1.png").
func (h *Handler) HandleMedia(w http.ResponseWriter, r *http.Request) {
	key := strings.TrimPrefix(r.PathValue("path"), "/")
	if key == "" {
		writeError(w, http.StatusBadRequest, "missing path")
		return
	}

	obj, err := h.S3.GetObject(r.Context(), key)
	if err != nil {
		writeError(w, http.StatusNotFound, "not found")
		return
	}
	defer obj.Body.Close()

	if obj.ContentType != "" {
		w.Header().Set("Content-Type", obj.ContentType)
	}
	w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")

	io.Copy(w, obj.Body)
}
