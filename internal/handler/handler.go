package handler

import (
	"encoding/json"
	"net/http"

	"github.com/zaffka/jigsaw/internal/store"
	"github.com/zaffka/jigsaw/pkg/s3"
	"go.uber.org/zap"
)

type Handler struct {
	Store        *store.Store
	S3           *s3.BucketCli
	Log          *zap.Logger
	CookieSecure bool
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if v != nil {
		json.NewEncoder(w).Encode(v)
	}
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

func decodeJSON(r *http.Request, v any) error {
	return json.NewDecoder(r.Body).Decode(v)
}
