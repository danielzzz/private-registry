package web_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"regexp"
	"strings"
	"testing"

	"github.com/danielzelisko/private-registry/internal/auth"
)

var apiTokenPlaintextRE = regexp.MustCompile(`prt_[^_]+_[0-9a-fA-F]{64}`)

func adminTokensSession(t *testing.T, h http.Handler, adminPass string) (cookie *http.Cookie, csrf string) {
	t.Helper()
	cookie = login(t, h, "admin", adminPass)

	req := httptest.NewRequest(http.MethodGet, "/admin/tokens", nil)
	req.AddCookie(cookie)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("tokens page status=%d body=%s", rec.Code, rec.Body.String())
	}
	csrf = extractCSRFToken(rec.Body.String())
	if csrf == "" {
		t.Fatal("expected csrf token on tokens page")
	}
	return cookie, csrf
}

func TestCreateAPITokenReturnsPlaintextOnce(t *testing.T) {
	s := openTestStore(t)
	adminPass, _ := seedUsers(t, s)
	h := newTestHandler(t, s)
	cookie, csrf := adminTokensSession(t, h, adminPass)

	admin, err := s.GetUserByUsername(context.Background(), "admin")
	if err != nil {
		t.Fatal(err)
	}

	body := url.Values{
		"csrf_token": {csrf},
		"user_id":    {admin.ID},
		"name":       {"ci-deploy"},
	}
	req := httptest.NewRequest(http.MethodPost, "/admin/tokens", strings.NewReader(body.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(cookie)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusSeeOther {
		t.Fatalf("create status=%d body=%s", rec.Code, rec.Body.String())
	}

	loc := rec.Header().Get("Location")
	m := apiTokenPlaintextRE.FindString(loc)
	if m == "" {
		t.Fatalf("expected plaintext token in redirect location, got %q", loc)
	}

	parts := strings.SplitN(strings.TrimPrefix(m, "prt_"), "_", 2)
	if len(parts) != 2 {
		t.Fatalf("invalid token format: %s", m)
	}
	tokenID, secretPart := parts[0], parts[1]

	tokens, err := s.ListAPITokensByUser(context.Background(), admin.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(tokens) != 1 {
		t.Fatalf("expected 1 token, got %d", len(tokens))
	}
	if tokens[0].ID != tokenID || tokens[0].Name != "ci-deploy" {
		t.Fatalf("token=%+v", tokens[0])
	}
	if !auth.CheckPassword(tokens[0].TokenHash, secretPart) {
		t.Fatal("expected token hash to match secret part")
	}

	req = httptest.NewRequest(http.MethodGet, "/admin/tokens", nil)
	req.AddCookie(cookie)
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("list status=%d", rec.Code)
	}
	if apiTokenPlaintextRE.MatchString(rec.Body.String()) {
		t.Fatal("plaintext token should not appear on subsequent GET")
	}
}

func TestRevokeAPIToken(t *testing.T) {
	s := openTestStore(t)
	adminPass, _ := seedUsers(t, s)
	h := newTestHandler(t, s)
	cookie, csrf := adminTokensSession(t, h, adminPass)

	admin, err := s.GetUserByUsername(context.Background(), "admin")
	if err != nil {
		t.Fatal(err)
	}

	body := url.Values{
		"csrf_token": {csrf},
		"user_id":    {admin.ID},
		"name":       {"to-revoke"},
	}
	req := httptest.NewRequest(http.MethodPost, "/admin/tokens", strings.NewReader(body.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(cookie)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusSeeOther {
		t.Fatalf("create status=%d body=%s", rec.Code, rec.Body.String())
	}

	tokens, err := s.ListAPITokensByUser(context.Background(), admin.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(tokens) != 1 || !tokens[0].Active {
		t.Fatalf("expected 1 active token, got %+v", tokens)
	}
	tokenID := tokens[0].ID

	body = url.Values{"csrf_token": {csrf}}
	req = httptest.NewRequest(http.MethodPost, "/admin/tokens/"+tokenID+"/revoke", strings.NewReader(body.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(cookie)
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusSeeOther {
		t.Fatalf("revoke status=%d body=%s", rec.Code, rec.Body.String())
	}

	tokens, err = s.ListAPITokensByUser(context.Background(), admin.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(tokens) != 1 || tokens[0].Active {
		t.Fatalf("expected revoked token, got %+v", tokens[0])
	}
}
