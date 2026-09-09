package bootstrap

import (
	"context"
	"errors"
	"fmt"

	"github.com/danielzelisko/private-registry/internal/auth"
	"github.com/danielzelisko/private-registry/internal/store"
)

var ErrAdminCredentialsRequired = errors.New("ADMIN_USER and ADMIN_PASSWORD are required when the database has no users")

func EnsureAdmin(ctx context.Context, s store.Store, adminUser, adminPassword string) error {
	count, err := s.CountUsers(ctx)
	if err != nil {
		return fmt.Errorf("count users: %w", err)
	}
	if count > 0 {
		return nil
	}
	if adminUser == "" || adminPassword == "" {
		return ErrAdminCredentialsRequired
	}
	hash, err := auth.HashPassword(adminPassword)
	if err != nil {
		return fmt.Errorf("hash admin password: %w", err)
	}
	_, err = s.CreateUser(ctx, adminUser, hash, true)
	if err != nil {
		return fmt.Errorf("create admin user: %w", err)
	}
	return nil
}
