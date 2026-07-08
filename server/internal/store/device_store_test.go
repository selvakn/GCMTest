package store

import (
	"context"
	"testing"
)

func TestUpsertDevice_NewEnrollment(t *testing.T) {
	db := newTestDB(t)
	ds := NewDeviceStore(db)

	result, err := ds.Upsert(context.Background(), "install-1", "android", "token-a")
	if err != nil {
		t.Fatalf("Upsert: %v", err)
	}
	if result != DeviceEnrolled {
		t.Fatalf("result = %v, want DeviceEnrolled", result)
	}

	var count int
	if err := db.QueryRow(`SELECT COUNT(*) FROM devices WHERE install_id = ?`, "install-1").Scan(&count); err != nil {
		t.Fatalf("count devices: %v", err)
	}
	if count != 1 {
		t.Errorf("device count = %d, want 1", count)
	}
}

func TestUpsertDevice_ReEnrollmentUpdatesExistingRowNotDuplicate(t *testing.T) {
	db := newTestDB(t)
	ds := NewDeviceStore(db)

	if _, err := ds.Upsert(context.Background(), "install-1", "android", "token-a"); err != nil {
		t.Fatalf("first Upsert: %v", err)
	}

	// Token rotation: same install_id, new push_token (FR-002, FR-019).
	result, err := ds.Upsert(context.Background(), "install-1", "android", "token-b")
	if err != nil {
		t.Fatalf("second Upsert: %v", err)
	}
	if result != DeviceUpdated {
		t.Fatalf("result = %v, want DeviceUpdated", result)
	}

	var count int
	if err := db.QueryRow(`SELECT COUNT(*) FROM devices WHERE install_id = ?`, "install-1").Scan(&count); err != nil {
		t.Fatalf("count devices: %v", err)
	}
	if count != 1 {
		t.Errorf("device count = %d, want 1 (no duplicate row)", count)
	}

	var pushToken string
	if err := db.QueryRow(`SELECT push_token FROM devices WHERE install_id = ?`, "install-1").Scan(&pushToken); err != nil {
		t.Fatalf("query push_token: %v", err)
	}
	if pushToken != "token-b" {
		t.Errorf("push_token = %q, want token-b (updated in place)", pushToken)
	}
}
