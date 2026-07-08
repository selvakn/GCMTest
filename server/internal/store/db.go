// Package store implements SQLite-backed persistence for devices, rounds, and deliveries.
package store

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	_ "modernc.org/sqlite"
)

// Open opens (creating if necessary) the SQLite database at dbPath and applies
// any pending *.sql migrations found in migrationsDir, in filename order,
// before returning.
func Open(dbPath, migrationsDir string) (*sql.DB, error) {
	// _time_format=sqlite makes the driver write time.Time values in a
	// format SQLite's own date/time functions (strftime, date, ...)
	// understand, so report queries can bucket by sent_at in SQL.
	db, err := sql.Open("sqlite", dbPath+"?_time_format=sqlite")
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}
	// SQLite only supports one writer at a time; a single connection avoids
	// SQLITE_BUSY errors under the low concurrency this service expects.
	db.SetMaxOpenConns(1)

	if err := migrate(db, migrationsDir); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("migrate: %w", err)
	}
	return db, nil
}

func migrate(db *sql.DB, migrationsDir string) error {
	entries, err := os.ReadDir(migrationsDir)
	if err != nil {
		return err
	}

	var names []string
	for _, entry := range entries {
		if !entry.IsDir() && filepath.Ext(entry.Name()) == ".sql" {
			names = append(names, entry.Name())
		}
	}
	sort.Strings(names)

	for _, name := range names {
		contents, err := os.ReadFile(filepath.Join(migrationsDir, name))
		if err != nil {
			return fmt.Errorf("read migration %s: %w", name, err)
		}
		if _, err := db.Exec(string(contents)); err != nil {
			return fmt.Errorf("apply migration %s: %w", name, err)
		}
	}
	return nil
}
