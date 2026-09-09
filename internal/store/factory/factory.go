package factory

import (
	"fmt"

	"github.com/danielzzz/private-registry/internal/store"
	"github.com/danielzzz/private-registry/internal/store/mysql"
	"github.com/danielzzz/private-registry/internal/store/sqlite"
)

func Open(driver, path, dsn string) (store.Store, error) {
	switch driver {
	case "sqlite":
		return sqlite.Open(path)
	case "mysql":
		return mysql.Open(dsn)
	default:
		return nil, fmt.Errorf("unknown database driver %q", driver)
	}
}
