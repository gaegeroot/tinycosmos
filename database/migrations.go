// Package database embeds the Tiny Cosmos SQL migrations so binaries and tests
// can apply them without depending on the working directory.
package database

import (
	"embed"
	"io/fs"
)

//go:embed migrations/*.sql
var migrationFiles embed.FS

// Migrations returns the migration scripts. Files sit at the root of the
// returned FS and are named NNN_description.sql, where NNN is the schema
// version the script produces.
func Migrations() fs.FS {
	sub, err := fs.Sub(migrationFiles, "migrations")
	if err != nil {
		panic(err) // the embedded path is fixed at compile time
	}
	return sub
}
