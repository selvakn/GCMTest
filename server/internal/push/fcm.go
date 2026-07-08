package push

import (
	"context"
	"time"

	firebase "firebase.google.com/go/v4"
	"firebase.google.com/go/v4/messaging"
	"google.golang.org/api/option"

	"github.com/selvakn/gcmtest/server/internal/model"
)

// fcmMessageSender is the subset of *messaging.Client this package depends
// on, so tests can inject a fake transport instead of calling real FCM.
type fcmMessageSender interface {
	Send(ctx context.Context, message *messaging.Message) (string, error)
}

// FCMSender sends round notifications to Android devices via Firebase Cloud
// Messaging. It implements Sender.
type FCMSender struct {
	client fcmMessageSender
}

// NewFCMSender constructs an FCMSender authenticated with the service-account
// credentials at serviceAccountJSONPath (FR-030 — never a hardcoded secret).
func NewFCMSender(ctx context.Context, serviceAccountJSONPath string) (*FCMSender, error) {
	app, err := firebase.NewApp(ctx, nil, option.WithAuthCredentialsFile(option.ServiceAccount, serviceAccountJSONPath))
	if err != nil {
		return nil, err
	}
	client, err := app.Messaging(ctx)
	if err != nil {
		return nil, err
	}
	return &FCMSender{client: client}, nil
}

// Send dispatches round to device as a high-priority data message with
// TTL=0, so FCM drops it immediately rather than queuing it for a
// currently-unreachable device (FR-005 — "fresh or dropped"). The payload
// carries both the round id and its authoritative sent_at so the client can
// compute and display latency without an extra round trip.
func (s *FCMSender) Send(ctx context.Context, device model.Device, round model.Round) error {
	ttl := time.Duration(0)
	msg := &messaging.Message{
		Token: device.PushToken,
		Data: map[string]string{
			"round_id": round.PublicID,
			"sent_at":  round.SentAt.Format(time.RFC3339Nano),
		},
		Android: &messaging.AndroidConfig{
			Priority: "high",
			TTL:      &ttl,
		},
	}
	_, err := s.client.Send(ctx, msg)
	return err
}
