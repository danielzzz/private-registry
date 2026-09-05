package bootstrap_test

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/danielzelisko/private-registry/internal/auth"
	"github.com/danielzelisko/private-registry/internal/bootstrap"
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

func TestEnsureAdminRequiresCredentialsOnEmptyDB(t *testing.T) {
	s := openTestDB(t)
	err := bootstrap.EnsureAdmin(context.Background(), s, "", "")
	if err == nil {
		t.Fatal("expected error when admin credentials missing on empty DB")
	}
}

func TestEnsureAdminCreatesAdminWhenCredentialsSet(t *testing.T) {
	s := openTestDB(t)
	ctx := context.Background()
	if err := bootstrap.EnsureAdmin(ctx, s, "admin", "secret"); err != nil {
		t.Fatal(err)
	}
	user, err := s.GetUserByUsername(ctx, "admin")
	if err != nil {
		t.Fatal(err)
	}
	if !user.Admin || !user.Active {
		t.Fatalf("user=%+v", user)
	}
	if !auth.CheckPassword(user.PasswordHash, "secret") {
		t.Fatal("password hash does not match")
	}
}

func TestEnsureAdminNoopWhenUsersExist(t *testing.T) {
	s := openTestDB(t)
	ctx := context.Background()
	_, err := s.CreateUser(ctx, "existing", "hash", true)
	if err != nil {
		t.Fatal(err)
	}
	if err := bootstrap.EnsureAdmin(ctx, s, "", ""); err != nil {
		t.Fatal(err)
	}
}
