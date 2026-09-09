package sqlite_test

import (
	"context"
	"testing"
)

func TestCreateGroupAndListGroupIDsForUser(t *testing.T) {
	s := openTestDB(t)
	ctx := context.Background()

	u, err := s.CreateUser(ctx, "alice", "hash", false)
	if err != nil {
		t.Fatal(err)
	}

	g, err := s.CreateGroup(ctx, "developers")
	if err != nil {
		t.Fatal(err)
	}
	if g.Name != "developers" || g.ID == "" {
		t.Fatalf("group=%+v", g)
	}

	if err := s.AddMember(ctx, g.ID, u.ID); err != nil {
		t.Fatal(err)
	}

	ids, err := s.ListGroupIDsForUser(ctx, u.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(ids) != 1 || ids[0] != g.ID {
		t.Fatalf("ids=%v", ids)
	}
}

func TestRemoveMember(t *testing.T) {
	s := openTestDB(t)
	ctx := context.Background()

	u, err := s.CreateUser(ctx, "alice", "hash", false)
	if err != nil {
		t.Fatal(err)
	}
	g, err := s.CreateGroup(ctx, "developers")
	if err != nil {
		t.Fatal(err)
	}
	if err := s.AddMember(ctx, g.ID, u.ID); err != nil {
		t.Fatal(err)
	}
	if err := s.RemoveMember(ctx, g.ID, u.ID); err != nil {
		t.Fatal(err)
	}

	ids, err := s.ListGroupIDsForUser(ctx, u.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(ids) != 0 {
		t.Fatalf("ids=%v", ids)
	}
}

func TestListGroups(t *testing.T) {
	s := openTestDB(t)
	ctx := context.Background()

	_, err := s.CreateGroup(ctx, "beta")
	if err != nil {
		t.Fatal(err)
	}
	_, err = s.CreateGroup(ctx, "alpha")
	if err != nil {
		t.Fatal(err)
	}

	groups, err := s.ListGroups(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(groups) != 2 || groups[0].Name != "alpha" || groups[1].Name != "beta" {
		t.Fatalf("groups=%+v", groups)
	}
}

func TestListGroupIDsForUserEmpty(t *testing.T) {
	s := openTestDB(t)
	ctx := context.Background()

	u, err := s.CreateUser(ctx, "alice", "hash", false)
	if err != nil {
		t.Fatal(err)
	}

	ids, err := s.ListGroupIDsForUser(ctx, u.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(ids) != 0 {
		t.Fatalf("ids=%v", ids)
	}
}
