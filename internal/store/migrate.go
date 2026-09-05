package store

const usersSchema = `
CREATE TABLE IF NOT EXISTS users (
  id TEXT PRIMARY KEY,
  username TEXT NOT NULL UNIQUE COLLATE NOCASE,
  password_hash TEXT NOT NULL,
  active INTEGER NOT NULL DEFAULT 1,
  admin INTEGER NOT NULL DEFAULT 0
);
`

func (s *Store) migrate() error {
	_, err := s.db.Exec(usersSchema)
	return err
}
