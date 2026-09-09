package migratecopy

import (
	"database/sql"
	"fmt"
)

// Copy copies all auth data from src to dst in FK-safe order, preserving row IDs.
// dst must be empty (no users). All inserts run inside a single transaction on dst.
func Copy(src, dst *sql.DB) error {
	tx, err := dst.Begin()
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback()

	if err := copyUsers(src, tx); err != nil {
		return err
	}
	if err := copyGroups(src, tx); err != nil {
		return err
	}
	if err := copyGroupMembers(src, tx); err != nil {
		return err
	}
	if err := copyAPITokens(src, tx); err != nil {
		return err
	}
	if err := copyACLRules(src, tx); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit: %w", err)
	}
	return nil
}

func copyUsers(src *sql.DB, tx *sql.Tx) error {
	rows, err := src.Query(`SELECT id, username, password_hash, active, admin FROM users`)
	if err != nil {
		return fmt.Errorf("select users: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var id, username, passwordHash string
		var active, admin int
		if err := rows.Scan(&id, &username, &passwordHash, &active, &admin); err != nil {
			return fmt.Errorf("scan user: %w", err)
		}
		_, err := tx.Exec(
			`INSERT INTO users (id, username, password_hash, active, admin) VALUES (?, ?, ?, ?, ?)`,
			id, username, passwordHash, active, admin,
		)
		if err != nil {
			return fmt.Errorf("insert user %s: %w", id, err)
		}
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("iterate users: %w", err)
	}
	return nil
}

func copyGroups(src *sql.DB, tx *sql.Tx) error {
	rows, err := src.Query(`SELECT id, name FROM groups`)
	if err != nil {
		return fmt.Errorf("select groups: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var id, name string
		if err := rows.Scan(&id, &name); err != nil {
			return fmt.Errorf("scan group: %w", err)
		}
		_, err := tx.Exec(`INSERT INTO groups (id, name) VALUES (?, ?)`, id, name)
		if err != nil {
			return fmt.Errorf("insert group %s: %w", id, err)
		}
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("iterate groups: %w", err)
	}
	return nil
}

func copyGroupMembers(src *sql.DB, tx *sql.Tx) error {
	rows, err := src.Query(`SELECT group_id, user_id FROM group_members`)
	if err != nil {
		return fmt.Errorf("select group_members: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var groupID, userID string
		if err := rows.Scan(&groupID, &userID); err != nil {
			return fmt.Errorf("scan group_member: %w", err)
		}
		_, err := tx.Exec(
			`INSERT INTO group_members (group_id, user_id) VALUES (?, ?)`,
			groupID, userID,
		)
		if err != nil {
			return fmt.Errorf("insert group_member %s/%s: %w", groupID, userID, err)
		}
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("iterate group_members: %w", err)
	}
	return nil
}

func copyAPITokens(src *sql.DB, tx *sql.Tx) error {
	rows, err := src.Query(`SELECT id, user_id, name, token_hash, expires_at, active FROM api_tokens`)
	if err != nil {
		return fmt.Errorf("select api_tokens: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var id, userID, name, tokenHash string
		var expires sql.NullInt64
		var active int
		if err := rows.Scan(&id, &userID, &name, &tokenHash, &expires, &active); err != nil {
			return fmt.Errorf("scan api_token: %w", err)
		}
		_, err := tx.Exec(
			`INSERT INTO api_tokens (id, user_id, name, token_hash, expires_at, active) VALUES (?, ?, ?, ?, ?, ?)`,
			id, userID, name, tokenHash, expires, active,
		)
		if err != nil {
			return fmt.Errorf("insert api_token %s: %w", id, err)
		}
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("iterate api_tokens: %w", err)
	}
	return nil
}

func copyACLRules(src *sql.DB, tx *sql.Tx) error {
	rows, err := src.Query(`SELECT id, subject_kind, subject_id, pattern, action FROM acl_rules`)
	if err != nil {
		return fmt.Errorf("select acl_rules: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var id, subjectKind, pattern, action string
		var subjectID sql.NullString
		if err := rows.Scan(&id, &subjectKind, &subjectID, &pattern, &action); err != nil {
			return fmt.Errorf("scan acl_rule: %w", err)
		}
		_, err := tx.Exec(
			`INSERT INTO acl_rules (id, subject_kind, subject_id, pattern, action) VALUES (?, ?, ?, ?, ?)`,
			id, subjectKind, subjectID, pattern, action,
		)
		if err != nil {
			return fmt.Errorf("insert acl_rule %s: %w", id, err)
		}
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("iterate acl_rules: %w", err)
	}
	return nil
}
