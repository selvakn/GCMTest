// Package push defines a platform-agnostic send boundary (FR-016) so
// additional platforms (iOS/APNs, Windows/WNS, ...) can be added later
// without changing round-triggering or reporting logic.
package push

import (
	"context"

	"github.com/selvakn/gcmtest/server/internal/model"
)

// Sender delivers one round's notification to one device on a best-effort,
// immediate basis (FR-005): unreachable devices must not be queued for later
// delivery. Implementations are looked up by model.Device.Platform.
type Sender interface {
	// Send dispatches round to device "now or never". The full Round (not
	// just its id) is passed so the payload can carry SentAt — the client
	// needs the authoritative send time to display latency (FR-024) without
	// an extra round trip. A returned error only indicates a send-time
	// failure (e.g., provider rejected the request); per Clarification Q3
	// this is not distinguished from ordinary non-receipt in the stored
	// Delivery outcome — callers should log it and move on.
	Send(ctx context.Context, device model.Device, round model.Round) error
}

// Registry dispatches to the right Sender for a device's platform.
type Registry map[string]Sender

// Send looks up the sender for device.Platform and delegates to it.
func (r Registry) Send(ctx context.Context, device model.Device, round model.Round) error {
	sender, ok := r[device.Platform]
	if !ok {
		return ErrUnsupportedPlatform{Platform: device.Platform}
	}
	return sender.Send(ctx, device, round)
}

// ErrUnsupportedPlatform is returned when no Sender is registered for a device's platform.
type ErrUnsupportedPlatform struct {
	Platform string
}

func (e ErrUnsupportedPlatform) Error() string {
	return "push: unsupported platform " + e.Platform
}
