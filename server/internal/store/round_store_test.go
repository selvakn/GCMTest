package store

import (
	"context"
	"testing"
	"time"

	"github.com/selvakn/gcmtest/server/internal/model"
)

func TestCreateRound_TargetsAllCurrentDevices(t *testing.T) {
	db := newTestDB(t)
	insertTestDevice(t, db, "device-1")
	insertTestDevice(t, db, "device-2")

	rs := NewRoundStore(db)
	sentAt := time.Now().UTC().Truncate(time.Millisecond)

	round, devices, err := rs.CreateRound(context.Background(), "round-abc", sentAt)
	if err != nil {
		t.Fatalf("CreateRound: %v", err)
	}

	if round.PublicID != "round-abc" {
		t.Errorf("PublicID = %q, want round-abc", round.PublicID)
	}
	if round.TargetedDeviceCount != 2 {
		t.Errorf("TargetedDeviceCount = %d, want 2", round.TargetedDeviceCount)
	}
	if len(devices) != 2 {
		t.Fatalf("len(devices) = %d, want 2", len(devices))
	}

	var pendingCount int
	if err := db.QueryRow(
		`SELECT COUNT(*) FROM deliveries WHERE round_id = ? AND status = ?`,
		round.ID, model.StatusPending,
	).Scan(&pendingCount); err != nil {
		t.Fatalf("count deliveries: %v", err)
	}
	if pendingCount != 2 {
		t.Errorf("pending deliveries = %d, want 2", pendingCount)
	}
}

func TestCreateRound_NoDevicesEnrolled(t *testing.T) {
	db := newTestDB(t)
	rs := NewRoundStore(db)

	round, devices, err := rs.CreateRound(context.Background(), "round-empty", time.Now().UTC())
	if err != nil {
		t.Fatalf("CreateRound: %v", err)
	}
	if round.TargetedDeviceCount != 0 {
		t.Errorf("TargetedDeviceCount = %d, want 0", round.TargetedDeviceCount)
	}
	if len(devices) != 0 {
		t.Errorf("len(devices) = %d, want 0", len(devices))
	}
}
