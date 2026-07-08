package push

import (
	"context"
	"errors"
	"testing"
	"time"

	"firebase.google.com/go/v4/messaging"

	"github.com/selvakn/gcmtest/server/internal/model"
)

type fakeFCMClient struct {
	lastMessage *messaging.Message
}

func (f *fakeFCMClient) Send(ctx context.Context, message *messaging.Message) (string, error) {
	f.lastMessage = message
	return "fake-message-id", nil
}

func TestFCMSender_SendsWithTTLZeroAndHighPriority(t *testing.T) {
	fake := &fakeFCMClient{}
	sender := &FCMSender{client: fake}

	device := model.Device{PushToken: "device-token-123"}
	round := model.Round{PublicID: "round-abc", SentAt: time.Date(2026, 7, 8, 12, 0, 0, 0, time.UTC)}
	if err := sender.Send(context.Background(), device, round); err != nil {
		t.Fatalf("Send: %v", err)
	}

	if fake.lastMessage == nil {
		t.Fatal("expected a message to have been sent")
	}
	if fake.lastMessage.Token != "device-token-123" {
		t.Errorf("Token = %q, want device-token-123", fake.lastMessage.Token)
	}
	if fake.lastMessage.Data["round_id"] != "round-abc" {
		t.Errorf("Data[round_id] = %q, want round-abc", fake.lastMessage.Data["round_id"])
	}
	if fake.lastMessage.Data["sent_at"] != round.SentAt.Format(time.RFC3339Nano) {
		t.Errorf("Data[sent_at] = %q, want %q", fake.lastMessage.Data["sent_at"], round.SentAt.Format(time.RFC3339Nano))
	}
	if fake.lastMessage.Android == nil {
		t.Fatal("expected AndroidConfig to be set")
	}
	if fake.lastMessage.Android.Priority != "high" {
		t.Errorf("Priority = %q, want high", fake.lastMessage.Android.Priority)
	}
	if fake.lastMessage.Android.TTL == nil || *fake.lastMessage.Android.TTL != 0 {
		t.Errorf("TTL = %v, want 0 (fresh or dropped, FR-005)", fake.lastMessage.Android.TTL)
	}
}

func TestFCMSender_PropagatesSendError(t *testing.T) {
	fake := &erroringFCMClient{}
	sender := &FCMSender{client: fake}

	err := sender.Send(context.Background(), model.Device{PushToken: "t"}, model.Round{PublicID: "round-x"})
	if err == nil {
		t.Fatal("expected an error to be propagated")
	}
}

type erroringFCMClient struct{}

func (erroringFCMClient) Send(ctx context.Context, message *messaging.Message) (string, error) {
	return "", errBoom
}

var errBoom = errors.New("boom")
