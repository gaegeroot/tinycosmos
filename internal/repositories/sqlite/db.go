// Package sqlite implements Tiny Cosmos repositories on top of SQLite.
package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/url"

	_ "modernc.org/sqlite" // registers the "sqlite" database/sql driver
)

// driverName is the database/sql driver name registered by modernc.org/sqlite.
const driverName = "sqlite"

// Open opens the SQLite database at path and verifies that foreign-key
// enforcement is active.
//
// Foreign keys are enabled through the DSN so that the PRAGMA runs on every
// pooled connection, not only the first one. A busy timeout is also set so
// concurrent writers wait briefly instead of failing immediately with
// SQLITE_BUSY.
func Open(ctx context.Context, path string) (*sql.DB, error) {
	if path == "" {
		return nil, errors.New("sqlite: database path is required")
	}

	q := url.Values{}
	q.Add("_pragma", "foreign_keys(1)")
	q.Add("_pragma", "busy_timeout(5000)")
	dsn := "file:" + path + "?" + q.Encode()

	db, err := sql.Open(driverName, dsn)
	if err != nil {
		return nil, fmt.Errorf("sqlite: open %q: %w", path, err)
	}

	var fk int
	if err := db.QueryRowContext(ctx, "PRAGMA foreign_keys").Scan(&fk); err != nil {
		db.Close()
		return nil, fmt.Errorf("sqlite: check foreign_keys pragma: %w", err)
	}
	if fk != 1 {
		db.Close()
		return nil, errors.New("sqlite: foreign key enforcement is not enabled")
	}
	return db, nil
}
