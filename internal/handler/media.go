package handler

import (
	"bytes"
	"io"
	"net/http"
	"strings"

	"github.com/zaffka/jigsaw/internal/crypto"
	"go.uber.org/zap"
)

// isPrivateKey reports whether the S3 key refers to private (encrypted) media.
func isPrivateKey(key string) bool {
	return strings.HasPrefix(key, "audio/") || strings.HasPrefix(key, "video/")
}

// HandleMedia streams an S3 object to the client.
// Route: GET /api/media/{path...}
// Private audio/ and video/ keys are decrypted on the fly when MediaEncryptionKey is set.
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

	// Decrypt private media if encryption is enabled.
	if len(h.MediaEncryptionKey) > 0 && isPrivateKey(key) {
		ciphertext, err := io.ReadAll(obj.Body)
		if err != nil {
			h.Log.Error("media: read encrypted body", zap.String("key", key), zap.Error(err))
			writeError(w, http.StatusInternalServerError, "internal error")
			return
		}
		plaintext, err := crypto.Decrypt(h.MediaEncryptionKey, ciphertext)
		if err != nil {
			h.Log.Error("media: decrypt", zap.String("key", key), zap.Error(err))
			writeError(w, http.StatusInternalServerError, "internal error")
			return
		}
		io.Copy(w, bytes.NewReader(plaintext))
		return
	}

	io.Copy(w, obj.Body)
}
