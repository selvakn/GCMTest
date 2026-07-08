package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/selvakn/gcmtest/server/internal/store"
)

func TestEnrollDevice_NewEnrollmentReturnsEnrolled(t *testing.T) {
	db := newTestDB(t)
	h := &DevicesHandler{Devices: store.NewDeviceStore(db)}

	body, _ := json.Marshal(map[string]string{
		"install_id": "install-1",
		"platform":   "android",
		"push_token": "token-a",
	})
	req := httptest.NewRequest(http.MethodPost, "/v1/devices", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	h.EnrollDevice(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200, body=%s", rec.Code, rec.Body.String())
	}
	var respBody map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &respBody); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if respBody["status"] != "enrolled" {
		t.Errorf("status = %q, want enrolled", respBody["status"])
	}
}

func TestEnrollDevice_ReEnrollmentReturnsUpdatedNotDuplicate(t *testing.T) {
	db := newTestDB(t)
	h := &DevicesHandler{Devices: store.NewDeviceStore(db)}

	enroll := func(token string) map[string]string {
		body, _ := json.Marshal(map[string]string{
			"install_id": "install-1",
			"platform":   "android",
			"push_token": token,
		})
		req := httptest.NewRequest(http.MethodPost, "/v1/devices", bytes.NewReader(body))
		rec := httptest.NewRecorder()
		h.EnrollDevice(rec, req)
		var respBody map[string]string
		if err := json.Unmarshal(rec.Body.Bytes(), &respBody); err != nil {
			t.Fatalf("decode response: %v", err)
		}
		return respBody
	}

	first := enroll("token-a")
	if first["status"] != "enrolled" {
		t.Fatalf("first status = %q, want enrolled", first["status"])
	}
	second := enroll("token-b")
	if second["status"] != "updated" {
		t.Fatalf("second status = %q, want updated", second["status"])
	}

	var count int
	if err := db.QueryRow(`SELECT COUNT(*) FROM devices WHERE install_id = 'install-1'`).Scan(&count); err != nil {
		t.Fatalf("count: %v", err)
	}
	if count != 1 {
		t.Errorf("device count = %d, want 1", count)
	}
}

func TestEnrollDevice_MalformedRequestReturns400(t *testing.T) {
	db := newTestDB(t)
	h := &DevicesHandler{Devices: store.NewDeviceStore(db)}

	req := httptest.NewRequest(http.MethodPost, "/v1/devices", bytes.NewReader([]byte("not json")))
	rec := httptest.NewRecorder()

	h.EnrollDevice(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
}

func TestEnrollDevice_MissingFieldsReturns400(t *testing.T) {
	db := newTestDB(t)
	h := &DevicesHandler{Devices: store.NewDeviceStore(db)}

	body, _ := json.Marshal(map[string]string{"install_id": "install-1"})
	req := httptest.NewRequest(http.MethodPost, "/v1/devices", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	h.EnrollDevice(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
}
