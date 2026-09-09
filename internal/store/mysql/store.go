package mysql

import (
	"database/sql"
	"fmt"

	"github.com/go-sql-driver/mysql"
)

type Store struct {
	db *sql.DB
}

func Open(dsn string) (*Store, error) {
	cfg, err := mysql.ParseDSN(dsn)
	if err != nil {
		return nil, fmt.Errorf("parse mysql dsn: %w", err)
	}
	// Match SQLite: RowsAffected counts matched rows, not only changed rows (no-op UPDATEs).
	cfg.ClientFoundRows = true
	db, err := sql.Open("mysql", cfg.FormatDSN())
	if err != nil {
		return nil, fmt.Errorf("open mysql: %w", err)
	}
	if err := db.Ping(); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("ping mysql: %w", err)
	}
	s := &Store{db: db}
	if err := s.migrate(); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("migrate: %w", err)
	}
	return s, nil
}

func (s *Store) Close() error {
	return s.db.Close()
}

// SQL exposes the underlying database for one-shot migration tools.
func (s *Store) SQL() *sql.DB {
	return s.db
}
