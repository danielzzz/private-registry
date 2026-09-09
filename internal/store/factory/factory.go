package factory

import (
	"fmt"

	"github.com/danielzelisko/private-registry/internal/store"
	"github.com/danielzelisko/private-registry/internal/store/sqlite"
)

func Open(driver, path, dsn string) (store.Store, error) {
	switch driver {
	case "sqlite":
		return sqlite.Open(path)
	case "mysql":
		return nil, fmt.Errorf("mysql backend not implemented yet")
	default:
		return nil, fmt.Errorf("unknown database driver %q", driver)
	}
}
