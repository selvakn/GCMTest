package api

import (
	"context"
	"database/sql"
	"sync"
	"testing"
	"time"

	"github.com/selvakn/gcmtest/server/internal/model"
	"github.com/selvakn/gcmtest/server/internal/store"
)

func newTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := store.Open(":memory:", "../../migrations")
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return db
}

func insertTestDevice(t *testing.T, db *sql.DB, installID string) {
	t.Helper()
	now := time.Now().UTC()
	if _, err := db.Exec(
		`INSERT INTO devices (install_id, platform, push_token, created_at, updated_at) VALUES (?, ?, ?, ?, ?)`,
		installID, "android", "token-"+installID, now, now,
	); err != nil {
		t.Fatalf("insert test device: %v", err)
	}
}

// fakeSender is a push.Sender test double that records every Send call
// instead of contacting a real push provider.
type fakeSender struct {
	mu   sync.Mutex
	sent []model.Device
	err  error
}

func (f *fakeSender) Send(ctx context.Context, device model.Device, round model.Round) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.sent = append(f.sent, device)
	return f.err
}

func (f *fakeSender) sentCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.sent)
}
