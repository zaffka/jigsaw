package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/zaffka/jigsaw/internal/store"
)

type contextKey string

const (
	contextKeyUser   contextKey = "user"
	contextKeyLocale contextKey = "locale"
)

func UserFromContext(ctx context.Context) *store.User {
	u, _ := ctx.Value(contextKeyUser).(*store.User)
	return u
}

func LocaleFromContext(ctx context.Context) string {
	if l, ok := ctx.Value(contextKeyLocale).(string); ok && l != "" {
		return l
	}
	return "ru"
}

func Auth(st *store.Store) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token := tokenFromRequest(r)
			if token != "" {
				sess, err := st.GetSessionByToken(r.Context(), token)
				if err == nil {
					user, err := st.GetUserByID(r.Context(), sess.UserID)
					if err == nil && !user.Blocked {
						ctx := context.WithValue(r.Context(), contextKeyUser, user)
						r = r.WithContext(ctx)
					}
				}
			}
			next.ServeHTTP(w, r)
		})
	}
}

func RequireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if UserFromContext(r.Context()) == nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte(`{"error":"unauthorized"}`))
			return
		}
		next.ServeHTTP(w, r)
	})
}

func RequireAdmin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user := UserFromContext(r.Context())
		if user == nil || user.Role != "admin" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusForbidden)
			w.Write([]byte(`{"error":"forbidden"}`))
			return
		}
		next.ServeHTTP(w, r)
	})
}

func Locale(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		locale := "ru"
		if user := UserFromContext(r.Context()); user != nil && user.Locale != "" {
			locale = user.Locale
		} else if lang := r.Header.Get("Accept-Language"); lang != "" {
			locale = parseLocale(lang)
		}
		ctx := context.WithValue(r.Context(), contextKeyLocale, locale)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func tokenFromRequest(r *http.Request) string {
	if auth := r.Header.Get("Authorization"); strings.HasPrefix(auth, "Bearer ") {
		return strings.TrimPrefix(auth, "Bearer ")
	}
	if c, err := r.Cookie("session"); err == nil {
		return c.Value
	}
	return ""
}

func parseLocale(acceptLang string) string {
	parts := strings.Split(acceptLang, ",")
	if len(parts) == 0 {
		return "ru"
	}
	tag := strings.TrimSpace(strings.Split(parts[0], ";")[0])
	lang := strings.ToLower(strings.Split(tag, "-")[0])
	switch lang {
	case "en":
		return "en"
	case "es":
		return "es"
	case "zh":
		return "zh"
	case "th":
		return "th"
	default:
		return "ru"
	}
}
