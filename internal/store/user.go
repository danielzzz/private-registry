package store

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"fmt"
)

type User struct {
	ID           string
	Username     string
	PasswordHash string
	Active       bool
	Admin        bool
}

func (s *Store) CreateUser(ctx context.Context, username, passwordHash string, admin bool) (User, error) {
	id, err := newID()
	if err != nil {
		return User{}, err
	}
	active := true
	_, err = s.db.ExecContext(ctx,
		`INSERT INTO users (id, username, password_hash, active, admin) VALUES (?, ?, ?, ?, ?)`,
		id, username, passwordHash, boolToInt(active), boolToInt(admin),
	)
	if err != nil {
		return User{}, fmt.Errorf("create user: %w", err)
	}
	return User{
		ID:           id,
		Username:     username,
		PasswordHash: passwordHash,
		Active:       active,
		Admin:        admin,
	}, nil
}

func (s *Store) GetUserByID(ctx context.Context, id string) (User, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT id, username, password_hash, active, admin FROM users WHERE id = ?`,
		id,
	)
	u, err := scanUser(row)
	if err != nil {
		return User{}, err
	}
	return u, nil
}

func (s *Store) GetUserByUsername(ctx context.Context, username string) (User, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT id, username, password_hash, active, admin FROM users WHERE username = ?`,
		username,
	)
	u, err := scanUser(row)
	if err != nil {
		return User{}, err
	}
	return u, nil
}

func (s *Store) CountUsers(ctx context.Context) (int, error) {
	var n int
	err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM users`).Scan(&n)
	if err != nil {
		return 0, fmt.Errorf("count users: %w", err)
	}
	return n, nil
}

func (s *Store) SetUserActive(ctx context.Context, id string, active bool) error {
	return s.updateUserField(ctx, id, "active", boolToInt(active))
}

func (s *Store) SetUserAdmin(ctx context.Context, id string, admin bool) error {
	return s.updateUserField(ctx, id, "admin", boolToInt(admin))
}

func (s *Store) UpdatePasswordHash(ctx context.Context, id string, passwordHash string) error {
	return s.updateUserField(ctx, id, "password_hash", passwordHash)
}

func (s *Store) ListUsers(ctx context.Context) ([]User, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, username, password_hash, active, admin FROM users ORDER BY username`,
	)
	if err != nil {
		return nil, fmt.Errorf("list users: %w", err)
	}
	defer rows.Close()

	var users []User
	for rows.Next() {
		u, err := scanUserRow(rows)
		if err != nil {
			return nil, err
		}
		users = append(users, u)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list users: %w", err)
	}
	return users, nil
}

func (s *Store) updateUserField(ctx context.Context, id, column string, value interface{}) error {
	query := fmt.Sprintf(`UPDATE users SET %s = ? WHERE id = ?`, column)
	res, err := s.db.ExecContext(ctx, query, value, id)
	if err != nil {
		return fmt.Errorf("update user %s: %w", column, err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("update user %s: %w", column, err)
	}
	if n == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func scanUser(row *sql.Row) (User, error) {
	var u User
	var active, admin int
	err := row.Scan(&u.ID, &u.Username, &u.PasswordHash, &active, &admin)
	if err != nil {
		if err == sql.ErrNoRows {
			return User{}, sql.ErrNoRows
		}
		return User{}, fmt.Errorf("scan user: %w", err)
	}
	u.Active = active != 0
	u.Admin = admin != 0
	return u, nil
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanUserRow(row rowScanner) (User, error) {
	var u User
	var active, admin int
	err := row.Scan(&u.ID, &u.Username, &u.PasswordHash, &active, &admin)
	if err != nil {
		return User{}, fmt.Errorf("scan user: %w", err)
	}
	u.Active = active != 0
	u.Admin = admin != 0
	return u, nil
}

func newID() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("generate id: %w", err)
	}
	return hex.EncodeToString(b), nil
}

func boolToInt(v bool) int {
	if v {
		return 1
	}
	return 0
}
