package store

import (
	"context"
	"database/sql"
	"time"
)

// DeviceResult reports whether an enrollment created a new device or updated
// an existing one.
type DeviceResult int

const (
	// DeviceEnrolled means install_id had never been seen before.
	DeviceEnrolled DeviceResult = iota
	// DeviceUpdated means install_id already existed; its push_token (and/or
	// platform) was updated in place — never duplicated (FR-002).
	DeviceUpdated
)

// DeviceStore persists enrolled devices.
type DeviceStore struct {
	db *sql.DB
}

// NewDeviceStore constructs a DeviceStore backed by db.
func NewDeviceStore(db *sql.DB) *DeviceStore {
	return &DeviceStore{db: db}
}

// Upsert enrolls installID if it is new, or updates its platform/push_token
// in place if it already exists (FR-002, FR-019). installID — not the push
// token — is the durable identity (Clarification Q1).
func (s *DeviceStore) Upsert(ctx context.Context, installID, platform, pushToken string) (DeviceResult, error) {
	now := time.Now().UTC()

	var existingID int64
	err := s.db.QueryRowContext(ctx, `SELECT id FROM devices WHERE install_id = ?`, installID).Scan(&existingID)
	switch {
	case err == sql.ErrNoRows:
		_, err := s.db.ExecContext(ctx,
			`INSERT INTO devices (install_id, platform, push_token, created_at, updated_at) VALUES (?, ?, ?, ?, ?)`,
			installID, platform, pushToken, now, now,
		)
		if err != nil {
			return DeviceEnrolled, err
		}
		return DeviceEnrolled, nil
	case err != nil:
		return DeviceEnrolled, err
	default:
		_, err := s.db.ExecContext(ctx,
			`UPDATE devices SET platform = ?, push_token = ?, updated_at = ? WHERE id = ?`,
			platform, pushToken, now, existingID,
		)
		if err != nil {
			return DeviceUpdated, err
		}
		return DeviceUpdated, nil
	}
}
