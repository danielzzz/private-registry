package mysql

const schema = `
CREATE TABLE IF NOT EXISTS users (
  id VARCHAR(64) PRIMARY KEY,
  username VARCHAR(255) NOT NULL,
  password_hash TEXT NOT NULL,
  active TINYINT(1) NOT NULL DEFAULT 1,
  admin TINYINT(1) NOT NULL DEFAULT 0,
  UNIQUE KEY users_username (username)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS api_tokens (
  id VARCHAR(64) PRIMARY KEY,
  user_id VARCHAR(64) NOT NULL,
  name VARCHAR(255) NOT NULL,
  token_hash TEXT NOT NULL,
  expires_at BIGINT NULL,
  active TINYINT(1) NOT NULL DEFAULT 1,
  CONSTRAINT fk_api_tokens_user FOREIGN KEY (user_id) REFERENCES users(id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS groups (
  id VARCHAR(64) PRIMARY KEY,
  name VARCHAR(255) NOT NULL,
  UNIQUE KEY groups_name (name)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS group_members (
  group_id VARCHAR(64) NOT NULL,
  user_id VARCHAR(64) NOT NULL,
  PRIMARY KEY (group_id, user_id),
  CONSTRAINT fk_gm_group FOREIGN KEY (group_id) REFERENCES groups(id),
  CONSTRAINT fk_gm_user FOREIGN KEY (user_id) REFERENCES users(id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS acl_rules (
  id VARCHAR(64) PRIMARY KEY,
  subject_kind VARCHAR(64) NOT NULL,
  subject_id VARCHAR(64) NULL,
  pattern TEXT NOT NULL,
  action VARCHAR(64) NOT NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
`

func (s *Store) migrate() error {
	_, err := s.db.Exec(schema)
	return err
}
