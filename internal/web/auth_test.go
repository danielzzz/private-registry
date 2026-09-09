package web_test

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/danielzelisko/private-registry/internal/auth"
	"github.com/danielzelisko/private-registry/internal/store"
	"github.com/danielzelisko/private-registry/internal/store/sqlite"
	"github.com/danielzelisko/private-registry/internal/web"
)

const testSessionSecret = "test-session-secret"

func openTestStore(t *testing.T) store.Store {
	t.Helper()
	path := filepath.Join(t.TempDir(), "test.db")
	s, err := sqlite.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = s.Close() })
	return s
}

func newTestHandler(t *testing.T, s store.Store) http.Handler {
	t.Helper()
	return web.NewHandler(web.Deps{
		SessionSecret: testSessionSecret,
		Auth:          &auth.Authenticator{Store: s},
		Store:         s,
	})
}

func seedUsers(t *testing.T, s store.Store) (adminPass, userPass string) {
	t.Helper()
	ctx := context.Background()
	adminPass = "admin-pass-123"
	userPass = "user-pass-456"
	adminHash, err := auth.HashPassword(adminPass)
	if err != nil {
		t.Fatal(err)
	}
	userHash, err := auth.HashPassword(userPass)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.CreateUser(ctx, "admin", adminHash, true); err != nil {
		t.Fatal(err)
	}
	if _, err := s.CreateUser(ctx, "user", userHash, false); err != nil {
		t.Fatal(err)
	}
	return adminPass, userPass
}

func login(t *testing.T, h http.Handler, username, password string) *http.Cookie {
	t.Helper()
	body := url.Values{"username": {username}, "password": {password}}
	req := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader(body.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusSeeOther {
		t.Fatalf("login status=%d body=%s", rec.Code, rec.Body.String())
	}
	for _, c := range rec.Result().Cookies() {
		if c.Name == "session" {
			return c
		}
	}
	t.Fatal("expected session cookie after login")
	return nil
}

var csrfTokenRE = regexp.MustCompile(`name="csrf_token"\s+value="([^"]+)"`)

func extractCSRFToken(html string) string {
	m := csrfTokenRE.FindStringSubmatch(html)
	if len(m) < 2 {
		return ""
	}
	return m[1]
}

func TestLoginSetsSessionCookie(t *testing.T) {
	s := openTestStore(t)
	adminPass, _ := seedUsers(t, s)
	h := newTestHandler(t, s)

	body := url.Values{"username": {"admin"}, "password": {adminPass}}
	req := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader(body.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusSeeOther {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var sessionCookie *http.Cookie
	for _, c := range rec.Result().Cookies() {
		if c.Name == "session" {
			sessionCookie = c
			break
		}
	}
	if sessionCookie == nil || sessionCookie.Value == "" {
		t.Fatal("expected non-empty session cookie")
	}
}

func TestNonAdminCannotLoginToAdmin(t *testing.T) {
	s := openTestStore(t)
	_, userPass := seedUsers(t, s)
	h := newTestHandler(t, s)

	body := url.Values{"username": {"user"}, "password": {userPass}}
	req := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader(body.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "Admin access required") {
		t.Fatalf("expected admin access message, body=%s", rec.Body.String())
	}
	for _, c := range rec.Result().Cookies() {
		if c.Name == "session" && c.Value != "" {
			t.Fatal("non-admin login must not set a session cookie")
		}
	}
}

func TestPostWithoutCSRFFails(t *testing.T) {
	s := openTestStore(t)
	adminPass, _ := seedUsers(t, s)
	h := newTestHandler(t, s)
	cookie := login(t, h, "admin", adminPass)

	req := httptest.NewRequest(http.MethodPost, "/logout", nil)
	req.AddCookie(cookie)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
}

func TestLogoutClearsSession(t *testing.T) {
	s := openTestStore(t)
	adminPass, _ := seedUsers(t, s)
	h := newTestHandler(t, s)
	cookie := login(t, h, "admin", adminPass)

	req := httptest.NewRequest(http.MethodGet, "/admin/", nil)
	req.AddCookie(cookie)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("admin status=%d", rec.Code)
	}
	csrf := extractCSRFToken(rec.Body.String())
	if csrf == "" {
		t.Fatal("expected csrf token on admin page")
	}

	body := url.Values{"csrf_token": {csrf}}
	req = httptest.NewRequest(http.MethodPost, "/logout", strings.NewReader(body.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(cookie)
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusSeeOther {
		t.Fatalf("logout status=%d body=%s", rec.Code, rec.Body.String())
	}

	var cleared bool
	for _, c := range rec.Result().Cookies() {
		if c.Name == "session" && (c.MaxAge < 0 || c.Value == "") {
			cleared = true
		}
	}
	if !cleared {
		t.Fatal("expected session cookie to be cleared on logout")
	}

	req = httptest.NewRequest(http.MethodGet, "/admin/", nil)
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusSeeOther {
		t.Fatalf("expected redirect to login after logout, status=%d", rec.Code)
	}
	if loc := rec.Header().Get("Location"); loc != "/login" {
		t.Fatalf("location=%q", loc)
	}
}

func TestUnauthenticatedAdminRedirectsToLogin(t *testing.T) {
	s := openTestStore(t)
	seedUsers(t, s)
	h := newTestHandler(t, s)

	req := httptest.NewRequest(http.MethodGet, "/admin/", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusSeeOther {
		t.Fatalf("status=%d", rec.Code)
	}
	if rec.Header().Get("Location") != "/login" {
		t.Fatalf("location=%q", rec.Header().Get("Location"))
	}
}

func TestLoginRateLimit(t *testing.T) {
	s := openTestStore(t)
	seedUsers(t, s)
	h := newTestHandler(t, s)

	body := url.Values{"username": {"admin"}, "password": {"wrong-password"}}
	for i := 0; i < 20; i++ {
		req := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader(body.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		if rec.Code == http.StatusTooManyRequests {
			t.Fatalf("request %d should not be rate limited", i+1)
		}
	}

	req := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader(body.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusTooManyRequests {
		t.Fatalf("expected 429 on 21st request, got %d body=%s", rec.Code, rec.Body.String())
	}
}

func TestInactiveAdminCannotAccessAdmin(t *testing.T) {
	s := openTestStore(t)
	adminPass, _ := seedUsers(t, s)
	h := newTestHandler(t, s)
	cookie := login(t, h, "admin", adminPass)

	admin, err := s.GetUserByUsername(context.Background(), "admin")
	if err != nil {
		t.Fatal(err)
	}
	if err := s.SetUserActive(context.Background(), admin.ID, false); err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodGet, "/admin/", nil)
	req.AddCookie(cookie)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusSeeOther {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	if rec.Header().Get("Location") != "/login" {
		t.Fatalf("location=%q", rec.Header().Get("Location"))
	}
}

func TestLoginPageRenders(t *testing.T) {
	s := openTestStore(t)
	seedUsers(t, s)
	h := newTestHandler(t, s)

	req := httptest.NewRequest(http.MethodGet, "/login", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d", rec.Code)
	}
	body, _ := io.ReadAll(rec.Body)
	if !strings.Contains(string(body), "tailwindcss.com") {
		t.Fatal("expected Tailwind CDN on login page")
	}
}
