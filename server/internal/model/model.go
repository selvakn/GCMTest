// Package model holds the domain types shared across the coordinator: Device,
// Round, and Delivery, per specs/001-push-notification-latency/data-model.md.
package model

import "time"

// Device is one enrolled fleet member, identified by its app-generated
// InstallID (stable across push-token rotation; Clarification Q1).
type Device struct {
	ID        int64
	InstallID string
	Platform  string
	PushToken string
	CreatedAt time.Time
	UpdatedAt time.Time
}

// Round is one "send now to everyone" instance.
type Round struct {
	ID                  int64
	PublicID            string
	SentAt              time.Time
	TargetedDeviceCount int
}

// DeliveryStatus is the lifecycle state of one Device's participation in one Round.
type DeliveryStatus string

const (
	// StatusPending means no receipt has been reported yet; this may be permanent.
	StatusPending DeliveryStatus = "pending"
	// StatusReceivedGood means a receipt was reported with a positive latency.
	StatusReceivedGood DeliveryStatus = "received_good"
	// StatusReceivedAnomalous means a receipt was reported with a non-positive
	// latency (clock-skew or measurement artifact) — never treated as "good".
	StatusReceivedAnomalous DeliveryStatus = "received_anomalous"
)

// Delivery is the record of one device's participation in one round.
type Delivery struct {
	RoundID    int64
	DeviceID   int64
	SentAt     time.Time
	ReceivedAt *time.Time
	LatencyMS  *int64
	Status     DeliveryStatus
}
