package sqlite

import (
	"errors"
	"fmt"
	"time"

	sqlite "modernc.org/sqlite"
	sqlite3 "modernc.org/sqlite/lib"
)

// timestampLayout matches the schema's convention and the migration's default
// expression strftime('%Y-%m-%dT%H:%M:%fZ', 'now'): ISO 8601, UTC, millisecond
// precision. Sub-millisecond precision is truncated on write.
const timestampLayout = "2006-01-02T15:04:05.000Z"

func formatTimestamp(t time.Time) string {
	return t.UTC().Format(timestampLayout)
}

func parseTimestamp(s string) (time.Time, error) {
	t, err := time.Parse(time.RFC3339Nano, s)
	if err != nil {
		return time.Time{}, fmt.Errorf("sqlite: parse timestamp %q: %w", s, err)
	}
	return t.UTC(), nil
}

// constraintCode returns the extended SQLite result code for constraint
// violations, or 0 if err is not one.
func constraintCode(err error) int {
	var se *sqlite.Error
	if !errors.As(err, &se) {
		return 0
	}
	switch c := se.Code(); c {
	case sqlite3.SQLITE_CONSTRAINT_UNIQUE,
		sqlite3.SQLITE_CONSTRAINT_PRIMARYKEY,
		sqlite3.SQLITE_CONSTRAINT_FOREIGNKEY:
		return c
	}
	return 0
}
