package store

import (
	"database/sql"
	"testing"
	"time"
)

func newTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := Open(":memory:", "../../migrations")
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return db
}

// insertTestDevice inserts a device directly (bypassing the enrollment API,
// which is a separate concern implemented in device_store.go) so store tests
// for rounds/deliveries have fixture devices to target.
func insertTestDevice(t *testing.T, db *sql.DB, installID string) int64 {
	t.Helper()
	now := time.Now().UTC()
	res, err := db.Exec(
		`INSERT INTO devices (install_id, platform, push_token, created_at, updated_at) VALUES (?, ?, ?, ?, ?)`,
		installID, "android", "token-"+installID, now, now,
	)
	if err != nil {
		t.Fatalf("insert test device: %v", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		t.Fatalf("last insert id: %v", err)
	}
	return id
}
