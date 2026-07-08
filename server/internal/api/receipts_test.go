package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/selvakn/gcmtest/server/internal/store"
)

func newReceiptRequest(t *testing.T, installID string, receivedAt time.Time) *http.Request {
	t.Helper()
	body, err := json.Marshal(map[string]any{
		"install_id":  installID,
		"received_at": receivedAt.Format(time.RFC3339Nano),
	})
	if err != nil {
		t.Fatalf("marshal request: %v", err)
	}
	return httptest.NewRequest(http.MethodPost, "/v1/rounds/round-1/receipts", bytes.NewReader(body))
}

func TestReportReceipt_RecordsForKnownRoundAndDevice(t *testing.T) {
	db := newTestDB(t)
	insertTestDevice(t, db, "device-1")
	rounds := store.NewRoundStore(db)
	sentAt := time.Now().UTC()
	if _, _, err := rounds.CreateRound(t.Context(), "round-1", sentAt); err != nil {
		t.Fatalf("CreateRound: %v", err)
	}

	h := &ReceiptsHandler{Deliveries: store.NewDeliveryStore(db)}
	req := newReceiptRequest(t, "device-1", sentAt.Add(200*time.Millisecond))
	req.SetPathValue("round_id", "round-1")
	rec := httptest.NewRecorder()

	h.ReportReceipt(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200, body=%s", rec.Code, rec.Body.String())
	}
	var body map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body["status"] != "recorded" {
		t.Errorf("status = %q, want recorded", body["status"])
	}
}

func TestReportReceipt_UnknownRoundIsHandledGracefully(t *testing.T) {
	db := newTestDB(t)
	insertTestDevice(t, db, "device-1")

	h := &ReceiptsHandler{Deliveries: store.NewDeliveryStore(db)}
	req := newReceiptRequest(t, "device-1", time.Now().UTC())
	req.SetPathValue("round_id", "no-such-round")
	rec := httptest.NewRecorder()

	h.ReportReceipt(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (graceful handling, FR-011), body=%s", rec.Code, rec.Body.String())
	}
	var body map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body["status"] != "invalid" {
		t.Errorf("status = %q, want invalid", body["status"])
	}
}

func TestReportReceipt_MalformedBodyReturns400(t *testing.T) {
	db := newTestDB(t)
	h := &ReceiptsHandler{Deliveries: store.NewDeliveryStore(db)}

	req := httptest.NewRequest(http.MethodPost, "/v1/rounds/round-1/receipts", bytes.NewReader([]byte("not json")))
	req.SetPathValue("round_id", "round-1")
	rec := httptest.NewRecorder()

	h.ReportReceipt(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
}
