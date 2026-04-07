package handler

import (
	"net/http"

	"go.uber.org/zap"
)

type childAuthRequest struct {
	ChildID string `json:"child_id"`
	PIN     string `json:"pin"`
}

type childAuthResponse struct {
	Token       string `json:"token"`
	ChildID     string `json:"child_id"`
	Name        string `json:"name"`
	AvatarEmoji string `json:"avatar_emoji"`
}

// HandleChildAuth POST /api/children/auth
// Authenticates a child by ID and PIN, returns a session token.
// Token is intended for sessionStorage on the frontend (no cookie).
func (h *Handler) HandleChildAuth(w http.ResponseWriter, r *http.Request) {
	var req childAuthRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.ChildID == "" || req.PIN == "" {
		writeError(w, http.StatusBadRequest, "child_id and pin are required")
		return
	}

	child, err := h.Store.GetChild(r.Context(), req.ChildID)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "invalid child_id or pin")
		return
	}

	ok, err := h.Store.VerifyChildPIN(r.Context(), req.ChildID, req.PIN)
	if err != nil {
		h.Log.Error("child auth: verify pin", zap.Error(err))
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	if !ok {
		writeError(w, http.StatusUnauthorized, "invalid child_id or pin")
		return
	}

	token, err := h.Store.CreateChildSession(r.Context(), req.ChildID)
	if err != nil {
		h.Log.Error("child auth: create session", zap.Error(err))
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	writeJSON(w, http.StatusOK, childAuthResponse{
		Token:       token,
		ChildID:     child.ID,
		Name:        child.Name,
		AvatarEmoji: child.AvatarEmoji,
	})
}
