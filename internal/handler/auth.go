package handler

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/zaffka/jigsaw/internal/middleware"
	"github.com/zaffka/jigsaw/internal/store"
	"go.uber.org/zap"
)

type registerRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	Locale   string `json:"locale"`
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type userResponse struct {
	ID     string `json:"id"`
	Email  string `json:"email"`
	Role   string `json:"role"`
	Locale string `json:"locale"`
}

func (h *Handler) HandleRegister(w http.ResponseWriter, r *http.Request) {
	var req registerRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	req.Email = strings.ToLower(strings.TrimSpace(req.Email))
	if req.Email == "" || req.Password == "" {
		writeError(w, http.StatusBadRequest, "email and password are required")
		return
	}
	if len(req.Password) < 8 {
		writeError(w, http.StatusBadRequest, "password must be at least 8 characters")
		return
	}
	if req.Locale == "" {
		req.Locale = middleware.LocaleFromContext(r.Context())
	}

	user, err := h.Store.CreateUser(r.Context(), req.Email, req.Password, req.Locale)
	if errors.Is(err, store.ErrEmailTaken) {
		writeError(w, http.StatusConflict, "email already registered")
		return
	}
	if err != nil {
		h.Log.Error("register: create user", zap.Error(err))
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	sess, err := h.Store.CreateSession(r.Context(), user.ID)
	if err != nil {
		h.Log.Error("register: create session", zap.Error(err))
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	setSessionCookie(w, sess.Token, sess.ExpiresAt)
	writeJSON(w, http.StatusCreated, userResponse{
		ID:     user.ID,
		Email:  user.Email,
		Role:   user.Role,
		Locale: user.Locale,
	})
}

func (h *Handler) HandleLogin(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	req.Email = strings.ToLower(strings.TrimSpace(req.Email))

	user, err := h.Store.GetUserByEmail(r.Context(), req.Email)
	if err != nil || !h.Store.CheckPassword(user.PasswordHash, req.Password) {
		writeError(w, http.StatusUnauthorized, "invalid email or password")
		return
	}

	sess, err := h.Store.CreateSession(r.Context(), user.ID)
	if err != nil {
		h.Log.Error("login: create session", zap.Error(err))
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	setSessionCookie(w, sess.Token, sess.ExpiresAt)
	writeJSON(w, http.StatusOK, userResponse{
		ID:     user.ID,
		Email:  user.Email,
		Role:   user.Role,
		Locale: user.Locale,
	})
}

func (h *Handler) HandleLogout(w http.ResponseWriter, r *http.Request) {
	c, err := r.Cookie("session")
	if err == nil {
		h.Store.DeleteSession(r.Context(), c.Value)
	}
	clearSessionCookie(w)
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) HandleMe(w http.ResponseWriter, r *http.Request) {
	user := middleware.UserFromContext(r.Context())
	if user == nil {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	writeJSON(w, http.StatusOK, userResponse{
		ID:     user.ID,
		Email:  user.Email,
		Role:   user.Role,
		Locale: user.Locale,
	})
}

func setSessionCookie(w http.ResponseWriter, token string, expires time.Time) {
	http.SetCookie(w, &http.Cookie{
		Name:     "session",
		Value:    token,
		Expires:  expires,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Path:     "/",
	})
}

func clearSessionCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     "session",
		Value:    "",
		MaxAge:   -1,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Path:     "/",
	})
}
