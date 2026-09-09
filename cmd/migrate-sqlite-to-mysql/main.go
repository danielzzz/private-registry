// Command migrate-sqlite-to-mysql copies auth data from SQLite to an empty MySQL database.
package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/danielzelisko/private-registry/internal/store/migratecopy"
	"github.com/danielzelisko/private-registry/internal/store/mysql"
	"github.com/danielzelisko/private-registry/internal/store/sqlite"
)

const usageHint = `Copy auth data from SQLite to MySQL with preserved IDs.

Requires an empty MySQL database (no users). If migration fails after a
partial commit, truncate all auth tables on MySQL and retry.`

func main() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [flags]\n\n%s\n\n", os.Args[0], usageHint)
		flag.PrintDefaults()
	}
	sqlitePath := flag.String("sqlite", "", "path to source SQLite database")
	mysqlDSN := flag.String("mysql", "", "destination MySQL DSN")
	flag.Parse()

	if *sqlitePath == "" || *mysqlDSN == "" {
		flag.Usage()
		os.Exit(2)
	}

	if err := run(*sqlitePath, *mysqlDSN); err != nil {
		log.Fatalf("migrate-sqlite-to-mysql: %v", err)
	}
	fmt.Println("ok")
}

func run(sqlitePath, mysqlDSN string) error {
	srcStore, err := sqlite.Open(sqlitePath)
	if err != nil {
		return err
	}
	defer srcStore.Close()

	dstStore, err := mysql.Open(mysqlDSN)
	if err != nil {
		return err
	}
	defer dstStore.Close()

	var userCount int
	if err := dstStore.SQL().QueryRow(`SELECT COUNT(*) FROM users`).Scan(&userCount); err != nil {
		return fmt.Errorf("count mysql users: %w", err)
	}
	if userCount > 0 {
		return fmt.Errorf("mysql database already has %d user(s); aborting to avoid overwriting data", userCount)
	}

	if err := migratecopy.Copy(srcStore.SQL(), dstStore.SQL()); err != nil {
		return err
	}
	return nil
}

func init() {
	log.SetOutput(os.Stderr)
	log.SetFlags(0)
}
