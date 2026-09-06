package web_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
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

func TestDashboardShowsUserCount(t *testing.T) {
	s := openTestStore(t)
	seedUsers(t, s)
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
	if !strings.Contains(body, "Users") || !strings.Contains(body, "2") {
		t.Fatalf("expected dashboard user count 2, body=%s", body)
	}
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
