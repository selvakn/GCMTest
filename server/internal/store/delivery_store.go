package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/selvakn/gcmtest/server/internal/model"
)

// ReceiptResult reports what happened when a receipt report was processed.
type ReceiptResult int

const (
	// ReceiptRecorded means this was the first receipt for this round+device
	// pairing and it was stored.
	ReceiptRecorded ReceiptResult = iota
	// ReceiptDuplicateIgnored means this pairing had already received a
	// receipt; the report was a safe-to-retry no-op (FR-012).
	ReceiptDuplicateIgnored
	// ReceiptInvalid means the round or device is unknown, or the device was
	// not targeted by this round (FR-011).
	ReceiptInvalid
)

// DeliveryStore records and queries Delivery outcomes.
type DeliveryStore struct {
	db *sql.DB
}

// NewDeliveryStore constructs a DeliveryStore backed by db.
func NewDeliveryStore(db *sql.DB) *DeliveryStore {
	return &DeliveryStore{db: db}
}

// RecordReceipt records that the device identified by installID received the
// round identified by roundPublicID at receivedAt. It is safe to call more
// than once for the same pairing (FR-012) and gracefully reports unknown
// rounds/devices instead of erroring (FR-011).
func (s *DeliveryStore) RecordReceipt(ctx context.Context, roundPublicID, installID string, receivedAt time.Time) (ReceiptResult, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return ReceiptInvalid, err
	}
	defer func() { _ = tx.Rollback() }()

	var roundID int64
	var sentAt time.Time
	err = tx.QueryRowContext(ctx, `SELECT id, sent_at FROM rounds WHERE public_id = ?`, roundPublicID).Scan(&roundID, &sentAt)
	if errors.Is(err, sql.ErrNoRows) {
		return ReceiptInvalid, nil
	}
	if err != nil {
		return ReceiptInvalid, err
	}

	var deviceID int64
	err = tx.QueryRowContext(ctx, `SELECT id FROM devices WHERE install_id = ?`, installID).Scan(&deviceID)
	if errors.Is(err, sql.ErrNoRows) {
		return ReceiptInvalid, nil
	}
	if err != nil {
		return ReceiptInvalid, err
	}

	var currentStatus string
	err = tx.QueryRowContext(ctx,
		`SELECT status FROM deliveries WHERE round_id = ? AND device_id = ?`, roundID, deviceID,
	).Scan(&currentStatus)
	if errors.Is(err, sql.ErrNoRows) {
		// Device was not targeted by this round.
		return ReceiptInvalid, nil
	}
	if err != nil {
		return ReceiptInvalid, err
	}
	if currentStatus != string(model.StatusPending) {
		return ReceiptDuplicateIgnored, nil
	}

	latencyMS := receivedAt.Sub(sentAt).Milliseconds()
	status := model.StatusReceivedGood
	if latencyMS <= 0 {
		status = model.StatusReceivedAnomalous
	}

	res, err := tx.ExecContext(ctx,
		`UPDATE deliveries SET received_at = ?, latency_ms = ?, status = ? WHERE round_id = ? AND device_id = ? AND status = ?`,
		receivedAt, latencyMS, status, roundID, deviceID, model.StatusPending,
	)
	if err != nil {
		return ReceiptInvalid, err
	}
	rows, err := res.RowsAffected()
	if err != nil {
		return ReceiptInvalid, err
	}
	if rows == 0 {
		// Lost a race with a concurrent duplicate report.
		return ReceiptDuplicateIgnored, nil
	}

	if err := tx.Commit(); err != nil {
		return ReceiptInvalid, err
	}
	return ReceiptRecorded, nil
}

// DeliveryRecord is one row returned by Query — a delivery joined with its
// round and device's public identifiers, for reporting (FR-013).
type DeliveryRecord struct {
	RoundPublicID   string
	DeviceInstallID string
	SentAt          time.Time
	ReceivedAt      *time.Time
	LatencyMS       *int64
	Status          model.DeliveryStatus
}

// DeliveryQueryOptions filters/orders/limits a Query call.
type DeliveryQueryOptions struct {
	// RoundPublicID, if non-empty, restricts results to one round.
	RoundPublicID string
	// Order is one of "latency_desc", "latency_asc", "sent_at_desc" (default).
	Order string
	// Limit caps the number of rows returned; <= 0 means the default of 50.
	Limit int
}

// Query returns deliveries matching opts, most useful for finding the
// slowest individual deliveries (FR-013).
func (s *DeliveryStore) Query(ctx context.Context, opts DeliveryQueryOptions) ([]DeliveryRecord, error) {
	limit := opts.Limit
	if limit <= 0 {
		limit = 50
	}

	orderClause := "d.sent_at DESC"
	switch opts.Order {
	case "latency_desc":
		orderClause = "d.latency_ms DESC NULLS LAST"
	case "latency_asc":
		orderClause = "d.latency_ms ASC NULLS LAST"
	case "sent_at_desc", "":
		orderClause = "d.sent_at DESC"
	}

	query := fmt.Sprintf(`
		SELECT r.public_id, dev.install_id, d.sent_at, d.received_at, d.latency_ms, d.status
		FROM deliveries d
		JOIN rounds r ON r.id = d.round_id
		JOIN devices dev ON dev.id = d.device_id
		WHERE (? = '' OR r.public_id = ?)
		ORDER BY %s
		LIMIT ?`, orderClause)

	rows, err := s.db.QueryContext(ctx, query, opts.RoundPublicID, opts.RoundPublicID, limit)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var records []DeliveryRecord
	for rows.Next() {
		var rec DeliveryRecord
		var status string
		if err := rows.Scan(&rec.RoundPublicID, &rec.DeviceInstallID, &rec.SentAt, &rec.ReceivedAt, &rec.LatencyMS, &status); err != nil {
			return nil, err
		}
		rec.Status = model.DeliveryStatus(status)
		records = append(records, rec)
	}
	return records, rows.Err()
}
