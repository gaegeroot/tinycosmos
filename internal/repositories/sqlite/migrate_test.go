package sqlite

import (
	"context"
	"database/sql"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/gaegeroot/tinycosmos/database"
)

func openEmpty(t *testing.T) *sql.DB {
	t.Helper()
	db, err := Open(context.Background(), filepath.Join(t.TempDir(), "m.db"))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}

func TestMigrateEmbeddedMigrations(t *testing.T) {
	ctx := context.Background()
	db := openEmpty(t)

	v, err := Migrate(ctx, db, database.Migrations())
	if err != nil {
		t.Fatalf("Migrate: %v", err)
	}
	if v < 1 {
		t.Fatalf("version = %d, want >= 1", v)
	}
	for _, table := range []string{"users", "user_identities", "groups", "group_integrations", "group_memberships"} {
		var n int
		if err := db.QueryRow(`SELECT count(*) FROM sqlite_schema WHERE type='table' AND name=?`, table).Scan(&n); err != nil || n != 1 {
			t.Errorf("table %s missing (n=%d, err=%v)", table, n, err)
		}
	}

	// Running again is a no-op.
	v2, err := Migrate(ctx, db, database.Migrations())
	if err != nil || v2 != v {
		t.Fatalf("second Migrate = %d, %v; want %d, nil", v2, err, v)
	}
}

func TestMigrateAppliesOnlyNewVersionsInOrder(t *testing.T) {
	ctx := context.Background()
	db := openEmpty(t)

	fsys := fstest.MapFS{
		"001_a.sql": {Data: []byte(`CREATE TABLE a (id INTEGER);`)},
	}
	if v, err := Migrate(ctx, db, fsys); err != nil || v != 1 {
		t.Fatalf("first Migrate = %d, %v", v, err)
	}

	// 001 must not re-run (it would fail: table a exists); 002 depends on a.
	fsys["002_b.sql"] = &fstest.MapFile{Data: []byte(`CREATE TABLE b (a_id INTEGER REFERENCES a(id));`)}
	fsys["README.md"] = &fstest.MapFile{Data: []byte("ignored")}
	if v, err := Migrate(ctx, db, fsys); err != nil || v != 2 {
		t.Fatalf("second Migrate = %d, %v; want 2, nil", v, err)
	}
}

func TestMigrateFailedScriptRollsBack(t *testing.T) {
	ctx := context.Background()
	db := openEmpty(t)

	fsys := fstest.MapFS{
		"001_ok.sql":  {Data: []byte(`CREATE TABLE a (id INTEGER);`)},
		"002_bad.sql": {Data: []byte(`CREATE TABLE b (id INTEGER); THIS IS NOT SQL;`)},
	}
	v, err := Migrate(ctx, db, fsys)
	if err == nil {
		t.Fatal("Migrate succeeded, want error")
	}
	if v != 1 {
		t.Fatalf("version after failure = %d, want 1", v)
	}
	var n int
	db.QueryRow(`SELECT count(*) FROM sqlite_schema WHERE name='b'`).Scan(&n)
	if n != 0 {
		t.Fatal("table b exists; failed migration was not rolled back")
	}
}

func TestMigrateRefusesUnversionedDatabaseWithTables(t *testing.T) {
	ctx := context.Background()
	db := openEmpty(t)
	if _, err := db.Exec(`CREATE TABLE users (id TEXT)`); err != nil {
		t.Fatal(err)
	}
	_, err := Migrate(ctx, db, database.Migrations())
	if err == nil || !strings.Contains(err.Error(), "no schema version") {
		t.Fatalf("Migrate = %v, want unversioned-database error", err)
	}
}

func TestMigrateRejectsBadFileNames(t *testing.T) {
	ctx := context.Background()
	for name, fsys := range map[string]fstest.MapFS{
		"no number":   {"init.sql": {Data: []byte(`SELECT 1;`)}},
		"duplicate":   {"001_a.sql": {Data: []byte(`SELECT 1;`)}, "001_b.sql": {Data: []byte(`SELECT 1;`)}},
		"none at all": {},
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := Migrate(ctx, openEmpty(t), fsys); err == nil {
				t.Fatal("Migrate succeeded, want error")
			}
		})
	}
}
