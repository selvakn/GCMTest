package api

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/selvakn/gcmtest/server/internal/store"
)

// ReceiptsHandler implements POST /v1/rounds/{round_id}/receipts (FR-009 through FR-012).
type ReceiptsHandler struct {
	Deliveries *store.DeliveryStore
}

type reportReceiptRequest struct {
	InstallID  string `json:"install_id"`
	ReceivedAt string `json:"received_at"`
}

// ReportReceipt records that a device received a round. Unknown rounds,
// unknown devices, and duplicate reports are all handled gracefully — this
// endpoint always answers 200 with a status field describing what happened,
// rather than erroring on data it cannot fully trust (FR-011, FR-012).
func (h *ReceiptsHandler) ReportReceipt(w http.ResponseWriter, r *http.Request) {
	roundPublicID := r.PathValue("round_id")

	var req reportReceiptRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "malformed request body")
		return
	}
	if req.InstallID == "" || req.ReceivedAt == "" {
		writeError(w, http.StatusBadRequest, "install_id and received_at are required")
		return
	}
	receivedAt, err := time.Parse(time.RFC3339Nano, req.ReceivedAt)
	if err != nil {
		writeError(w, http.StatusBadRequest, "received_at must be RFC3339")
		return
	}

	result, err := h.Deliveries.RecordReceipt(r.Context(), roundPublicID, req.InstallID, receivedAt)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to record receipt")
		return
	}

	status := "invalid"
	switch result {
	case store.ReceiptRecorded:
		status = "recorded"
	case store.ReceiptDuplicateIgnored:
		status = "duplicate_ignored"
	case store.ReceiptInvalid:
		status = "invalid"
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": status})
}
