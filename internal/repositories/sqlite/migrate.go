package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io/fs"
	"path"
	"sort"
	"strconv"
	"strings"
)

// migration is one numbered SQL script.
type migration struct {
	version int
	name    string
	script  string
}

// Migrate applies every migration in fsys whose version is greater than the
// database's current schema version, in order, and returns the resulting
// version.
//
// Migration files must be named NNN_description.sql at the root of fsys. The
// applied version is tracked with SQLite's PRAGMA user_version. Each migration
// runs in its own transaction together with the version bump, so a failed
// script leaves the database at the previous version.
//
// Migrate refuses to run on a database that already has tables but no
// recorded version (for example, one created by hand with the sqlite3 CLI),
// because it cannot tell which migrations that database already contains.
func Migrate(ctx context.Context, db *sql.DB, fsys fs.FS) (int, error) {
	migrations, err := loadMigrations(fsys)
	if err != nil {
		return 0, err
	}

	current, err := schemaVersion(ctx, db)
	if err != nil {
		return 0, err
	}
	if current == 0 {
		var tables int
		err := db.QueryRowContext(ctx,
			`SELECT count(*) FROM sqlite_schema WHERE type = 'table' AND name NOT LIKE 'sqlite_%'`,
		).Scan(&tables)
		if err != nil {
			return 0, fmt.Errorf("sqlite: inspect schema: %w", err)
		}
		if tables > 0 {
			return 0, errors.New("sqlite: database has tables but no schema version; " +
				"it was probably created outside the migrator. Recreate it, or set " +
				"PRAGMA user_version to the last migration it contains")
		}
	}

	for _, m := range migrations {
		if m.version <= current {
			continue
		}
		if err := applyMigration(ctx, db, m); err != nil {
			return current, err
		}
		current = m.version
	}
	return current, nil
}

func applyMigration(ctx context.Context, db *sql.DB, m migration) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("sqlite: begin migration %s: %w", m.name, err)
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx, m.script); err != nil {
		return fmt.Errorf("sqlite: apply migration %s: %w", m.name, err)
	}
	// PRAGMA statements cannot take bound parameters; version is an int we
	// parsed ourselves, so formatting it is safe.
	if _, err := tx.ExecContext(ctx, "PRAGMA user_version = "+strconv.Itoa(m.version)); err != nil {
		return fmt.Errorf("sqlite: record version %d: %w", m.version, err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("sqlite: commit migration %s: %w", m.name, err)
	}
	return nil
}

func schemaVersion(ctx context.Context, db *sql.DB) (int, error) {
	var v int
	if err := db.QueryRowContext(ctx, "PRAGMA user_version").Scan(&v); err != nil {
		return 0, fmt.Errorf("sqlite: read schema version: %w", err)
	}
	return v, nil
}

func loadMigrations(fsys fs.FS) ([]migration, error) {
	entries, err := fs.ReadDir(fsys, ".")
	if err != nil {
		return nil, fmt.Errorf("sqlite: list migrations: %w", err)
	}

	var out []migration
	seen := map[int]string{}
	for _, e := range entries {
		if e.IsDir() || path.Ext(e.Name()) != ".sql" {
			continue
		}
		prefix, _, ok := strings.Cut(e.Name(), "_")
		v, err := strconv.Atoi(prefix)
		if !ok || err != nil || v <= 0 {
			return nil, fmt.Errorf("sqlite: migration %q must be named NNN_description.sql", e.Name())
		}
		if prev, dup := seen[v]; dup {
			return nil, fmt.Errorf("sqlite: migrations %q and %q share version %d", prev, e.Name(), v)
		}
		seen[v] = e.Name()

		b, err := fs.ReadFile(fsys, e.Name())
		if err != nil {
			return nil, fmt.Errorf("sqlite: read migration %q: %w", e.Name(), err)
		}
		out = append(out, migration{version: v, name: e.Name(), script: string(b)})
	}
	if len(out) == 0 {
		return nil, errors.New("sqlite: no migrations found")
	}
	sort.Slice(out, func(i, j int) bool { return out[i].version < out[j].version })
	return out, nil
}
