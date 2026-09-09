package sqlite_test

import (
	"context"
	"testing"

	"github.com/danielzelisko/private-registry/internal/acl"
	"github.com/danielzelisko/private-registry/internal/store"
)

func TestCreateAndListACLRules(t *testing.T) {
	s := openTestDB(t)
	ctx := context.Background()

	u, err := s.CreateUser(ctx, "alice", "hash", false)
	if err != nil {
		t.Fatal(err)
	}

	created, err := s.CreateACLRule(ctx, store.ACLRuleInput{
		SubjectKind: acl.SubjectUser,
		SubjectID:   u.ID,
		Pattern:     "library/*",
		Action:      acl.ActionPull,
	})
	if err != nil {
		t.Fatal(err)
	}
	if created.ID == "" {
		t.Fatal("expected rule id")
	}

	rules, err := s.ListACLRules(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(rules) != 1 {
		t.Fatalf("rules=%+v", rules)
	}
	if rules[0].ID != created.ID || rules[0].Pattern != "library/*" {
		t.Fatalf("rules=%+v", rules)
	}
}

func TestCreateAnonymousACLRule(t *testing.T) {
	s := openTestDB(t)
	ctx := context.Background()

	created, err := s.CreateACLRule(ctx, store.ACLRuleInput{
		SubjectKind: acl.SubjectAnonymous,
		Pattern:     "public/*",
		Action:      acl.ActionPull,
	})
	if err != nil {
		t.Fatal(err)
	}

	rules, err := s.ListACLRules(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(rules) != 1 || rules[0].SubjectKind != acl.SubjectAnonymous || rules[0].SubjectID != "" {
		t.Fatalf("rules=%+v", rules)
	}
	if created.SubjectID != "" {
		t.Fatalf("created=%+v", created)
	}
}

func TestUpdateACLRule(t *testing.T) {
	s := openTestDB(t)
	ctx := context.Background()

	created, err := s.CreateACLRule(ctx, store.ACLRuleInput{
		SubjectKind: acl.SubjectAnonymous,
		Pattern:     "public/*",
		Action:      acl.ActionPull,
	})
	if err != nil {
		t.Fatal(err)
	}

	updated := created
	updated.Pattern = "public/**"
	updated.Action = acl.ActionPush
	if err := s.UpdateACLRule(ctx, updated); err != nil {
		t.Fatal(err)
	}

	rules, err := s.ListACLRules(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(rules) != 1 || rules[0].Pattern != "public/**" || rules[0].Action != acl.ActionPush {
		t.Fatalf("rules=%+v", rules)
	}
}

func TestDeleteACLRule(t *testing.T) {
	s := openTestDB(t)
	ctx := context.Background()

	created, err := s.CreateACLRule(ctx, store.ACLRuleInput{
		SubjectKind: acl.SubjectAnonymous,
		Pattern:     "public/*",
		Action:      acl.ActionPull,
	})
	if err != nil {
		t.Fatal(err)
	}

	if err := s.DeleteACLRule(ctx, created.ID); err != nil {
		t.Fatal(err)
	}

	rules, err := s.ListACLRules(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(rules) != 0 {
		t.Fatalf("rules=%+v", rules)
	}
}
