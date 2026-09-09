package web_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/danielzzz/private-registry/internal/acl"
)

func adminACLRulesSession(t *testing.T, h http.Handler, adminPass string) (cookie *http.Cookie, csrf string) {
	t.Helper()
	cookie = login(t, h, "admin", adminPass)

	req := httptest.NewRequest(http.MethodGet, "/admin/acl", nil)
	req.AddCookie(cookie)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("acl page status=%d body=%s", rec.Code, rec.Body.String())
	}
	csrf = extractCSRFToken(rec.Body.String())
	if csrf == "" {
		t.Fatal("expected csrf token on acl page")
	}
	return cookie, csrf
}

func TestCreateACLRuleListed(t *testing.T) {
	s := openTestStore(t)
	adminPass, _ := seedUsers(t, s)
	h := newTestHandler(t, s)
	cookie, csrf := adminACLRulesSession(t, h, adminPass)

	admin, err := s.GetUserByUsername(context.Background(), "admin")
	if err != nil {
		t.Fatal(err)
	}

	body := url.Values{
		"csrf_token":   {csrf},
		"subject_kind": {"user"},
		"subject_id":   {admin.ID},
		"pattern":      {"app/*"},
		"action":       {"pull"},
	}
	req := httptest.NewRequest(http.MethodPost, "/admin/acl", strings.NewReader(body.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(cookie)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusSeeOther {
		t.Fatalf("create acl status=%d body=%s", rec.Code, rec.Body.String())
	}

	rules, err := s.ListACLRules(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(rules) != 1 {
		t.Fatalf("expected 1 rule, got %d", len(rules))
	}
	if rules[0].SubjectKind != acl.SubjectUser || rules[0].SubjectID != admin.ID ||
		rules[0].Pattern != "app/*" || rules[0].Action != acl.ActionPull {
		t.Fatalf("rule=%+v", rules[0])
	}

	req = httptest.NewRequest(http.MethodGet, "/admin/acl", nil)
	req.AddCookie(cookie)
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("list status=%d", rec.Code)
	}
	page := rec.Body.String()
	if !strings.Contains(page, "app/*") {
		t.Fatalf("expected pattern in list page, body=%s", page)
	}
}

func TestCreateAnonymousACLRule(t *testing.T) {
	s := openTestStore(t)
	adminPass, _ := seedUsers(t, s)
	h := newTestHandler(t, s)
	cookie, csrf := adminACLRulesSession(t, h, adminPass)

	body := url.Values{
		"csrf_token":   {csrf},
		"subject_kind": {"anonymous"},
		"pattern":      {"public/*"},
		"action":       {"pull"},
	}
	req := httptest.NewRequest(http.MethodPost, "/admin/acl", strings.NewReader(body.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(cookie)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusSeeOther {
		t.Fatalf("create anonymous acl status=%d body=%s", rec.Code, rec.Body.String())
	}

	rules, err := s.ListACLRules(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(rules) != 1 {
		t.Fatalf("expected 1 rule, got %d", len(rules))
	}
	if rules[0].SubjectKind != acl.SubjectAnonymous || rules[0].SubjectID != "" ||
		rules[0].Pattern != "public/*" || rules[0].Action != acl.ActionPull {
		t.Fatalf("rule=%+v", rules[0])
	}
}

func TestRejectAnonymousPushACLRule(t *testing.T) {
	s := openTestStore(t)
	adminPass, _ := seedUsers(t, s)
	h := newTestHandler(t, s)
	cookie, csrf := adminACLRulesSession(t, h, adminPass)

	body := url.Values{
		"csrf_token":   {csrf},
		"subject_kind": {"anonymous"},
		"pattern":      {"public/*"},
		"action":       {"push"},
	}
	req := httptest.NewRequest(http.MethodPost, "/admin/acl", strings.NewReader(body.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(cookie)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusSeeOther {
		t.Fatalf("create anonymous push acl status=%d body=%s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Header().Get("Location"), "error=") {
		t.Fatalf("expected error redirect, got %q", rec.Header().Get("Location"))
	}

	rules, err := s.ListACLRules(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(rules) != 0 {
		t.Fatalf("expected 0 rules, got %d", len(rules))
	}
}

func TestDeleteACLRule(t *testing.T) {
	s := openTestStore(t)
	adminPass, _ := seedUsers(t, s)
	h := newTestHandler(t, s)
	cookie, csrf := adminACLRulesSession(t, h, adminPass)

	admin, err := s.GetUserByUsername(context.Background(), "admin")
	if err != nil {
		t.Fatal(err)
	}

	body := url.Values{
		"csrf_token":   {csrf},
		"subject_kind": {"user"},
		"subject_id":   {admin.ID},
		"pattern":      {"app/*"},
		"action":       {"pull"},
	}
	req := httptest.NewRequest(http.MethodPost, "/admin/acl", strings.NewReader(body.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(cookie)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusSeeOther {
		t.Fatalf("create acl status=%d body=%s", rec.Code, rec.Body.String())
	}

	rules, err := s.ListACLRules(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(rules) != 1 {
		t.Fatalf("expected 1 rule, got %d", len(rules))
	}
	ruleID := rules[0].ID

	body = url.Values{"csrf_token": {csrf}}
	req = httptest.NewRequest(http.MethodPost, "/admin/acl/"+ruleID+"/delete", strings.NewReader(body.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(cookie)
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusSeeOther {
		t.Fatalf("delete acl status=%d body=%s", rec.Code, rec.Body.String())
	}

	rules, err = s.ListACLRules(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(rules) != 0 {
		t.Fatalf("expected 0 rules after delete, got %d", len(rules))
	}
}
