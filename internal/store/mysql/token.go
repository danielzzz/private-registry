package mysql

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/danielzelisko/private-registry/internal/store"
)

func (s *Store) CreateAPIToken(ctx context.Context, userID, name, hash string, expiresAt *time.Time) (string, error) {
	id, err := store.NewID()
	if err != nil {
		return "", err
	}
	var expires sql.NullInt64
	if expiresAt != nil {
		expires = sql.NullInt64{Int64: expiresAt.Unix(), Valid: true}
	}
	_, err = s.db.ExecContext(ctx,
		`INSERT INTO api_tokens (id, user_id, name, token_hash, expires_at, active) VALUES (?, ?, ?, ?, ?, 1)`,
		id, userID, name, hash, expires,
	)
	if err != nil {
		return "", fmt.Errorf("create api token: %w", err)
	}
	return id, nil
}

func (s *Store) ListAPITokensByUser(ctx context.Context, userID string) ([]store.APIToken, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, user_id, name, token_hash, expires_at, active FROM api_tokens WHERE user_id = ? ORDER BY name`,
		userID,
	)
	if err != nil {
		return nil, fmt.Errorf("list api tokens: %w", err)
	}
	defer rows.Close()

	var tokens []store.APIToken
	for rows.Next() {
		t, err := scanAPIToken(rows)
		if err != nil {
			return nil, err
		}
		tokens = append(tokens, t)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list api tokens: %w", err)
	}
	return tokens, nil
}

func (s *Store) CountAPITokens(ctx context.Context) (int, error) {
	var n int
	err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM api_tokens`).Scan(&n)
	if err != nil {
		return 0, fmt.Errorf("count api tokens: %w", err)
	}
	return n, nil
}

func (s *Store) RevokeAPIToken(ctx context.Context, id string) error {
	res, err := s.db.ExecContext(ctx, `UPDATE api_tokens SET active = 0 WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("revoke api token: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("revoke api token: %w", err)
	}
	if n == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func scanAPIToken(row rowScanner) (store.APIToken, error) {
	var t store.APIToken
	var expires sql.NullInt64
	var active int
	err := row.Scan(&t.ID, &t.UserID, &t.Name, &t.TokenHash, &expires, &active)
	if err != nil {
		return store.APIToken{}, fmt.Errorf("scan api token: %w", err)
	}
	t.Active = active != 0
	if expires.Valid {
		ts := time.Unix(expires.Int64, 0).UTC()
		t.ExpiresAt = &ts
	}
	return t, nil
}
