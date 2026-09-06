package web

import (
	"context"
	"database/sql"
	"errors"
	"net/http"
	"time"

	"github.com/danielzelisko/private-registry/internal/auth"
	"github.com/danielzelisko/private-registry/internal/store"
)

type Deps struct {
	SessionSecret string
	Auth          *auth.Authenticator
	Store         *store.Store
	Token         http.Handler
}

func NewHandler(deps Deps) http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	sessions := NewSessionManager(deps.SessionSecret)
	loginLimiter := newRateLimiter(20, time.Minute)
	tokenLimiter := newRateLimiter(20, time.Minute)

	if deps.Token != nil {
		mux.Handle("GET /token", tokenLimiter.middleware(deps.Token))
	}

	mux.HandleFunc("GET /login", handleLoginGET)
	mux.Handle("POST /login", loginLimiter.middleware(http.HandlerFunc(handleLoginPOST(sessions, deps.Auth))))

	adminHandler := requireAdmin(deps.Store, sessions, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userID, ok := userIDFromContext(r)
		if !ok {
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}
		user, err := deps.Store.GetUserByID(r.Context(), userID)
		if err != nil {
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_ = adminTemplate.Execute(w, adminPageData{
			Username:  user.Username,
			CSRFToken: csrfFromContext(r),
		})
	}))
	mux.Handle("GET /admin/", adminHandler)

	mux.Handle("POST /logout", requireSession(sessions, http.HandlerFunc(handleLogout(sessions))))

	return withSession(sessions, mux)
}

func withSession(sessions *SessionManager, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		if sess, ok := sessions.Get(r); ok {
			ctx = context.WithValue(ctx, userIDKey, sess.UserID)
			ctx = withCSRF(ctx, sess.CSRFToken)
		}
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func requireSession(sessions *SessionManager, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, ok := userIDFromContext(r); !ok {
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func requireAdmin(st *store.Store, sessions *SessionManager, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userID, ok := userIDFromContext(r)
		if !ok {
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}
		user, err := st.GetUserByID(r.Context(), userID)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				sessions.Clear(w)
				http.Redirect(w, r, "/login", http.StatusSeeOther)
				return
			}
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		if !user.Active {
			sessions.Clear(w)
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}
		if !user.Admin {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, r)
	})
}
