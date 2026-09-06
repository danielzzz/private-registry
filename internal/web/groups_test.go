package web_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func adminGroupsSession(t *testing.T, h http.Handler, adminPass string) (cookie *http.Cookie, csrf string) {
	t.Helper()
	cookie = login(t, h, "admin", adminPass)

	req := httptest.NewRequest(http.MethodGet, "/admin/groups", nil)
	req.AddCookie(cookie)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("groups page status=%d body=%s", rec.Code, rec.Body.String())
	}
	csrf = extractCSRFToken(rec.Body.String())
	if csrf == "" {
		t.Fatal("expected csrf token on groups page")
	}
	return cookie, csrf
}

func TestGroupMembership(t *testing.T) {
	s := openTestStore(t)
	adminPass, _ := seedUsers(t, s)
	h := newTestHandler(t, s)
	cookie, csrf := adminGroupsSession(t, h, adminPass)

	target, err := s.GetUserByUsername(context.Background(), "user")
	if err != nil {
		t.Fatal(err)
	}

	body := url.Values{
		"csrf_token": {csrf},
		"name":       {"developers"},
	}
	req := httptest.NewRequest(http.MethodPost, "/admin/groups", strings.NewReader(body.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(cookie)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusSeeOther {
		t.Fatalf("create group status=%d body=%s", rec.Code, rec.Body.String())
	}

	groups, err := s.ListGroups(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(groups) != 1 || groups[0].Name != "developers" {
		t.Fatalf("groups=%+v", groups)
	}
	groupID := groups[0].ID

	body = url.Values{
		"csrf_token": {csrf},
		"user_id":    {target.ID},
	}
	req = httptest.NewRequest(http.MethodPost, "/admin/groups/"+groupID+"/add-member", strings.NewReader(body.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(cookie)
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusSeeOther {
		t.Fatalf("add member status=%d body=%s", rec.Code, rec.Body.String())
	}

	ids, err := s.ListGroupIDsForUser(context.Background(), target.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(ids) != 1 || ids[0] != groupID {
		t.Fatalf("expected user in group, ids=%v", ids)
	}

	body = url.Values{
		"csrf_token": {csrf},
		"user_id":    {target.ID},
	}
	req = httptest.NewRequest(http.MethodPost, "/admin/groups/"+groupID+"/remove-member", strings.NewReader(body.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(cookie)
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusSeeOther {
		t.Fatalf("remove member status=%d body=%s", rec.Code, rec.Body.String())
	}

	ids, err = s.ListGroupIDsForUser(context.Background(), target.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(ids) != 0 {
		t.Fatalf("expected user removed from group, ids=%v", ids)
	}
}

func TestGroupsPageRemoveMemberFormURL(t *testing.T) {
	s := openTestStore(t)
	adminPass, _ := seedUsers(t, s)
	h := newTestHandler(t, s)
	cookie, csrf := adminGroupsSession(t, h, adminPass)

	target, err := s.GetUserByUsername(context.Background(), "user")
	if err != nil {
		t.Fatal(err)
	}

	body := url.Values{
		"csrf_token": {csrf},
		"name":       {"developers"},
	}
	req := httptest.NewRequest(http.MethodPost, "/admin/groups", strings.NewReader(body.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(cookie)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusSeeOther {
		t.Fatalf("create group status=%d body=%s", rec.Code, rec.Body.String())
	}

	groups, err := s.ListGroups(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	groupID := groups[0].ID

	body = url.Values{
		"csrf_token": {csrf},
		"user_id":    {target.ID},
	}
	req = httptest.NewRequest(http.MethodPost, "/admin/groups/"+groupID+"/add-member", strings.NewReader(body.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(cookie)
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusSeeOther {
		t.Fatalf("add member status=%d body=%s", rec.Code, rec.Body.String())
	}

	req = httptest.NewRequest(http.MethodGet, "/admin/groups", nil)
	req.AddCookie(cookie)
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("groups page status=%d body=%s", rec.Code, rec.Body.String())
	}
	expected := "/admin/groups/" + groupID + "/remove-member"
	if !strings.Contains(rec.Body.String(), expected) {
		t.Fatalf("expected remove-member form action %q in page body", expected)
	}
}
