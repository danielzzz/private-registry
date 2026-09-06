package acl_test

import (
	"testing"

	"github.com/danielzelisko/private-registry/internal/acl"
)

func contains(actions []string, action string) bool {
	for _, a := range actions {
		if a == action {
			return true
		}
	}
	return false
}

func TestEvaluate_UserDirectPull(t *testing.T) {
	rules := []acl.Rule{{
		SubjectKind: acl.SubjectUser, SubjectID: "u1",
		Pattern: "danielzelisko/test-project*", Action: acl.ActionPull,
	}}
	id := acl.Identity{UserID: "u1"}
	req := []acl.Scope{{Type: "repository", Name: "danielzelisko/test-project-app", Actions: []string{"pull"}}}
	got := acl.Evaluate(id, rules, req)
	if len(got) != 1 || got[0].Name != req[0].Name || !contains(got[0].Actions, "pull") {
		t.Fatalf("got=%v", got)
	}
}

func TestEvaluate_PushImpliesPull(t *testing.T) {
	rules := []acl.Rule{{
		SubjectKind: acl.SubjectUser, SubjectID: "u1",
		Pattern: "app/*", Action: acl.ActionPush,
	}}
	id := acl.Identity{UserID: "u1"}
	req := []acl.Scope{{Type: "repository", Name: "app/web", Actions: []string{"push", "pull"}}}
	got := acl.Evaluate(id, rules, req)
	if len(got) != 1 || !contains(got[0].Actions, "push") || !contains(got[0].Actions, "pull") {
		t.Fatalf("got=%v", got)
	}
}

func TestEvaluate_PushRuleGrantsPullWhenClientRequestsPushOnly(t *testing.T) {
	rules := []acl.Rule{{
		SubjectKind: acl.SubjectUser, SubjectID: "u1",
		Pattern: "app/*", Action: acl.ActionPush,
	}}
	id := acl.Identity{UserID: "u1"}
	req := []acl.Scope{{Type: "repository", Name: "app/web", Actions: []string{"push"}}}
	got := acl.Evaluate(id, rules, req)
	if len(got) != 1 {
		t.Fatalf("got=%v", got)
	}
	if !contains(got[0].Actions, "push") || !contains(got[0].Actions, "pull") {
		t.Fatalf("expected both pull and push, got=%v", got[0].Actions)
	}
}

func TestEvaluate_PullRuleDoesNotGrantPush(t *testing.T) {
	rules := []acl.Rule{{
		SubjectKind: acl.SubjectUser, SubjectID: "u1",
		Pattern: "app/*", Action: acl.ActionPull,
	}}
	id := acl.Identity{UserID: "u1"}
	req := []acl.Scope{{Type: "repository", Name: "app/web", Actions: []string{"push", "pull"}}}
	got := acl.Evaluate(id, rules, req)
	if len(got) != 1 {
		t.Fatalf("got=%v", got)
	}
	if !contains(got[0].Actions, "pull") || contains(got[0].Actions, "push") {
		t.Fatalf("expected pull only, got=%v", got[0].Actions)
	}
}

func TestEvaluate_GroupAndAnonymous(t *testing.T) {
	rules := []acl.Rule{
		{SubjectKind: acl.SubjectGroup, SubjectID: "g1", Pattern: "team/*", Action: acl.ActionPull},
		{SubjectKind: acl.SubjectAnonymous, Pattern: "public/*", Action: acl.ActionPull},
	}
	anon := acl.Evaluate(acl.Identity{Anonymous: true}, rules, []acl.Scope{
		{Type: "repository", Name: "public/foo", Actions: []string{"pull"}},
		{Type: "repository", Name: "team/bar", Actions: []string{"pull"}},
	})
	if len(anon) != 1 || anon[0].Name != "public/foo" {
		t.Fatalf("anon=%v", anon)
	}
	member := acl.Evaluate(acl.Identity{UserID: "u1", GroupIDs: []string{"g1"}}, rules, []acl.Scope{
		{Type: "repository", Name: "team/bar", Actions: []string{"pull"}},
	})
	if len(member) != 1 {
		t.Fatalf("member=%v", member)
	}
}

func TestEvaluate_DenyByOmission(t *testing.T) {
	got := acl.Evaluate(acl.Identity{UserID: "u1"}, nil, []acl.Scope{
		{Type: "repository", Name: "x/y", Actions: []string{"pull"}},
	})
	if len(got) != 0 {
		t.Fatalf("got=%v", got)
	}
}

func TestEvaluate_AnonymousPushDenied(t *testing.T) {
	rules := []acl.Rule{{
		SubjectKind: acl.SubjectAnonymous,
		Pattern:     "public/*",
		Action:      acl.ActionPush,
	}}
	got := acl.Evaluate(acl.Identity{Anonymous: true}, rules, []acl.Scope{
		{Type: "repository", Name: "public/foo", Actions: []string{"push", "pull"}},
	})
	if len(got) != 0 {
		t.Fatalf("anonymous push should be denied, got=%v", got)
	}
}
