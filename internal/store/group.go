package store

import (
	"context"
	"database/sql"
	"fmt"
)

type Group struct {
	ID   string
	Name string
}

func (s *Store) CreateGroup(ctx context.Context, name string) (Group, error) {
	id, err := newID()
	if err != nil {
		return Group{}, err
	}
	_, err = s.db.ExecContext(ctx,
		`INSERT INTO groups (id, name) VALUES (?, ?)`,
		id, name,
	)
	if err != nil {
		return Group{}, fmt.Errorf("create group: %w", err)
	}
	return Group{ID: id, Name: name}, nil
}

func (s *Store) AddMember(ctx context.Context, groupID, userID string) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO group_members (group_id, user_id) VALUES (?, ?)`,
		groupID, userID,
	)
	if err != nil {
		return fmt.Errorf("add member: %w", err)
	}
	return nil
}

func (s *Store) RemoveMember(ctx context.Context, groupID, userID string) error {
	res, err := s.db.ExecContext(ctx,
		`DELETE FROM group_members WHERE group_id = ? AND user_id = ?`,
		groupID, userID,
	)
	if err != nil {
		return fmt.Errorf("remove member: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("remove member: %w", err)
	}
	if n == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func (s *Store) ListGroups(ctx context.Context) ([]Group, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id, name FROM groups ORDER BY name`)
	if err != nil {
		return nil, fmt.Errorf("list groups: %w", err)
	}
	defer rows.Close()

	var groups []Group
	for rows.Next() {
		var g Group
		if err := rows.Scan(&g.ID, &g.Name); err != nil {
			return nil, fmt.Errorf("scan group: %w", err)
		}
		groups = append(groups, g)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list groups: %w", err)
	}
	return groups, nil
}

func (s *Store) CountGroups(ctx context.Context) (int, error) {
	var n int
	err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM groups`).Scan(&n)
	if err != nil {
		return 0, fmt.Errorf("count groups: %w", err)
	}
	return n, nil
}

func (s *Store) ListGroupIDsForUser(ctx context.Context, userID string) ([]string, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT group_id FROM group_members WHERE user_id = ? ORDER BY group_id`,
		userID,
	)
	if err != nil {
		return nil, fmt.Errorf("list group ids for user: %w", err)
	}
	defer rows.Close()

	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("scan group id: %w", err)
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list group ids for user: %w", err)
	}
	return ids, nil
}
