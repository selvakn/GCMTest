package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/selvakn/gcmtest/server/internal/store"
)

func TestTriggerRound_RequiresBearerAuth(t *testing.T) {
	db := newTestDB(t)
	h := &RoundsHandler{Rounds: store.NewRoundStore(db), Sender: &fakeSender{}}
	protected := RequireOperatorToken("secret-token", h.TriggerRound)

	req := httptest.NewRequest(http.MethodPost, "/v1/rounds", nil)
	rec := httptest.NewRecorder()
	protected(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", rec.Code)
	}
}

func TestTriggerRound_DispatchesToFleetAndReturns202(t *testing.T) {
	db := newTestDB(t)
	insertTestDevice(t, db, "device-1")
	insertTestDevice(t, db, "device-2")

	sender := &fakeSender{}
	h := &RoundsHandler{Rounds: store.NewRoundStore(db), Sender: sender}
	protected := RequireOperatorToken("secret-token", h.TriggerRound)

	req := httptest.NewRequest(http.MethodPost, "/v1/rounds", nil)
	req.Header.Set("Authorization", "Bearer secret-token")
	rec := httptest.NewRecorder()
	protected(rec, req)

	if rec.Code != http.StatusAccepted {
		t.Fatalf("status = %d, want 202, body=%s", rec.Code, rec.Body.String())
	}
	if sender.sentCount() != 2 {
		t.Errorf("sentCount = %d, want 2", sender.sentCount())
	}

	var body map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body["round_id"] == "" || body["round_id"] == nil {
		t.Errorf("round_id missing in response: %v", body)
	}
	if int(body["targeted_device_count"].(float64)) != 2 {
		t.Errorf("targeted_device_count = %v, want 2", body["targeted_device_count"])
	}
}
