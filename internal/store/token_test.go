package store_test

import (
	"context"
	"testing"
	"time"
)

func TestCreateAndListAPITokens(t *testing.T) {
	s := openTestDB(t)
	ctx := context.Background()

	u, err := s.CreateUser(ctx, "alice", "hash", false)
	if err != nil {
		t.Fatal(err)
	}

	expires := time.Now().Add(24 * time.Hour).UTC().Truncate(time.Second)
	id, err := s.CreateAPIToken(ctx, u.ID, "ci-token", "tokenhash", &expires)
	if err != nil {
		t.Fatal(err)
	}
	if id == "" {
		t.Fatal("expected token id")
	}

	tokens, err := s.ListAPITokensByUser(ctx, u.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(tokens) != 1 {
		t.Fatalf("tokens=%+v", tokens)
	}
	if tokens[0].ID != id || tokens[0].Name != "ci-token" || tokens[0].TokenHash != "tokenhash" {
		t.Fatalf("tokens=%+v", tokens)
	}
	if !tokens[0].Active {
		t.Fatalf("token not active: %+v", tokens[0])
	}
	if tokens[0].ExpiresAt == nil || !tokens[0].ExpiresAt.Equal(expires) {
		t.Fatalf("expires=%v want=%v", tokens[0].ExpiresAt, expires)
	}
}

func TestCreateAPITokenNoExpiry(t *testing.T) {
	s := openTestDB(t)
	ctx := context.Background()

	u, err := s.CreateUser(ctx, "alice", "hash", false)
	if err != nil {
		t.Fatal(err)
	}

	_, err = s.CreateAPIToken(ctx, u.ID, "permanent", "hash", nil)
	if err != nil {
		t.Fatal(err)
	}

	tokens, err := s.ListAPITokensByUser(ctx, u.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(tokens) != 1 || tokens[0].ExpiresAt != nil {
		t.Fatalf("tokens=%+v", tokens)
	}
}

func TestRevokeAPIToken(t *testing.T) {
	s := openTestDB(t)
	ctx := context.Background()

	u, err := s.CreateUser(ctx, "alice", "hash", false)
	if err != nil {
		t.Fatal(err)
	}

	id, err := s.CreateAPIToken(ctx, u.ID, "ci-token", "tokenhash", nil)
	if err != nil {
		t.Fatal(err)
	}

	if err := s.RevokeAPIToken(ctx, id); err != nil {
		t.Fatal(err)
	}

	tokens, err := s.ListAPITokensByUser(ctx, u.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(tokens) != 1 || tokens[0].Active {
		t.Fatalf("tokens=%+v", tokens)
	}
}
