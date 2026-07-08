package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/selvakn/gcmtest/server/internal/store"
)

func TestListDeliveries_ReturnsOrderedSlowestFirst(t *testing.T) {
	db := newTestDB(t)
	insertTestDevice(t, db, "device-1")
	insertTestDevice(t, db, "device-2")
	rounds := store.NewRoundStore(db)
	deliveries := store.NewDeliveryStore(db)
	sentAt := time.Now().UTC().Truncate(time.Millisecond)
	if _, _, err := rounds.CreateRound(t.Context(), "round-1", sentAt); err != nil {
		t.Fatalf("CreateRound: %v", err)
	}
	if _, err := deliveries.RecordReceipt(t.Context(), "round-1", "device-1", sentAt.Add(100*time.Millisecond)); err != nil {
		t.Fatalf("RecordReceipt: %v", err)
	}
	if _, err := deliveries.RecordReceipt(t.Context(), "round-1", "device-2", sentAt.Add(900*time.Millisecond)); err != nil {
		t.Fatalf("RecordReceipt: %v", err)
	}

	h := &ReportsHandler{Deliveries: deliveries, Reports: store.NewReportStore(db)}
	req := httptest.NewRequest(http.MethodGet, "/v1/reports/deliveries?order=latency_desc&limit=10", nil)
	rec := httptest.NewRecorder()

	h.ListDeliveries(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200, body=%s", rec.Code, rec.Body.String())
	}
	var body []map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(body) != 2 {
		t.Fatalf("len(body) = %d, want 2", len(body))
	}
	if int(body[0]["latency_ms"].(float64)) != 900 {
		t.Errorf("body[0].latency_ms = %v, want 900 (slowest first)", body[0]["latency_ms"])
	}
}

func TestSuccessRateReport_ReturnsBuckets(t *testing.T) {
	db := newTestDB(t)
	insertTestDevice(t, db, "device-1")
	rounds := store.NewRoundStore(db)
	sentAt := time.Now().UTC().Truncate(time.Millisecond)
	if _, _, err := rounds.CreateRound(t.Context(), "round-1", sentAt); err != nil {
		t.Fatalf("CreateRound: %v", err)
	}

	h := &ReportsHandler{Deliveries: store.NewDeliveryStore(db), Reports: store.NewReportStore(db)}
	req := httptest.NewRequest(http.MethodGet, "/v1/reports/success-rate?bucket=hour", nil)
	rec := httptest.NewRecorder()

	h.SuccessRateReport(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200, body=%s", rec.Code, rec.Body.String())
	}
	var body []map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(body) != 1 {
		t.Fatalf("len(body) = %d, want 1", len(body))
	}
	if int(body[0]["targeted"].(float64)) != 1 {
		t.Errorf("targeted = %v, want 1", body[0]["targeted"])
	}
}
