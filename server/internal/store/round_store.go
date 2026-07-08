package store

import (
	"context"
	"database/sql"
	"time"

	"github.com/selvakn/gcmtest/server/internal/model"
)

// RoundStore persists Rounds and the Deliveries created when a Round is triggered.
type RoundStore struct {
	db *sql.DB
}

// NewRoundStore constructs a RoundStore backed by db.
func NewRoundStore(db *sql.DB) *RoundStore {
	return &RoundStore{db: db}
}

// CreateRound records a new round with a unique publicID and sentAt (assigned
// before any device could plausibly have responded — FR-006), snapshots the
// currently enrolled fleet, and creates one pending Delivery per device
// (FR-007). It returns the created Round and the snapshot of targeted
// devices, which the caller uses to actually dispatch pushes.
func (s *RoundStore) CreateRound(ctx context.Context, publicID string, sentAt time.Time) (model.Round, []model.Device, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return model.Round{}, nil, err
	}
	defer func() { _ = tx.Rollback() }()

	devices, err := allDevicesTx(ctx, tx)
	if err != nil {
		return model.Round{}, nil, err
	}

	res, err := tx.ExecContext(ctx,
		`INSERT INTO rounds (public_id, sent_at, targeted_device_count) VALUES (?, ?, ?)`,
		publicID, sentAt, len(devices),
	)
	if err != nil {
		return model.Round{}, nil, err
	}
	roundID, err := res.LastInsertId()
	if err != nil {
		return model.Round{}, nil, err
	}

	if len(devices) > 0 {
		stmt, err := tx.PrepareContext(ctx,
			`INSERT INTO deliveries (round_id, device_id, sent_at, status) VALUES (?, ?, ?, ?)`,
		)
		if err != nil {
			return model.Round{}, nil, err
		}
		defer func() { _ = stmt.Close() }()

		for _, device := range devices {
			if _, err := stmt.ExecContext(ctx, roundID, device.ID, sentAt, model.StatusPending); err != nil {
				return model.Round{}, nil, err
			}
		}
	}

	if err := tx.Commit(); err != nil {
		return model.Round{}, nil, err
	}

	return model.Round{
		ID:                  roundID,
		PublicID:            publicID,
		SentAt:              sentAt,
		TargetedDeviceCount: len(devices),
	}, devices, nil
}

// allDevicesTx reads every enrolled device within tx.
func allDevicesTx(ctx context.Context, tx *sql.Tx) ([]model.Device, error) {
	rows, err := tx.QueryContext(ctx,
		`SELECT id, install_id, platform, push_token, created_at, updated_at FROM devices`,
	)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var devices []model.Device
	for rows.Next() {
		var d model.Device
		if err := rows.Scan(&d.ID, &d.InstallID, &d.Platform, &d.PushToken, &d.CreatedAt, &d.UpdatedAt); err != nil {
			return nil, err
		}
		devices = append(devices, d)
	}
	return devices, rows.Err()
}
