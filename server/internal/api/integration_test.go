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

// TestEndToEnd_QuickstartFlow exercises the full wired router the way
// quickstart.md's curl walkthrough does: enroll a device, trigger a round,
// have the device report receipt, then confirm it shows up in both report
// endpoints. This is the closest verifiable proxy to the real Android +
// FCM flow available in an environment without a device/emulator or live
// FCM credentials — the push transport is swapped for fakeSender, but
// every HTTP contract, store, and handler in the chain is real.
func TestEndToEnd_QuickstartFlow(t *testing.T) {
	db := newTestDB(t)
	sender := &fakeSender{}

	handlers := Handlers{
		OperatorToken: "test-operator-token",
		Devices:       &DevicesHandler{Devices: store.NewDeviceStore(db)},
		Rounds:        &RoundsHandler{Rounds: store.NewRoundStore(db), Sender: sender},
		Receipts:      &ReceiptsHandler{Deliveries: store.NewDeliveryStore(db)},
		Reports:       &ReportsHandler{Deliveries: store.NewDeliveryStore(db), Reports: store.NewReportStore(db)},
	}
	server := httptest.NewServer(NewRouter(handlers))
	defer server.Close()

	client := server.Client()

	// 1. Enroll a device (as the app does automatically on first launch).
	enrollBody, _ := json.Marshal(map[string]string{
		"install_id": "e2e-device-1",
		"platform":   "android",
		"push_token": "fcm-token-e2e-1",
	})
	enrollResp, err := client.Post(server.URL+"/v1/devices", "application/json", bytes.NewReader(enrollBody))
	if err != nil {
		t.Fatalf("enroll: %v", err)
	}
	if enrollResp.StatusCode != http.StatusOK {
		t.Fatalf("enroll status = %d", enrollResp.StatusCode)
	}
	_ = enrollResp.Body.Close()

	// 2. Trigger a round (operator action, requires bearer auth).
	req, _ := http.NewRequest(http.MethodPost, server.URL+"/v1/rounds", nil)
	req.Header.Set("Authorization", "Bearer test-operator-token")
	triggerResp, err := client.Do(req)
	if err != nil {
		t.Fatalf("trigger round: %v", err)
	}
	if triggerResp.StatusCode != http.StatusAccepted {
		t.Fatalf("trigger status = %d", triggerResp.StatusCode)
	}
	var triggerBody struct {
		RoundID             string `json:"round_id"`
		TargetedDeviceCount int    `json:"targeted_device_count"`
	}
	if err := json.NewDecoder(triggerResp.Body).Decode(&triggerBody); err != nil {
		t.Fatalf("decode trigger response: %v", err)
	}
	_ = triggerResp.Body.Close()
	if triggerBody.TargetedDeviceCount != 1 {
		t.Fatalf("targeted_device_count = %d, want 1", triggerBody.TargetedDeviceCount)
	}
	if sender.sentCount() != 1 {
		t.Fatalf("sender.sentCount() = %d, want 1 (push dispatched to the enrolled device)", sender.sentCount())
	}

	// 3. The device "receives" the push and reports receipt.
	receiptBody, _ := json.Marshal(map[string]string{
		"install_id":  "e2e-device-1",
		"received_at": time.Now().UTC().Add(120 * time.Millisecond).Format(time.RFC3339Nano),
	})
	receiptResp, err := client.Post(
		server.URL+"/v1/rounds/"+triggerBody.RoundID+"/receipts",
		"application/json",
		bytes.NewReader(receiptBody),
	)
	if err != nil {
		t.Fatalf("report receipt: %v", err)
	}
	var receiptRespBody map[string]string
	if err := json.NewDecoder(receiptResp.Body).Decode(&receiptRespBody); err != nil {
		t.Fatalf("decode receipt response: %v", err)
	}
	_ = receiptResp.Body.Close()
	if receiptRespBody["status"] != "recorded" {
		t.Fatalf("receipt status = %q, want recorded", receiptRespBody["status"])
	}

	// 4. The operator reviews reports: the delivery shows up with its latency...
	deliveriesResp, err := client.Get(server.URL + "/v1/reports/deliveries?order=latency_desc")
	if err != nil {
		t.Fatalf("list deliveries: %v", err)
	}
	var deliveries []map[string]any
	if err := json.NewDecoder(deliveriesResp.Body).Decode(&deliveries); err != nil {
		t.Fatalf("decode deliveries: %v", err)
	}
	_ = deliveriesResp.Body.Close()
	if len(deliveries) != 1 || deliveries[0]["status"] != "received_good" {
		t.Fatalf("deliveries = %+v, want one received_good row", deliveries)
	}

	// ...and success-rate reporting shows 100% for this round's bucket.
	successResp, err := client.Get(server.URL + "/v1/reports/success-rate?bucket=hour")
	if err != nil {
		t.Fatalf("success rate: %v", err)
	}
	var buckets []map[string]any
	if err := json.NewDecoder(successResp.Body).Decode(&buckets); err != nil {
		t.Fatalf("decode success-rate: %v", err)
	}
	_ = successResp.Body.Close()
	if len(buckets) != 1 || buckets[0]["success_rate"].(float64) != 1.0 {
		t.Fatalf("buckets = %+v, want one bucket with success_rate=1.0", buckets)
	}
}

// TestEndToEnd_UnreachableDeviceStaysPendingForever mirrors the spec's
// "offline device" scenario (spec.md §4.2): a targeted device that never
// reports receipt must remain visibly pending, not silently absent.
func TestEndToEnd_UnreachableDeviceStaysPendingForever(t *testing.T) {
	db := newTestDB(t)
	sender := &fakeSender{}

	handlers := Handlers{
		OperatorToken: "test-operator-token",
		Devices:       &DevicesHandler{Devices: store.NewDeviceStore(db)},
		Rounds:        &RoundsHandler{Rounds: store.NewRoundStore(db), Sender: sender},
		Reports:       &ReportsHandler{Deliveries: store.NewDeliveryStore(db), Reports: store.NewReportStore(db)},
	}
	server := httptest.NewServer(NewRouter(handlers))
	defer server.Close()
	client := server.Client()

	enrollBody, _ := json.Marshal(map[string]string{
		"install_id": "e2e-offline-device",
		"platform":   "android",
		"push_token": "fcm-token-offline",
	})
	resp, err := client.Post(server.URL+"/v1/devices", "application/json", bytes.NewReader(enrollBody))
	if err != nil {
		t.Fatalf("enroll: %v", err)
	}
	_ = resp.Body.Close()

	req, _ := http.NewRequest(http.MethodPost, server.URL+"/v1/rounds", nil)
	req.Header.Set("Authorization", "Bearer test-operator-token")
	triggerResp, err := client.Do(req)
	if err != nil {
		t.Fatalf("trigger round: %v", err)
	}
	_ = triggerResp.Body.Close()

	// No receipt is ever reported for this device — check it still shows as pending.
	deliveriesResp, err := client.Get(server.URL + "/v1/reports/deliveries")
	if err != nil {
		t.Fatalf("list deliveries: %v", err)
	}
	defer func() { _ = deliveriesResp.Body.Close() }()
	var deliveries []map[string]any
	if err := json.NewDecoder(deliveriesResp.Body).Decode(&deliveries); err != nil {
		t.Fatalf("decode deliveries: %v", err)
	}
	if len(deliveries) != 1 || deliveries[0]["status"] != "pending" {
		t.Fatalf("deliveries = %+v, want one pending row (never silently absent)", deliveries)
	}
}
