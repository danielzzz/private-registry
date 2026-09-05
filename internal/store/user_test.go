package store_test

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"

	"github.com/danielzelisko/private-registry/internal/store"
)

func openTestDB(t *testing.T) *store.Store {
	t.Helper()
	path := filepath.Join(t.TempDir(), "test.db")
	s, err := store.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = s.Close() })
	return s
}

func TestCreateAndGetUser(t *testing.T) {
	s := openTestDB(t)
	u, err := s.CreateUser(context.Background(), "alice", "hash", true)
	if err != nil {
		t.Fatal(err)
	}
	got, err := s.GetUserByUsername(context.Background(), "alice")
	if err != nil || got.ID != u.ID || !got.Admin || !got.Active {
		t.Fatalf("got=%+v err=%v", got, err)
	}
}

func TestCountUsers(t *testing.T) {
	s := openTestDB(t)
	ctx := context.Background()
	if n, err := s.CountUsers(ctx); err != nil || n != 0 {
		t.Fatalf("initial count=%d err=%v", n, err)
	}
	_, err := s.CreateUser(ctx, "alice", "hash", false)
	if err != nil {
		t.Fatal(err)
	}
	if n, err := s.CountUsers(ctx); err != nil || n != 1 {
		t.Fatalf("count=%d err=%v", n, err)
	}
}

func TestSetUserActive(t *testing.T) {
	s := openTestDB(t)
	ctx := context.Background()
	u, err := s.CreateUser(ctx, "alice", "hash", false)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.SetUserActive(ctx, u.ID, false); err != nil {
		t.Fatal(err)
	}
	got, err := s.GetUserByUsername(ctx, "alice")
	if err != nil || got.Active {
		t.Fatalf("got=%+v err=%v", got, err)
	}
}

func TestSetUserAdmin(t *testing.T) {
	s := openTestDB(t)
	ctx := context.Background()
	u, err := s.CreateUser(ctx, "alice", "hash", false)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.SetUserAdmin(ctx, u.ID, true); err != nil {
		t.Fatal(err)
	}
	got, err := s.GetUserByUsername(ctx, "alice")
	if err != nil || !got.Admin {
		t.Fatalf("got=%+v err=%v", got, err)
	}
}

func TestUpdatePasswordHash(t *testing.T) {
	s := openTestDB(t)
	ctx := context.Background()
	u, err := s.CreateUser(ctx, "alice", "hash", false)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.UpdatePasswordHash(ctx, u.ID, "newhash"); err != nil {
		t.Fatal(err)
	}
	got, err := s.GetUserByUsername(ctx, "alice")
	if err != nil || got.PasswordHash != "newhash" {
		t.Fatalf("got=%+v err=%v", got, err)
	}
}

func TestListUsers(t *testing.T) {
	s := openTestDB(t)
	ctx := context.Background()
	_, err := s.CreateUser(ctx, "bob", "hash", false)
	if err != nil {
		t.Fatal(err)
	}
	_, err = s.CreateUser(ctx, "alice", "hash", true)
	if err != nil {
		t.Fatal(err)
	}
	users, err := s.ListUsers(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(users) != 2 || users[0].Username != "alice" || users[1].Username != "bob" {
		t.Fatalf("users=%+v", users)
	}
}

func TestGetUserByUsernameCaseInsensitive(t *testing.T) {
	s := openTestDB(t)
	ctx := context.Background()
	_, err := s.CreateUser(ctx, "alice", "hash", false)
	if err != nil {
		t.Fatal(err)
	}
	got, err := s.GetUserByUsername(ctx, "ALICE")
	if err != nil {
		t.Fatal(err)
	}
	if got.Username != "alice" {
		t.Fatalf("username=%q", got.Username)
	}
}

func TestGetUserByUsernameNotFound(t *testing.T) {
	s := openTestDB(t)
	_, err := s.GetUserByUsername(context.Background(), "missing")
	if err != sql.ErrNoRows {
		t.Fatalf("err=%v", err)
	}
}

func TestSetUserActiveNotFound(t *testing.T) {
	s := openTestDB(t)
	err := s.SetUserActive(context.Background(), "missing-id", false)
	if err != sql.ErrNoRows {
		t.Fatalf("err=%v", err)
	}
}

func TestSetUserAdminNotFound(t *testing.T) {
	s := openTestDB(t)
	err := s.SetUserAdmin(context.Background(), "missing-id", true)
	if err != sql.ErrNoRows {
		t.Fatalf("err=%v", err)
	}
}

func TestUpdatePasswordHashNotFound(t *testing.T) {
	s := openTestDB(t)
	err := s.UpdatePasswordHash(context.Background(), "missing-id", "newhash")
	if err != sql.ErrNoRows {
		t.Fatalf("err=%v", err)
	}
}
