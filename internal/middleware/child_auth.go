package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/zaffka/jigsaw/internal/store"
)

const contextKeyChild contextKey = "child"

// ChildFromContext retrieves the authenticated child from the request context.
func ChildFromContext(ctx context.Context) *store.Child {
	c, _ := ctx.Value(contextKeyChild).(*store.Child)
	return c
}

// ChildAuth reads a Bearer token from Authorization header or child_session cookie,
// looks up the child session, and stores the child in the request context.
// If no valid token is found, the request proceeds without a child in context.
func ChildAuth(st *store.Store) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token := childTokenFromRequest(r)
			if token != "" {
				child, err := st.GetChildByToken(r.Context(), token)
				if err == nil {
					ctx := context.WithValue(r.Context(), contextKeyChild, child)
					r = r.WithContext(ctx)
				}
			}
			next.ServeHTTP(w, r)
		})
	}
}

// RequireChild returns 401 if no child is present in the context.
func RequireChild(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if ChildFromContext(r.Context()) == nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte(`{"error":"unauthorized"}`))
			return
		}
		next.ServeHTTP(w, r)
	})
}

// childTokenFromRequest extracts a token from Authorization: Bearer <token>
// or from the child_session cookie.
func childTokenFromRequest(r *http.Request) string {
	if auth := r.Header.Get("Authorization"); strings.HasPrefix(auth, "Bearer ") {
		return strings.TrimPrefix(auth, "Bearer ")
	}
	if c, err := r.Cookie("child_session"); err == nil {
		return c.Value
	}
	return ""
}
