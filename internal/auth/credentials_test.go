package auth_test

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"path/filepath"
	"testing"

	"github.com/danielzelisko/private-registry/internal/auth"
	"github.com/danielzelisko/private-registry/internal/store"
)

func openTestStore(t *testing.T) *store.Store {
	t.Helper()
	path := filepath.Join(t.TempDir(), "test.db")
	s, err := store.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = s.Close() })
	return s
}

func randomHexSecret(t *testing.T) string {
	t.Helper()
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		t.Fatal(err)
	}
	return hex.EncodeToString(b)
}

func TestAuthenticateWithPassword(t *testing.T) {
	s := openTestStore(t)
	ctx := context.Background()

	hash, err := auth.HashPassword("s3cr3t")
	if err != nil {
		t.Fatal(err)
	}
	_, err = s.CreateUser(ctx, "alice", hash, false)
	if err != nil {
		t.Fatal(err)
	}

	a := &auth.Authenticator{Store: s}
	u, err := a.Authenticate(ctx, "alice", "s3cr3t")
	if err != nil {
		t.Fatal(err)
	}
	if u.Username != "alice" || !u.Active {
		t.Fatalf("user=%+v", u)
	}
}

func TestAuthenticateWithAPIToken(t *testing.T) {
	s := openTestStore(t)
	ctx := context.Background()

	u, err := s.CreateUser(ctx, "alice", "unused", false)
	if err != nil {
		t.Fatal(err)
	}

	secretPart := randomHexSecret(t)
	tokenHash, err := auth.HashPassword(secretPart)
	if err != nil {
		t.Fatal(err)
	}
	tokenID, err := s.CreateAPIToken(ctx, u.ID, "ci", tokenHash, nil)
	if err != nil {
		t.Fatal(err)
	}
	plaintext := "prt_" + tokenID + "_" + secretPart

	a := &auth.Authenticator{Store: s}
	got, err := a.Authenticate(ctx, "alice", plaintext)
	if err != nil {
		t.Fatal(err)
	}
	if got.ID != u.ID {
		t.Fatalf("got=%+v want id=%s", got, u.ID)
	}
}

func TestAuthenticateRejectsBadSecret(t *testing.T) {
	s := openTestStore(t)
	ctx := context.Background()

	hash, err := auth.HashPassword("s3cr3t")
	if err != nil {
		t.Fatal(err)
	}
	_, err = s.CreateUser(ctx, "alice", hash, false)
	if err != nil {
		t.Fatal(err)
	}

	a := &auth.Authenticator{Store: s}
	_, err = a.Authenticate(ctx, "alice", "wrong")
	if !errors.Is(err, auth.ErrInvalidCredentials) {
		t.Fatalf("err=%v", err)
	}
}

func TestAuthenticateRejectsInactiveUser(t *testing.T) {
	s := openTestStore(t)
	ctx := context.Background()

	hash, err := auth.HashPassword("s3cr3t")
	if err != nil {
		t.Fatal(err)
	}
	u, err := s.CreateUser(ctx, "alice", hash, false)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.SetUserActive(ctx, u.ID, false); err != nil {
		t.Fatal(err)
	}

	a := &auth.Authenticator{Store: s}
	_, err = a.Authenticate(ctx, "alice", "s3cr3t")
	if !errors.Is(err, auth.ErrUserInactive) {
		t.Fatalf("err=%v", err)
	}
}
