package api

import (
	"net/http"
	"strconv"
	"time"

	"github.com/selvakn/gcmtest/server/internal/store"
)

// ReportsHandler implements GET /v1/reports/deliveries and
// GET /v1/reports/success-rate (FR-013, FR-014, FR-015).
type ReportsHandler struct {
	Deliveries *store.DeliveryStore
	Reports    *store.ReportStore
}

// ListDeliveries answers GET /v1/reports/deliveries: individual delivery
// latencies, filterable/orderable/limitable — the way an operator finds the
// slowest deliveries (FR-013).
func (h *ReportsHandler) ListDeliveries(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	opts := store.DeliveryQueryOptions{
		RoundPublicID: q.Get("round_id"),
		Order:         q.Get("order"),
	}
	if limitStr := q.Get("limit"); limitStr != "" {
		limit, err := strconv.Atoi(limitStr)
		if err != nil || limit <= 0 {
			writeError(w, http.StatusBadRequest, "limit must be a positive integer")
			return
		}
		opts.Limit = limit
	}

	records, err := h.Deliveries.Query(r.Context(), opts)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to query deliveries")
		return
	}

	type deliveryResponse struct {
		RoundID    string     `json:"round_id"`
		DeviceID   string     `json:"device_id"`
		SentAt     time.Time  `json:"sent_at"`
		ReceivedAt *time.Time `json:"received_at,omitempty"`
		LatencyMS  *int64     `json:"latency_ms,omitempty"`
		Status     string     `json:"status"`
	}

	resp := make([]deliveryResponse, 0, len(records))
	for _, rec := range records {
		resp = append(resp, deliveryResponse{
			RoundID:    rec.RoundPublicID,
			DeviceID:   rec.DeviceInstallID,
			SentAt:     rec.SentAt,
			ReceivedAt: rec.ReceivedAt,
			LatencyMS:  rec.LatencyMS,
			Status:     string(rec.Status),
		})
	}
	writeJSON(w, http.StatusOK, resp)
}

// SuccessRateReport answers GET /v1/reports/success-rate: delivery success
// rate sliced by when rounds were sent (FR-014).
func (h *ReportsHandler) SuccessRateReport(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	bucket := q.Get("bucket")
	if bucket == "" {
		bucket = "hour"
	}
	if bucket != "hour" && bucket != "day" {
		writeError(w, http.StatusBadRequest, "bucket must be 'hour' or 'day'")
		return
	}

	since, err := parseOptionalTime(q.Get("since"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "since must be RFC3339")
		return
	}
	until, err := parseOptionalTime(q.Get("until"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "until must be RFC3339")
		return
	}

	buckets, err := h.Reports.SuccessRateByBucket(r.Context(), bucket, since, until)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to compute success rate")
		return
	}

	type bucketResponse struct {
		BucketStart time.Time `json:"bucket_start"`
		Targeted    int       `json:"targeted"`
		Received    int       `json:"received"`
		SuccessRate float64   `json:"success_rate"`
	}

	resp := make([]bucketResponse, 0, len(buckets))
	for _, b := range buckets {
		resp = append(resp, bucketResponse{
			BucketStart: b.BucketStart,
			Targeted:    b.Targeted,
			Received:    b.Received,
			SuccessRate: b.SuccessRate,
		})
	}
	writeJSON(w, http.StatusOK, resp)
}

func parseOptionalTime(raw string) (*time.Time, error) {
	if raw == "" {
		return nil, nil
	}
	t, err := time.Parse(time.RFC3339, raw)
	if err != nil {
		return nil, err
	}
	return &t, nil
}
