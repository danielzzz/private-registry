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
	Store         store.Store
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

	mux.Handle("GET /admin/{$}", requireAdmin(deps.Store, sessions, http.HandlerFunc(handleDashboardGET(deps.Store))))
	mux.Handle("GET /admin/users", requireAdmin(deps.Store, sessions, http.HandlerFunc(handleUsersGET(deps.Store))))
	mux.Handle("POST /admin/users", requireAdmin(deps.Store, sessions, http.HandlerFunc(handleUsersPOST(deps.Store))))
	mux.Handle("POST /admin/users/{id}/disable", requireAdmin(deps.Store, sessions, http.HandlerFunc(handleUserDisablePOST(deps.Store))))
	mux.Handle("POST /admin/users/{id}/enable", requireAdmin(deps.Store, sessions, http.HandlerFunc(handleUserEnablePOST(deps.Store))))
	mux.Handle("POST /admin/users/{id}/toggle-admin", requireAdmin(deps.Store, sessions, http.HandlerFunc(handleUserToggleAdminPOST(deps.Store))))
	mux.Handle("POST /admin/users/{id}/reset-password", requireAdmin(deps.Store, sessions, http.HandlerFunc(handleUserResetPasswordPOST(deps.Store))))

	mux.Handle("GET /admin/tokens", requireAdmin(deps.Store, sessions, http.HandlerFunc(handleAPITokensGET(deps.Store))))
	mux.Handle("POST /admin/tokens", requireAdmin(deps.Store, sessions, http.HandlerFunc(handleAPITokensPOST(deps.Store))))
	mux.Handle("POST /admin/tokens/{id}/revoke", requireAdmin(deps.Store, sessions, http.HandlerFunc(handleAPITokenRevokePOST(deps.Store))))

	mux.Handle("GET /admin/groups", requireAdmin(deps.Store, sessions, http.HandlerFunc(handleGroupsGET(deps.Store))))
	mux.Handle("POST /admin/groups", requireAdmin(deps.Store, sessions, http.HandlerFunc(handleGroupsPOST(deps.Store))))
	mux.Handle("POST /admin/groups/{id}/add-member", requireAdmin(deps.Store, sessions, http.HandlerFunc(handleGroupAddMemberPOST(deps.Store))))
	mux.Handle("POST /admin/groups/{id}/remove-member", requireAdmin(deps.Store, sessions, http.HandlerFunc(handleGroupRemoveMemberPOST(deps.Store))))

	mux.Handle("GET /admin/acl", requireAdmin(deps.Store, sessions, http.HandlerFunc(handleACLRulesGET(deps.Store))))
	mux.Handle("POST /admin/acl", requireAdmin(deps.Store, sessions, http.HandlerFunc(handleACLRulesPOST(deps.Store))))
	mux.Handle("POST /admin/acl/{id}/delete", requireAdmin(deps.Store, sessions, http.HandlerFunc(handleACLRuleDeletePOST(deps.Store))))

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

func requireAdmin(st store.Store, sessions *SessionManager, next http.Handler) http.Handler {
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
