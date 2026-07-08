package api

import (
	"encoding/json"
	"net/http"

	"github.com/selvakn/gcmtest/server/internal/store"
)

// DevicesHandler implements POST /v1/devices — fleet enrollment. Unlike
// round-triggering, this endpoint is intentionally open (no auth): any
// device running the client app must be able to join automatically
// (FR-001, spec.md §8).
type DevicesHandler struct {
	Devices *store.DeviceStore
}

type enrollDeviceRequest struct {
	InstallID string `json:"install_id"`
	Platform  string `json:"platform"`
	PushToken string `json:"push_token"`
}

// EnrollDevice enrolls a new device, or updates an existing one's platform/
// push_token in place if its install_id is already known (FR-002).
func (h *DevicesHandler) EnrollDevice(w http.ResponseWriter, r *http.Request) {
	var req enrollDeviceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "malformed request body")
		return
	}
	if req.InstallID == "" || req.Platform == "" || req.PushToken == "" {
		writeError(w, http.StatusBadRequest, "install_id, platform, and push_token are required")
		return
	}

	result, err := h.Devices.Upsert(r.Context(), req.InstallID, req.Platform, req.PushToken)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to enroll device")
		return
	}

	status := "enrolled"
	if result == store.DeviceUpdated {
		status = "updated"
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": status})
}
