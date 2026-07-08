// Package api implements the coordinator's HTTP handlers and routing.
package api

import "net/http"

// Handlers bundles everything the router needs to build request handlers.
// Fields are added incrementally as each endpoint is implemented.
type Handlers struct {
	OperatorToken string
	Devices       *DevicesHandler
	Rounds        *RoundsHandler
	Receipts      *ReceiptsHandler
	Reports       *ReportsHandler
}

// NewRouter builds the coordinator's HTTP routing table.
func NewRouter(h Handlers) *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})

	if h.Devices != nil {
		mux.HandleFunc("POST /v1/devices", h.Devices.EnrollDevice)
	}
	if h.Rounds != nil {
		mux.HandleFunc("POST /v1/rounds", RequireOperatorToken(h.OperatorToken, h.Rounds.TriggerRound))
	}
	if h.Receipts != nil {
		mux.HandleFunc("POST /v1/rounds/{round_id}/receipts", h.Receipts.ReportReceipt)
	}
	if h.Reports != nil {
		mux.HandleFunc("GET /v1/reports/deliveries", h.Reports.ListDeliveries)
		mux.HandleFunc("GET /v1/reports/success-rate", h.Reports.SuccessRateReport)
	}

	return mux
}
