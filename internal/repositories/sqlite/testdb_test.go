package sqlite

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"sort"
	"testing"
)

// migrationsDir is relative to this package directory, which is the working
// directory during `go test`.
const migrationsDir = "../../../database/migrations"

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

	files, err := filepath.Glob(filepath.Join(migrationsDir, "*.sql"))
	if err != nil || len(files) == 0 {
		t.Fatalf("find migrations in %s: files=%v err=%v", migrationsDir, files, err)
	}
	sort.Strings(files)
	for _, f := range files {
		script, err := os.ReadFile(f)
		if err != nil {
			t.Fatalf("read migration %s: %v", f, err)
		}
		if _, err := db.ExecContext(ctx, string(script)); err != nil {
			t.Fatalf("apply migration %s: %v", f, err)
		}
	}
	return db
}
