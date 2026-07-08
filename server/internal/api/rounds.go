package api

import (
	"context"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/selvakn/gcmtest/server/internal/model"
	"github.com/selvakn/gcmtest/server/internal/push"
	"github.com/selvakn/gcmtest/server/internal/store"
)

// RoundsHandler implements POST /v1/rounds (FR-003 through FR-008).
type RoundsHandler struct {
	Rounds *store.RoundStore
	Sender push.Sender
}

// TriggerRound creates a round, assigns it a unique id and send time before
// any device could plausibly respond (FR-006), fans the push out to the
// currently enrolled fleet, and returns a confirmation that dispatch
// happened — independent of any individual device's eventual outcome (FR-008).
func (h *RoundsHandler) TriggerRound(w http.ResponseWriter, r *http.Request) {
	publicID := uuid.NewString()
	sentAt := time.Now().UTC()

	round, devices, err := h.Rounds.CreateRound(r.Context(), publicID, sentAt)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create round")
		return
	}

	dispatch(r.Context(), h.Sender, devices, round)

	writeJSON(w, http.StatusAccepted, map[string]any{
		"round_id":              round.PublicID,
		"sent_at":               round.SentAt,
		"targeted_device_count": round.TargetedDeviceCount,
	})
}

// dispatch sends round to every device concurrently. A per-device send error
// is logged, not surfaced to the caller: per Clarification Q3, a
// provider-level send failure is recorded the same way as ordinary
// non-receipt — there is no distinct failure status.
func dispatch(ctx context.Context, sender push.Sender, devices []model.Device, round model.Round) {
	var wg sync.WaitGroup
	for _, device := range devices {
		wg.Add(1)
		go func(d model.Device) {
			defer wg.Done()
			if err := sender.Send(ctx, d, round); err != nil {
				log.Printf("push send failed for device %d: %v", d.ID, err)
			}
		}(device)
	}
	wg.Wait()
}
