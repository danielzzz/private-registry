package web_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"github.com/danielzelisko/private-registry/internal/acl"
	"github.com/danielzelisko/private-registry/internal/auth"
	"github.com/danielzelisko/private-registry/internal/store"
)

func adminSession(t *testing.T, h http.Handler, adminPass string) (cookie *http.Cookie, csrf string) {
	t.Helper()
	cookie = login(t, h, "admin", adminPass)

	req := httptest.NewRequest(http.MethodGet, "/admin/users", nil)
	req.AddCookie(cookie)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("users page status=%d body=%s", rec.Code, rec.Body.String())
	}
	csrf = extractCSRFToken(rec.Body.String())
	if csrf == "" {
		t.Fatal("expected csrf token on users page")
	}
	return cookie, csrf
}

func dashboardCountRE(tile string) *regexp.Regexp {
	return regexp.MustCompile(`data-dashboard-count="` + tile + `"[^>]*>(\d+)</p>`)
}

func assertDashboardCount(t *testing.T, html, tile string, want int) {
	t.Helper()
	m := dashboardCountRE(tile).FindStringSubmatch(html)
	if len(m) < 2 {
		t.Fatalf("dashboard tile %q not found in body=%s", tile, html)
	}
	if m[1] != strconv.Itoa(want) {
		t.Fatalf("dashboard tile %q: got %s want %d", tile, m[1], want)
	}
}

func seedDashboardEntities(t *testing.T, s *store.Store) {
	t.Helper()
	ctx := context.Background()
	admin, err := s.GetUserByUsername(ctx, "admin")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.CreateGroup(ctx, "developers"); err != nil {
		t.Fatal(err)
	}
	if _, err := s.CreateAPIToken(ctx, admin.ID, "ci-token", "hash-placeholder", nil); err != nil {
		t.Fatal(err)
	}
	if _, err := s.CreateACLRule(ctx, store.ACLRuleInput{
		SubjectKind: acl.SubjectUser,
		SubjectID:   admin.ID,
		Pattern:     "app/*",
		Action:      acl.ActionPull,
	}); err != nil {
		t.Fatal(err)
	}
}

func TestDashboardShowsEntityCounts(t *testing.T) {
	s := openTestStore(t)
	seedUsers(t, s)
	seedDashboardEntities(t, s)
	h := newTestHandler(t, s)
	cookie := login(t, h, "admin", "admin-pass-123")

	req := httptest.NewRequest(http.MethodGet, "/admin/", nil)
	req.AddCookie(cookie)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	body := rec.Body.String()
	assertDashboardCount(t, body, "users", 2)
	assertDashboardCount(t, body, "groups", 1)
	assertDashboardCount(t, body, "tokens", 1)
	assertDashboardCount(t, body, "acl", 1)
}

func TestCreateUserViaPost(t *testing.T) {
	s := openTestStore(t)
	adminPass, _ := seedUsers(t, s)
	h := newTestHandler(t, s)
	cookie, csrf := adminSession(t, h, adminPass)

	body := url.Values{
		"csrf_token": {csrf},
		"username":   {"newuser"},
		"password":   {"new-pass-789"},
	}
	req := httptest.NewRequest(http.MethodPost, "/admin/users", strings.NewReader(body.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(cookie)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusSeeOther {
		t.Fatalf("create status=%d body=%s", rec.Code, rec.Body.String())
	}

	u, err := s.GetUserByUsername(context.Background(), "newuser")
	if err != nil {
		t.Fatalf("user not in store: %v", err)
	}
	if !u.Active {
		t.Fatal("expected new user to be active")
	}

	req = httptest.NewRequest(http.MethodGet, "/admin/users", nil)
	req.AddCookie(cookie)
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("list status=%d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "newuser") {
		t.Fatalf("expected newuser in list, body=%s", rec.Body.String())
	}
}

func TestCreateUserWithoutCSRFFails(t *testing.T) {
	s := openTestStore(t)
	adminPass, _ := seedUsers(t, s)
	h := newTestHandler(t, s)
	cookie := login(t, h, "admin", adminPass)

	body := url.Values{
		"username": {"newuser"},
		"password": {"new-pass-789"},
	}
	req := httptest.NewRequest(http.MethodPost, "/admin/users", strings.NewReader(body.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(cookie)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
}

func TestDisableUser(t *testing.T) {
	s := openTestStore(t)
	adminPass, _ := seedUsers(t, s)
	h := newTestHandler(t, s)
	cookie, csrf := adminSession(t, h, adminPass)

	target, err := s.GetUserByUsername(context.Background(), "user")
	if err != nil {
		t.Fatal(err)
	}

	body := url.Values{"csrf_token": {csrf}}
	req := httptest.NewRequest(http.MethodPost, "/admin/users/"+target.ID+"/disable", strings.NewReader(body.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(cookie)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusSeeOther {
		t.Fatalf("disable status=%d body=%s", rec.Code, rec.Body.String())
	}

	updated, err := s.GetUserByID(context.Background(), target.ID)
	if err != nil {
		t.Fatal(err)
	}
	if updated.Active {
		t.Fatal("expected user to be disabled")
	}
}

func TestEnableUser(t *testing.T) {
	s := openTestStore(t)
	adminPass, _ := seedUsers(t, s)
	h := newTestHandler(t, s)
	cookie, csrf := adminSession(t, h, adminPass)

	target, err := s.GetUserByUsername(context.Background(), "user")
	if err != nil {
		t.Fatal(err)
	}
	if err := s.SetUserActive(context.Background(), target.ID, false); err != nil {
		t.Fatal(err)
	}

	body := url.Values{"csrf_token": {csrf}}
	req := httptest.NewRequest(http.MethodPost, "/admin/users/"+target.ID+"/enable", strings.NewReader(body.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(cookie)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusSeeOther {
		t.Fatalf("enable status=%d body=%s", rec.Code, rec.Body.String())
	}

	updated, err := s.GetUserByID(context.Background(), target.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !updated.Active {
		t.Fatal("expected user to be enabled")
	}
}

func TestToggleUserAdmin(t *testing.T) {
	s := openTestStore(t)
	adminPass, _ := seedUsers(t, s)
	h := newTestHandler(t, s)
	cookie, csrf := adminSession(t, h, adminPass)

	target, err := s.GetUserByUsername(context.Background(), "user")
	if err != nil {
		t.Fatal(err)
	}
	if target.Admin {
		t.Fatal("expected user to start as non-admin")
	}

	body := url.Values{"csrf_token": {csrf}}
	req := httptest.NewRequest(http.MethodPost, "/admin/users/"+target.ID+"/toggle-admin", strings.NewReader(body.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(cookie)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusSeeOther {
		t.Fatalf("toggle-admin status=%d body=%s", rec.Code, rec.Body.String())
	}

	updated, err := s.GetUserByID(context.Background(), target.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !updated.Admin {
		t.Fatal("expected user to be admin after toggle")
	}
}

func TestResetUserPassword(t *testing.T) {
	s := openTestStore(t)
	adminPass, _ := seedUsers(t, s)
	h := newTestHandler(t, s)
	cookie, csrf := adminSession(t, h, adminPass)

	target, err := s.GetUserByUsername(context.Background(), "user")
	if err != nil {
		t.Fatal(err)
	}

	newPass := "brand-new-pass-999"
	body := url.Values{
		"csrf_token": {csrf},
		"password":   {newPass},
	}
	req := httptest.NewRequest(http.MethodPost, "/admin/users/"+target.ID+"/reset-password", strings.NewReader(body.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(cookie)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusSeeOther {
		t.Fatalf("reset-password status=%d body=%s", rec.Code, rec.Body.String())
	}

	updated, err := s.GetUserByID(context.Background(), target.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !auth.CheckPassword(updated.PasswordHash, newPass) {
		t.Fatal("expected password hash to match new password")
	}
}

func TestCannotDisableLastActiveAdmin(t *testing.T) {
	s := openTestStore(t)
	adminPass, _ := seedUsers(t, s)
	h := newTestHandler(t, s)
	cookie, csrf := adminSession(t, h, adminPass)

	admin, err := s.GetUserByUsername(context.Background(), "admin")
	if err != nil {
		t.Fatal(err)
	}

	body := url.Values{"csrf_token": {csrf}}
	req := httptest.NewRequest(http.MethodPost, "/admin/users/"+admin.ID+"/disable", strings.NewReader(body.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(cookie)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusSeeOther {
		t.Fatalf("disable status=%d body=%s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Header().Get("Location"), "error=") {
		t.Fatalf("expected error redirect, got %q", rec.Header().Get("Location"))
	}

	updated, err := s.GetUserByID(context.Background(), admin.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !updated.Active || !updated.Admin {
		t.Fatal("expected admin to remain active")
	}
}

func TestCannotRemoveLastActiveAdmin(t *testing.T) {
	s := openTestStore(t)
	adminPass, _ := seedUsers(t, s)
	h := newTestHandler(t, s)
	cookie, csrf := adminSession(t, h, adminPass)

	admin, err := s.GetUserByUsername(context.Background(), "admin")
	if err != nil {
		t.Fatal(err)
	}

	body := url.Values{"csrf_token": {csrf}}
	req := httptest.NewRequest(http.MethodPost, "/admin/users/"+admin.ID+"/toggle-admin", strings.NewReader(body.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(cookie)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusSeeOther {
		t.Fatalf("toggle-admin status=%d body=%s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Header().Get("Location"), "error=") {
		t.Fatalf("expected error redirect, got %q", rec.Header().Get("Location"))
	}

	updated, err := s.GetUserByID(context.Background(), admin.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !updated.Admin {
		t.Fatal("expected admin to remain admin")
	}
}
