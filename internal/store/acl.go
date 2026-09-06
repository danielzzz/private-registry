package store

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/danielzelisko/private-registry/internal/acl"
)

type ACLRule struct {
	ID          string
	SubjectKind acl.SubjectKind
	SubjectID   string
	Pattern     string
	Action      acl.Action
}

type ACLRuleInput struct {
	SubjectKind acl.SubjectKind
	SubjectID   string
	Pattern     string
	Action      acl.Action
}

func (s *Store) CreateACLRule(ctx context.Context, input ACLRuleInput) (ACLRule, error) {
	id, err := newID()
	if err != nil {
		return ACLRule{}, err
	}
	subjectID := sql.NullString{String: input.SubjectID, Valid: input.SubjectID != ""}
	_, err = s.db.ExecContext(ctx,
		`INSERT INTO acl_rules (id, subject_kind, subject_id, pattern, action) VALUES (?, ?, ?, ?, ?)`,
		id, string(input.SubjectKind), subjectID, input.Pattern, string(input.Action),
	)
	if err != nil {
		return ACLRule{}, fmt.Errorf("create acl rule: %w", err)
	}
	return ACLRule{
		ID:          id,
		SubjectKind: input.SubjectKind,
		SubjectID:   input.SubjectID,
		Pattern:     input.Pattern,
		Action:      input.Action,
	}, nil
}

func (s *Store) ListACLRules(ctx context.Context) ([]ACLRule, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, subject_kind, subject_id, pattern, action FROM acl_rules ORDER BY id`,
	)
	if err != nil {
		return nil, fmt.Errorf("list acl rules: %w", err)
	}
	defer rows.Close()

	var rules []ACLRule
	for rows.Next() {
		r, err := scanACLRule(rows)
		if err != nil {
			return nil, err
		}
		rules = append(rules, r)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list acl rules: %w", err)
	}
	return rules, nil
}

func (s *Store) CountACLRules(ctx context.Context) (int, error) {
	var n int
	err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM acl_rules`).Scan(&n)
	if err != nil {
		return 0, fmt.Errorf("count acl rules: %w", err)
	}
	return n, nil
}

func (s *Store) UpdateACLRule(ctx context.Context, rule ACLRule) error {
	subjectID := sql.NullString{String: rule.SubjectID, Valid: rule.SubjectID != ""}
	res, err := s.db.ExecContext(ctx,
		`UPDATE acl_rules SET subject_kind = ?, subject_id = ?, pattern = ?, action = ? WHERE id = ?`,
		string(rule.SubjectKind), subjectID, rule.Pattern, string(rule.Action), rule.ID,
	)
	if err != nil {
		return fmt.Errorf("update acl rule: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("update acl rule: %w", err)
	}
	if n == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func (s *Store) DeleteACLRule(ctx context.Context, id string) error {
	res, err := s.db.ExecContext(ctx, `DELETE FROM acl_rules WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("delete acl rule: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("delete acl rule: %w", err)
	}
	if n == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func scanACLRule(row rowScanner) (ACLRule, error) {
	var r ACLRule
	var subjectKind, action string
	var subjectID sql.NullString
	err := row.Scan(&r.ID, &subjectKind, &subjectID, &r.Pattern, &action)
	if err != nil {
		return ACLRule{}, fmt.Errorf("scan acl rule: %w", err)
	}
	r.SubjectKind = acl.SubjectKind(subjectKind)
	if subjectID.Valid {
		r.SubjectID = subjectID.String
	}
	r.Action = acl.Action(action)
	return r, nil
}
