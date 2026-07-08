package store

import (
	"context"
	"testing"
	"time"
)

func TestSuccessRateByBucket_GroupsByHour(t *testing.T) {
	db := newTestDB(t)
	insertTestDevice(t, db, "device-1")
	insertTestDevice(t, db, "device-2")
	rs := NewRoundStore(db)
	ds := NewDeliveryStore(db)

	hourOne := time.Date(2026, 7, 8, 10, 15, 0, 0, time.UTC)
	hourTwo := time.Date(2026, 7, 8, 11, 5, 0, 0, time.UTC)

	// Round in the 10:00 bucket: both devices receive it.
	if _, _, err := rs.CreateRound(context.Background(), "round-1", hourOne); err != nil {
		t.Fatalf("CreateRound: %v", err)
	}
	if _, err := ds.RecordReceipt(context.Background(), "round-1", "device-1", hourOne.Add(time.Second)); err != nil {
		t.Fatalf("RecordReceipt: %v", err)
	}
	if _, err := ds.RecordReceipt(context.Background(), "round-1", "device-2", hourOne.Add(time.Second)); err != nil {
		t.Fatalf("RecordReceipt: %v", err)
	}

	// Round in the 11:00 bucket: only one of two devices receives it.
	if _, _, err := rs.CreateRound(context.Background(), "round-2", hourTwo); err != nil {
		t.Fatalf("CreateRound: %v", err)
	}
	if _, err := ds.RecordReceipt(context.Background(), "round-2", "device-1", hourTwo.Add(time.Second)); err != nil {
		t.Fatalf("RecordReceipt: %v", err)
	}

	report := NewReportStore(db)
	buckets, err := report.SuccessRateByBucket(context.Background(), "hour", nil, nil)
	if err != nil {
		t.Fatalf("SuccessRateByBucket: %v", err)
	}
	if len(buckets) != 2 {
		t.Fatalf("len(buckets) = %d, want 2", len(buckets))
	}

	first := buckets[0]
	if first.Targeted != 2 || first.Received != 2 || first.SuccessRate != 1.0 {
		t.Errorf("first bucket = %+v, want targeted=2 received=2 rate=1.0", first)
	}
	second := buckets[1]
	if second.Targeted != 2 || second.Received != 1 || second.SuccessRate != 0.5 {
		t.Errorf("second bucket = %+v, want targeted=2 received=1 rate=0.5", second)
	}
}

func TestSuccessRateByBucket_NoRounds(t *testing.T) {
	db := newTestDB(t)
	report := NewReportStore(db)
	buckets, err := report.SuccessRateByBucket(context.Background(), "hour", nil, nil)
	if err != nil {
		t.Fatalf("SuccessRateByBucket: %v", err)
	}
	if len(buckets) != 0 {
		t.Errorf("len(buckets) = %d, want 0", len(buckets))
	}
}
