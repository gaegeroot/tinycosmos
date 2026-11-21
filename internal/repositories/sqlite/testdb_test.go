package sqlite

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"

	"github.com/gaegeroot/tinycosmos/database"
)

// newTestDB returns a fresh, fully migrated SQLite database stored in a
// per-test temporary directory. It is closed and removed when the test ends.
func newTestDB(t *testing.T) *sql.DB {
	t.Helper()
	ctx := context.Background()

	db, err := Open(ctx, filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	if _, err := Migrate(ctx, db, database.Migrations()); err != nil {
		t.Fatalf("migrate test db: %v", err)
	}
	return db
}
