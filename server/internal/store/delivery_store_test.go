package store

import (
	"context"
	"testing"
	"time"

	"github.com/selvakn/gcmtest/server/internal/model"
)

func TestRecordReceipt_ComputesPositiveLatency(t *testing.T) {
	db := newTestDB(t)
	insertTestDevice(t, db, "device-1")
	rs := NewRoundStore(db)
	sentAt := time.Now().UTC().Truncate(time.Millisecond)
	round, _, err := rs.CreateRound(context.Background(), "round-1", sentAt)
	if err != nil {
		t.Fatalf("CreateRound: %v", err)
	}

	ds := NewDeliveryStore(db)
	receivedAt := sentAt.Add(150 * time.Millisecond)
	result, err := ds.RecordReceipt(context.Background(), "round-1", "device-1", receivedAt)
	if err != nil {
		t.Fatalf("RecordReceipt: %v", err)
	}
	if result != ReceiptRecorded {
		t.Fatalf("result = %v, want ReceiptRecorded", result)
	}

	var status string
	var latencyMS int64
	if err := db.QueryRow(
		`SELECT status, latency_ms FROM deliveries WHERE round_id = ? AND device_id = (SELECT id FROM devices WHERE install_id = ?)`,
		round.ID, "device-1",
	).Scan(&status, &latencyMS); err != nil {
		t.Fatalf("query delivery: %v", err)
	}
	if status != string(model.StatusReceivedGood) {
		t.Errorf("status = %q, want %q", status, model.StatusReceivedGood)
	}
	if latencyMS != 150 {
		t.Errorf("latency_ms = %d, want 150", latencyMS)
	}
}

func TestRecordReceipt_NonPositiveLatencyIsAnomalous(t *testing.T) {
	db := newTestDB(t)
	insertTestDevice(t, db, "device-1")
	rs := NewRoundStore(db)
	sentAt := time.Now().UTC().Truncate(time.Millisecond)
	if _, _, err := rs.CreateRound(context.Background(), "round-1", sentAt); err != nil {
		t.Fatalf("CreateRound: %v", err)
	}

	ds := NewDeliveryStore(db)
	// receivedAt at-or-before sentAt: clock-skew/measurement artifact.
	result, err := ds.RecordReceipt(context.Background(), "round-1", "device-1", sentAt)
	if err != nil {
		t.Fatalf("RecordReceipt: %v", err)
	}
	if result != ReceiptRecorded {
		t.Fatalf("result = %v, want ReceiptRecorded", result)
	}

	var status string
	if err := db.QueryRow(
		`SELECT status FROM deliveries WHERE round_id = (SELECT id FROM rounds WHERE public_id = 'round-1')`,
	).Scan(&status); err != nil {
		t.Fatalf("query delivery: %v", err)
	}
	if status != string(model.StatusReceivedAnomalous) {
		t.Errorf("status = %q, want %q (non-positive latency must never be 'good')", status, model.StatusReceivedAnomalous)
	}
}

func TestRecordReceipt_DuplicateReportIsIgnored(t *testing.T) {
	db := newTestDB(t)
	insertTestDevice(t, db, "device-1")
	rs := NewRoundStore(db)
	sentAt := time.Now().UTC().Truncate(time.Millisecond)
	if _, _, err := rs.CreateRound(context.Background(), "round-1", sentAt); err != nil {
		t.Fatalf("CreateRound: %v", err)
	}

	ds := NewDeliveryStore(db)
	first, err := ds.RecordReceipt(context.Background(), "round-1", "device-1", sentAt.Add(100*time.Millisecond))
	if err != nil || first != ReceiptRecorded {
		t.Fatalf("first RecordReceipt = %v, %v", first, err)
	}

	// A retried network call reports the same receipt again, with a different
	// (later) timestamp — the original recorded value must not be overwritten.
	second, err := ds.RecordReceipt(context.Background(), "round-1", "device-1", sentAt.Add(9999*time.Millisecond))
	if err != nil {
		t.Fatalf("second RecordReceipt: %v", err)
	}
	if second != ReceiptDuplicateIgnored {
		t.Fatalf("second result = %v, want ReceiptDuplicateIgnored", second)
	}

	var latencyMS int64
	if err := db.QueryRow(
		`SELECT latency_ms FROM deliveries WHERE round_id = (SELECT id FROM rounds WHERE public_id = 'round-1')`,
	).Scan(&latencyMS); err != nil {
		t.Fatalf("query delivery: %v", err)
	}
	if latencyMS != 100 {
		t.Errorf("latency_ms = %d, want 100 (unchanged by duplicate report)", latencyMS)
	}
}

func TestRecordReceipt_UnknownRoundOrDeviceIsInvalid(t *testing.T) {
	db := newTestDB(t)
	insertTestDevice(t, db, "device-1")
	ds := NewDeliveryStore(db)

	result, err := ds.RecordReceipt(context.Background(), "no-such-round", "device-1", time.Now().UTC())
	if err != nil {
		t.Fatalf("RecordReceipt (unknown round): %v", err)
	}
	if result != ReceiptInvalid {
		t.Errorf("result = %v, want ReceiptInvalid for unknown round", result)
	}

	rs := NewRoundStore(db)
	if _, _, err := rs.CreateRound(context.Background(), "round-1", time.Now().UTC()); err != nil {
		t.Fatalf("CreateRound: %v", err)
	}
	result, err = ds.RecordReceipt(context.Background(), "round-1", "no-such-device", time.Now().UTC())
	if err != nil {
		t.Fatalf("RecordReceipt (unknown device): %v", err)
	}
	if result != ReceiptInvalid {
		t.Errorf("result = %v, want ReceiptInvalid for unknown device", result)
	}
}

func TestQueryDeliveries_OrdersByLatencyDescAndLimits(t *testing.T) {
	db := newTestDB(t)
	insertTestDevice(t, db, "device-1")
	insertTestDevice(t, db, "device-2")
	insertTestDevice(t, db, "device-3")
	rs := NewRoundStore(db)
	sentAt := time.Now().UTC().Truncate(time.Millisecond)
	if _, _, err := rs.CreateRound(context.Background(), "round-1", sentAt); err != nil {
		t.Fatalf("CreateRound: %v", err)
	}

	ds := NewDeliveryStore(db)
	mustRecord := func(installID string, latency time.Duration) {
		if _, err := ds.RecordReceipt(context.Background(), "round-1", installID, sentAt.Add(latency)); err != nil {
			t.Fatalf("RecordReceipt(%s): %v", installID, err)
		}
	}
	mustRecord("device-1", 500*time.Millisecond)
	mustRecord("device-2", 100*time.Millisecond)
	mustRecord("device-3", 900*time.Millisecond)

	records, err := ds.Query(context.Background(), DeliveryQueryOptions{Order: "latency_desc", Limit: 2})
	if err != nil {
		t.Fatalf("Query: %v", err)
	}
	if len(records) != 2 {
		t.Fatalf("len(records) = %d, want 2 (limit applied)", len(records))
	}
	if *records[0].LatencyMS != 900 || *records[1].LatencyMS != 500 {
		t.Errorf("latencies = %d, %d, want 900, 500 (descending)", *records[0].LatencyMS, *records[1].LatencyMS)
	}
}

func TestQueryDeliveries_FiltersByRound(t *testing.T) {
	db := newTestDB(t)
	insertTestDevice(t, db, "device-1")
	rs := NewRoundStore(db)
	sentAt := time.Now().UTC().Truncate(time.Millisecond)
	if _, _, err := rs.CreateRound(context.Background(), "round-1", sentAt); err != nil {
		t.Fatalf("CreateRound: %v", err)
	}
	if _, _, err := rs.CreateRound(context.Background(), "round-2", sentAt); err != nil {
		t.Fatalf("CreateRound: %v", err)
	}

	ds := NewDeliveryStore(db)
	records, err := ds.Query(context.Background(), DeliveryQueryOptions{RoundPublicID: "round-1"})
	if err != nil {
		t.Fatalf("Query: %v", err)
	}
	if len(records) != 1 || records[0].RoundPublicID != "round-1" {
		t.Fatalf("records = %+v, want exactly one row for round-1", records)
	}
}
